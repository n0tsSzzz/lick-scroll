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
	// Like methods
	CreateLike(userID, postID string) error
	DeleteLike(userID, postID string) error
	IsLiked(userID, postID string) (bool, error)
	GetLikedPosts(userID string, limit, offset int) ([]*models.Post, error)
	GetLikeCount(postID string) (int64, error)
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

func (r *postRepository) CreateLike(userID, postID string) error {
	like := &models.Like{
		UserID: userID,
		PostID: postID,
	}
	return r.db.Create(like).Error
}

func (r *postRepository) DeleteLike(userID, postID string) error {
	return r.db.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&models.Like{}).Error
}

func (r *postRepository) IsLiked(userID, postID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Like{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}

func (r *postRepository) GetLikedPosts(userID string, limit, offset int) ([]*models.Post, error) {
	var posts []*models.Post
	query := r.db.Table("posts").
		Joins("INNER JOIN likes ON posts.id = likes.post_id").
		Where("likes.user_id = ? AND likes.deleted_at IS NULL", userID).
		Order("likes.created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	
	err := query.Find(&posts).Error
	return posts, err
}

func (r *postRepository) GetLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Like{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

