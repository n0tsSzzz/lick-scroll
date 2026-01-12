package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FanoutHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewFanoutHandler(db *gorm.DB, redisClient *redis.Client, logger *logger.Logger) *FanoutHandler {
	return &FanoutHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

type FanoutPostRequest struct {
	PostID     string `json:"post_id" binding:"required"`
	CreatorID  string `json:"creator_id" binding:"required"`
	Category   string `json:"category"`
}

func (h *FanoutHandler) FanoutPost(c *gin.Context) {
	var req FanoutPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get all subscribers of the creator
	var subscriptions []models.Subscription
	if err := h.db.Where("creator_id = ?", req.CreatorID).Find(&subscriptions).Error; err != nil {
		h.logger.Error("Failed to get subscriptions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscriptions"})
		return
	}

	ctx := context.Background()

	// Add post to each subscriber's feed
	for _, sub := range subscriptions {
		feedKey := fmt.Sprintf("feed:%s", sub.ViewerID)
		
		// Add to general feed
		h.redisClient.LPush(ctx, feedKey, req.PostID)
		h.redisClient.LTrim(ctx, feedKey, 0, 999) // Keep last 1000 posts
		h.redisClient.Expire(ctx, feedKey, 7*24*time.Hour)

		// Add to category feed if category is specified
		if req.Category != "" {
			categoryFeedKey := fmt.Sprintf("feed:%s:%s", sub.ViewerID, req.Category)
			h.redisClient.LPush(ctx, categoryFeedKey, req.PostID)
			h.redisClient.LTrim(ctx, categoryFeedKey, 0, 999)
			h.redisClient.Expire(ctx, categoryFeedKey, 7*24*time.Hour)
		}
	}

	// Post metadata should already be cached by post service
	// We only cache it here if it doesn't exist
	postKey := fmt.Sprintf("post:%s", req.PostID)
	exists, _ := h.redisClient.Exists(ctx, postKey).Result()
	if exists == 0 {
		// Cache basic metadata if post service hasn't cached it yet
		h.redisClient.HSet(ctx, postKey, "id", req.PostID)
		h.redisClient.HSet(ctx, postKey, "creator_id", req.CreatorID)
		if req.Category != "" {
			h.redisClient.HSet(ctx, postKey, "category", req.Category)
		}
		h.redisClient.Expire(ctx, postKey, 24*time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Post fanned out successfully",
		"subscribers": len(subscriptions),
	})
}

func (h *FanoutHandler) Subscribe(c *gin.Context) {
	viewerID := c.GetHeader("X-User-ID")
	creatorID := c.Param("creator_id")

	if viewerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	// Check if already subscribed
	var existing models.Subscription
	if err := h.db.Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Already subscribed"})
		return
	}

	subscription := &models.Subscription{
		ViewerID:  viewerID,
		CreatorID: creatorID,
	}

	if err := h.db.Create(subscription).Error; err != nil {
		h.logger.Error("Failed to create subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Subscribed successfully"})
}

func (h *FanoutHandler) Unsubscribe(c *gin.Context) {
	viewerID := c.GetHeader("X-User-ID")
	creatorID := c.Param("creator_id")

	if viewerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	if err := h.db.Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).Delete(&models.Subscription{}).Error; err != nil {
		h.logger.Error("Failed to delete subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

