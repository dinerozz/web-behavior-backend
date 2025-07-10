package entity

type PeriodInfo struct {
	Key         string `json:"key" db:"key"`
	Label       string `json:"label" db:"label"`
	Description string `json:"description" db:"description"`
}
