package http

import (
	"net/http"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/analytics/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsUseCase usecase.AnalyticsUseCase
	logger           *logger.Logger
}

func NewAnalyticsHandler(analyticsUseCase usecase.AnalyticsUseCase, logger *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsUseCase: analyticsUseCase,
		logger:           logger,
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

	stats, err := h.analyticsUseCase.GetCreatorStats(userID)
	if err != nil {
		h.logger.Error("Failed to get creator stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
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

	stats, err := h.analyticsUseCase.GetPostStats(postID, userID)
	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if err.Error() == "you can only view stats for your own posts" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("Failed to get post stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, stats)
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

	revenue, err := h.analyticsUseCase.GetRevenue(userID)
	if err != nil {
		h.logger.Error("Failed to get revenue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"creator_id": userID,
		"revenue":    revenue,
	})
}
