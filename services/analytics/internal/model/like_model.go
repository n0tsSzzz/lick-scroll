package model

import "time"

type LikeModel struct {
	ID        string     `gorm:"column:id;type:uuid;primaryKey"`
	UserID    string     `gorm:"column:user_id;type:uuid;not null"`
	PostID    string     `gorm:"column:post_id;type:uuid;not null"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamp"`
	DeletedAt *time.Time `gorm:"column:deleted_at;type:timestamp"`
}

func (LikeModel) TableName() string {
	return "likes"
}
