package response

type PaginationMeta struct {
    CurrentPage  int `json:"current_page"`
    PerPage     int `json:"per_page"`
    TotalItems  int `json:"total_items"`
    TotalPages  int `json:"total_pages"`
}