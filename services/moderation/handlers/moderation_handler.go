package handlers

import (
	"context"
	"fmt"
	"net/http"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/services/moderation/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ModerationHandler struct {
	moderationRepo repository.ModerationRepository
	redisClient    *redis.Client
	logger         *logger.Logger
}

func NewModerationHandler(moderationRepo repository.ModerationRepository, redisClient *redis.Client, logger *logger.Logger) *ModerationHandler {
	return &ModerationHandler{
		moderationRepo: moderationRepo,
		redisClient:    redisClient,
		logger:         logger,
	}
}

type ReviewRequest struct {
	Status  string `json:"status" binding:"required,oneof=approved rejected"`
	Comment string `json:"comment"`
}

func (h *ModerationHandler) ReviewPost(c *gin.Context) {
	postID := c.Param("post_id")

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.moderationRepo.GetPostByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	status := models.PostStatus(req.Status)
	if err := h.moderationRepo.UpdatePostStatus(postID, status); err != nil {
		h.logger.Error("Failed to update post status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post status"})
		return
	}

	// If approved, trigger fanout
	if status == models.StatusApproved {
		ctx := context.Background()
		fanoutKey := fmt.Sprintf("fanout:post:%s", postID)
		h.redisClient.Publish(ctx, "fanout_queue", fanoutKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Post reviewed successfully",
		"post_id": postID,
		"status":  status,
	})
}

func (h *ModerationHandler) GetPendingPosts(c *gin.Context) {
	limit := 50
	offset := 0

	posts, err := h.moderationRepo.GetPendingPosts(limit, offset)
	if err != nil {
		h.logger.Error("Failed to get pending posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts)})
}

func (h *ModerationHandler) ApprovePost(c *gin.Context) {
	postID := c.Param("post_id")

	_, err := h.moderationRepo.GetPostByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	if err := h.moderationRepo.UpdatePostStatus(postID, models.StatusApproved); err != nil {
		h.logger.Error("Failed to approve post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve post"})
		return
	}

	// Trigger fanout
	ctx := context.Background()
	fanoutKey := fmt.Sprintf("fanout:post:%s", postID)
	h.redisClient.Publish(ctx, "fanout_queue", fanoutKey)

	c.JSON(http.StatusOK, gin.H{"message": "Post approved successfully"})
}

func (h *ModerationHandler) RejectPost(c *gin.Context) {
	postID := c.Param("post_id")

	if err := h.moderationRepo.UpdatePostStatus(postID, models.StatusRejected); err != nil {
		h.logger.Error("Failed to reject post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post rejected successfully"})
}

