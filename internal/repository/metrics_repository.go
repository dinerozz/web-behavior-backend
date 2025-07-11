package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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

type engagedTimeResult struct {
	ActiveMinutes      int            `db:"active_minutes"`
	ActiveEventsCount  int            `db:"active_events_count"`
	SessionsCount      int            `db:"sessions_count"`
	PeriodStart        time.Time      `db:"period_start"`
	PeriodEnd          time.Time      `db:"period_end"`
	TotalMinutes       float64        `db:"total_minutes"`
	UniqueDomainsCount int            `db:"unique_domains_count"`
	DomainsList        pq.StringArray `db:"domains_list"`

	DeepSessionsCount int             `db:"deep_sessions_count"`
	TotalDeepMinutes  float64         `db:"total_deep_minutes"`
	AvgDeepMinutes    float64         `db:"avg_deep_minutes"`
	MaxDeepMinutes    float64         `db:"max_deep_minutes"`
	TopDeepDomains    json.RawMessage `db:"top_deep_domains"`
}

type UserMetricsRepository interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error)
	GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error)
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

func (qb *queryBuilder) buildQueryWithDeepWork() string {
	eventPlaceholders := qb.addActiveEventsPlaceholders()

	return fmt.Sprintf(`
    %s,
    %s,
    %s,
    %s
    SELECT 
        LEAST(COALESCE(ad.raw_active_minutes, 0), FLOOR(COALESCE(tb.total_minutes, 0))::int) as active_minutes,
        COALESCE(ad.active_events_count, 0) as active_events_count,
        COALESCE(ad.sessions_count, 0) as sessions_count,
        COALESCE(tb.actual_start, $2::timestamp) as period_start,
        COALESCE(tb.actual_end, $3::timestamp) as period_end,
        COALESCE(tb.total_minutes, 0) as total_minutes,
        COALESCE(dd.unique_domains_count, 0) as unique_domains_count,
        COALESCE(dd.domains_list, ARRAY[]::text[]) as domains_list,
        
        -- Deep Work данные
        COALESCE(dws.deep_sessions_count, 0) as deep_sessions_count,
        COALESCE(dws.total_deep_minutes, 0) as total_deep_minutes,
        COALESCE(dws.avg_deep_minutes, 0) as avg_deep_minutes,
        COALESCE(dws.max_deep_minutes, 0) as max_deep_minutes,
        
        -- Top domains для deep work (JSON)
        COALESCE(
            json_agg(
                json_build_object(
                    'domain', dwd.domain,
                    'minutes', dwd.domain_minutes,
                    'sessions', dwd.domain_sessions
                ) ORDER BY dwd.domain_minutes DESC
            ) FILTER (WHERE dwd.domain IS NOT NULL),
            '[]'::json
        ) as top_deep_domains
        
    FROM time_bounds tb
    FULL OUTER JOIN active_data ad ON true
    FULL OUTER JOIN domains_data dd ON true
    FULL OUTER JOIN deep_work_stats dws ON true
    LEFT JOIN deep_work_domains dwd ON true
    GROUP BY tb.actual_start, tb.actual_end, tb.total_minutes, 
             ad.raw_active_minutes, ad.active_events_count, ad.sessions_count,
             dd.unique_domains_count, dd.domains_list,
             dws.deep_sessions_count, dws.total_deep_minutes, dws.avg_deep_minutes, dws.max_deep_minutes`,

		fmt.Sprintf(timeBoundsQuery, qb.sessionFilter),
		fmt.Sprintf(activeDataQuery, eventPlaceholders, qb.sessionFilter),
		fmt.Sprintf(domainsDataQuery, qb.sessionFilter),
		fmt.Sprintf(deepWorkQuery, eventPlaceholders, qb.sessionFilter))
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

	// Парсим top domains для deep work
	var topDomains []entity.DeepWorkDomain
	if len(result.TopDeepDomains) > 0 && string(result.TopDeepDomains) != "[]" {
		if err := json.Unmarshal(result.TopDeepDomains, &topDomains); err != nil {
			topDomains = []entity.DeepWorkDomain{}
		}
	}

	// Рассчитываем deep work rate
	var deepWorkRate float64
	if result.TotalMinutes > 0 {
		deepWorkRate = utils.RoundToTwoDecimals((result.TotalDeepMinutes / result.TotalMinutes) * 100)
	}

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
		DeepWork: entity.DeepWorkData{
			SessionsCount:  result.DeepSessionsCount,
			TotalMinutes:   utils.RoundToTwoDecimals(result.TotalDeepMinutes),
			TotalHours:     utils.RoundToTwoDecimals(result.TotalDeepMinutes / 60),
			AverageMinutes: utils.RoundToTwoDecimals(result.AvgDeepMinutes),
			LongestMinutes: utils.RoundToTwoDecimals(result.MaxDeepMinutes),
			DeepWorkRate:   deepWorkRate,
			TopDomains:     topDomains,
		},
	}
}

func (r *metricsRepository) GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error) {
	qb := newQueryBuilder(filter)
	query := qb.buildQueryWithDeepWork() // Используем новый метод

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

	// Добавляем Deep Work запрос
	deepWorkQuery = `
    domain_events AS (
        SELECT 
            ub.timestamp,
            CASE 
                WHEN ub.url ~ '^https?://' THEN 
                    split_part(split_part(ub.url, '://', 2), '/', 1)
                ELSE 
                    split_part(ub.url, '/', 1)
            END as domain,
            LAG(ub.timestamp) OVER (
                PARTITION BY CASE 
                    WHEN ub.url ~ '^https?://' THEN 
                        split_part(split_part(ub.url, '://', 2), '/', 1)
                    ELSE 
                        split_part(ub.url, '/', 1)
                END 
                ORDER BY ub.timestamp
            ) as prev_timestamp
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end
            AND ub.event_type IN (%s) %s
    ),
    session_breaks AS (
        SELECT 
            timestamp,
            domain,
            CASE 
                WHEN prev_timestamp IS NULL 
                    OR EXTRACT(EPOCH FROM (timestamp - prev_timestamp)) > 120 
                THEN 1 
                ELSE 0 
            END as is_new_session
        FROM domain_events
    ),
    session_groups AS (
        SELECT 
            domain,
            timestamp,
            SUM(is_new_session) OVER (
                PARTITION BY domain 
                ORDER BY timestamp 
                ROWS UNBOUNDED PRECEDING
            ) as session_group
        FROM session_breaks
    ),
    domain_sessions AS (
        SELECT 
            domain,
            session_group,
            MIN(timestamp) as session_start,
            MAX(timestamp) as session_end,
            EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60.0 as duration_minutes,
            COUNT(*) as events_count
        FROM session_groups
        GROUP BY domain, session_group
        HAVING COUNT(*) > 1
    ),
    deep_work_sessions AS (
        SELECT 
            domain,
            session_start,
            session_end,
            duration_minutes,
            events_count
        FROM domain_sessions
        WHERE duration_minutes >= 15
    ),
    deep_work_stats AS (
        SELECT 
            COUNT(*) as deep_sessions_count,
            COALESCE(SUM(duration_minutes), 0) as total_deep_minutes,
            COALESCE(AVG(duration_minutes), 0) as avg_deep_minutes,
            COALESCE(MAX(duration_minutes), 0) as max_deep_minutes
        FROM deep_work_sessions
    ),
    deep_work_domains AS (
        SELECT 
            domain,
            SUM(duration_minutes) as domain_minutes,
            COUNT(*) as domain_sessions
        FROM deep_work_sessions
        GROUP BY domain
        ORDER BY domain_minutes DESC
        LIMIT 3
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

func (r *metricsRepository) GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 10 // По умолчанию показываем топ 10
	}

	sessionCondition := ""
	args := []interface{}{filter.UserID, limit}

	if filter.SessionID != nil {
		sessionCondition = " AND session_id = $3"
		args = append(args, *filter.SessionID)
	}

	query := fmt.Sprintf(`
		WITH domain_stats AS (
			SELECT 
				CASE 
					WHEN url ~ '^https?://' THEN 
						split_part(split_part(url, '://', 2), '/', 1)
					ELSE 
						split_part(url, '/', 1)
				END as domain,
				COUNT(*) as events_count,
				COUNT(DISTINCT DATE_TRUNC('minute', timestamp)) as active_minutes,
				MIN(timestamp) as first_visit,
				MAX(timestamp) as last_visit
			FROM user_behaviors 
			WHERE user_id = $1 
				AND url IS NOT NULL 
				AND url != '' %s
			GROUP BY domain
		),
		total_stats AS (
			SELECT 
				COUNT(DISTINCT domain) as total_domains,
				SUM(events_count) as total_events
			FROM domain_stats
		)
		SELECT 
			ds.domain,
			ds.events_count,
			ds.active_minutes,
			ROUND((ds.events_count::numeric / ts.total_events::numeric * 100), 2) as percentage,
			ds.first_visit,
			ds.last_visit,
			ts.total_domains,
			ts.total_events
		FROM domain_stats ds
		CROSS JOIN total_stats ts
		ORDER BY ds.events_count DESC, ds.active_minutes DESC
		LIMIT $2`, sessionCondition)

	type queryResult struct {
		Domain        string    `db:"domain"`
		EventsCount   int       `db:"events_count"`
		ActiveMinutes int       `db:"active_minutes"`
		Percentage    float64   `db:"percentage"`
		FirstVisit    time.Time `db:"first_visit"`
		LastVisit     time.Time `db:"last_visit"`
		TotalDomains  int       `db:"total_domains"`
		TotalEvents   int       `db:"total_events"`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get top domains: %w", err)
	}
	defer rows.Close()

	var domains []entity.DomainStats
	var totalDomains, totalEvents int

	for rows.Next() {
		var result queryResult
		err := rows.Scan(
			&result.Domain,
			&result.EventsCount,
			&result.ActiveMinutes,
			&result.Percentage,
			&result.FirstVisit,
			&result.LastVisit,
			&result.TotalDomains,
			&result.TotalEvents,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan domain stats: %w", err)
		}

		domains = append(domains, entity.DomainStats{
			Domain:        result.Domain,
			EventsCount:   result.EventsCount,
			ActiveMinutes: result.ActiveMinutes,
			Percentage:    result.Percentage,
			FirstVisit:    result.FirstVisit,
			LastVisit:     result.LastVisit,
		})

		if totalDomains == 0 {
			totalDomains = result.TotalDomains
			totalEvents = result.TotalEvents
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domain stats: %w", err)
	}

	return &entity.TopDomainsResponse{
		UserID:       filter.UserID,
		TotalDomains: totalDomains,
		TotalEvents:  totalEvents,
		Domains:      domains,
	}, nil
}
