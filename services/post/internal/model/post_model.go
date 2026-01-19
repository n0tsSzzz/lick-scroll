package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostModel struct {
	ID           string         `gorm:"type:uuid;primary_key" json:"id"`
	CreatorID    string         `gorm:"type:uuid;not null;index" json:"creator_id"`
	Title        string         `gorm:"type:varchar(255);not null" json:"title"`
	Description  string         `gorm:"type:text" json:"description"`
	Type         string         `gorm:"type:varchar(20);not null" json:"type"`
	MediaURL     string         `gorm:"type:varchar(500)" json:"media_url"`
	ThumbnailURL string         `gorm:"type:varchar(500)" json:"thumbnail_url"`
	Category     string         `gorm:"type:varchar(100)" json:"category"`
	Status       string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Views        int            `gorm:"default:0" json:"views"`
	Purchases    int            `gorm:"default:0" json:"purchases"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Images       []PostImageModel `gorm:"foreignKey:PostID" json:"images,omitempty"`
}

func (p *PostModel) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

type PostImageModel struct {
	ID           string         `gorm:"type:uuid;primary_key" json:"id"`
	PostID       string         `gorm:"type:uuid;not null;index" json:"post_id"`
	ImageURL     string         `gorm:"type:varchar(500);not null" json:"image_url"`
	ThumbnailURL string         `gorm:"type:varchar(500)" json:"thumbnail_url"`
	Order        int            `gorm:"default:0;index" json:"order"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (pi *PostImageModel) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == "" {
		pi.ID = uuid.New().String()
	}
	return nil
}
