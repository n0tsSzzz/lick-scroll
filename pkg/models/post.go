package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostStatus string

const (
	StatusPending   PostStatus = "pending"
	StatusApproved  PostStatus = "approved"
	StatusRejected  PostStatus = "rejected"
)

type PostType string

const (
	PostTypePhoto PostType = "photo"
	PostTypeVideo PostType = "video"
)

type Post struct {
	ID          string    `gorm:"type:uuid;primary_key" json:"id"`
	CreatorID   string    `gorm:"type:uuid;not null;index" json:"creator_id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `json:"description"`
	Type        PostType  `gorm:"type:varchar(10);not null" json:"type"`
	MediaURL    string    `gorm:"not null" json:"media_url"` // Deprecated
	ThumbnailURL string   `json:"thumbnail_url"` // Deprecated
	Category    string    `gorm:"index" json:"category"`
	Status      PostStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Views       int       `gorm:"default:0" json:"views"`
	Purchases   int       `gorm:"default:0" json:"purchases"`
	Images      []PostImage `gorm:"foreignKey:PostID" json:"images"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

