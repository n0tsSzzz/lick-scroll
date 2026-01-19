package entity

import "time"

type UserRole string

const (
	RoleViewer   UserRole = "viewer"
	RoleCreator  UserRole = "creator"
	RoleModerator UserRole = "moderator"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	AvatarURL string    `json:"avatar_url"`
	Role      UserRole  `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
