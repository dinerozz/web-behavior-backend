package entity

import "github.com/gofrs/uuid"

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}
