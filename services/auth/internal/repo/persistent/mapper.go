package persistent

import (
	"lick-scroll/services/auth/internal/entity"
	"lick-scroll/services/auth/internal/model"
)

func ToUserEntity(m *model.UserModel) *entity.User {
	if m == nil {
		return nil
	}

	return &entity.User{
		ID:        m.ID,
		Email:     m.Email,
		Username:  m.Username,
		Password:  m.Password,
		AvatarURL: m.AvatarURL,
		Role:      entity.UserRole(m.Role),
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToUserModel(e *entity.User) *model.UserModel {
	if e == nil {
		return nil
	}

	return &model.UserModel{
		ID:        e.ID,
		Email:     e.Email,
		Username:  e.Username,
		Password:  e.Password,
		AvatarURL: e.AvatarURL,
		Role:      string(e.Role),
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func ToSubscriptionEntity(m *model.SubscriptionModel) *entity.Subscription {
	if m == nil {
		return nil
	}

	return &entity.Subscription{
		ID:        m.ID,
		ViewerID:  m.ViewerID,
		CreatorID: m.CreatorID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToSubscriptionModel(e *entity.Subscription) *model.SubscriptionModel {
	if e == nil {
		return nil
	}

	return &model.SubscriptionModel{
		ID:        e.ID,
		ViewerID:  e.ViewerID,
		CreatorID: e.CreatorID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
