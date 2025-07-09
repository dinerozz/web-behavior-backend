package wrapper

import (
	"github.com/dinerozz/web-behavior-backend/internal/entity"
)

type ResponseWrapper struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
}

type PaginatedResponseWrapper struct {
	Data    interface{}           `json:"data"`
	Meta    entity.PaginationInfo `json:"meta"`
	Success bool                  `json:"success"`
}

type SuccessWrapper struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
