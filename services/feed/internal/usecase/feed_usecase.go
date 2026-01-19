package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"lick-scroll/pkg/config"
	"lick-scroll/pkg/logger"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FeedUseCase interface {
	GetFeed(userID string, limit, offset int, authToken string) ([]map[string]interface{}, int, error)
	GetFeedByCategory(userID, category string, limit, offset int) ([]map[string]interface{}, error)
}

type feedUseCase struct {
	db             *gorm.DB
	redisClient    *redis.Client
	logger         *logger.Logger
	authServiceURL string
	config         *config.Config
}

func NewFeedUseCase(db *gorm.DB, redisClient *redis.Client, logger *logger.Logger, authServiceURL string, cfg *config.Config) FeedUseCase {
	return &feedUseCase{
		db:             db,
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
		query := uc.db.Table("posts").
			Select("posts.*, post_images.id as image_id, post_images.image_url, post_images.thumbnail_url, post_images.\"order\" as image_order").
			Joins("LEFT JOIN post_images ON posts.id = post_images.post_id").
			Where("posts.creator_id IN ? AND posts.deleted_at IS NULL", subscriptions).
			Order("posts.created_at DESC").
			Limit(limit * 2)

		rows, err := query.Rows()
		if err == nil {
			defer rows.Close()
			subscribedPosts = uc.scanPostsFromRows(rows)
		} else {
			uc.logger.Error("Failed to get posts from subscribed creators: %v", err)
		}
	}

	var otherPosts []map[string]interface{}
	query := uc.db.Table("posts").
		Select("posts.*, post_images.id as image_id, post_images.image_url, post_images.thumbnail_url, post_images.\"order\" as image_order").
		Joins("LEFT JOIN post_images ON posts.id = post_images.post_id").
		Where("posts.creator_id != ? AND posts.deleted_at IS NULL", userID).
		Order("posts.created_at DESC").
		Limit(limit * 2)

	if len(subscriptions) > 0 {
		query = query.Where("posts.creator_id NOT IN ?", subscriptions)
	}

	rows, err := query.Rows()
	if err == nil {
		defer rows.Close()
		otherPosts = uc.scanPostsFromRows(rows)
	} else {
		uc.logger.Error("Failed to get other posts: %v", err)
		otherPosts = []map[string]interface{}{}
	}

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
		postID := post["id"].(string)

		isLiked := false
		var likeCount int64
		if userID != "" {
			var count int64
			if err := uc.db.Table("likes").Where("user_id = ? AND post_id = ? AND deleted_at IS NULL", userID, postID).Count(&count).Error; err == nil {
				isLiked = count > 0
			}
			uc.db.Table("likes").Where("post_id = ? AND deleted_at IS NULL", postID).Count(&likeCount)
		}

		var creator struct {
			AvatarURL string
			Username  string
		}
		uc.db.Table("users").Select("avatar_url, username").Where("id = ?", post["creator_id"]).Scan(&creator)

		images := uc.formatPostImages(post)

		postItem := map[string]interface{}{
			"id":               post["id"],
			"title":            post["title"],
			"description":      post["description"],
			"type":             post["type"],
			"creator_id":       post["creator_id"],
			"creator_avatar":   creator.AvatarURL,
			"creator_username": creator.Username,
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

func (uc *feedUseCase) scanPostsFromRows(rows *sql.Rows) []map[string]interface{} {
	postMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var postID, creatorID, title, description, postType, mediaURL, thumbnailURL, category, status sql.NullString
		var views, purchases sql.NullInt32
		var createdAt, updatedAt sql.NullTime
		var imageID, imageURL, imageThumbnailURL sql.NullString
		var imageOrder sql.NullInt32

		if err := rows.Scan(&postID, &creatorID, &title, &description, &postType, &mediaURL, &thumbnailURL, &category, &status, &views, &purchases, &createdAt, &updatedAt, &imageID, &imageURL, &imageThumbnailURL, &imageOrder); err != nil {
			continue
		}

		if _, exists := postMap[postID.String]; !exists {
			postMap[postID.String] = map[string]interface{}{
				"id":           postID.String,
				"creator_id":   creatorID.String,
				"title":        title.String,
				"description":  description.String,
				"type":         postType.String,
				"media_url":    mediaURL.String,
				"thumbnail_url": thumbnailURL.String,
				"category":     category.String,
				"status":       status.String,
				"views":        int(views.Int32),
				"purchases":    int(purchases.Int32),
				"created_at":   createdAt.Time,
				"updated_at":   updatedAt.Time,
				"images":       []map[string]interface{}{},
			}
		}

		if imageID.Valid {
			images := postMap[postID.String]["images"].([]map[string]interface{})
			images = append(images, map[string]interface{}{
				"id":           imageID.String,
				"post_id":      postID.String,
				"image_url":    imageURL.String,
				"thumbnail_url": imageThumbnailURL.String,
				"order":        int(imageOrder.Int32),
			})
			postMap[postID.String]["images"] = images
		}
	}

	results := make([]map[string]interface{}, 0, len(postMap))
	for _, post := range postMap {
		results = append(results, post)
	}

	return results
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
