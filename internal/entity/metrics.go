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
	UserID         string    `json:"user_id" db:"user_id"`
	ActiveMinutes  int       `json:"active_minutes" db:"active_minutes"`
	ActiveHours    float64   `json:"active_hours" db:"active_hours"`
	ActiveEvents   int       `json:"active_events" db:"active_events"`
	Sessions       int       `json:"sessions" db:"sessions"`
	TrackedMinutes float64   `json:"tracked_minutes" db:"tracked_minutes"`
	TrackedHours   float64   `json:"tracked_hours" db:"tracked_hours"`
	EngagementRate float64   `json:"engagement_rate" db:"engagement_rate"`
	StartTime      time.Time `json:"start_time" db:"start_time"`
	EndTime        time.Time `json:"end_time" db:"end_time"`
	Period         string    `json:"period" db:"period"`

	DeepWork DeepWorkData `json:"deep_work"`

	UniqueDomainsCount int      `json:"unique_domains_count" db:"unique_domains_count"`
	DomainsList        []string `json:"domains_list" db:"domains_list"`
	FocusLevel         string   `json:"focus_level" db:"focus_level"` // "high", "medium", "low"
	FocusInsight       string   `json:"focus_insight" db:"focus_insight"`

	HourlyBreakdown []HourlyData `json:"hourly_breakdown"`
}

type HourlyData struct {
	Hour         int     `json:"hour"`         // час (0-23)
	Timestamp    string  `json:"timestamp"`    // "8:00 AM", "9:00 AM"
	EngagedMins  int     `json:"engaged_mins"` // активные минуты в этом часе
	IdleMins     int     `json:"idle_mins"`    // неактивные минуты в этом часе
	TotalMins    int     `json:"total_mins"`   // общее время в этом часе
	Events       int     `json:"events"`       // количество событий
	Sessions     int     `json:"sessions"`     // количество сессий
	Productivity float64 `json:"productivity"` // engaged_mins / total_mins * 100
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

type DeepWorkData struct {
	SessionsCount  int              `json:"sessions_count"`        // количество deep work сессий
	TotalMinutes   float64          `json:"total_minutes"`         // общее время в deep work
	TotalHours     float64          `json:"total_hours"`           // общее время в часах
	AverageMinutes float64          `json:"average_minutes"`       // средняя длительность сессии
	LongestMinutes float64          `json:"longest_minutes"`       // самая длинная сессия
	DeepWorkRate   float64          `json:"deep_work_rate"`        // % от tracked time
	TopDomains     []DeepWorkDomain `json:"top_domains,omitempty"` // топ доменов для deep work
}

type DeepWorkDomain struct {
	Domain   string  `json:"domain"`
	Minutes  float64 `json:"minutes"`
	Sessions int     `json:"sessions"`
}

func (e *EngagedTimeMetric) GetFocusLevelDescription() string {
	switch e.FocusLevel {
	case "high":
		return "Высокая концентрация"
	case "medium":
		return "Сбалансированная многозадачность"
	case "low":
		return "Высокая многозадачность"
	default:
		return "Неопределено"
	}
}

func (e *EngagedTimeMetric) GetTopDomains(limit int) []string {
	if limit <= 0 || limit >= len(e.DomainsList) {
		return e.DomainsList
	}
	return e.DomainsList[:limit]
}

func (e *EngagedTimeMetric) IsHighFocus() bool {
	return e.FocusLevel == "high"
}

func (e *EngagedTimeMetric) IsLowFocus() bool {
	return e.FocusLevel == "low"
}
