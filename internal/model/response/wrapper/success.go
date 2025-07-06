package wrapper

import "github.com/dinerozz/web-behavior-backend/internal/model/response"

type ResponseWrapper struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
}

type PaginatedResponseWrapper struct {
	Data    interface{}             `json:"data"`
	Meta    response.PaginationMeta `json:"meta"`
	Success bool                    `json:"success"`
}

type SuccessWrapper struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
