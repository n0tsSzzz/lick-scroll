package entity

import "time"

type PostType string

const (
	PostTypePhoto PostType = "photo"
	PostTypeVideo PostType = "video"
)

type PostStatus string

const (
	StatusPending  PostStatus = "pending"
	StatusApproved PostStatus = "approved"
	StatusRejected PostStatus = "rejected"
)

type Post struct {
	ID           string      `json:"id"`
	CreatorID   string      `json:"creator_id"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	Type         PostType    `json:"type"`
	MediaURL     string      `json:"media_url"`
	ThumbnailURL string      `json:"thumbnail_url"`
	Category     string      `json:"category"`
	Status       PostStatus  `json:"status"`
	Views        int         `json:"views"`
	Purchases    int         `json:"purchases"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Images       []PostImage `json:"images,omitempty"`
}

type PostImage struct {
	ID           string    `json:"id"`
	PostID       string    `json:"post_id"`
	ImageURL     string    `json:"image_url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Order        int       `json:"order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
