package repository

import (
	"lick-scroll/pkg/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepository interface {
	Create(post *models.Post) error
	GetByID(id string) (*models.Post, error)
	GetByCreatorID(creatorID string, limit, offset int) ([]*models.Post, error)
	List(limit, offset int, category string, status models.PostStatus) ([]*models.Post, error)
	Update(post *models.Post) error
	Delete(id string) error
	IncrementViews(id string) error
	IncrementPurchases(id string) error
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) GetByID(id string) (*models.Post, error) {
	var post models.Post
	if err := r.db.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) GetByCreatorID(creatorID string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := r.db.Where("creator_id = ?", creatorID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *postRepository) List(limit, offset int, category string, status models.PostStatus) ([]*models.Post, error) {
	var posts []*models.Post
	query := r.db.Where("status = ?", status).Order("created_at DESC")
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	
	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *postRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

func (r *postRepository) Delete(id string) error {
	return r.db.Delete(&models.Post{}, "id = ?", id).Error
}

func (r *postRepository) IncrementViews(id string) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).UpdateColumn("views", clause.Expr{SQL: "views + ?", Vars: []interface{}{1}}).Error
}

func (r *postRepository) IncrementPurchases(id string) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).UpdateColumn("purchases", clause.Expr{SQL: "purchases + ?", Vars: []interface{}{1}}).Error
}

