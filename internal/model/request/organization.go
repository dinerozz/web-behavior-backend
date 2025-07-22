package request

type CreateOrganization struct {
	Name        string `json:"name" binding:"required" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=255"`
}

type UpdateOrganization struct {
	Name        string `json:"name" binding:"required" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=255"`
}

type AddUserToOrganization struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required" validate:"oneof=admin member viewer"`
}
