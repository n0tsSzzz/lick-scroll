package usecase

import (
	"fmt"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/analytics/internal/repo/persistent"
)

type AnalyticsUseCase interface {
	GetCreatorStats(creatorID string) (map[string]interface{}, error)
	GetPostStats(postID, creatorID string) (map[string]interface{}, error)
	GetRevenue(creatorID string) (int, error)
}

type analyticsUseCase struct {
	analyticsRepo persistent.AnalyticsRepository
	logger         *logger.Logger
}

func NewAnalyticsUseCase(analyticsRepo persistent.AnalyticsRepository, logger *logger.Logger) AnalyticsUseCase {
	return &analyticsUseCase{
		analyticsRepo: analyticsRepo,
		logger:        logger,
	}
}

func (uc *analyticsUseCase) GetCreatorStats(creatorID string) (map[string]interface{}, error) {
	posts, err := uc.analyticsRepo.GetCreatorPosts(creatorID)
	if err != nil {
		uc.logger.Error("Failed to get creator posts: %v", err)
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	totalViews := 0
	totalDonations := int64(0)
	totalLikes := int64(0)
	for _, post := range posts {
		if post != nil {
			totalViews += post.Views
			if post.ID != "" {
				donationCount, err := uc.analyticsRepo.GetPostDonations(post.ID)
				if err == nil {
					totalDonations += donationCount
				}
				likeCount, err := uc.analyticsRepo.GetPostLikeCount(post.ID)
				if err == nil {
					totalLikes += likeCount
				}
			}
		}
	}

	revenue, err := uc.analyticsRepo.GetCreatorRevenue(creatorID)
	if err != nil {
		uc.logger.Error("Failed to get revenue: %v", err)
		revenue = 0
	}

	subscribers, err := uc.analyticsRepo.GetCreatorSubscriberCount(creatorID)
	if err != nil {
		uc.logger.Error("Failed to get subscriber count: %v", err)
		subscribers = 0
	}

	return map[string]interface{}{
		"total_posts":       len(posts),
		"total_views":       totalViews,
		"total_donations":   totalDonations,
		"total_likes":       totalLikes,
		"total_revenue":     revenue,
		"total_subscribers": subscribers,
	}, nil
}

func (uc *analyticsUseCase) GetPostStats(postID, creatorID string) (map[string]interface{}, error) {
	post, err := uc.analyticsRepo.GetPostByID(postID)
	if err != nil {
		return nil, fmt.Errorf("post not found")
	}

	if post.CreatorID != creatorID {
		return nil, fmt.Errorf("you can only view stats for your own posts")
	}

	donations, err := uc.analyticsRepo.GetPostDonations(postID)
	if err != nil {
		uc.logger.Error("Failed to get donations: %v", err)
		donations = 0
	}

	donationAmount, err := uc.analyticsRepo.GetPostDonationAmount(postID)
	if err != nil {
		uc.logger.Error("Failed to get donation amount: %v", err)
		donationAmount = 0
	}

	likes, err := uc.analyticsRepo.GetPostLikeCount(postID)
	if err != nil {
		uc.logger.Error("Failed to get likes: %v", err)
		likes = 0
	}

	return map[string]interface{}{
		"post_id":         postID,
		"views":           post.Views,
		"likes":           likes,
		"donations_count": donations,
		"donations_total": donationAmount,
	}, nil
}

func (uc *analyticsUseCase) GetRevenue(creatorID string) (int, error) {
	revenue, err := uc.analyticsRepo.GetCreatorRevenue(creatorID)
	if err != nil {
		uc.logger.Error("Failed to get revenue: %v", err)
		return 0, fmt.Errorf("failed to get revenue: %w", err)
	}
	return revenue, nil
}
