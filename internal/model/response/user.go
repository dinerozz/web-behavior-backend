package response

import (
	"github.com/gofrs/uuid"
	"time"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Password     *string    `json:"-"`
	IsSuperAdmin *bool      `json:"is_super_admin,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}
