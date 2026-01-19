package persistent

import (
	"lick-scroll/services/notification/internal/model"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	GetCreatorUsername(creatorID string) (string, error)
	GetSubscribers(creatorID string) ([]string, error)
	GetLikerUsername(likerID string) (string, error)
	GetSubscriberUsername(subscriberID string) (string, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) GetCreatorUsername(creatorID string) (string, error) {
	var userModel model.UserModel
	err := r.db.Where("id = ?", creatorID).Select("username").First(&userModel).Error
	if err != nil {
		return "", err
	}
	return ToUserEntity(&userModel), nil
}

func (r *notificationRepository) GetSubscribers(creatorID string) ([]string, error) {
	var subscriptionModels []model.SubscriptionModel
	if err := r.db.Where("creator_id = ? AND deleted_at IS NULL", creatorID).Select("viewer_id").Find(&subscriptionModels).Error; err != nil {
		return nil, err
	}
	return ToSubscriptionEntity(subscriptionModels), nil
}

func (r *notificationRepository) GetLikerUsername(likerID string) (string, error) {
	var userModel model.UserModel
	err := r.db.Where("id = ?", likerID).Select("username").First(&userModel).Error
	if err != nil {
		return "", err
	}
	return ToUserEntity(&userModel), nil
}

func (r *notificationRepository) GetSubscriberUsername(subscriberID string) (string, error) {
	var userModel model.UserModel
	err := r.db.Where("id = ?", subscriberID).Select("username").First(&userModel).Error
	if err != nil {
		return "", err
	}
	return ToUserEntity(&userModel), nil
}
