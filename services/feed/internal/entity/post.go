package entity

import "time"

type Post struct {
	ID           string
	CreatorID    string
	Title        string
	Description  string
	Type         string
	MediaURL     string
	ThumbnailURL string
	Category     string
	Status       string
	Views        int
	Purchases    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Images       []PostImage
}

type PostImage struct {
	ID           string
	PostID       string
	ImageURL     string
	ThumbnailURL string
	Order        int
}
