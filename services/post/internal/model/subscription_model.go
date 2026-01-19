package model

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionModel struct {
	ID        string         `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ViewerID  string         `gorm:"type:uuid;not null" json:"viewer_id"`
	CreatorID string         `gorm:"type:uuid;not null" json:"creator_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SubscriptionModel) TableName() string {
	return "subscriptions"
}
