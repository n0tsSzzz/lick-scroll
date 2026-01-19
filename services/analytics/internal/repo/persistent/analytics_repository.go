package persistent

import (
	"lick-scroll/services/analytics/internal/entity"
	"lick-scroll/services/analytics/internal/model"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetCreatorPosts(creatorID string) ([]*entity.Post, error)
	GetPostByID(postID string) (*entity.Post, error)
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

func (r *analyticsRepository) GetCreatorPosts(creatorID string) ([]*entity.Post, error) {
	var postModels []model.PostModel
	if err := r.db.Where("creator_id = ? AND deleted_at IS NULL", creatorID).Find(&postModels).Error; err != nil {
		return nil, err
	}
	return ToPostEntities(postModels), nil
}

func (r *analyticsRepository) GetPostByID(postID string) (*entity.Post, error) {
	var postModel model.PostModel
	if err := r.db.Where("id = ? AND deleted_at IS NULL", postID).First(&postModel).Error; err != nil {
		return nil, err
	}
	return ToPostEntity(&postModel), nil
}

func (r *analyticsRepository) GetPostDonations(postID string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.TransactionModel{}).
		Where("post_id = ? AND type = ?", postID, "donation").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *analyticsRepository) GetPostDonationAmount(postID string) (int, error) {
	var totalAmount int64
	if err := r.db.Model(&model.TransactionModel{}).
		Where("post_id = ? AND type = ?", postID, "donation").
		Select("COALESCE(SUM(ABS(amount)), 0)").
		Scan(&totalAmount).Error; err != nil {
		return 0, err
	}
	return int(totalAmount), nil
}

func (r *analyticsRepository) GetCreatorRevenue(creatorID string) (int, error) {
	var totalRevenue int64
	err := r.db.Model(&model.TransactionModel{}).
		Where("user_id = ? AND type = ? AND amount > 0", creatorID, "earn").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRevenue).Error
	if err != nil {
		return 0, err
	}
	return int(totalRevenue), nil
}

func (r *analyticsRepository) GetPostLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.LikeModel{}).Where("post_id = ? AND deleted_at IS NULL", postID).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) GetCreatorSubscriberCount(creatorID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.SubscriptionModel{}).Where("creator_id = ? AND deleted_at IS NULL", creatorID).Count(&count).Error
	return count, err
}
