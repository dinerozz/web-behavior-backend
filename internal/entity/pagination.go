package entity

type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Success    bool           `json:"success"`
	Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
