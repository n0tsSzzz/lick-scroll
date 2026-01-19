package persistent

import (
	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetCreatorPosts(creatorID string) ([]map[string]interface{}, error)
	GetPostByID(postID string) (map[string]interface{}, error)
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

func (r *analyticsRepository) GetCreatorPosts(creatorID string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	if err := r.db.Table("posts").
		Where("creator_id = ? AND deleted_at IS NULL", creatorID).
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *analyticsRepository) GetPostByID(postID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := r.db.Table("posts").
		Where("id = ? AND deleted_at IS NULL", postID).
		First(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (r *analyticsRepository) GetPostDonations(postID string) (int64, error) {
	var count int64
	if err := r.db.Table("transactions").
		Where("post_id = ? AND type = ?", postID, "donation").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *analyticsRepository) GetPostDonationAmount(postID string) (int, error) {
	var totalAmount int64
	if err := r.db.Table("transactions").
		Where("post_id = ? AND type = ?", postID, "donation").
		Select("COALESCE(SUM(ABS(amount)), 0)").
		Scan(&totalAmount).Error; err != nil {
		return 0, err
	}
	return int(totalAmount), nil
}

func (r *analyticsRepository) GetCreatorRevenue(creatorID string) (int, error) {
	var totalRevenue int64
	err := r.db.Table("transactions").
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
	err := r.db.Table("likes").Where("post_id = ? AND deleted_at IS NULL", postID).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) GetCreatorSubscriberCount(creatorID string) (int64, error) {
	var count int64
	err := r.db.Table("subscriptions").Where("creator_id = ? AND deleted_at IS NULL", creatorID).Count(&count).Error
	return count, err
}
