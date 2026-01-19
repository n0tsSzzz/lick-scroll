package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostImage struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	PostID    string    `gorm:"type:uuid;not null;index" json:"post_id"`
	ImageURL  string    `gorm:"not null" json:"image_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Order     int       `gorm:"default:0;index" json:"order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (pi *PostImage) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == "" {
		pi.ID = uuid.New().String()
	}
	return nil
}
