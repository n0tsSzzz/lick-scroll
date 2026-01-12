package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/pkg/s3"
	"lick-scroll/services/post/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PostHandler struct {
	postRepo   repository.PostRepository
	s3Client   *s3.Client
	redisClient *redis.Client
	logger     *logger.Logger
}

func NewPostHandler(postRepo repository.PostRepository, s3Client *s3.Client, redisClient *redis.Client, logger *logger.Logger) *PostHandler {
	return &PostHandler{
		postRepo:    postRepo,
		s3Client:    s3Client,
		redisClient: redisClient,
		logger:      logger,
	}
}

type CreatePostRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	Type        string `form:"type" binding:"required,oneof=photo video"`
	Category    string `form:"category"`
	Price       int    `form:"price"`
}

// CreatePost godoc
// @Summary      Create a new post
// @Description  Create a new post with media file (photo or video up to 30s). Only creators can create posts.
// @Tags         posts
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        title formData string true "Post title"
// @Param        description formData string false "Post description"
// @Param        type formData string true "Post type (photo or video)" Enums(photo, video)
// @Param        category formData string false "Post category"
// @Param        price formData int false "Post price in internal currency"
// @Param        media formData file true "Media file (photo: jpg/jpeg/png, video: mp4/mov/avi)"
// @Success      201  {object}  models.Post
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	// Only creators can create posts
	if userRole != string(models.RoleCreator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only creators can create posts"})
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get file
	file, err := c.FormFile("media")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Media file is required"})
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	if req.Type == "video" {
		if ext != ".mp4" && ext != ".mov" && ext != ".avi" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video format. Only mp4, mov, avi are allowed"})
			return
		}
		// Check video duration (should be <= 30 seconds)
		// In production, you would use ffmpeg or similar to check duration
	} else {
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format. Only jpg, jpeg, png are allowed"})
			return
		}
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		h.logger.Error("Failed to open file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	defer src.Close()

	// Upload to S3
	fileKey := fmt.Sprintf("posts/%s/%s%s", userID, uuid.New().String(), ext)
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		if req.Type == "video" {
			contentType = "video/mp4"
		} else {
			contentType = "image/jpeg"
		}
	}

	mediaURL, err := h.s3Client.UploadFile(fileKey, src, contentType)
	if err != nil {
		h.logger.Error("Failed to upload file to S3: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Create post
	post := &models.Post{
		CreatorID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Type:        models.PostType(req.Type),
		MediaURL:    mediaURL,
		Category:    req.Category,
		Price:       req.Price,
		Status:      models.StatusPending, // Needs moderation
	}

	if err := h.postRepo.Create(post); err != nil {
		h.logger.Error("Failed to create post: %v", err)
		// Try to delete from S3 if post creation fails
		_ = h.s3Client.DeleteFile(fileKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	// Cache post metadata for quick access
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", post.ID)
	postData := map[string]interface{}{
		"id":          post.ID,
		"creator_id":  post.CreatorID,
		"title":       post.Title,
		"media_url":   post.MediaURL,
		"price":       fmt.Sprintf("%d", post.Price),
		"category":    post.Category,
		"status":      string(post.Status),
	}
	for k, v := range postData {
		h.redisClient.HSet(ctx, postKey, k, v)
	}
	h.redisClient.Expire(ctx, postKey, 24*time.Hour)

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

	// Try to get from cache first
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", postID)
	cached := h.redisClient.Get(ctx, postKey)
	if cached.Err() == nil {
		// Post exists in cache, get from DB
	}

	post, err := h.postRepo.GetByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Increment views
	go h.postRepo.IncrementViews(postID)

	c.JSON(http.StatusOK, post)
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
	status := models.StatusApproved // Only show approved posts

	posts, err := h.postRepo.List(limit, offset, category, status)
	if err != nil {
		h.logger.Error("Failed to list posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts)})
}

// UpdatePost godoc
// @Summary      Update post
// @Description  Update post details. Only the creator can update their own posts.
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Param        request body object true "Update data" SchemaExample({"title":"Updated title","description":"Updated description","category":"fetish","price":150})
// @Success      200  {object}  models.Post
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /posts/{id} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetString("user_id")

	post, err := h.postRepo.GetByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Only creator can update their own post
	if post.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own posts"})
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Price       int    `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != "" {
		post.Title = req.Title
	}
	if req.Description != "" {
		post.Description = req.Description
	}
	if req.Category != "" {
		post.Category = req.Category
	}
	if req.Price >= 0 {
		post.Price = req.Price
	}

	if err := h.postRepo.Update(post); err != nil {
		h.logger.Error("Failed to update post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	c.JSON(http.StatusOK, post)
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

	post, err := h.postRepo.GetByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Only creator can delete their own post
	if post.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own posts"})
		return
	}

	if err := h.postRepo.Delete(postID); err != nil {
		h.logger.Error("Failed to delete post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}

	// Remove from cache
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

	posts, err := h.postRepo.GetByCreatorID(creatorID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get creator posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts)})
}

