package repository

import (
	"lick-scroll/pkg/models"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetCreatorPosts(creatorID string) ([]*models.Post, error)
	GetPostByID(postID string) (*models.Post, error)
	GetPostPurchases(postID string) (int64, error)
	GetCreatorRevenue(creatorID string) (int, error)
	GetPostLikeCount(postID string) (int64, error)
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

func (r *analyticsRepository) GetPostPurchases(postID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Transaction{}).
		Where("post_id = ? AND type = ?", postID, models.TransactionTypePurchase).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *analyticsRepository) GetCreatorRevenue(creatorID string) (int, error) {
	var totalRevenue int64
	err := r.db.Model(&models.Transaction{}).
		Joins("JOIN posts ON transactions.post_id = posts.id").
		Where("posts.creator_id = ? AND transactions.type = ?", creatorID, models.TransactionTypePurchase).
		Select("COALESCE(SUM(ABS(transactions.amount)), 0)").
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

