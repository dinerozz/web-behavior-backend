package response

import (
	"github.com/gofrs/uuid"
	"time"
)

type Organization struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type OrganizationWithMembers struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Members     []OrganizationMember `json:"members"`
}

type OrganizationMember struct {
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	Username string    `json:"username" db:"username"`
	Role     string    `json:"role" db:"role"`
	JoinedAt time.Time `json:"joined_at" db:"created_at"`
}

type UserOrganizations struct {
	UserID        uuid.UUID       `json:"user_id"`
	Organizations []UserOrgAccess `json:"organizations"`
}

type UserOrgAccess struct {
	ID          uuid.UUID `json:"id" db:"organization_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	Role        string    `json:"role" db:"role"`
	JoinedAt    time.Time `json:"joined_at" db:"created_at"`
}
