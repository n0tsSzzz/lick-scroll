package persistent

import (
	"database/sql"

	"gorm.io/gorm"
)

type FeedRepository interface {
	GetPostsByCreatorIDs(creatorIDs []string, limit int) ([]map[string]interface{}, error)
	GetOtherPosts(userID string, excludeCreatorIDs []string, limit int) ([]map[string]interface{}, error)
	IsLiked(userID, postID string) (bool, error)
	GetLikeCount(postID string) (int64, error)
	GetCreatorInfo(creatorID string) (map[string]interface{}, error)
}

type feedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) FeedRepository {
	return &feedRepository{db: db}
}

func (r *feedRepository) GetPostsByCreatorIDs(creatorIDs []string, limit int) ([]map[string]interface{}, error) {
	if len(creatorIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	query := r.db.Table("posts").
		Select("posts.id, posts.creator_id, posts.title, posts.description, posts.type, posts.media_url, posts.thumbnail_url, posts.category, posts.status, posts.views, posts.purchases, posts.created_at, posts.updated_at, post_images.id as image_id, post_images.image_url, post_images.thumbnail_url, post_images.\"order\" as image_order").
		Joins("LEFT JOIN post_images ON posts.id = post_images.post_id").
		Where("posts.creator_id IN ? AND posts.deleted_at IS NULL AND (posts.status IS NULL OR posts.status = '' OR posts.status != ?)", creatorIDs, "rejected").
		Order("posts.created_at DESC").
		Limit(limit)

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPostsFromRows(rows), nil
}

func (r *feedRepository) GetOtherPosts(userID string, excludeCreatorIDs []string, limit int) ([]map[string]interface{}, error) {
	query := r.db.Table("posts").
		Select("posts.id, posts.creator_id, posts.title, posts.description, posts.type, posts.media_url, posts.thumbnail_url, posts.category, posts.status, posts.views, posts.purchases, posts.created_at, posts.updated_at, post_images.id as image_id, post_images.image_url, post_images.thumbnail_url, post_images.\"order\" as image_order").
		Joins("LEFT JOIN post_images ON posts.id = post_images.post_id").
		Where("posts.deleted_at IS NULL AND (posts.status IS NULL OR posts.status = '' OR posts.status != ?)", "rejected").
		Order("posts.created_at DESC").
		Limit(limit)

	if userID != "" {
		query = query.Where("posts.creator_id != ?", userID)
	}

	if len(excludeCreatorIDs) > 0 {
		query = query.Where("posts.creator_id NOT IN ?", excludeCreatorIDs)
	}

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPostsFromRows(rows), nil
}

func (r *feedRepository) IsLiked(userID, postID string) (bool, error) {
	var count int64
	err := r.db.Table("likes").Where("user_id = ? AND post_id = ? AND deleted_at IS NULL", userID, postID).Count(&count).Error
	return count > 0, err
}

func (r *feedRepository) GetLikeCount(postID string) (int64, error) {
	var count int64
	err := r.db.Table("likes").Where("post_id = ? AND deleted_at IS NULL", postID).Count(&count).Error
	return count, err
}

func (r *feedRepository) GetCreatorInfo(creatorID string) (map[string]interface{}, error) {
	var creator struct {
		AvatarURL string
		Username  string
	}
	err := r.db.Table("users").Select("avatar_url, username").Where("id = ?", creatorID).Scan(&creator).Error
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"avatar_url": creator.AvatarURL,
		"username":   creator.Username,
	}, nil
}

func (r *feedRepository) scanPostsFromRows(rows *sql.Rows) []map[string]interface{} {
	postMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var postID, creatorID, title, description, postType, mediaURL, thumbnailURL, category, status sql.NullString
		var views, purchases sql.NullInt32
		var createdAt, updatedAt sql.NullTime
		var imageID, imageURL, imageThumbnailURL sql.NullString
		var imageOrder sql.NullInt32

		if err := rows.Scan(&postID, &creatorID, &title, &description, &postType, &mediaURL, &thumbnailURL, &category, &status, &views, &purchases, &createdAt, &updatedAt, &imageID, &imageURL, &imageThumbnailURL, &imageOrder); err != nil {
			continue
		}

		if _, exists := postMap[postID.String]; !exists {
			postMap[postID.String] = map[string]interface{}{
				"id":           postID.String,
				"creator_id":   creatorID.String,
				"title":        title.String,
				"description":  description.String,
				"type":         postType.String,
				"media_url":    mediaURL.String,
				"thumbnail_url": thumbnailURL.String,
				"category":     category.String,
				"status":       status.String,
				"views":        int(views.Int32),
				"purchases":    int(purchases.Int32),
				"created_at":   createdAt.Time,
				"updated_at":   updatedAt.Time,
				"images":       []map[string]interface{}{},
			}
		}

		if imageID.Valid {
			images := postMap[postID.String]["images"].([]map[string]interface{})
			images = append(images, map[string]interface{}{
				"id":           imageID.String,
				"post_id":      postID.String,
				"image_url":    imageURL.String,
				"thumbnail_url": imageThumbnailURL.String,
				"order":        int(imageOrder.Int32),
			})
			postMap[postID.String]["images"] = images
		}
	}

	results := make([]map[string]interface{}, 0, len(postMap))
	for _, post := range postMap {
		results = append(results, post)
	}

	return results
}
