package model

import "time"

type PostModel struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey"`
	CreatorID  string    `gorm:"column:creator_id;type:uuid;not null"`
	Title      string    `gorm:"column:title;type:varchar(255)"`
	Description string   `gorm:"column:description;type:text"`
	Type       string    `gorm:"column:type;type:varchar(50)"`
	MediaURL   string    `gorm:"column:media_url;type:text"`
	ThumbnailURL string  `gorm:"column:thumbnail_url;type:text"`
	Category   string    `gorm:"column:category;type:varchar(100)"`
	Status     string    `gorm:"column:status;type:varchar(50)"`
	Views      int       `gorm:"column:views;type:integer;default:0"`
	Purchases  int       `gorm:"column:purchases;type:integer;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp"`
	DeletedAt  *time.Time `gorm:"column:deleted_at;type:timestamp"`
}

func (PostModel) TableName() string {
	return "posts"
}
