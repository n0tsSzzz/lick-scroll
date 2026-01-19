package persistent

import (
	"lick-scroll/services/interaction/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InteractionRepository interface {
	CreateLike(userID, postID string) error
	DeleteLike(userID, postID string) error
	IsLiked(userID, postID string) (bool, error)
	GetLikedPosts(userID string, limit, offset int) ([]map[string]interface{}, error)
	GetLikeCount(postID string) (int64, error)
	IncrementViews(postID string) error
	GetViewCount(postID string) (int64, error)
}

type interactionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) InteractionRepository {
	return &interactionRepository{db: db}
}

func (r *interactionRepository) CreateLike(userID, postID string) error {
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

func (r *interactionRepository) DeleteLike(userID, postID string) error {
	return r.db.Unscoped().Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.LikeModel{}).Error
}

func (r *interactionRepository) IsLiked(userID, postID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.LikeModel{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}

func (r *interactionRepository) GetLikedPosts(userID string, limit, offset int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := r.db.Table("posts").
		Select("posts.*, post_images.id as image_id, post_images.image_url, post_images.thumbnail_url, post_images.\"order\" as image_order").
		Joins("INNER JOIN likes ON posts.id = likes.post_id").
		Joins("LEFT JOIN post_images ON posts.id = post_images.post_id").
		Where("likes.user_id = ? AND likes.deleted_at IS NULL AND posts.deleted_at IS NULL", userID).
		Order("likes.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var postID, creatorID, title, description, postType, mediaURL, thumbnailURL, category, status string
		var views, purchases int
		var createdAt, updatedAt interface{}
		var imageID, imageURL, imageThumbnailURL interface{}
		var imageOrder interface{}

		if err := rows.Scan(&postID, &creatorID, &title, &description, &postType, &mediaURL, &thumbnailURL, &category, &status, &views, &purchases, &createdAt, &updatedAt, &imageID, &imageURL, &imageThumbnailURL, &imageOrder); err != nil {
			continue
		}

		if _, exists := postMap[postID]; !exists {
			postMap[postID] = map[string]interface{}{
				"id":           postID,
				"creator_id":   creatorID,
				"title":        title,
				"description": description,
				"type":         postType,
				"media_url":    mediaURL,
				"thumbnail_url": thumbnailURL,
				"category":     category,
				"status":       status,
				"views":        views,
				"purchases":    purchases,
				"created_at":   createdAt,
				"updated_at":   updatedAt,
				"images":       []map[string]interface{}{},
			}
		}

		if imageID != nil {
			images := postMap[postID]["images"].([]map[string]interface{})
			images = append(images, map[string]interface{}{
				"id":           imageID,
				"post_id":      postID,
				"image_url":    imageURL,
				"thumbnail_url": imageThumbnailURL,
				"order":        imageOrder,
			})
			postMap[postID]["images"] = images
		}
	}

	results = make([]map[string]interface{}, 0, len(postMap))
	for _, post := range postMap {
		results = append(results, post)
	}

	return results, nil
}

func (r *interactionRepository) GetLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.LikeModel{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *interactionRepository) IncrementViews(postID string) error {
	return r.db.Table("posts").Where("id = ?", postID).UpdateColumn("views", clause.Expr{SQL: "views + ?", Vars: []interface{}{1}}).Error
}

func (r *interactionRepository) GetViewCount(postID string) (int64, error) {
	var views int
	if err := r.db.Table("posts").Select("views").Where("id = ?", postID).Scan(&views).Error; err != nil {
		return 0, err
	}
	return int64(views), nil
}
