package http

import (
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/feed/internal/usecase"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	feedUseCase usecase.FeedUseCase
	logger      *logger.Logger
}

func NewFeedHandler(feedUseCase usecase.FeedUseCase, logger *logger.Logger) *FeedHandler {
	return &FeedHandler{
		feedUseCase: feedUseCase,
		logger:      logger,
	}
}

// GetFeed godoc
// @Summary      Get personalized feed
// @Description  Get personalized feed based on subscriptions: posts from subscribed creators first, then other posts
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Router       /feed [get]
func (h *FeedHandler) GetFeed(c *gin.Context) {
	userID := c.GetString("user_id")
	limit := 100
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

	posts, count, err := h.feedUseCase.GetFeed(userID, limit, offset, c.GetHeader("Authorization"))
	if err != nil {
		h.logger.Error("Failed to get feed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": count, "offset": offset})
}

// GetFeedByCategory godoc
// @Summary      Get feed by category
// @Description  Get global feed filtered by category
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        category path string true "Category name"
// @Param        limit query int false "Number of posts to return (max 100)"
// @Param        offset query int false "Offset for pagination"
// @Success      200  {object}  map[string]interface{}
// @Router       /feed/category/{category} [get]
func (h *FeedHandler) GetFeedByCategory(c *gin.Context) {
	userID := c.GetString("user_id")
	category := c.Param("category")
	limit := 100
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

	posts, err := h.feedUseCase.GetFeedByCategory(userID, category, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get feed by category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "count": len(posts), "category": category, "offset": offset})
}
