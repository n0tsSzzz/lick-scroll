package entity

// Notification represents a notification sent to a user
type Notification struct {
	UserID    string                 `json:"user_id"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt string                 `json:"created_at"`
}
