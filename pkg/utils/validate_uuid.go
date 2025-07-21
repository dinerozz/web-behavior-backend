package utils

import "github.com/gofrs/uuid"

func ValidateUUID(uuidStr string) bool {
	_, err := uuid.FromString(uuidStr)
	return err == nil
}
