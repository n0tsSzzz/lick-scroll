package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/services/notification/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type NotificationHandler struct {
	notificationUseCase usecase.NotificationUseCase
	redisClient         *redis.Client
	logger              *logger.Logger
	jwtService          *jwt.Service
}

func NewNotificationHandler(notificationUseCase usecase.NotificationUseCase, redisClient *redis.Client, logger *logger.Logger, jwtService *jwt.Service) *NotificationHandler {
	return &NotificationHandler{
		notificationUseCase: notificationUseCase,
		redisClient:         redisClient,
		logger:               logger,
		jwtService:           jwtService,
	}
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

	notification, err := h.notificationUseCase.SendNotification(req.UserID, req.Title, req.Message, req.Type, req.Data)
	if err != nil {
		h.logger.Error("Failed to send notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

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

	sentCount, err := h.notificationUseCase.BroadcastNotification(req.UserIDs, req.Title, req.Message, req.Type, req.Data)
	if err != nil {
		h.logger.Error("Failed to broadcast notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to broadcast notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Notifications sent successfully",
		"sent_count": sentCount,
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
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	notifications, totalCount, err := h.notificationUseCase.GetNotifications(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
		"total":         totalCount,
		"offset":        offset,
	})
}

// DeleteNotificationByPostID godoc
// @Summary      Delete notification by post ID
// @Description  Delete notification for a specific post when user views it
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /notifications/{post_id} [delete]
func (h *NotificationHandler) DeleteNotificationByPostID(c *gin.Context) {
	userID := c.GetString("user_id")
	postID := c.Param("post_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Post ID required"})
		return
	}

	deletedCount, err := h.notificationUseCase.DeleteNotificationByPostID(userID, postID)
	if err != nil {
		h.logger.Error("Failed to delete notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification deleted",
		"deleted": deletedCount,
	})
}

// GetNotificationSettings godoc
// @Summary      Get notification settings for a creator
// @Description  Check if notifications are enabled for a specific creator
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        creator_id path string true "Creator ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /notifications/settings/{creator_id} [get]
func (h *NotificationHandler) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	creatorID := c.Param("creator_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	enabled, err := h.notificationUseCase.GetNotificationSettings(userID, creatorID)
	if err != nil {
		h.logger.Error("Failed to get notification settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

// EnableNotifications godoc
// @Summary      Enable notifications for a creator
// @Description  Enable notifications for a specific creator
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        creator_id path string true "Creator ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /notifications/settings/{creator_id} [post]
func (h *NotificationHandler) EnableNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	creatorID := c.Param("creator_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.notificationUseCase.EnableNotifications(userID, creatorID); err != nil {
		h.logger.Error("Failed to enable notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications enabled", "enabled": true})
}

// DisableNotifications godoc
// @Summary      Disable notifications for a creator
// @Description  Disable notifications for a specific creator
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        creator_id path string true "Creator ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /notifications/settings/{creator_id} [delete]
func (h *NotificationHandler) DisableNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	creatorID := c.Param("creator_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.notificationUseCase.DisableNotifications(userID, creatorID); err != nil {
		h.logger.Error("Failed to disable notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications disabled", "enabled": false})
}

func (h *NotificationHandler) ProcessNotificationQueue(c *gin.Context) {
	queueLength, err := h.notificationUseCase.ProcessNotificationQueue()
	if err != nil {
		h.logger.Error("Failed to get queue length: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue length"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Queue processing is handled automatically in main.go. This endpoint shows queue status only.",
		"queue_length": queueLength,
	})
}

func (h *NotificationHandler) HandleWebSocket(c *gin.Context) {
	userID := c.GetString("user_id")
	
	if userID == "" {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			return
		}
		
		claims, err := h.jwtService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		
		userID = claims.UserID
	}
	
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	h.logger.Info("WebSocket connected for user %s", userID)

	ctx := context.Background()
	pubsub := h.redisClient.Subscribe(ctx, fmt.Sprintf("notifications:%s", userID))
	defer pubsub.Close()

	redisChannel := pubsub.Channel()
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case msg := <-redisChannel:
				if msg == nil {
					continue
				}
				if err := conn.WriteMessage(1, []byte(msg.Payload)); err != nil {
					h.logger.Error("Failed to write WebSocket message: %v", err)
					return
				}
			}
		}
	}()

	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			h.logger.Warn("WebSocket read error: %v", err)
			break
		}
		if messageType == 8 {
			break
		}
		if messageType == 9 {
			conn.WriteMessage(10, nil)
		}
	}

	close(done)
	h.logger.Info("WebSocket disconnected for user %s", userID)
}
