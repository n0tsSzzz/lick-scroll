package entity

import "time"

type Subscription struct {
	ID        string    `json:"id"`
	ViewerID  string    `json:"viewer_id"`
	CreatorID string    `json:"creator_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
