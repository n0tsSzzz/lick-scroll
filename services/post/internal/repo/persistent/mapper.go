package persistent

import (
	"lick-scroll/services/post/internal/entity"
	"lick-scroll/services/post/internal/model"
)

func ToPostEntity(m *model.PostModel) *entity.Post {
	if m == nil {
		return nil
	}

	post := &entity.Post{
		ID:           m.ID,
		CreatorID:    m.CreatorID,
		Title:        m.Title,
		Description:  m.Description,
		Type:         entity.PostType(m.Type),
		MediaURL:     m.MediaURL,
		ThumbnailURL: m.ThumbnailURL,
		Category:     m.Category,
		Status:       entity.PostStatus(m.Status),
		Views:        m.Views,
		Purchases:    m.Purchases,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if len(m.Images) > 0 {
		post.Images = make([]entity.PostImage, len(m.Images))
		for i, img := range m.Images {
			post.Images[i] = ToPostImageEntity(&img)
		}
	}

	return post
}

func ToPostModel(e *entity.Post) *model.PostModel {
	if e == nil {
		return nil
	}

	post := &model.PostModel{
		ID:           e.ID,
		CreatorID:    e.CreatorID,
		Title:        e.Title,
		Description:  e.Description,
		Type:         string(e.Type),
		MediaURL:     e.MediaURL,
		ThumbnailURL: e.ThumbnailURL,
		Category:     e.Category,
		Status:       string(e.Status),
		Views:        e.Views,
		Purchases:    e.Purchases,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}

	if len(e.Images) > 0 {
		post.Images = make([]model.PostImageModel, len(e.Images))
		for i, img := range e.Images {
			post.Images[i] = *ToPostImageModel(&img)
		}
	}

	return post
}

func ToPostImageEntity(m *model.PostImageModel) entity.PostImage {
	if m == nil {
		return entity.PostImage{}
	}

	return entity.PostImage{
		ID:           m.ID,
		PostID:       m.PostID,
		ImageURL:     m.ImageURL,
		ThumbnailURL: m.ThumbnailURL,
		Order:        m.Order,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func ToPostImageModel(e *entity.PostImage) *model.PostImageModel {
	if e == nil {
		return nil
	}

	return &model.PostImageModel{
		ID:           e.ID,
		PostID:       e.PostID,
		ImageURL:     e.ImageURL,
		ThumbnailURL: e.ThumbnailURL,
		Order:        e.Order,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func ToLikeEntity(m *model.LikeModel) *entity.Like {
	if m == nil {
		return nil
	}

	return &entity.Like{
		ID:        m.ID,
		UserID:    m.UserID,
		PostID:    m.PostID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToLikeModel(e *entity.Like) *model.LikeModel {
	if e == nil {
		return nil
	}

	return &model.LikeModel{
		ID:        e.ID,
		UserID:    e.UserID,
		PostID:    e.PostID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func ToSubscriptionEntity(m *model.SubscriptionModel) *entity.Subscription {
	if m == nil {
		return nil
	}

	return &entity.Subscription{
		ID:        m.ID,
		ViewerID:  m.ViewerID,
		CreatorID: m.CreatorID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToSubscriptionModel(e *entity.Subscription) *model.SubscriptionModel {
	if e == nil {
		return nil
	}

	return &model.SubscriptionModel{
		ID:        e.ID,
		ViewerID:  e.ViewerID,
		CreatorID: e.CreatorID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
