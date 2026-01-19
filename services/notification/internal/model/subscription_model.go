package model

import "time"

type SubscriptionModel struct {
	ID        string     `gorm:"column:id;type:uuid;primaryKey"`
	CreatorID string     `gorm:"column:creator_id;type:uuid;not null"`
	ViewerID  string     `gorm:"column:viewer_id;type:uuid;not null"`
	DeletedAt *time.Time `gorm:"column:deleted_at;type:timestamp"`
}

func (SubscriptionModel) TableName() string {
	return "subscriptions"
}
