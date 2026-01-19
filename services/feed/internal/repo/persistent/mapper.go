package persistent

import (
	"lick-scroll/services/feed/internal/entity"
	"lick-scroll/services/feed/internal/model"
)

func ToPostEntity(m *model.PostModel) *entity.Post {
	if m == nil {
		return nil
	}

	images := make([]entity.PostImage, len(m.Images))
	for i, img := range m.Images {
		images[i] = entity.PostImage{
			ID:           img.ID,
			PostID:       img.PostID,
			ImageURL:     img.ImageURL,
			ThumbnailURL: img.ThumbnailURL,
			Order:        img.Order,
		}
	}

	return &entity.Post{
		ID:           m.ID,
		CreatorID:    m.CreatorID,
		Title:        m.Title,
		Description:  m.Description,
		Type:         m.Type,
		MediaURL:     m.MediaURL,
		ThumbnailURL: m.ThumbnailURL,
		Category:     m.Category,
		Status:       m.Status,
		Views:        m.Views,
		Purchases:    m.Purchases,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Images:       images,
	}
}

func ToPostModel(e *entity.Post) *model.PostModel {
	if e == nil {
		return nil
	}

	images := make([]model.PostImageModel, len(e.Images))
	for i, img := range e.Images {
		images[i] = model.PostImageModel{
			ID:           img.ID,
			PostID:       img.PostID,
			ImageURL:     img.ImageURL,
			ThumbnailURL: img.ThumbnailURL,
			Order:        img.Order,
		}
	}

	return &model.PostModel{
		ID:           e.ID,
		CreatorID:    e.CreatorID,
		Title:        e.Title,
		Description:  e.Description,
		Type:         e.Type,
		MediaURL:     e.MediaURL,
		ThumbnailURL: e.ThumbnailURL,
		Category:     e.Category,
		Status:       e.Status,
		Views:        e.Views,
		Purchases:    e.Purchases,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		Images:       images,
	}
}
