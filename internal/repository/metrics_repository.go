package repository

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/jmoiron/sqlx"
	"time"
)

type MetricsRepository struct {
	db *sqlx.DB
}

func NewMetricsRepository(db *sqlx.DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

func (r *MetricsRepository) GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error) {
	query := `
			SELECT 
				user_id,
				session_id,
				MIN(timestamp) as session_start,
				MAX(timestamp) as session_end,
				EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60 as duration_minutes
			FROM user_behaviors 
			WHERE user_id = $1 
				AND timestamp >= $2 
				AND timestamp <= $3 group by user_id, session_id`

	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime}
	argIndex := 4

	// Если указан конкретный session_id
	if filter.SessionID != nil {
		query += fmt.Sprintf(" AND session_id = $%d", argIndex)
		args = append(args, *filter.SessionID)
		argIndex++
	}

	query += `
        GROUP BY user_id, session_id
        HAVING COUNT(*) > 1`

	type sessionResult struct {
		UserID          string    `db:"user_id"`
		SessionID       string    `db:"session_id"`
		SessionStart    time.Time `db:"session_start"`
		SessionEnd      time.Time `db:"session_end"`
		DurationMinutes float64   `db:"duration_minutes"`
	}

	var sessions []sessionResult
	err := r.db.SelectContext(ctx, &sessions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked time: %w", err)
	}

	if len(sessions) == 0 {
		return &entity.TrackedTimeMetric{
			UserID:       filter.UserID,
			TotalMinutes: 0,
			TotalHours:   0,
			Sessions:     0,
			StartTime:    filter.StartTime,
			EndTime:      filter.EndTime,
			Period:       utils.FormatPeriod(filter.StartTime, filter.EndTime),
		}, nil
	}

	var totalMinutes float64
	var globalStart, globalEnd time.Time

	for i, session := range sessions {
		totalMinutes += session.DurationMinutes

		if i == 0 {
			globalStart = session.SessionStart
			globalEnd = session.SessionEnd
		} else {
			if session.SessionStart.Before(globalStart) {
				globalStart = session.SessionStart
			}
			if session.SessionEnd.After(globalEnd) {
				globalEnd = session.SessionEnd
			}
		}
	}

	return &entity.TrackedTimeMetric{
		UserID:       filter.UserID,
		TotalMinutes: totalMinutes,
		TotalHours:   totalMinutes / 60,
		Sessions:     len(sessions),
		StartTime:    globalStart,
		EndTime:      globalEnd,
		Period:       utils.FormatPeriod(filter.StartTime, filter.EndTime),
	}, nil
}
