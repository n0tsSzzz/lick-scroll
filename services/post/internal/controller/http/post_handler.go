package http

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/post/internal/entity"
	"lick-scroll/services/post/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type PostHandler struct {
	postUseCase  usecase.PostUseCase
	redisClient  *redis.Client
	logger       *logger.Logger
}

func NewPostHandler(postUseCase usecase.PostUseCase, redisClient *redis.Client, logger *logger.Logger) *PostHandler {
	return &PostHandler{
		postUseCase: postUseCase,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (h *PostHandler) formatPostResponse(post *entity.Post, likeCount int64) map[string]interface{} {
	response := map[string]interface{}{
		"id":          post.ID,
		"creator_id":  post.CreatorID,
		"title":       post.Title,
		"description": post.Description,
		"type":        post.Type,
		"category":    post.Category,
		"status":      post.Status,
		"views":       post.Views,
		"likes_count": likeCount,
		"images":      post.Images,
		"created_at":  post.CreatedAt,
		"updated_at":  post.UpdatedAt,
	}

	if post.MediaURL != "" && len(post.Images) == 0 {
		response["media_url"] = post.MediaURL
	}

	if post.ThumbnailURL != "" {
		response["thumbnail_url"] = post.ThumbnailURL
	}

	return response
}

type CreatePostRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	Type        string `form:"type" binding:"required,oneof=photo video"`
	Category    string `form:"category"`
}

// CreatePost godoc
// @Summary      Create a new post
// @Description  Create a new post with media files. For photo posts, you can upload multiple images. For video posts, upload one video file (up to 30s).
// @Tags         posts
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        title formData string true "Post title"
// @Param        description formData string false "Post description"
// @Param        type formData string true "Post type (photo or video)" Enums(photo, video)
// @Param        category formData string false "Post category"
// @Param        media formData file false "Media file (for video: mp4/mov/avi, for photo: jpg/jpeg/png) - deprecated, use images[] instead"
// @Param        images formData file false "Image files (jpg/jpeg/png) - multiple files allowed for photo posts"
// @Success      201  {object}  models.Post
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreatePostRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mediaFile *multipart.FileHeader
	var imageFiles []*multipart.FileHeader

	if req.Type == "video" {
		file, err := c.FormFile("media")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Media file is required for video posts"})
			return
		}
		mediaFile = file
	} else {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
			return
		}

		files := form.File["images"]
		if len(files) == 0 {
			mediaFile, err := c.FormFile("media")
			if err == nil {
				files = []*multipart.FileHeader{mediaFile}
			}
		}

		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "At least one image file is required for photo posts"})
			return
		}

		imageFiles = files
	}

	post, err := h.postUseCase.CreatePost(userID, req.Title, req.Description, req.Type, req.Category, mediaFile, imageFiles)
	if err != nil {
		h.logger.Error("Failed to create post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, post)
}

// GetPost godoc
// @Summary      Get post by ID
// @Description  Get post details by ID and increment view count
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  models.Post
// @Failure      404  {object}  map[string]string
// @Router       /posts/{id} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	post, likeCount, isLiked, err := h.postUseCase.GetPost(postID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	response := gin.H{
		"id":           post.ID,
		"creator_id":   post.CreatorID,
		"title":        post.Title,
		"description":  post.Description,
		"type":         post.Type,
		"media_url":    post.MediaURL,
		"thumbnail_url": post.ThumbnailURL,
		"category":     post.Category,
		"status":       post.Status,
		"views":        post.Views,
		"likes_count":  likeCount,
		"is_liked":     isLiked,
		"images":       post.Images,
		"created_at":   post.CreatedAt,
		"updated_at":   post.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// ListPosts godoc
// @Summary      List posts
// @Description  Get list of approved posts with optional category filter
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        category query string false "Filter by category"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /posts [get]
func (h *PostHandler) ListPosts(c *gin.Context) {
	limit := 20
	offset := 0
	category := c.Query("category")

	posts, err := h.postUseCase.ListPosts(limit, offset, category)
	if err != nil {
		h.logger.Error("Failed to list posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	postsWithLikes := make([]map[string]interface{}, len(posts))
	for i, post := range posts {
		likeCount, _ := h.postUseCase.GetLikeCount(post.ID)
		postsWithLikes[i] = h.formatPostResponse(post, likeCount)
	}

	c.JSON(http.StatusOK, gin.H{"posts": postsWithLikes, "count": len(postsWithLikes)})
}

// UpdatePost godoc
// @Summary      Update post
// @Description  Update post details. Only the creator can update their own posts.
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Param        request body object true "Update data" SchemaExample({"title":"Updated title","description":"Updated description","category":"fetish"})
// @Success      200  {object}  models.Post
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts/{id} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var title, description, category *string
	if req.Title != "" {
		title = &req.Title
	}
	if req.Description != "" {
		description = &req.Description
	}
	if req.Category != "" {
		category = &req.Category
	}

	post, err := h.postUseCase.UpdatePost(postID, userID, title, description, category)
	if err != nil {
		if err.Error() == "you can only update your own posts" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Failed to update post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	likeCount, _ := h.postUseCase.GetLikeCount(post.ID)
	response := h.formatPostResponse(post, likeCount)
	c.JSON(http.StatusOK, response)
}

// DeletePost godoc
// @Summary      Delete post
// @Description  Delete a post. Only the creator can delete their own posts.
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts/{id} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	if err := h.postUseCase.DeletePost(postID, userID); err != nil {
		if err.Error() == "you can only delete your own posts" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Failed to delete post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}

	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", postID)
	h.redisClient.Del(ctx, postKey)

	c.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}

// GetCreatorPosts godoc
// @Summary      Get creator posts
// @Description  Get all posts by a specific creator
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        creator_id path string true "Creator ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /posts/creator/{creator_id} [get]
func (h *PostHandler) GetCreatorPosts(c *gin.Context) {
	creatorID := c.Param("creator_id")
	limit := 20
	offset := 0

	posts, err := h.postUseCase.GetCreatorPosts(creatorID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get creator posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	postsWithLikes := make([]map[string]interface{}, len(posts))
	for i, post := range posts {
		likeCount, _ := h.postUseCase.GetLikeCount(post.ID)
		postsWithLikes[i] = h.formatPostResponse(post, likeCount)
	}

	c.JSON(http.StatusOK, gin.H{"posts": postsWithLikes, "count": len(postsWithLikes)})
}

// LikePost godoc
// @Summary      Like a post
// @Description  Like a post (toggle - if already liked, removes like)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts/{id}/like [post]
func (h *PostHandler) LikePost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	liked, err := h.postUseCase.LikePost(userID, postID)
	if err != nil {
		h.logger.Error("Failed to like post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like post"})
		return
	}

	message := "Post liked"
	if !liked {
		message = "Post unliked"
	}

	c.JSON(http.StatusOK, gin.H{"message": message, "liked": liked})
}

// GetLikedPosts godoc
// @Summary      Get liked posts
// @Description  Get all posts liked by the authenticated user
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /posts/liked [get]
func (h *PostHandler) GetLikedPosts(c *gin.Context) {
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

	posts, err := h.postUseCase.GetLikedPosts(userID, limit, offset)
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
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts/{id}/view [post]
func (h *PostHandler) IncrementView(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	ctx := context.Background()
	viewKey := fmt.Sprintf("post_viewed:%s:%s", postID, userID)

	set, err := h.redisClient.SetNX(ctx, viewKey, "1", 365*24*time.Hour).Result()
	if err != nil {
		h.logger.Error("Failed to set view key in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track view"})
		return
	}

	if set {
		if err := h.postUseCase.IncrementView(postID); err != nil {
			h.logger.Error("Failed to increment views: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment views"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "View counted",
			"viewed": true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "View already counted",
			"viewed": false,
		})
	}
}
