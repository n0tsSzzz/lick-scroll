package persistent

import (
	"lick-scroll/services/interaction/internal/entity"
	"lick-scroll/services/interaction/internal/model"
)

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
