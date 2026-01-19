package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LikeModel struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string         `gorm:"type:uuid;not null;index" json:"user_id"`
	PostID    string         `gorm:"type:uuid;not null;index" json:"post_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LikeModel) TableName() string {
	return "likes"
}

func (l *LikeModel) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}
