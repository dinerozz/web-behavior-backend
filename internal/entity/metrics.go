// entity/metrics_service.go
package entity

import "time"

type TrackedTimeMetric struct {
	UserID       string    `json:"user_id"`
	TotalMinutes float64   `json:"total_minutes"`
	TotalHours   float64   `json:"total_hours"`
	Sessions     int       `json:"sessions"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Period       string    `json:"period"`
}

type TrackedTimeFilter struct {
	UserID    string    `form:"user_id" json:"user_id" binding:"required"`
	StartTime time.Time `form:"start_time" json:"start_time" binding:"required"`
	EndTime   time.Time `form:"end_time" json:"end_time" binding:"required"`
	SessionID *string   `form:"session_id" json:"session_id,omitempty"`
}

type TrackedTimeResponse struct {
	Data    *TrackedTimeMetric `json:"data"`
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
}

type EngagedTimeMetric struct {
	UserID        string  `json:"user_id"`
	ActiveMinutes int     `json:"active_minutes"`
	ActiveHours   float64 `json:"active_hours"`
	ActiveEvents  int     `json:"active_events"`

	TrackedMinutes float64 `json:"tracked_minutes"`
	TrackedHours   float64 `json:"tracked_hours"`
	EngagementRate float64 `json:"engagement_rate"` // active_minutes / tracked_minutes * 100

	Sessions  int       `json:"sessions"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Period    string    `json:"period"`
}

type EngagedTimeFilter struct {
	UserID    string    `form:"user_id" json:"user_id" binding:"required"`
	StartTime time.Time `form:"start_time" json:"start_time" binding:"required"`
	EndTime   time.Time `form:"end_time" json:"end_time" binding:"required"`
	SessionID *string   `form:"session_id" json:"session_id,omitempty"`
}

type EngagedTimeResponse struct {
	Data    *EngagedTimeMetric `json:"data"`
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
}
