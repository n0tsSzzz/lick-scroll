package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	"lick-scroll/services/interaction/internal/repo/persistent"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type InteractionHandler struct {
	repo        persistent.InteractionRepository
	postRepo    *gorm.DB // For checking if post exists
	redisClient *redis.Client
	queueClient *queue.Client
	logger      *logger.Logger
}

func NewInteractionHandler(repo persistent.InteractionRepository, db *gorm.DB, redisClient *redis.Client, queueClient *queue.Client, logger *logger.Logger) *InteractionHandler {
	return &InteractionHandler{
		repo:        repo,
		postRepo:    db,
		redisClient: redisClient,
		queueClient: queueClient,
		logger:      logger,
	}
}

// LikePost godoc
// @Summary      Like a post
// @Description  Like a post (toggle - if already liked, removes like)
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /interactions/posts/{post_id}/like [post]
func (h *InteractionHandler) LikePost(c *gin.Context) {
	postID := c.Param("post_id")
	userID := c.GetString("user_id")

	// Check if post exists
	var count int64
	if err := h.postRepo.Table("posts").Where("id = ? AND deleted_at IS NULL", postID).Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Check if already liked
	isLiked, err := h.repo.IsLiked(userID, postID)
	if err != nil {
		h.logger.Error("Failed to check like status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check like status"})
		return
	}

	ctx := context.Background()
	redisKey := fmt.Sprintf("post:likes:%s", postID)

	if isLiked {
		// Unlike
		if err := h.repo.DeleteLike(userID, postID); err != nil {
			h.logger.Error("Failed to delete like: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike post"})
			return
		}
		// Decrement Redis counter
		h.redisClient.Decr(ctx, redisKey)
		c.JSON(http.StatusOK, gin.H{"message": "Post unliked", "liked": false})
	} else {
		// Like
		if err := h.repo.CreateLike(userID, postID); err != nil {
			h.logger.Error("Failed to create like: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like post"})
			return
		}
		// Increment Redis counter
		h.redisClient.Incr(ctx, redisKey)
		
		// Get creator ID for notification
		var creatorID string
		if err := h.postRepo.Table("posts").Select("creator_id").Where("id = ?", postID).Scan(&creatorID).Error; err == nil {
			// Send notification to post creator via RabbitMQ (if not the same user)
			if creatorID != userID && h.queueClient != nil {
				go func() {
					task := map[string]interface{}{
						"type":       "like",
						"user_id":    creatorID,
						"liker_id":   userID,
						"post_id":    postID,
						"priority":   3, // Lower priority for likes
					}
				
				h.logger.Info("[NOTIFICATION QUEUE] Publishing like notification task to RabbitMQ: liker_id=%s, creator_id=%s, post_id=%s", userID, creatorID, postID)
				if err := h.queueClient.PublishNotificationTask(task); err != nil {
					h.logger.Error("[NOTIFICATION QUEUE] Failed to publish like notification task to RabbitMQ: %v", err)
				} else {
						h.logger.Info("[NOTIFICATION QUEUE] Successfully published like notification task to RabbitMQ")
					}
				}()
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Post liked", "liked": true})
	}
}

// GetLikeCount godoc
// @Summary      Get like count for a post
// @Description  Get the number of likes for a post (from Redis cache first, fallback to DB)
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Router       /interactions/posts/{post_id}/likes [get]
func (h *InteractionHandler) GetLikeCount(c *gin.Context) {
	postID := c.Param("post_id")

	ctx := context.Background()
	redisKey := fmt.Sprintf("post:likes:%s", postID)

	// Try Redis first
	countStr, err := h.redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		c.JSON(http.StatusOK, gin.H{"post_id": postID, "likes_count": count})
		return
	}

	// Fallback to DB
	count, err := h.repo.GetLikeCount(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Cache in Redis
	h.redisClient.Set(ctx, redisKey, count, 0)

	c.JSON(http.StatusOK, gin.H{"post_id": postID, "likes_count": count})
}

// IsLiked godoc
// @Summary      Check if user liked a post
// @Description  Check if the authenticated user has liked a specific post
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /interactions/posts/{post_id}/liked [get]
func (h *InteractionHandler) IsLiked(c *gin.Context) {
	postID := c.Param("post_id")
	userID := c.GetString("user_id")

	isLiked, err := h.repo.IsLiked(userID, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check like status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"post_id": postID, "liked": isLiked})
}

// GetLikedPosts godoc
// @Summary      Get liked posts
// @Description  Get all posts liked by the authenticated user
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /interactions/posts/liked [get]
func (h *InteractionHandler) GetLikedPosts(c *gin.Context) {
	userID := c.GetString("user_id")
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

	posts, err := h.repo.GetLikedPosts(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get liked posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch liked posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "offset": offset})
}

// IncrementView godoc
// @Summary      Increment post view count
// @Description  Increment view count for a post (only once per user)
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /interactions/posts/{post_id}/view [post]
func (h *InteractionHandler) IncrementView(c *gin.Context) {
	postID := c.Param("post_id")
	userID := c.GetString("user_id")

	// Check if post exists
	var count int64
	if err := h.postRepo.Table("posts").Where("id = ? AND deleted_at IS NULL", postID).Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Use Redis to track views - only increment once per user
	ctx := context.Background()
	viewKey := fmt.Sprintf("post_viewed:%s:%s", postID, userID)
	redisViewCountKey := fmt.Sprintf("post:views:%s", postID)

	set, err := h.redisClient.SetNX(ctx, viewKey, "1", 365*24*3600*time.Second).Result()
	if err != nil {
		h.logger.Error("Failed to set view key in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track view"})
		return
	}

	// Only increment if this is the first time this user views the post
	if set {
		if err := h.repo.IncrementViews(postID); err != nil {
			h.logger.Error("Failed to increment views: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment views"})
			return
		}
		// Increment Redis counter
		h.redisClient.Incr(ctx, redisViewCountKey)
		c.JSON(http.StatusOK, gin.H{
			"message": "View counted",
			"viewed":  true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "View already counted",
			"viewed":  false,
		})
	}
}

// GetViewCount godoc
// @Summary      Get view count for a post
// @Description  Get the number of views for a post (from Redis cache first, fallback to DB)
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Router       /interactions/posts/{post_id}/views [get]
func (h *InteractionHandler) GetViewCount(c *gin.Context) {
	postID := c.Param("post_id")

	ctx := context.Background()
	redisKey := fmt.Sprintf("post:views:%s", postID)

	// Try Redis first
	countStr, err := h.redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		c.JSON(http.StatusOK, gin.H{"post_id": postID, "views_count": count})
		return
	}

	// Fallback to DB
	count, err := h.repo.GetViewCount(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Cache in Redis
	h.redisClient.Set(ctx, redisKey, count, 0)

	c.JSON(http.StatusOK, gin.H{"post_id": postID, "views_count": count})
}
