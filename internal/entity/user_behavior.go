package entity

import (
	"github.com/gofrs/uuid"
	"time"
)

type UserBehavior struct {
	ID        uuid.UUID `json:"id" db:"id"`
	SessionID string    `json:"sessionId" db:"session_id" binding:"required"`
	Timestamp time.Time `json:"ts" db:"timestamp" binding:"required"`
	Type      string    `json:"type" db:"event_type" binding:"required"`
	URL       string    `json:"url" db:"url" binding:"required"`
	UserID    *string   `json:"userId" db:"user_id"`
	UserName  *string   `json:"userName" db:"user_name"`
	X         *int      `json:"x,omitempty" db:"x"`
	Y         *int      `json:"y,omitempty" db:"y"`
	Key       *string   `json:"key,omitempty" db:"key"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type CreateUserBehaviorRequest struct {
	SessionID string    `json:"sessionId" binding:"required"`
	Timestamp time.Time `json:"ts" binding:"required"`
	Type      string    `json:"type" binding:"required"`
	URL       string    `json:"url" binding:"required"`
	UserID    *string   `json:"userId"`
	UserName  *string   `json:"userName"`
	X         *int      `json:"x,omitempty"`
	Y         *int      `json:"y,omitempty"`
	Key       *string   `json:"key,omitempty"`
}

type BatchCreateUserBehaviorRequest struct {
	Events []CreateUserBehaviorRequest `json:"events" binding:"required,dive"`
}

// UserBehaviorFilter фильтры для поиска событий
type UserBehaviorFilter struct {
	UserID    *string    `form:"userId"`
	SessionID *string    `form:"sessionId"`
	EventType *string    `form:"eventType"`
	URL       *string    `form:"url"`
	StartTime *time.Time `form:"startTime"`
	EndTime   *time.Time `form:"endTime"`
	Limit     int        `form:"limit"`
	Offset    int        `form:"offset"`
}

type UserBehaviorStats struct {
	TotalEvents    int64            `json:"totalEvents"`
	UniqueUsers    int64            `json:"uniqueUsers"`
	UniqueSessions int64            `json:"uniqueSessions"`
	EventsByType   map[string]int64 `json:"eventsByType"`
	PopularURLs    []URLStats       `json:"popularUrls"`
}

type URLStats struct {
	URL   string `json:"url"`
	Count int64  `json:"count"`
}

type SessionSummary struct {
	SessionID   string    `json:"sessionId"`
	UserID      *string   `json:"userId"`
	UserName    *string   `json:"userName"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Duration    int64     `json:"duration"` // в секундах
	EventsCount int64     `json:"eventsCount"`
	URLs        []string  `json:"urls"`
}
