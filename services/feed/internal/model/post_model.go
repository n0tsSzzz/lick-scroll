package model

import "time"

type PostModel struct {
	ID           string           `gorm:"type:uuid;primary_key"`
	CreatorID    string           `gorm:"type:uuid;not null;index"`
	Title        string           `gorm:"type:varchar(255);not null"`
	Description  string           `gorm:"type:text"`
	Type         string           `gorm:"type:varchar(20);not null"`
	MediaURL     string           `gorm:"type:varchar(500)"`
	ThumbnailURL string           `gorm:"type:varchar(500)"`
	Category     string           `gorm:"type:varchar(100)"`
	Status       string           `gorm:"type:varchar(20);default:'pending'"`
	Views        int              `gorm:"default:0"`
	Purchases    int              `gorm:"default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time       `gorm:"index"`
	Images       []PostImageModel `gorm:"foreignKey:PostID"`
}

func (PostModel) TableName() string {
	return "posts"
}

type PostImageModel struct {
	ID           string `gorm:"type:uuid;primary_key"`
	PostID       string `gorm:"type:uuid;not null;index"`
	ImageURL     string `gorm:"type:varchar(500);not null"`
	ThumbnailURL string `gorm:"type:varchar(500)"`
	Order        int    `gorm:"default:0;index"`
}

func (PostImageModel) TableName() string {
	return "post_images"
}
