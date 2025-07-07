// internal/entity/extension_user.go
package entity

import (
	"github.com/gofrs/uuid"
	"time"
)

type ExtensionUser struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	Username   string     `json:"username" db:"username"`
	APIKey     string     `json:"apiKey" db:"api_key"`
	IsActive   bool       `json:"isActive" db:"is_active"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
	LastUsedAt *time.Time `json:"lastUsedAt" db:"last_used_at"`
}

type ExtensionUserPublic struct {
	ID         uuid.UUID  `json:"id"`
	Username   string     `json:"username"`
	IsActive   bool       `json:"isActive"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

type CreateExtensionUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
}

type UpdateExtensionUserRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=3,max=100"`
	IsActive *bool   `json:"isActive,omitempty"`
}

type RegenerateAPIKeyResponse struct {
	ID     uuid.UUID `json:"id"`
	APIKey string    `json:"apiKey"`
}

type ExtensionUserFilter struct {
	Username string `form:"username"`
	IsActive *bool  `form:"isActive"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}

type ExtensionUserStats struct {
	TotalUsers        int64 `json:"totalUsers"`
	ActiveUsers       int64 `json:"activeUsers"`
	InactiveUsers     int64 `json:"inactiveUsers"`
	UsersUsedToday    int64 `json:"usersUsedToday"`
	UsersUsedThisWeek int64 `json:"usersUsedThisWeek"`
}
