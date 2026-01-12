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

// GetFeed godoc
// @Summary      Get global feed (like TikTok)
// @Description  Get global feed with all posts - all viewers see all posts
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Router       /feed [get]
func (h *FeedHandler) GetFeed(c *gin.Context) {
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := context.Background()
	feedKey := "feed:global" // Global feed - all posts

	// Get post IDs from global feed cache
	end := int64(offset + limit - 1)
	postIDs, err := h.redisClient.LRange(ctx, feedKey, int64(offset), end).Result()
	if err != nil && err != redis.Nil {
		h.logger.Error("Failed to get feed from cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	// If feed is empty, return empty list
	if len(postIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"posts": []interface{}{}, "count": 0, "offset": offset})
		return
	}

	// Get post details from cache
	var posts []map[string]interface{}
	for _, postID := range postIDs {
		postKey := fmt.Sprintf("post:%s", postID)
		postData, err := h.redisClient.HGetAll(ctx, postKey).Result()
		if err == nil && len(postData) > 0 {
			posts = append(posts, map[string]interface{}{
				"id":         postData["id"],
				"title":      postData["title"],
				"media_url":  postData["media_url"],
				"creator_id": postData["creator_id"],
				"price":      postData["price"],
				"category":   postData["category"],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "offset": offset})
}

// GetFeedByCategory godoc
// @Summary      Get feed by category
// @Description  Get global feed filtered by category
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        category path string true "Category name"
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Router       /feed/category/{category} [get]
func (h *FeedHandler) GetFeedByCategory(c *gin.Context) {
	category := c.Param("category")
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := context.Background()
	feedKey := fmt.Sprintf("feed:global:%s", category) // Global category feed

	// Get post IDs from global category feed cache
	end := int64(offset + limit - 1)
	postIDs, err := h.redisClient.LRange(ctx, feedKey, int64(offset), end).Result()
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
				"id":         postData["id"],
				"title":      postData["title"],
				"media_url":  postData["media_url"],
				"creator_id": postData["creator_id"],
				"price":      postData["price"],
				"category":   postData["category"],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "category": category, "offset": offset})
}

