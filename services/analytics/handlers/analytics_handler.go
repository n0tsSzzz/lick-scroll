package handlers

import (
	"net/http"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/analytics/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type AnalyticsHandler struct {
	analyticsRepo repository.AnalyticsRepository
	redisClient   *redis.Client
	logger        *logger.Logger
}

func NewAnalyticsHandler(analyticsRepo repository.AnalyticsRepository, redisClient *redis.Client, logger *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsRepo: analyticsRepo,
		redisClient:   redisClient,
		logger:        logger,
	}
}

// GetCreatorStats godoc
// @Summary      Get creator statistics
// @Description  Get overall statistics for the authenticated creator
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Router       /analytics/creator/stats [get]
func (h *AnalyticsHandler) GetCreatorStats(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	// Only creators can view their stats
	if userRole != "creator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only creators can view analytics"})
		return
	}

	posts, err := h.analyticsRepo.GetCreatorPosts(userID)
	if err != nil {
		h.logger.Error("Failed to get creator posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	totalViews := 0
	totalPurchases := 0
	totalLikes := int64(0)
	for _, post := range posts {
		totalViews += post.Views
		totalPurchases += post.Purchases
		likeCount, err := h.analyticsRepo.GetPostLikeCount(post.ID)
		if err == nil {
			totalLikes += likeCount
		}
	}

	revenue, err := h.analyticsRepo.GetCreatorRevenue(userID)
	if err != nil {
		h.logger.Error("Failed to get revenue: %v", err)
		revenue = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"total_posts":     len(posts),
		"total_views":     totalViews,
		"total_purchases": totalPurchases,
		"total_likes":     totalLikes,
		"total_revenue":   revenue,
	})
}

// GetPostStats godoc
// @Summary      Get post statistics
// @Description  Get statistics for a specific post
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /analytics/creator/posts/{post_id} [get]
func (h *AnalyticsHandler) GetPostStats(c *gin.Context) {
	postID := c.Param("post_id")
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	// Only creators can view their post stats
	if userRole != "creator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only creators can view analytics"})
		return
	}

	post, err := h.analyticsRepo.GetPostByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Verify post belongs to creator
	if post.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view stats for your own posts"})
		return
	}

	purchases, err := h.analyticsRepo.GetPostPurchases(postID)
	if err != nil {
		h.logger.Error("Failed to get purchases: %v", err)
		purchases = 0
	}

	likes, err := h.analyticsRepo.GetPostLikeCount(postID)
	if err != nil {
		h.logger.Error("Failed to get likes: %v", err)
		likes = 0
	}

	revenue := post.Price * int(purchases)

	c.JSON(http.StatusOK, gin.H{
		"post_id":   postID,
		"views":     post.Views,
		"likes":     likes,
		"purchases": purchases,
		"revenue":   revenue,
	})
}

// GetRevenue godoc
// @Summary      Get creator revenue
// @Description  Get total revenue for the authenticated creator
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Router       /analytics/creator/revenue [get]
func (h *AnalyticsHandler) GetRevenue(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	// Only creators can view revenue
	if userRole != "creator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only creators can view revenue"})
		return
	}

	revenue, err := h.analyticsRepo.GetCreatorRevenue(userID)
	if err != nil {
		h.logger.Error("Failed to get revenue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get revenue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"creator_id": userID,
		"revenue":    revenue,
	})
}

