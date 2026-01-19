package persistent

import (
	"lick-scroll/services/auth/internal/entity"
	"lick-scroll/services/auth/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *entity.User) error
	GetByEmail(email string) (*entity.User, error)
	GetByID(id string) (*entity.User, error)
	GetByUsername(username string) (*entity.User, error)
	Update(user *entity.User) error
	GetSubscriptions(userID string) ([]*entity.Subscription, error)
	CreateSubscription(viewerID, creatorID string) error
	DeleteSubscription(viewerID, creatorID string) error
	GetSubscription(viewerID, creatorID string) (*entity.Subscription, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *entity.User) error {
	userModel := ToUserModel(user)
	if userModel.ID == "" {
		userModel.ID = uuid.New().String()
	}
	if err := r.db.Create(userModel).Error; err != nil {
		return err
	}
	*user = *ToUserEntity(userModel)
	return nil
}

func (r *userRepository) GetByEmail(email string) (*entity.User, error) {
	var userModel model.UserModel
	if err := r.db.Where("email = ?", email).First(&userModel).Error; err != nil {
		return nil, err
	}
	return ToUserEntity(&userModel), nil
}

func (r *userRepository) GetByID(id string) (*entity.User, error) {
	var userModel model.UserModel
	if err := r.db.Where("id = ?", id).First(&userModel).Error; err != nil {
		return nil, err
	}
	return ToUserEntity(&userModel), nil
}

func (r *userRepository) GetByUsername(username string) (*entity.User, error) {
	var userModel model.UserModel
	if err := r.db.Where("username = ?", username).First(&userModel).Error; err != nil {
		return nil, err
	}
	return ToUserEntity(&userModel), nil
}

func (r *userRepository) Update(user *entity.User) error {
	userModel := ToUserModel(user)
	return r.db.Save(userModel).Error
}

func (r *userRepository) GetSubscriptions(userID string) ([]*entity.Subscription, error) {
	var subscriptionModels []model.SubscriptionModel
	if err := r.db.Where("viewer_id = ?", userID).Find(&subscriptionModels).Error; err != nil {
		return nil, err
	}

	subscriptions := make([]*entity.Subscription, len(subscriptionModels))
	for i := range subscriptionModels {
		subscriptions[i] = ToSubscriptionEntity(&subscriptionModels[i])
	}
	return subscriptions, nil
}

func (r *userRepository) CreateSubscription(viewerID, creatorID string) error {
	var existing model.SubscriptionModel
	err := r.db.Unscoped().Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			if err := r.db.Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			return nil
		}
		return nil
	}

	subscriptionModel := &model.SubscriptionModel{
		ID:        uuid.New().String(),
		ViewerID:  viewerID,
		CreatorID: creatorID,
	}
	return r.db.Create(subscriptionModel).Error
}

func (r *userRepository) DeleteSubscription(viewerID, creatorID string) error {
	return r.db.Unscoped().Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).Delete(&model.SubscriptionModel{}).Error
}

func (r *userRepository) GetSubscription(viewerID, creatorID string) (*entity.Subscription, error) {
	var subscriptionModel model.SubscriptionModel
	err := r.db.Where("viewer_id = ? AND creator_id = ?", viewerID, creatorID).First(&subscriptionModel).Error
	if err != nil {
		return nil, err
	}
	return ToSubscriptionEntity(&subscriptionModel), nil
}
