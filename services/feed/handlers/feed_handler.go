package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type FeedHandler struct {
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewFeedHandler(redisClient *redis.Client, logger *logger.Logger) *FeedHandler {
	return &FeedHandler{
		redisClient: redisClient,
		logger:      logger,
	}
}

func (h *FeedHandler) GetFeed(c *gin.Context) {
	userID := c.GetString("user_id")
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	ctx := context.Background()
	feedKey := fmt.Sprintf("feed:%s", userID)

	// Get post IDs from cache
	postIDs, err := h.redisClient.LRange(ctx, feedKey, 0, int64(limit-1)).Result()
	if err != nil && err != redis.Nil {
		h.logger.Error("Failed to get feed from cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	// If feed is empty, return empty list
	if len(postIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"posts": []string{}, "count": 0})
		return
	}

	// Get post details from cache
	var posts []map[string]interface{}
	for _, postID := range postIDs {
		postKey := fmt.Sprintf("post:%s", postID)
		postData, err := h.redisClient.HGetAll(ctx, postKey).Result()
		if err == nil && len(postData) > 0 {
			posts = append(posts, map[string]interface{}{
				"id":   postData["id"],
				"title": postData["title"],
				"media_url": postData["media_url"],
				"creator_id": postData["creator_id"],
				"price": postData["price"],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts)})
}

func (h *FeedHandler) GetFeedByCategory(c *gin.Context) {
	userID := c.GetString("user_id")
	category := c.Param("category")
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	ctx := context.Background()
	feedKey := fmt.Sprintf("feed:%s:%s", userID, category)

	// Get post IDs from cache filtered by category
	postIDs, err := h.redisClient.LRange(ctx, feedKey, 0, int64(limit-1)).Result()
	if err != nil && err != redis.Nil {
		h.logger.Error("Failed to get feed from cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	// Get post details from cache
	var posts []map[string]interface{}
	for _, postID := range postIDs {
		postKey := fmt.Sprintf("post:%s", postID)
		postData, err := h.redisClient.HGetAll(ctx, postKey).Result()
		if err == nil && len(postData) > 0 {
			posts = append(posts, map[string]interface{}{
				"id":   postData["id"],
				"title": postData["title"],
				"media_url": postData["media_url"],
				"creator_id": postData["creator_id"],
				"price": postData["price"],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "category": category})
}

