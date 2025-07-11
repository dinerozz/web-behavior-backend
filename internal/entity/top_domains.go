// entity/top_domains.go
package entity

import "time"

type TopDomainsFilter struct {
	UserID    string  `json:"user_id"`
	Limit     int     `json:"limit,omitempty"` // По умолчанию 10
	SessionID *string `json:"session_id,omitempty"`
}

type DomainStats struct {
	Domain        string    `json:"domain"`
	EventsCount   int       `json:"events_count"`
	ActiveMinutes int       `json:"active_minutes"`
	Percentage    float64   `json:"percentage"`
	FirstVisit    time.Time `json:"first_visit"`
	LastVisit     time.Time `json:"last_visit"`
}

type TopDomainsResponse struct {
	UserID       string        `json:"user_id"`
	TotalDomains int           `json:"total_domains"`
	TotalEvents  int           `json:"total_events"`
	Domains      []DomainStats `json:"domains"`
}
