package persistent

import (
	"lick-scroll/services/post/internal/entity"
	"lick-scroll/services/post/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepository interface {
	Create(post *entity.Post) error
	GetByID(id string) (*entity.Post, error)
	GetByCreatorID(creatorID string, limit, offset int) ([]*entity.Post, error)
	List(limit, offset int, category string, status entity.PostStatus) ([]*entity.Post, error)
	Update(post *entity.Post) error
	Delete(id string) error
	IncrementViews(id string) error
	IncrementPurchases(id string) error
	CreateLike(userID, postID string) error
	DeleteLike(userID, postID string) error
	IsLiked(userID, postID string) (bool, error)
	GetLikedPosts(userID string, limit, offset int) ([]*entity.Post, error)
	GetLikeCount(postID string) (int64, error)
	GetSubscription(userID, creatorID string) (*entity.Subscription, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *entity.Post) error {
	postModel := ToPostModel(post)
	if postModel.ID == "" {
		postModel.ID = uuid.New().String()
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		images := postModel.Images
		postModel.Images = nil

		if err := tx.Create(postModel).Error; err != nil {
			return err
		}

		if len(images) > 0 {
			usedIDs := make(map[string]bool)
			for i := range images {
				images[i].PostID = postModel.ID
				if images[i].ID == "" {
					images[i].ID = uuid.New().String()
				}
				for usedIDs[images[i].ID] {
					images[i].ID = uuid.New().String()
				}
				usedIDs[images[i].ID] = true

				if err := tx.Create(&images[i]).Error; err != nil {
					return err
				}
			}
			postModel.Images = images
		}

		*post = *ToPostEntity(postModel)
		return nil
	})
}

func (r *postRepository) GetByID(id string) (*entity.Post, error) {
	var postModel model.PostModel
	if err := r.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("post_images.order ASC")
	}).Where("id = ?", id).First(&postModel).Error; err != nil {
		return nil, err
	}
	return ToPostEntity(&postModel), nil
}

func (r *postRepository) GetByCreatorID(creatorID string, limit, offset int) ([]*entity.Post, error) {
	var postModels []model.PostModel
	query := r.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("post_images.order ASC")
	}).Where("creator_id = ?", creatorID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	posts := make([]*entity.Post, len(postModels))
	for i := range postModels {
		posts[i] = ToPostEntity(&postModels[i])
	}
	return posts, nil
}

func (r *postRepository) List(limit, offset int, category string, status entity.PostStatus) ([]*entity.Post, error) {
	var postModels []model.PostModel
	query := r.db.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("post_images.order ASC")
	}).Where("status = ?", string(status)).Order("created_at DESC")

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	posts := make([]*entity.Post, len(postModels))
	for i := range postModels {
		posts[i] = ToPostEntity(&postModels[i])
	}
	return posts, nil
}

func (r *postRepository) Update(post *entity.Post) error {
	postModel := ToPostModel(post)
	return r.db.Save(postModel).Error
}

func (r *postRepository) Delete(id string) error {
	return r.db.Delete(&model.PostModel{}, "id = ?", id).Error
}

func (r *postRepository) IncrementViews(id string) error {
	return r.db.Model(&model.PostModel{}).Where("id = ?", id).UpdateColumn("views", clause.Expr{SQL: "views + ?", Vars: []interface{}{1}}).Error
}

func (r *postRepository) IncrementPurchases(id string) error {
	return r.db.Model(&model.PostModel{}).Where("id = ?", id).UpdateColumn("purchases", clause.Expr{SQL: "purchases + ?", Vars: []interface{}{1}}).Error
}

func (r *postRepository) CreateLike(userID, postID string) error {
	var existing model.LikeModel
	err := r.db.Unscoped().Where("user_id = ? AND post_id = ?", userID, postID).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			if err := r.db.Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			return nil
		}
		return nil
	}

	likeModel := &model.LikeModel{
		ID:     uuid.New().String(),
		UserID: userID,
		PostID: postID,
	}
	return r.db.Create(likeModel).Error
}

func (r *postRepository) DeleteLike(userID, postID string) error {
	return r.db.Unscoped().Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.LikeModel{}).Error
}

func (r *postRepository) IsLiked(userID, postID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.LikeModel{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}

func (r *postRepository) GetLikedPosts(userID string, limit, offset int) ([]*entity.Post, error) {
	var postModels []model.PostModel
	query := r.db.Model(&model.PostModel{}).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("post_images.order ASC")
		}).
		Joins("INNER JOIN likes ON posts.id = likes.post_id").
		Where("likes.user_id = ? AND likes.deleted_at IS NULL", userID).
		Order("likes.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&postModels).Error; err != nil {
		return nil, err
	}

	posts := make([]*entity.Post, len(postModels))
	for i := range postModels {
		posts[i] = ToPostEntity(&postModels[i])
	}
	return posts, nil
}

func (r *postRepository) GetLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.LikeModel{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *postRepository) GetSubscription(userID, creatorID string) (*entity.Subscription, error) {
	var subscriptionModel model.SubscriptionModel
	err := r.db.Where("viewer_id = ? AND creator_id = ?", userID, creatorID).First(&subscriptionModel).Error
	if err != nil {
		return nil, err
	}
	return ToSubscriptionEntity(&subscriptionModel), nil
}
