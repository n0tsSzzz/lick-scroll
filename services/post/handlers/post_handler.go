package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
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
	postRepo      repository.PostRepository
	s3Client      *s3.Client
	redisClient   *redis.Client
	logger        *logger.Logger
	fanoutServiceURL string
}

func NewPostHandler(postRepo repository.PostRepository, s3Client *s3.Client, redisClient *redis.Client, logger *logger.Logger, fanoutServiceURL string) *PostHandler {
	return &PostHandler{
		postRepo:        postRepo,
		s3Client:        s3Client,
		redisClient:     redisClient,
		logger:          logger,
		fanoutServiceURL: fanoutServiceURL,
	}
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
	// All users can create posts (like TikTok)

	var req CreatePostRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mediaURL string
	var postImages []models.PostImage

	if req.Type == "video" {
		// For video, get single file
		file, err := c.FormFile("media")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Media file is required for video posts"})
			return
		}

		// Validate video file type
		ext := filepath.Ext(file.Filename)
		if ext != ".mp4" && ext != ".mov" && ext != ".avi" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video format. Only mp4, mov, avi are allowed"})
			return
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
			contentType = "video/mp4"
		}

		uploadedURL, err := h.s3Client.UploadFile(fileKey, src, contentType)
		if err != nil {
			h.logger.Error("Failed to upload file to S3: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
			return
		}
		mediaURL = uploadedURL
	} else {
		// For photo posts, get multiple images
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
			return
		}

		files := form.File["images[]"]
		// Fallback to single "media" file for backward compatibility
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

		// Validate and upload each image
		for i, file := range files {
			ext := filepath.Ext(file.Filename)
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid image format for file %s. Only jpg, jpeg, png are allowed", file.Filename)})
				return
			}

			// Open file
			src, err := file.Open()
			if err != nil {
				h.logger.Error("Failed to open file: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
				return
			}

			// Upload to S3
			fileKey := fmt.Sprintf("posts/%s/%s%s", userID, uuid.New().String(), ext)
			contentType := file.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "image/jpeg"
			}

			imageURL, err := h.s3Client.UploadFile(fileKey, src, contentType)
			src.Close()
			if err != nil {
				h.logger.Error("Failed to upload file to S3: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
				return
			}

			// For backward compatibility, set MediaURL to first image
			if i == 0 {
				mediaURL = imageURL
			}

			postImages = append(postImages, models.PostImage{
				ImageURL: imageURL,
				Order:    i,
			})
		}
	}

	// Create post
	post := &models.Post{
		CreatorID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Type:        models.PostType(req.Type),
		MediaURL:    mediaURL, // For backward compatibility
		Category:    req.Category,
		Price:       0, // All posts are free now
		Status:      models.StatusPending, // Needs moderation
		Images:      postImages,
	}

	if err := h.postRepo.Create(post); err != nil {
		h.logger.Error("Failed to create post: %v", err)
		// Note: In production, you should delete uploaded files from S3 if post creation fails
		// For now, we just log the error
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
		"media_url":   post.MediaURL, // For backward compatibility
		"category":    post.Category,
		"status":      string(post.Status),
	}
	
	// Add images data if available
	if len(post.Images) > 0 {
		imagesJSON, _ := json.Marshal(post.Images)
		postData["images"] = string(imagesJSON)
	}
	
	for k, v := range postData {
		h.redisClient.HSet(ctx, postKey, k, v)
	}
	h.redisClient.Expire(ctx, postKey, 24*time.Hour)

	// Add post to global feed (like TikTok - all viewers see all posts)
	// In production, this should happen after moderation approves
	globalFeedKey := "feed:global"
	h.redisClient.LPush(ctx, globalFeedKey, post.ID)
	h.redisClient.LTrim(ctx, globalFeedKey, 0, 9999) // Keep last 10000 posts
	h.redisClient.Expire(ctx, globalFeedKey, 7*24*time.Hour)

	// Also add to global category feed if category is specified
	if post.Category != "" {
		categoryFeedKey := fmt.Sprintf("feed:global:%s", post.Category)
		h.redisClient.LPush(ctx, categoryFeedKey, post.ID)
		h.redisClient.LTrim(ctx, categoryFeedKey, 0, 9999)
		h.redisClient.Expire(ctx, categoryFeedKey, 7*24*time.Hour)
	}

	// Call fanout service to notify subscribers (async)
	go func() {
		fanoutURL := fmt.Sprintf("%s/api/v1/fanout/post/%s", h.fanoutServiceURL, post.ID)
		fanoutData := map[string]interface{}{
			"post_id":    post.ID,
			"creator_id": post.CreatorID,
			"category":   post.Category,
		}
		
		jsonData, err := json.Marshal(fanoutData)
		if err != nil {
			h.logger.Error("Failed to marshal fanout data: %v", err)
			return
		}

		req, err := http.NewRequest("POST", fanoutURL, bytes.NewBuffer(jsonData))
		if err != nil {
			h.logger.Error("Failed to create fanout request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", c.GetHeader("Authorization"))

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			h.logger.Error("Failed to call fanout service: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			h.logger.Error("Fanout service returned error: %d", resp.StatusCode)
			return
		}

		h.logger.Info("Post fanned out successfully: %s", post.ID)
	}()

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

	// All posts are free now - no access restrictions

	// Increment views when someone directly views the post
	// Views are counted only for direct post views (GET /posts/{id}), not for feed views
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
	
	// For MVP: Show both pending and approved posts (exclude rejected)
	// Get more posts to combine and sort properly
	approvedPosts, err := h.postRepo.List(limit*2, 0, category, models.StatusApproved)
	if err != nil {
		h.logger.Error("Failed to list approved posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}
	
	// Get pending posts
	pendingPosts, err := h.postRepo.List(limit*2, 0, category, models.StatusPending)
	if err != nil {
		h.logger.Error("Failed to list pending posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}
	
	// Combine posts (approved first, then pending)
	// Simple merge: approved posts first, then pending
	// In production, you'd want proper sorting by CreatedAt, but this works for MVP
	result := approvedPosts
	result = append(result, pendingPosts...)
	
	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(result) {
		result = []*models.Post{}
	} else {
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	c.JSON(http.StatusOK, gin.H{"posts": result, "count": len(result)})
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
	// All users can like posts (like TikTok)

	// Check if post exists
	_, err := h.postRepo.GetByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Check if already liked
	isLiked, err := h.postRepo.IsLiked(userID, postID)
	if err != nil {
		h.logger.Error("Failed to check like status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check like status"})
		return
	}

	if isLiked {
		// Unlike
		if err := h.postRepo.DeleteLike(userID, postID); err != nil {
			h.logger.Error("Failed to delete like: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike post"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Post unliked", "liked": false})
	} else {
		// Like
		if err := h.postRepo.CreateLike(userID, postID); err != nil {
			h.logger.Error("Failed to create like: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like post"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Post liked", "liked": true})
	}
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

	posts, err := h.postRepo.GetLikedPosts(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get liked posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch liked posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "offset": offset})
}

