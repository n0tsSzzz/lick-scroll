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
// @Description  Get overall statistics for the authenticated creator. Views are incremented when someone views a post via GET /posts/{id}. Revenue is calculated from donations received.
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Router       /analytics/creator/stats [get]
func (h *AnalyticsHandler) GetCreatorStats(c *gin.Context) {
	userID := c.GetString("user_id")
	// All users can view their stats (like TikTok)

	posts, err := h.analyticsRepo.GetCreatorPosts(userID)
	if err != nil {
		h.logger.Error("Failed to get creator posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	totalViews := 0
	totalDonations := int64(0)
	totalLikes := int64(0)
	for _, post := range posts {
		totalViews += post.Views
		donationCount, err := h.analyticsRepo.GetPostDonations(post.ID)
		if err == nil {
			totalDonations += donationCount
		}
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

	subscribers, err := h.analyticsRepo.GetCreatorSubscriberCount(userID)
	if err != nil {
		h.logger.Error("Failed to get subscriber count: %v", err)
		subscribers = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"total_posts":      len(posts),
		"total_views":      totalViews,
		"total_donations":  totalDonations,
		"total_likes":      totalLikes,
		"total_revenue":    revenue,
		"total_subscribers": subscribers,
	})
}

// GetPostStats godoc
// @Summary      Get post statistics
// @Description  Get statistics for a specific post. Views are incremented when someone views the post via GET /posts/{id}. Donations are counted from TransactionTypeDonation.
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
	// All users can view their post stats (like TikTok)

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

	donations, err := h.analyticsRepo.GetPostDonations(postID)
	if err != nil {
		h.logger.Error("Failed to get donations: %v", err)
		donations = 0
	}

	donationAmount, err := h.analyticsRepo.GetPostDonationAmount(postID)
	if err != nil {
		h.logger.Error("Failed to get donation amount: %v", err)
		donationAmount = 0
	}

	likes, err := h.analyticsRepo.GetPostLikeCount(postID)
	if err != nil {
		h.logger.Error("Failed to get likes: %v", err)
		likes = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"post_id":         postID,
		"views":           post.Views,
		"likes":           likes,
		"donations_count": donations,
		"donations_total": donationAmount,
	})
}

// GetRevenue godoc
// @Summary      Get creator revenue
// @Description  Get total revenue for the authenticated creator. Revenue is calculated from donations received (TransactionTypeEarn transactions with positive amounts).
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Router       /analytics/creator/revenue [get]
func (h *AnalyticsHandler) GetRevenue(c *gin.Context) {
	userID := c.GetString("user_id")
	// All users can view their revenue (like TikTok)

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

