package repository

import (
	"lick-scroll/pkg/models"

	"gorm.io/gorm"
)

type ModerationRepository interface {
	GetPostByID(id string) (*models.Post, error)
	GetPendingPosts(limit, offset int) ([]*models.Post, error)
	UpdatePostStatus(id string, status models.PostStatus) error
}

type moderationRepository struct {
	db *gorm.DB
}

func NewModerationRepository(db *gorm.DB) ModerationRepository {
	return &moderationRepository{db: db}
}

func (r *moderationRepository) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	if err := r.db.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *moderationRepository) GetPendingPosts(limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := r.db.Where("status = ?", models.StatusPending).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *moderationRepository) UpdatePostStatus(id string, status models.PostStatus) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).Update("status", status).Error
}

