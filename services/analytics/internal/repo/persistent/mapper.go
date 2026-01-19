package persistent

import (
	"lick-scroll/services/analytics/internal/entity"
	"lick-scroll/services/analytics/internal/model"
)

func ToPostEntity(m *model.PostModel) *entity.Post {
	if m == nil {
		return nil
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
	}
}

func ToPostEntities(models []model.PostModel) []*entity.Post {
	results := make([]*entity.Post, len(models))
	for i := range models {
		results[i] = ToPostEntity(&models[i])
	}
	return results
}

func ToPostModel(e *entity.Post) *model.PostModel {
	if e == nil {
		return nil
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
	}
}
