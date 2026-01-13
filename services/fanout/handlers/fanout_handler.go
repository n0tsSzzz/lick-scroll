package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/pkg/queue"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FanoutHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	queueClient *queue.Client
	logger      *logger.Logger
}

func NewFanoutHandler(db *gorm.DB, redisClient *redis.Client, queueClient *queue.Client, logger *logger.Logger) *FanoutHandler {
	return &FanoutHandler{
		db:          db,
		redisClient: redisClient,
		queueClient: queueClient,
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

	// Send notifications to all subscribers (free subscriptions get notifications)
	// Create notification tasks and publish to RabbitMQ priority queue
	notificationTasks := 0
	for _, sub := range subscriptions {
		// Free subscribers get notifications about new posts
		if sub.Type == models.SubscriptionTypeFree {
			task := map[string]interface{}{
				"user_id":    sub.ViewerID,
				"post_id":    req.PostID,
				"creator_id": req.CreatorID,
				"type":       "new_post",
				"priority":   1, // Normal priority
			}
			if err := h.queueClient.PublishNotificationTask(task); err != nil {
				h.logger.Error("Failed to publish notification task: %v", err)
				// Continue processing other tasks even if one fails
			} else {
				notificationTasks++
			}
		}
	}

	if notificationTasks > 0 {
		h.logger.Info("Published %d notification tasks to RabbitMQ queue", notificationTasks)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Post fanned out successfully",
		"subscribers":        len(subscriptions),
		"notifications_sent": notificationTasks,
	})
}

type SubscribeRequest struct {
	Type string `json:"type" binding:"omitempty,oneof=free paid"` // Default: free
}

func (h *FanoutHandler) Subscribe(c *gin.Context) {
	viewerID := c.GetString("user_id")
	creatorID := c.Param("creator_id")

	if viewerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		// If no body, default to free
		req.Type = "free"
	} else if req.Type == "" {
		req.Type = "free"
	}

	// Check if already subscribed
	var existing models.Subscription
	if err := h.db.Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).First(&existing).Error; err == nil {
		// Update subscription type if different
		if existing.Type != models.SubscriptionType(req.Type) {
			existing.Type = models.SubscriptionType(req.Type)
			if err := h.db.Save(&existing).Error; err != nil {
				h.logger.Error("Failed to update subscription: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Subscription updated", "type": req.Type})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "Already subscribed"})
		return
	}

	subscription := &models.Subscription{
		ViewerID:  viewerID,
		CreatorID: creatorID,
		Type:      models.SubscriptionType(req.Type),
	}

	if err := h.db.Create(subscription).Error; err != nil {
		h.logger.Error("Failed to create subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Subscribed successfully", "type": req.Type})
}

func (h *FanoutHandler) Unsubscribe(c *gin.Context) {
	viewerID := c.GetString("user_id")
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

// GetSubscriptionStatus godoc
// @Summary      Get subscription status
// @Description  Get the subscription status for a specific creator by the authenticated user.
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        creator_id path string true "Creator ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Router       /subscribe/{creator_id} [get]
func (h *FanoutHandler) GetSubscriptionStatus(c *gin.Context) {
	viewerID := c.GetString("user_id")
	creatorID := c.Param("creator_id")

	if viewerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	var subscription models.Subscription
	err := h.db.Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).First(&subscription).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"subscribed": false, "type": ""})
			return
		}
		h.logger.Error("Failed to get subscription status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscribed": true, "type": subscription.Type})
}

