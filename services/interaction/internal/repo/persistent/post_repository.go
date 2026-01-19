package persistent

import (
	"gorm.io/gorm"
)

type PostRepository interface {
	PostExists(postID string) (bool, error)
	GetCreatorID(postID string) (string, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) PostExists(postID string) (bool, error) {
	var count int64
	err := r.db.Table("posts").Where("id = ? AND deleted_at IS NULL", postID).Count(&count).Error
	return count > 0, err
}

func (r *postRepository) GetCreatorID(postID string) (string, error) {
	var creatorID string
	err := r.db.Table("posts").Select("creator_id").Where("id = ?", postID).Scan(&creatorID).Error
	return creatorID, err
}
