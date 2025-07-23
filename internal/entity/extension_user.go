// internal/entity/extension_user.go
package entity

import (
	"github.com/gofrs/uuid"
	"time"
)

type ExtensionUser struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	Username       string            `json:"username" db:"username"`
	APIKey         string            `json:"apiKey" db:"api_key"`
	IsActive       bool              `json:"isActive" db:"is_active"`
	CreatedAt      time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time         `json:"updatedAt" db:"updated_at"`
	LastUsedAt     *time.Time        `json:"lastUsedAt" db:"last_used_at"`
	OrganizationID uuid.UUID         `json:"organization_id,omitzero" db:"organization_id"`
	Organization   *OrganizationInfo `json:"organization,omitempty"`
}

type ExtensionUserPublic struct {
	ID           uuid.UUID         `json:"id"`
	Username     string            `json:"username"`
	IsActive     bool              `json:"isActive"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastUsedAt   *time.Time        `json:"lastUsedAt"`
	Organization *OrganizationInfo `json:"organization,omitempty"`
}

type OrganizationInfo struct {
	ID   *uuid.UUID `json:"id"`
	Name string     `json:"name"`
}

type CreateExtensionUserRequest struct {
	Username       string     `json:"username" binding:"required,min=3,max=100"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`
}

type UpdateExtensionUserRequest struct {
	Username       *string    `json:"username,omitempty" binding:"omitempty,min=3,max=100"`
	IsActive       *bool      `json:"isActive,omitempty"`
	APIKey         *string    `json:"apiKey,omitempty"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`
}

type RegenerateAPIKeyResponse struct {
	ID     uuid.UUID `json:"id"`
	APIKey string    `json:"apiKey"`
}

type ExtensionUserFilter struct {
	Username       string     `form:"username" json:"username"`
	IsActive       *bool      `form:"isActive" json:"is_active"`
	OrganizationID *uuid.UUID `form:"organization_id" json:"organization_id"`
	Limit          int        `form:"limit" json:"limit"`
	Offset         int        `form:"offset" json:"offset"`
	Page           int        `form:"page" json:"page"`
	PerPage        int        `form:"per_page" json:"per_page"`
}

type ExtensionUserStats struct {
	TotalUsers        int64 `json:"totalUsers"`
	ActiveUsers       int64 `json:"activeUsers"`
	InactiveUsers     int64 `json:"inactiveUsers"`
	UsersUsedToday    int64 `json:"usersUsedToday"`
	UsersUsedThisWeek int64 `json:"usersUsedThisWeek"`
}
