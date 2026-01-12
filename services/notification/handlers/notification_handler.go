package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lick-scroll/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type NotificationHandler struct {
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewNotificationHandler(redisClient *redis.Client, logger *logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		redisClient: redisClient,
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

