package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscription struct {
	ID         string         `gorm:"type:uuid;primary_key" json:"id"`
	ViewerID   string         `gorm:"type:uuid;not null;index" json:"viewer_id"`
	CreatorID  string         `gorm:"type:uuid;not null;index" json:"creator_id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

