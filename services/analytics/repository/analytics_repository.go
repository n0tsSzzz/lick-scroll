package repository

import (
	"lick-scroll/pkg/models"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetCreatorPosts(creatorID string) ([]*models.Post, error)
	GetPostByID(postID string) (*models.Post, error)
	GetPostDonations(postID string) (int64, error)
	GetPostDonationAmount(postID string) (int, error)
	GetCreatorRevenue(creatorID string) (int, error)
	GetPostLikeCount(postID string) (int64, error)
	GetCreatorSubscriberCount(creatorID string) (int64, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) GetCreatorPosts(creatorID string) ([]*models.Post, error) {
	var posts []*models.Post
	if err := r.db.Where("creator_id = ?", creatorID).Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *analyticsRepository) GetPostByID(postID string) (*models.Post, error) {
	var post models.Post
	if err := r.db.Where("id = ?", postID).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *analyticsRepository) GetPostDonations(postID string) (int64, error) {
	var count int64
	// Count donations - transactions where user donated to this post
	if err := r.db.Model(&models.Transaction{}).
		Where("post_id = ? AND type = ?", postID, models.TransactionTypeDonation).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *analyticsRepository) GetPostDonationAmount(postID string) (int, error) {
	var totalAmount int64
	// Sum all donations for this post (donations are negative amounts, so we take ABS)
	if err := r.db.Model(&models.Transaction{}).
		Where("post_id = ? AND type = ?", postID, models.TransactionTypeDonation).
		Select("COALESCE(SUM(ABS(amount)), 0)").
		Scan(&totalAmount).Error; err != nil {
		return 0, err
	}
	return int(totalAmount), nil
}

func (r *analyticsRepository) GetCreatorRevenue(creatorID string) (int, error) {
	var totalRevenue int64
	// Revenue is calculated from TransactionTypeEarn - these are transactions where creator received money
	// (when someone donates to creator's post, creator gets TransactionTypeEarn with positive amount)
	err := r.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND amount > 0", creatorID, models.TransactionTypeEarn).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRevenue).Error
	if err != nil {
		return 0, err
	}
	return int(totalRevenue), nil
}

func (r *analyticsRepository) GetPostLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Like{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) GetCreatorSubscriberCount(creatorID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Subscription{}).Where("creator_id = ?", creatorID).Count(&count).Error
	return count, err
}
