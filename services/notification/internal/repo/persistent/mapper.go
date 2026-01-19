package persistent

import (
	"lick-scroll/services/notification/internal/model"
)

func ToUserEntity(m *model.UserModel) string {
	if m == nil {
		return ""
	}
	return m.Username
}

func ToSubscriptionEntity(models []model.SubscriptionModel) []string {
	viewerIDs := make([]string, len(models))
	for i, sub := range models {
		viewerIDs[i] = sub.ViewerID
	}
	return viewerIDs
}
