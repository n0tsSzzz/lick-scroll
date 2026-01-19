package http

import (
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/interaction/internal/usecase"

	"github.com/gin-gonic/gin"
)

type InteractionHandler struct {
	interactionUseCase usecase.InteractionUseCase
	logger             *logger.Logger
}

func NewInteractionHandler(interactionUseCase usecase.InteractionUseCase, logger *logger.Logger) *InteractionHandler {
	return &InteractionHandler{
		interactionUseCase: interactionUseCase,
		logger:             logger,
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

	liked, err := h.interactionUseCase.LikePost(userID, postID)
	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("Failed to like post: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if liked {
		c.JSON(http.StatusOK, gin.H{"message": "Post liked", "liked": true})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Post unliked", "liked": false})
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

	count, err := h.interactionUseCase.GetLikeCount(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

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

	isLiked, err := h.interactionUseCase.IsLiked(userID, postID)
	if err != nil {
		h.logger.Error("Failed to check like status: %v", err)
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

	posts, err := h.interactionUseCase.GetLikedPosts(userID, limit, offset)
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

	viewed, err := h.interactionUseCase.IncrementView(userID, postID)
	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("Failed to increment view: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if viewed {
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

	count, err := h.interactionUseCase.GetViewCount(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"post_id": postID, "views_count": count})
}
