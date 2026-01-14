package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type NotificationHandler struct {
	redisClient *redis.Client
	queueClient *queue.Client
	logger      *logger.Logger
}

func NewNotificationHandler(redisClient *redis.Client, queueClient *queue.Client, logger *logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		redisClient: redisClient,
		queueClient: queueClient,
		logger:      logger,
	}
}

type Notification struct {
	UserID  string                 `json:"user_id"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type SendNotificationRequest struct {
	UserID  string                 `json:"user_id" binding:"required"`
	Title   string                 `json:"title" binding:"required"`
	Message string                 `json:"message" binding:"required"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type BroadcastNotificationRequest struct {
	UserIDs []string               `json:"user_ids" binding:"required"`
	Title   string                 `json:"title" binding:"required"`
	Message string                 `json:"message" binding:"required"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification := Notification{
		UserID:  req.UserID,
		Title:   req.Title,
		Message: req.Message,
		Type:    req.Type,
		Data:    req.Data,
	}

	// Store notification in Redis
	ctx := context.Background()
	notificationJSON, _ := json.Marshal(notification)
	
	// Add to user's notification list
	userNotificationsKey := fmt.Sprintf("notifications:%s", req.UserID)
	h.redisClient.LPush(ctx, userNotificationsKey, notificationJSON)
	h.redisClient.LTrim(ctx, userNotificationsKey, 0, 99) // Keep last 100 notifications
	h.redisClient.Expire(ctx, userNotificationsKey, 30*24*time.Hour)

	// Publish to Redis pub/sub for real-time notifications
	h.redisClient.Publish(ctx, fmt.Sprintf("notifications:%s", req.UserID), notificationJSON)

	h.logger.Info("Notification sent to user %s: %s", req.UserID, req.Title)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Notification sent successfully",
		"notification": notification,
	})
}

func (h *NotificationHandler) BroadcastNotification(c *gin.Context) {
	var req BroadcastNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	sentCount := 0

	for _, userID := range req.UserIDs {
		notification := Notification{
			UserID:  userID,
			Title:   req.Title,
			Message: req.Message,
			Type:    req.Type,
			Data:    req.Data,
		}

		notificationJSON, _ := json.Marshal(notification)
		
		// Add to user's notification list
		userNotificationsKey := fmt.Sprintf("notifications:%s", userID)
		h.redisClient.LPush(ctx, userNotificationsKey, notificationJSON)
		h.redisClient.LTrim(ctx, userNotificationsKey, 0, 99)
		h.redisClient.Expire(ctx, userNotificationsKey, 30*24*time.Hour)

		// Publish to Redis pub/sub
		h.redisClient.Publish(ctx, fmt.Sprintf("notifications:%s", userID), notificationJSON)
		sentCount++
	}

	h.logger.Info("Broadcast notification sent to %d users: %s", sentCount, req.Title)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Notifications sent successfully",
		"sent_count": sentCount,
	})
}

// ProcessNotificationQueue starts consuming notifications from RabbitMQ queue
// This endpoint starts a background consumer that processes notifications
func (h *NotificationHandler) ProcessNotificationQueue(c *gin.Context) {
	ctx := context.Background()

	// Start consuming from RabbitMQ queue
	err := h.queueClient.ConsumeNotificationTasks(func(task map[string]interface{}) error {
		userID, _ := task["user_id"].(string)
		postID, _ := task["post_id"].(string)
		creatorID, _ := task["creator_id"].(string)

		// Create notification
		notification := Notification{
			UserID:  userID,
			Title:   "New Post Alert!",
			Message: fmt.Sprintf("Creator %s just posted new content!", creatorID),
			Type:    "new_post",
			Data: map[string]interface{}{
				"post_id":    postID,
				"creator_id": creatorID,
			},
		}

		// Send notification (store in Redis and publish via pub/sub)
		notificationJSON, _ := json.Marshal(notification)
		userNotificationsKey := fmt.Sprintf("notifications:%s", userID)
		h.redisClient.LPush(ctx, userNotificationsKey, notificationJSON)
		h.redisClient.LTrim(ctx, userNotificationsKey, 0, 99)
		h.redisClient.Expire(ctx, userNotificationsKey, 30*24*time.Hour)

		// Publish to Redis pub/sub for real-time notifications
		h.redisClient.Publish(ctx, fmt.Sprintf("notifications:%s", userID), notificationJSON)

		h.logger.Info("Processed notification for user %s: post %s", userID, postID)
		return nil
	})

	if err != nil {
		h.logger.Error("Failed to start consuming from queue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start queue consumer"})
		return
	}

	// Get queue length
	queueLength, _ := h.queueClient.GetQueueLength()

	c.JSON(http.StatusOK, gin.H{
		"message":      "Notification queue consumer started",
		"queue_length": queueLength,
	})
}

// GetNotifications godoc
// @Summary      Get user notifications
// @Description  Get all notifications for the authenticated user
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of notifications to return (max 100)"
// @Success      200  {object}  map[string]interface{}
// @Router       /notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	ctx := context.Background()
	userNotificationsKey := fmt.Sprintf("notifications:%s", userID)

	// Get notifications from Redis list
	notificationsJSON, err := h.redisClient.LRange(ctx, userNotificationsKey, 0, int64(limit-1)).Result()
	if err != nil && err != redis.Nil {
		h.logger.Error("Failed to get notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	var notifications []Notification
	for _, notifJSON := range notificationsJSON {
		var notification Notification
		if err := json.Unmarshal([]byte(notifJSON), &notification); err == nil {
			notifications = append(notifications, notification)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

