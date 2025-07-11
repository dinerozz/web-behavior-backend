package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"strings"
	"time"
)

var ActiveEvents = []string{
	"pageshow", "click", "focus", "keydown",
	"scrollend", "pagehide", "visibility_visible",
}

type queryBuilder struct {
	args          []interface{}
	argIndex      int
	sessionFilter string
}

type UserMetricsRepository interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error)
}

type metricsRepository struct {
	db *sqlx.DB
}

func NewMetricsRepository(db *sqlx.DB) *metricsRepository {
	return &metricsRepository{db: db}
}

func (r *metricsRepository) GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error) {
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
				AND timestamp <= $3`

	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime}
	argIndex := 4

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
		TotalMinutes: utils.RoundToTwoDecimals(totalMinutes),
		TotalHours:   utils.RoundToTwoDecimals(totalMinutes / 60),
		Sessions:     len(sessions),
		StartTime:    globalStart,
		EndTime:      globalEnd,
		Period:       utils.FormatPeriod(filter.StartTime, filter.EndTime),
	}, nil
}

func (r *metricsRepository) GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error) {
	query := `
        SELECT 
            user_id,
            MIN(timestamp) as actual_start,
            MAX(timestamp) as actual_end,
            EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60 as total_minutes,
            COUNT(DISTINCT session_id) as sessions_count
        FROM user_behaviors 
        WHERE user_id = $1`

	args := []interface{}{filter.UserID}
	argIndex := 2

	if filter.SessionID != nil {
		query += fmt.Sprintf(" AND session_id = $%d", argIndex)
		args = append(args, *filter.SessionID)
	}

	query += " GROUP BY user_id"

	type result struct {
		UserID        string    `db:"user_id"`
		ActualStart   time.Time `db:"actual_start"`
		ActualEnd     time.Time `db:"actual_end"`
		TotalMinutes  float64   `db:"total_minutes"`
		SessionsCount int       `db:"sessions_count"`
	}

	var res result
	err := r.db.GetContext(ctx, &res, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.TrackedTimeMetric{
				UserID:       filter.UserID,
				TotalMinutes: 0,
				TotalHours:   0,
				Sessions:     0,
				Period:       utils.FormatPeriod(filter.StartTime, filter.EndTime),
			}, nil
		}
		return nil, fmt.Errorf("failed to get total tracked time: %w", err)
	}

	return &entity.TrackedTimeMetric{
		UserID:       res.UserID,
		TotalMinutes: utils.RoundToTwoDecimals(res.TotalMinutes),
		TotalHours:   utils.RoundToTwoDecimals(res.TotalMinutes / 60),
		Sessions:     res.SessionsCount,
		StartTime:    res.ActualStart,
		EndTime:      res.ActualEnd,
		Period:       utils.FormatPeriod(res.ActualStart, res.ActualEnd),
	}, nil
}

func newQueryBuilder(filter entity.EngagedTimeFilter) *queryBuilder {
	qb := &queryBuilder{
		args:     []interface{}{filter.UserID, filter.StartTime, filter.EndTime},
		argIndex: 4,
	}

	if filter.SessionID != nil {
		qb.sessionFilter = fmt.Sprintf(" AND session_id = $%d", qb.argIndex)
		qb.args = append(qb.args, *filter.SessionID)
		qb.argIndex++
	}

	return qb
}

func (qb *queryBuilder) addActiveEventsPlaceholders() string {
	placeholders := make([]string, len(ActiveEvents))
	for i, event := range ActiveEvents {
		placeholders[i] = fmt.Sprintf("$%d", qb.argIndex)
		qb.args = append(qb.args, event)
		qb.argIndex++
	}
	return strings.Join(placeholders, ",")
}

func (qb *queryBuilder) buildQuery() string {
	eventPlaceholders := qb.addActiveEventsPlaceholders()

	return fmt.Sprintf(`
    WITH time_bounds AS (
        SELECT 
            MIN(timestamp) as actual_start,
            MAX(timestamp) as actual_end,
            EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60.0 as total_minutes
        FROM user_behaviors 
        WHERE user_id = $1 
            AND timestamp >= $2 
            AND timestamp <= $3 %s
    ),
    active_data AS (
        SELECT 
            COUNT(DISTINCT DATE_TRUNC('minute', ub.timestamp)) as raw_active_minutes,
            COUNT(*) as active_events_count,
            COUNT(DISTINCT session_id) as sessions_count
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end
            AND ub.event_type IN (%s) %s
    ),
    domains_data AS (
        SELECT 
            COUNT(DISTINCT 
                CASE 
                    WHEN url ~ '^https?://' THEN 
                        split_part(split_part(url, '://', 2), '/', 1)
                    ELSE 
                        split_part(url, '/', 1)
                END
            ) as unique_domains_count,
            array_agg(DISTINCT 
                CASE 
                    WHEN url ~ '^https?://' THEN 
                        split_part(split_part(url, '://', 2), '/', 1)
                    ELSE 
                        split_part(url, '/', 1)
                END
            ) FILTER (WHERE url IS NOT NULL AND url != '') as domains_list
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end %s
    )
    SELECT 
        LEAST(COALESCE(ad.raw_active_minutes, 0), FLOOR(COALESCE(tb.total_minutes, 0))::int) as active_minutes,
        COALESCE(ad.active_events_count, 0) as active_events_count,
        COALESCE(ad.sessions_count, 0) as sessions_count,
        COALESCE(tb.actual_start, $2::timestamp) as period_start,
        COALESCE(tb.actual_end, $3::timestamp) as period_end,
        COALESCE(tb.total_minutes, 0) as total_minutes,
        COALESCE(dd.unique_domains_count, 0) as unique_domains_count,
        COALESCE(dd.domains_list, ARRAY[]::text[]) as domains_list
    FROM time_bounds tb
    FULL OUTER JOIN active_data ad ON true
    FULL OUTER JOIN domains_data dd ON true`,
		qb.sessionFilter, eventPlaceholders, qb.sessionFilter, qb.sessionFilter)
}

type engagedTimeResult struct {
	ActiveMinutes      int            `db:"active_minutes"`
	ActiveEventsCount  int            `db:"active_events_count"`
	SessionsCount      int            `db:"sessions_count"`
	PeriodStart        time.Time      `db:"period_start"`
	PeriodEnd          time.Time      `db:"period_end"`
	TotalMinutes       float64        `db:"total_minutes"`
	UniqueDomainsCount int            `db:"unique_domains_count"`
	DomainsList        pq.StringArray `db:"domains_list"`
}

// Вычисляет процент вовлеченности
func calculateEngagementRate(activeMinutes int, totalMinutes float64) float64 {
	if totalMinutes <= 0 {
		return 0
	}
	return utils.RoundToTwoDecimals((float64(activeMinutes) / totalMinutes) * 100)
}

// Создает финальную метрику из результата запроса
func (r *metricsRepository) buildEngagedTimeMetric(filter entity.EngagedTimeFilter, result engagedTimeResult) *entity.EngagedTimeMetric {
	engagementRate := calculateEngagementRate(result.ActiveMinutes, result.TotalMinutes)
	focusLevel := r.determineFocusLevel(result.UniqueDomainsCount)
	focusInsight := r.generateFocusInsight(result.UniqueDomainsCount, result.DomainsList)

	return &entity.EngagedTimeMetric{
		UserID:             filter.UserID,
		ActiveMinutes:      result.ActiveMinutes,
		ActiveHours:        utils.RoundToTwoDecimals(float64(result.ActiveMinutes) / 60),
		ActiveEvents:       result.ActiveEventsCount,
		Sessions:           result.SessionsCount,
		TrackedMinutes:     utils.RoundToTwoDecimals(result.TotalMinutes),
		TrackedHours:       utils.RoundToTwoDecimals(result.TotalMinutes / 60),
		EngagementRate:     engagementRate,
		StartTime:          filter.StartTime,
		EndTime:            filter.EndTime,
		Period:             utils.FormatPeriod(filter.StartTime, filter.EndTime),
		UniqueDomainsCount: result.UniqueDomainsCount,
		DomainsList:        []string(result.DomainsList),
		FocusLevel:         focusLevel,
		FocusInsight:       focusInsight,
	}
}

func (r *metricsRepository) GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error) {
	qb := newQueryBuilder(filter)
	query := qb.buildQuery()

	var result engagedTimeResult
	err := r.db.GetContext(ctx, &result, query, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get engaged time: %w", err)
	}

	return r.buildEngagedTimeMetric(filter, result), nil
}

const (
	timeBoundsQuery = `
    WITH time_bounds AS (
        SELECT 
            MIN(timestamp) as actual_start,
            MAX(timestamp) as actual_end,
            EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60.0 as total_minutes
        FROM user_behaviors 
        WHERE user_id = $1 
            AND timestamp >= $2 
            AND timestamp <= $3 %s
    )`

	activeDataQuery = `
    active_data AS (
        SELECT 
            COUNT(DISTINCT DATE_TRUNC('minute', ub.timestamp)) as raw_active_minutes,
            COUNT(*) as active_events_count,
            COUNT(DISTINCT session_id) as sessions_count
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end
            AND ub.event_type IN (%s) %s
    )`

	domainsDataQuery = `
    domains_data AS (
        SELECT 
            COUNT(DISTINCT 
                CASE 
                    WHEN url ~ '^https?://' THEN 
                        split_part(split_part(url, '://', 2), '/', 1)
                    ELSE 
                        split_part(url, '/', 1)
                END
            ) as unique_domains_count,
            array_agg(DISTINCT 
                CASE 
                    WHEN url ~ '^https?://' THEN 
                        split_part(split_part(url, '://', 2), '/', 1)
                    ELSE 
                        split_part(url, '/', 1)
                END
            ) FILTER (WHERE url IS NOT NULL AND url != '') as domains_list
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end %s
    )`
)

// determineFocusLevel определяет уровень фокуса на основе количества доменов
func (r *metricsRepository) determineFocusLevel(domainsCount int) string {
	switch {
	case domainsCount <= 5:
		return "high" // Высокая концентрация
	case domainsCount <= 15:
		return "medium" // Сбалансированная многозадачность
	default:
		return "low" // Высокая многозадачность/расфокус
	}
}

func (r *metricsRepository) generateFocusInsight(domainsCount int, domains []string) string {
	switch {
	case domainsCount <= 5:
		return fmt.Sprintf("Пользователь работал в ограниченном числе сайтов (%d доменов), что свидетельствует о высокой концентрации и глубокой работе в рамках одного контекста.", domainsCount)

	case domainsCount <= 15:
		return fmt.Sprintf("За сессию пользователь посетил %d уникальных сайтов. Сбалансированная многозадачность без явных признаков расфокуса.", domainsCount)

	case domainsCount <= 25:
		return fmt.Sprintf("За период зафиксировано %d уникальных сайтов. Повышенная контекстная нагрузка - возможно, сотрудник работает в режиме постоянных переключений, что может снижать продуктивность.", domainsCount)

	default:
		return fmt.Sprintf("Зафиксировано %d уникальных сайтов - очень высокий уровень переключений между контекстами. Рекомендуется проверить фокус задач и возможную декомпозицию работы.", domainsCount)
	}
}
