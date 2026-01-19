package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"lick-scroll/pkg/config"
	"lick-scroll/pkg/logger"
	"lick-scroll/services/feed/internal/repo/persistent"

	"github.com/redis/go-redis/v9"
)

type FeedUseCase interface {
	GetFeed(userID string, limit, offset int, authToken string) ([]map[string]interface{}, int, error)
	GetFeedByCategory(userID, category string, limit, offset int) ([]map[string]interface{}, error)
}

type feedUseCase struct {
	feedRepo       persistent.FeedRepository
	redisClient    *redis.Client
	logger         *logger.Logger
	authServiceURL string
	config         *config.Config
}

func NewFeedUseCase(feedRepo persistent.FeedRepository, redisClient *redis.Client, logger *logger.Logger, authServiceURL string, cfg *config.Config) FeedUseCase {
	return &feedUseCase{
		feedRepo:       feedRepo,
		redisClient:    redisClient,
		logger:         logger,
		authServiceURL: authServiceURL,
		config:         cfg,
	}
}

func (uc *feedUseCase) GetFeed(userID string, limit, offset int, authToken string) ([]map[string]interface{}, int, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("feed:user:%s", userID)

	if cachedFeed, err := uc.redisClient.Get(ctx, cacheKey).Result(); err == nil {
		var cachedPosts []map[string]interface{}
		if err := json.Unmarshal([]byte(cachedFeed), &cachedPosts); err == nil {
			if offset+limit <= len(cachedPosts) {
				feed := cachedPosts[offset : offset+limit]
				return feed, len(feed), nil
			}
		}
	}

	subscriptions, err := uc.getSubscriptionsFromAuthService(userID, authToken)
	if err != nil {
		uc.logger.Warn("Failed to get subscriptions from Auth Service: %v, continuing with empty subscriptions", err)
		subscriptions = []string{}
	}

	var subscribedPosts []map[string]interface{}
	if len(subscriptions) > 0 {
		posts, err := uc.feedRepo.GetPostsByCreatorIDs(subscriptions, limit*2)
		if err != nil {
			uc.logger.Error("Failed to get posts from subscribed creators: %v", err)
		} else {
			subscribedPosts = posts
		}
	}

	otherPosts, err := uc.feedRepo.GetOtherPosts(userID, subscriptions, limit*2)
	if err != nil {
		uc.logger.Error("Failed to get other posts: %v", err)
		otherPosts = []map[string]interface{}{}
	}

	uc.logger.Info("GetFeed: subscribedPosts=%d, otherPosts=%d", len(subscribedPosts), len(otherPosts))

	allPosts := make([]map[string]interface{}, 0, len(subscribedPosts)+len(otherPosts))
	allPosts = append(allPosts, subscribedPosts...)
	allPosts = append(allPosts, otherPosts...)

	sort.Slice(allPosts, func(i, j int) bool {
		timeI := uc.parseTime(allPosts[i]["created_at"])
		timeJ := uc.parseTime(allPosts[j]["created_at"])
		return timeI.After(timeJ)
	})

	var formattedPosts []map[string]interface{}
	for _, post := range allPosts {
		postIDStr, ok := post["id"].(string)
		if !ok {
			uc.logger.Warn("Post ID is not a string: %v", post["id"])
			continue
		}
		postID := postIDStr

		isLiked := false
		var likeCount int64
		if userID != "" {
			var err error
			isLiked, err = uc.feedRepo.IsLiked(userID, postID)
			if err != nil {
				uc.logger.Warn("Failed to check like status: %v", err)
			}
			likeCount, err = uc.feedRepo.GetLikeCount(postID)
			if err != nil {
				uc.logger.Warn("Failed to get like count: %v", err)
			}
		}

		creatorIDStr, ok := post["creator_id"].(string)
		if !ok {
			uc.logger.Warn("Creator ID is not a string: %v", post["creator_id"])
			creatorIDStr = ""
		}
		creatorInfo, err := uc.feedRepo.GetCreatorInfo(creatorIDStr)
		if err != nil {
			uc.logger.Warn("Failed to get creator info: %v", err)
			creatorInfo = map[string]interface{}{
				"avatar_url": "",
				"username":   "",
			}
		}

		images := uc.formatPostImages(post)

		postItem := map[string]interface{}{
			"id":               post["id"],
			"title":            post["title"],
			"description":      post["description"],
			"type":             post["type"],
			"creator_id":       post["creator_id"],
			"creator_avatar":   creatorInfo["avatar_url"],
			"creator_username": creatorInfo["username"],
			"category":         post["category"],
			"images":           images,
			"likes_count":      likeCount,
			"is_liked":         isLiked,
			"created_at":       post["created_at"],
		}

		if mediaURL, ok := post["media_url"].(string); ok && mediaURL != "" && len(images) == 0 {
			postItem["media_url"] = mediaURL
		}

		formattedPosts = append(formattedPosts, postItem)
	}

	if len(formattedPosts) > 0 {
		feedJSON, _ := json.Marshal(formattedPosts)
		uc.redisClient.Set(ctx, cacheKey, feedJSON, 10*time.Minute)
	}

	start := offset
	end := offset + limit
	if start > len(formattedPosts) {
		formattedPosts = []map[string]interface{}{}
	} else {
		if end > len(formattedPosts) {
			end = len(formattedPosts)
		}
		formattedPosts = formattedPosts[start:end]
	}

	return formattedPosts, len(formattedPosts), nil
}

func (uc *feedUseCase) GetFeedByCategory(userID, category string, limit, offset int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	feedKey := fmt.Sprintf("feed:global:%s", category)

	// Get post IDs from global category feed cache
	end := int64(offset + limit - 1)
	postIDs, err := uc.redisClient.LRange(ctx, feedKey, int64(offset), end).Result()
	if err != nil && err != redis.Nil {
		uc.logger.Error("Failed to get feed from cache: %v", err)
		return nil, fmt.Errorf("failed to fetch feed")
	}

	// Get post details from cache
	var posts []map[string]interface{}
	for _, postID := range postIDs {
		postKey := fmt.Sprintf("post:%s", postID)
		postData, err := uc.redisClient.HGetAll(ctx, postKey).Result()
		if err == nil && len(postData) > 0 {
			// Skip own posts
			if postData["creator_id"] == userID {
				continue
			}

			postItem := map[string]interface{}{
				"id":         postData["id"],
				"title":      postData["title"],
				"creator_id": postData["creator_id"],
				"category":   postData["category"],
				"media_url":  postData["media_url"],
			}

			// Add images if available
			if imagesJSON, ok := postData["images"]; ok && imagesJSON != "" {
				var images []map[string]interface{}
				if err := json.Unmarshal([]byte(imagesJSON), &images); err == nil {
					postItem["images"] = images
				}
			}

			posts = append(posts, postItem)
		}
	}

	return posts, nil
}

func (uc *feedUseCase) getSubscriptionsFromAuthService(userID, authToken string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/subscriptions", uc.authServiceURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth service returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Subscriptions []struct {
			CreatorID string `json:"creator_id"`
		} `json:"subscriptions"`
		Count int `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	creatorIDs := make([]string, len(result.Subscriptions))
	for i, sub := range result.Subscriptions {
		creatorIDs[i] = sub.CreatorID
	}

	return creatorIDs, nil
}

func (uc *feedUseCase) formatPostImages(post map[string]interface{}) []map[string]interface{} {
	images, ok := post["images"].([]map[string]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	formatted := make([]map[string]interface{}, len(images))
	for i, img := range images {
		formatted[i] = map[string]interface{}{
			"id":           img["id"],
			"image_url":    img["image_url"],
			"thumbnail_url": img["thumbnail_url"],
			"order":        img["order"],
		}
	}

	return formatted
}

func (uc *feedUseCase) parseTime(t interface{}) time.Time {
	if t == nil {
		return time.Time{}
	}
	
	switch v := t.(type) {
	case time.Time:
		return v
	case string:
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed
		}
		if parsed, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", v); err == nil {
			return parsed
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}
