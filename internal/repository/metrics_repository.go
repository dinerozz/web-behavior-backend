package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var ActiveEvents = []string{
	"pageshow", "click", "focus", "keyup", "keydown",
	"scrollend", "pagehide", "visibility_visible",
}

const (
	DeepWorkMinDurationMinutes  = 25  // Минимальная длительность Deep Work блока (25 минут)
	ActivityGapThresholdSeconds = 300 // Максимальный разрыв между событиями (5 минут)
	MinEventsPerBlock           = 10  // Минимальное количество событий в блоке

	// Пороги для определения уровня фокуса (переключения контекста в час)
	HighFocusThreshold   = 5  // <= 5 переключений/час = высокий фокус
	MediumFocusThreshold = 15 // <= 15 переключений/час = средний фокус
	// > 15 переключений/час = низкий фокус
)

// Структуры результатов запросов
type engagedTimeResult struct {
	ActiveMinutes       int            `db:"active_minutes"`
	ActiveEventsCount   int            `db:"active_events_count"`
	TotalTrackedMinutes int            `db:"total_tracked_minutes"`
	IdleMinutes         int            `db:"idle_minutes"`
	SessionsCount       int            `db:"sessions_count"`
	PeriodStart         time.Time      `db:"period_start"`
	PeriodEnd           time.Time      `db:"period_end"`
	UniqueDomainsCount  int            `db:"unique_domains_count"`
	DomainsList         pq.StringArray `db:"domains_list"`
}

type hourlyBreakdownResult struct {
	Hour           int    `db:"hour"`
	Date           string `db:"date"`
	EngagedMinutes int    `db:"engaged_minutes"`
	TotalMinutes   int    `db:"total_minutes"`
	IdleMinutes    int    `db:"idle_minutes"`
	ActiveEvents   int    `db:"active_events"`
	SessionsCount  int    `db:"sessions_count"`
}

type deepWorkDomainResult struct {
	Domain   string  `db:"domain"`
	Minutes  float64 `db:"minutes"`
	Sessions int     `db:"sessions"`
}

type deepWorkStatsResult struct {
	DeepSessionsCount int     `db:"deep_sessions_count"`
	TotalDeepMinutes  float64 `db:"total_deep_minutes"`
	AvgDeepMinutes    float64 `db:"avg_deep_minutes"`
	MaxDeepMinutes    float64 `db:"max_deep_minutes"`
}

type deepWorkSessionsResult struct {
	SessionsCount        int             `db:"sessions_count"`
	TotalMinutes         float64         `db:"total_minutes"`
	AverageMinutes       float64         `db:"average_minutes"`
	LongestMinutes       float64         `db:"longest_minutes"`
	ShortestMinutes      float64         `db:"shortest_minutes"`
	UniqueDomains        int             `db:"unique_domains"`
	TotalContextSwitches int             `db:"total_context_switches"`
	AvgSwitchesPerHour   float64         `db:"avg_switches_per_hour"`
	HighFocusBlocks      int             `db:"high_focus_blocks"`
	MediumFocusBlocks    int             `db:"medium_focus_blocks"`
	LowFocusBlocks       int             `db:"low_focus_blocks"`
	DeepWorkContextRatio float64         `db:"deep_work_context_ratio"`
	SessionsData         json.RawMessage `db:"sessions_data"`
	TotalTrackedMinutes  int             `db:"total_tracked_minutes"`
	HourlyBreakdownData  json.RawMessage `db:"hourly_breakdown_data"`
}

type UserMetricsRepository interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error)
	GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error)
	GetDeepWorkSessions(ctx context.Context, filter entity.DeepWorkSessionsFilter) (*entity.DeepWorkSessionsResponse, error)
}

type metricsRepository struct {
	db *sqlx.DB
}

func NewMetricsRepository(db *sqlx.DB) *metricsRepository {
	return &metricsRepository{db: db}
}

// Единая CTE для Deep Work анализа - используется в обоих эндпоинтах
const deepWorkCoreCTE = `
WITH active_events_filtered AS (
	SELECT
		user_id,
		timestamp,
		event_type,
		url,
		session_id,
		CASE 
			WHEN url ~ '^https?://' THEN 
				split_part(split_part(url, '://', 2), '/', 1)
			ELSE 
				split_part(url, '/', 1)
		END as domain,
		LAG(timestamp) OVER (PARTITION BY user_id ORDER BY timestamp) AS prev_timestamp,
		LAG(
			CASE 
				WHEN url ~ '^https?://' THEN 
					split_part(split_part(url, '://', 2), '/', 1)
				ELSE 
					split_part(url, '/', 1)
			END
		) OVER (PARTITION BY user_id ORDER BY timestamp) AS prev_domain
	FROM user_behaviors 
	WHERE user_id = $1 
		AND timestamp >= $2 
		AND timestamp <= $3
		AND event_type = ANY($4::text[]) %s
),
activity_gaps AS (
	SELECT *,
		CASE 
			WHEN prev_timestamp IS NULL 
				OR EXTRACT(EPOCH FROM (timestamp - prev_timestamp)) > %d
			THEN 1 ELSE 0 
		END AS is_new_block
	FROM active_events_filtered
),
numbered_blocks AS (
	SELECT *,
		SUM(is_new_block) OVER (PARTITION BY user_id ORDER BY timestamp) AS block_id
	FROM activity_gaps
),
block_stats AS (
	SELECT
		user_id,
		block_id,
		MIN(timestamp) AS start_time,
		MAX(timestamp) AS end_time,
		COUNT(*) AS total_events,
		COUNT(DISTINCT domain) AS unique_domains,
		EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) / 60.0 AS duration_minutes,
		-- Подсчет переключений контекста
		COUNT(CASE WHEN domain IS DISTINCT FROM prev_domain AND prev_domain IS NOT NULL THEN 1 END) AS context_switches
	FROM numbered_blocks
	GROUP BY user_id, block_id
),
deep_work_blocks AS (
	SELECT *,
		-- Расчет переключений в час
		CASE 
			WHEN duration_minutes > 0 
			THEN ROUND((context_switches * 60.0 / duration_minutes)::numeric, 2)
			ELSE 0::numeric 
		END AS switches_per_hour,
		-- Определение уровня фокуса
		CASE 
			WHEN duration_minutes = 0 THEN 'low'
			WHEN (context_switches * 60.0 / duration_minutes) <= %d THEN 'high'
			WHEN (context_switches * 60.0 / duration_minutes) <= %d THEN 'medium'
			ELSE 'low'
		END AS focus_level
	FROM block_stats
	WHERE duration_minutes >= %d  -- Минимальная длительность
		AND total_events >= %d     -- Минимальное количество событий
)`

// Основной запрос для базовых метрик engaged time (без deep work)
const optimizedEngagedTimeQuery = `
WITH minute_activity AS (
    SELECT
        DATE_TRUNC('minute', timestamp) AS minute,
        MAX(CASE WHEN event_type = ANY($4::text[]) THEN 1 ELSE 0 END) AS is_active,
        1 AS is_tracked,
        COUNT(CASE WHEN event_type = ANY($4::text[]) THEN 1 END) AS active_events_in_minute,
        COUNT(DISTINCT session_id) AS sessions_in_minute,
        CASE 
            WHEN url ~ '^https?://' THEN 
                split_part(split_part(url, '://', 2), '/', 1)
            ELSE 
                split_part(url, '/', 1)
        END as domain
    FROM user_behaviors 
    WHERE user_id = $1 
        AND timestamp >= $2 
        AND timestamp <= $3 %s
    GROUP BY DATE_TRUNC('minute', timestamp), 
             CASE 
                 WHEN url ~ '^https?://' THEN 
                     split_part(split_part(url, '://', 2), '/', 1)
                 ELSE 
                     split_part(url, '/', 1)
             END
),
base_stats AS (
    SELECT 
        COALESCE(SUM(is_active), 0) as active_minutes,
        COALESCE(SUM(is_tracked), 0) as total_tracked_minutes,
        COALESCE(SUM(CASE WHEN is_active = 0 THEN 1 ELSE 0 END), 0) as idle_minutes,
        COALESCE(SUM(active_events_in_minute), 0) as active_events_count,
        COALESCE(COUNT(DISTINCT CASE WHEN sessions_in_minute > 0 THEN minute END), 0) as sessions_count,
        COALESCE(MIN(minute), $2::timestamp) as period_start,
        COALESCE(MAX(minute), $3::timestamp) as period_end,
        COALESCE(COUNT(DISTINCT domain) FILTER (WHERE domain IS NOT NULL AND domain != ''), 0) as unique_domains_count,
        COALESCE(array_agg(DISTINCT domain) FILTER (WHERE domain IS NOT NULL AND domain != ''), ARRAY[]::text[]) as domains_list
    FROM minute_activity
)
SELECT 
    bs.active_minutes,
    bs.total_tracked_minutes,
    bs.idle_minutes,
    bs.active_events_count,
    bs.sessions_count,
    bs.period_start,
    bs.period_end,
    bs.unique_domains_count,
    bs.domains_list
FROM base_stats bs`

// Запрос для hourly breakdown
const hourlyBreakdownQuery = `
WITH hourly_minute_activity AS (
    SELECT 
        EXTRACT(HOUR FROM timestamp)::integer as hour,
        DATE(timestamp)::text as date,
        DATE_TRUNC('minute', timestamp) AS minute,
        MAX(CASE WHEN event_type = ANY($4::text[]) THEN 1 ELSE 0 END) AS is_active,
        COUNT(CASE WHEN event_type = ANY($4::text[]) THEN 1 END) AS active_events_in_minute,
        COUNT(DISTINCT session_id) AS sessions_in_minute
    FROM user_behaviors 
    WHERE user_id = $1 
        AND timestamp >= $2 
        AND timestamp <= $3 %s
    GROUP BY EXTRACT(HOUR FROM timestamp), DATE(timestamp), DATE_TRUNC('minute', timestamp)
)
SELECT 
    hour,
    date,
    COALESCE(SUM(is_active), 0)::integer as engaged_minutes,
    COALESCE(COUNT(*), 0)::integer as total_minutes,
    COALESCE(SUM(CASE WHEN is_active = 0 THEN 1 ELSE 0 END), 0)::integer as idle_minutes,
    COALESCE(SUM(active_events_in_minute), 0)::integer as active_events,
    COALESCE(COUNT(DISTINCT CASE WHEN sessions_in_minute > 0 THEN minute END), 0)::integer as sessions_count
FROM hourly_minute_activity 
GROUP BY hour, date
ORDER BY date, hour`

// Функции-билдеры для Deep Work запросов
func buildDeepWorkStatsQuery(sessionFilter string) string {
	cte := fmt.Sprintf(deepWorkCoreCTE,
		sessionFilter,
		ActivityGapThresholdSeconds,
		HighFocusThreshold,
		MediumFocusThreshold,
		DeepWorkMinDurationMinutes,
		MinEventsPerBlock,
	)

	return fmt.Sprintf(`%s
	SELECT 
		COALESCE(COUNT(*), 0) as deep_sessions_count,
		COALESCE(SUM(duration_minutes), 0) as total_deep_minutes,
		COALESCE(AVG(duration_minutes), 0) as avg_deep_minutes,
		COALESCE(MAX(duration_minutes), 0) as max_deep_minutes
	FROM deep_work_blocks`, cte)
}

func buildDeepWorkTopDomainsQuery(sessionFilter string) string {
	cte := fmt.Sprintf(deepWorkCoreCTE,
		sessionFilter,
		ActivityGapThresholdSeconds,
		HighFocusThreshold,
		MediumFocusThreshold,
		DeepWorkMinDurationMinutes,
		MinEventsPerBlock,
	)

	return fmt.Sprintf(`%s,
	domain_stats AS (
		SELECT 
			nb.domain,
			SUM(dwb.duration_minutes) as total_minutes,
			COUNT(DISTINCT dwb.block_id) as sessions_count
		FROM deep_work_blocks dwb
		JOIN numbered_blocks nb ON dwb.block_id = nb.block_id
		WHERE nb.domain IS NOT NULL AND nb.domain != ''
		GROUP BY nb.domain
	)
	SELECT 
		domain,
		total_minutes as minutes,
		sessions_count as sessions
	FROM domain_stats
	ORDER BY total_minutes DESC
	LIMIT 3`, cte)
}

func buildDeepWorkSessionsQuery(sessionFilter string) string {
	cte := fmt.Sprintf(deepWorkCoreCTE,
		sessionFilter,
		ActivityGapThresholdSeconds,
		HighFocusThreshold,
		MediumFocusThreshold,
		DeepWorkMinDurationMinutes,
		MinEventsPerBlock,
	)

	return fmt.Sprintf(`%s,
	-- Hourly статистика (ИСПРАВЛЕНО)
	hourly_stats AS (
		SELECT 
			EXTRACT(HOUR FROM hour_start)::integer as hour,
			DATE(hour_start)::text as date,
			SUM(hour_duration) as deep_work_minutes,
			COUNT(DISTINCT dwb.block_id) as sessions_count,
			SUM(dwb.context_switches * (hour_duration / dwb.duration_minutes)) as context_switches,
			AVG(dwb.switches_per_hour) as avg_switches_per_hour
		FROM deep_work_blocks dwb
		CROSS JOIN LATERAL (
			SELECT 
				generate_series(
					DATE_TRUNC('hour', dwb.start_time),
					DATE_TRUNC('hour', dwb.end_time),
					'1 hour'::interval
				) as hour_start
		) hours
		CROSS JOIN LATERAL (
			SELECT 
				CASE 
					-- Полный час внутри сессии
					WHEN hour_start >= DATE_TRUNC('hour', dwb.start_time) 
						AND hour_start + INTERVAL '1 hour' <= dwb.end_time 
					THEN 60.0
					-- Частичный час в начале
					WHEN hour_start = DATE_TRUNC('hour', dwb.start_time) 
						AND hour_start + INTERVAL '1 hour' > dwb.end_time
					THEN EXTRACT(EPOCH FROM (dwb.end_time - dwb.start_time)) / 60.0
					-- Частичный час в начале (но сессия продолжается)
					WHEN hour_start = DATE_TRUNC('hour', dwb.start_time)
					THEN EXTRACT(EPOCH FROM (hour_start + INTERVAL '1 hour' - dwb.start_time)) / 60.0
					-- Частичный час в конце
					WHEN hour_start + INTERVAL '1 hour' > dwb.end_time
					THEN EXTRACT(EPOCH FROM (dwb.end_time - hour_start)) / 60.0
					-- Полный час
					ELSE 60.0
				END as hour_duration
		) duration_calc
		WHERE hour_duration > 0
		GROUP BY EXTRACT(HOUR FROM hour_start), DATE(hour_start)
	),
	-- Общая статистика по всем минутам для расчета процента
	total_tracked AS (
		SELECT 
			COUNT(DISTINCT DATE_TRUNC('minute', timestamp)) as total_minutes
		FROM user_behaviors 
		WHERE user_id = $1 
			AND timestamp >= $2 
			AND timestamp <= $3 %s
	),
	-- Финальная агрегация
	aggregated_stats AS (
		SELECT 
			COUNT(*)::integer as sessions_count,
			COALESCE(SUM(duration_minutes), 0) as total_minutes,
			COALESCE(AVG(duration_minutes), 0) as average_minutes,
			COALESCE(MAX(duration_minutes), 0) as longest_minutes,
			COALESCE(MIN(duration_minutes), 0) as shortest_minutes,
			(SELECT COUNT(DISTINCT nb.domain) 
			 FROM numbered_blocks nb 
			 WHERE nb.block_id IN (SELECT block_id FROM deep_work_blocks) 
			 AND nb.domain IS NOT NULL AND nb.domain != '') as unique_domains,
			COALESCE(SUM(context_switches), 0)::integer as total_context_switches,
			COALESCE(AVG(switches_per_hour), 0) as avg_switches_per_hour,
			COUNT(CASE WHEN focus_level = 'high' THEN 1 END)::integer as high_focus_blocks,
			COUNT(CASE WHEN focus_level = 'medium' THEN 1 END)::integer as medium_focus_blocks,
			COUNT(CASE WHEN focus_level = 'low' THEN 1 END)::integer as low_focus_blocks,
			-- Соотношение deep work сессий к переключениям контекста
			CASE 
				WHEN SUM(context_switches) > 0 
				THEN COUNT(*)::numeric / SUM(context_switches)::numeric
				ELSE CASE WHEN COUNT(*) > 0 THEN 99.0 ELSE 0.0 END
			END as deep_work_context_ratio
		FROM deep_work_blocks
	),
	-- JSON данные сессий
	sessions_json AS (
		SELECT 
			COALESCE(
				json_agg(
					json_build_object(
						'block_id', block_id,
						'start_time', start_time,
						'end_time', end_time,
						'duration_minutes', ROUND(duration_minutes::numeric, 2),
						'total_events', total_events,
						'context_switches', context_switches,
						'switches_per_hour', switches_per_hour,
						'focus_level', focus_level
					) ORDER BY start_time
				),
				'[]'::json
			) as sessions_data
		FROM deep_work_blocks
	),
	-- JSON данные hourly breakdown
	hourly_json AS (
		SELECT 
			COALESCE(
				json_agg(
					json_build_object(
						'hour', hour,
						'date', date,
						'timestamp', CASE 
							WHEN hour = 0 THEN '12:00 AM'
							WHEN hour = 12 THEN '12:00 PM'
							WHEN hour < 12 THEN hour || ':00 AM'
							ELSE (hour - 12) || ':00 PM'
						END,
						'total_mins', ROUND(deep_work_minutes::numeric, 0),
						'sessions', sessions_count,
						'deep_work_mins', ROUND(deep_work_minutes::numeric, 0),
						'context_switches', context_switches,
						'switches_per_hour', ROUND(avg_switches_per_hour::numeric, 2),
						'deep_work_rate', 100.0  -- В hourly мы показываем только deep work время
					) ORDER BY date, hour
				),
				'[]'::json
			) as hourly_data
		FROM hourly_stats
	)
	SELECT 
		COALESCE(ag.sessions_count, 0) as sessions_count,
		COALESCE(ag.total_minutes, 0) as total_minutes,
		COALESCE(ag.average_minutes, 0) as average_minutes,
		COALESCE(ag.longest_minutes, 0) as longest_minutes,
		COALESCE(ag.shortest_minutes, 0) as shortest_minutes,
		COALESCE(ag.unique_domains, 0) as unique_domains,
		COALESCE(ag.total_context_switches, 0) as total_context_switches,
		COALESCE(ag.avg_switches_per_hour, 0) as avg_switches_per_hour,
		COALESCE(ag.high_focus_blocks, 0) as high_focus_blocks,
		COALESCE(ag.medium_focus_blocks, 0) as medium_focus_blocks,
		COALESCE(ag.low_focus_blocks, 0) as low_focus_blocks,
		COALESCE(ag.deep_work_context_ratio, 0) as deep_work_context_ratio,
		sj.sessions_data,
		COALESCE(tt.total_minutes, 0) as total_tracked_minutes,
		hj.hourly_data as hourly_breakdown_data
	FROM aggregated_stats ag
	CROSS JOIN sessions_json sj
	CROSS JOIN total_tracked tt
	CROSS JOIN hourly_json hj`, cte, sessionFilter)
}

func (r *metricsRepository) getDeepWorkStats(ctx context.Context, filter entity.EngagedTimeFilter) (*deepWorkStatsResult, error) {
	sessionFilter := ""
	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime, pq.Array(ActiveEvents)}

	if filter.SessionID != nil {
		sessionFilter = " AND session_id = $5"
		args = append(args, *filter.SessionID)
	}

	query := buildDeepWorkStatsQuery(sessionFilter)

	var result deepWorkStatsResult
	err := r.db.GetContext(ctx, &result, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return &deepWorkStatsResult{}, nil
		}
		return nil, fmt.Errorf("failed to get deep work stats: %w", err)
	}

	return &result, nil
}

func (r *metricsRepository) getDeepWorkTopDomains(ctx context.Context, filter entity.EngagedTimeFilter) ([]deepWorkDomainResult, error) {
	sessionFilter := ""
	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime, pq.Array(ActiveEvents)}

	if filter.SessionID != nil {
		sessionFilter = " AND session_id = $5"
		args = append(args, *filter.SessionID)
	}

	query := buildDeepWorkTopDomainsQuery(sessionFilter)

	var results []deepWorkDomainResult
	err := r.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get deep work top domains: %w", err)
	}

	return results, nil
}

func (r *metricsRepository) GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error) {
	sessionFilter := ""
	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime, pq.Array(ActiveEvents)}

	if filter.SessionID != nil {
		sessionFilter = " AND session_id = $5"
		args = append(args, *filter.SessionID)
	}

	mainQuery := fmt.Sprintf(optimizedEngagedTimeQuery, sessionFilter)

	var result engagedTimeResult
	err := r.db.GetContext(ctx, &result, mainQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return r.buildEmptyEngagedTimeMetric(filter), nil
		}
		return nil, fmt.Errorf("failed to get engaged time base stats: %w", err)
	}

	// 2. Deep Work статистика (единая логика)
	deepWorkStats, err := r.getDeepWorkStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get deep work stats: %w", err)
	}

	// 3. Hourly breakdown
	hourlyQuery := fmt.Sprintf(hourlyBreakdownQuery, sessionFilter)
	var hourlyResults []hourlyBreakdownResult
	err = r.db.SelectContext(ctx, &hourlyResults, hourlyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly breakdown: %w", err)
	}

	// 4. Top domains для deep work (только если есть deep work)
	var topDomains []deepWorkDomainResult
	if deepWorkStats.DeepSessionsCount > 0 {
		topDomains, err = r.getDeepWorkTopDomains(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get deep work top domains: %w", err)
		}
	}

	return r.buildEngagedTimeMetricWithDeepWork(filter, result, deepWorkStats, hourlyResults, topDomains), nil
}

func (r *metricsRepository) GetDeepWorkSessions(ctx context.Context, filter entity.DeepWorkSessionsFilter) (*entity.DeepWorkSessionsResponse, error) {
	sessionFilter := ""
	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime, pq.Array(ActiveEvents)}

	if filter.SessionID != nil {
		sessionFilter = " AND session_id = $5"
		args = append(args, *filter.SessionID)
	}

	query := buildDeepWorkSessionsQuery(sessionFilter)

	var result deepWorkSessionsResult
	err := r.db.GetContext(ctx, &result, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return r.buildEmptyDeepWorkSessionsResponse(filter), nil
		}
		return nil, fmt.Errorf("failed to get deep work sessions: %w", err)
	}

	return r.buildDeepWorkSessionsResponse(filter, result), nil
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

func (r *metricsRepository) GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
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
			GROUP BY CASE 
						WHEN url ~ '^https?://' THEN 
							split_part(split_part(url, '://', 2), '/', 1)
						ELSE 
							split_part(url, '/', 1)
					END
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

// Билдеры результатов
func (r *metricsRepository) buildEngagedTimeMetricWithDeepWork(
	filter entity.EngagedTimeFilter,
	result engagedTimeResult,
	deepWorkStats *deepWorkStatsResult,
	hourlyResults []hourlyBreakdownResult,
	topDomainsResults []deepWorkDomainResult,
) *entity.EngagedTimeMetric {

	engagementRate := calculateEngagementRate(result.ActiveMinutes, result.TotalTrackedMinutes)

	hourlyBreakdown := make([]entity.HourlyData, len(hourlyResults))
	for i, hourly := range hourlyResults {
		totalMins := hourly.TotalMinutes
		idleMins := hourly.IdleMinutes

		if totalMins != (hourly.EngagedMinutes + idleMins) {
			idleMins = totalMins - hourly.EngagedMinutes
			if idleMins < 0 {
				idleMins = 0
				totalMins = hourly.EngagedMinutes
			}
		}

		var productivity float64
		if totalMins > 0 {
			productivity = utils.RoundToTwoDecimals((float64(hourly.EngagedMinutes) / float64(totalMins)) * 100)
		}

		hourlyBreakdown[i] = entity.HourlyData{
			Hour:         hourly.Hour,
			Date:         hourly.Date,
			Timestamp:    utils.FormatHourTimestamp(hourly.Hour),
			EngagedMins:  hourly.EngagedMinutes,
			IdleMins:     idleMins,
			TotalMins:    totalMins,
			Events:       hourly.ActiveEvents,
			Sessions:     hourly.SessionsCount,
			Productivity: productivity,
		}
	}

	topDomains := make([]entity.DeepWorkDomain, len(topDomainsResults))
	for i, domain := range topDomainsResults {
		topDomains[i] = entity.DeepWorkDomain{
			Domain:   domain.Domain,
			Minutes:  utils.RoundToTwoDecimals(domain.Minutes),
			Sessions: domain.Sessions,
		}
	}

	var deepWorkRate float64
	if result.TotalTrackedMinutes > 0 {
		deepWorkRate = utils.RoundToTwoDecimals((deepWorkStats.TotalDeepMinutes / float64(result.TotalTrackedMinutes)) * 100)
	}

	return &entity.EngagedTimeMetric{
		UserID:             filter.UserID,
		ActiveMinutes:      result.ActiveMinutes,
		ActiveHours:        utils.RoundToTwoDecimals(float64(result.ActiveMinutes) / 60),
		ActiveEvents:       result.ActiveEventsCount,
		Sessions:           result.SessionsCount,
		TrackedMinutes:     utils.RoundToTwoDecimals(float64(result.TotalTrackedMinutes)),
		TrackedHours:       utils.RoundToTwoDecimals(float64(result.TotalTrackedMinutes) / 60),
		EngagementRate:     engagementRate,
		StartTime:          filter.StartTime,
		EndTime:            filter.EndTime,
		Period:             utils.FormatPeriod(filter.StartTime, filter.EndTime),
		UniqueDomainsCount: result.UniqueDomainsCount,
		DomainsList:        result.DomainsList,
		DeepWork: entity.DeepWorkData{
			SessionsCount:  int(deepWorkStats.DeepSessionsCount),
			TotalMinutes:   utils.RoundToTwoDecimals(deepWorkStats.TotalDeepMinutes),
			TotalHours:     utils.RoundToTwoDecimals(deepWorkStats.TotalDeepMinutes / 60),
			AverageMinutes: utils.RoundToTwoDecimals(deepWorkStats.AvgDeepMinutes),
			LongestMinutes: utils.RoundToTwoDecimals(deepWorkStats.MaxDeepMinutes),
			DeepWorkRate:   deepWorkRate,
			TopDomains:     topDomains,
		},
		HourlyBreakdown: hourlyBreakdown,
	}
}

func (r *metricsRepository) buildEmptyDeepWorkSessionsResponse(filter entity.DeepWorkSessionsFilter) *entity.DeepWorkSessionsResponse {
	return &entity.DeepWorkSessionsResponse{
		UserID:    filter.UserID,
		StartTime: filter.StartTime,
		EndTime:   filter.EndTime,
		Period:    utils.FormatPeriod(filter.StartTime, filter.EndTime),
		Sessions:  []entity.DeepWorkSession{},
		ContextSwitches: entity.ContextSwitchesStats{
			TotalSwitches:      0,
			AvgSwitchesPerHour: 0,
			HighFocusBlocks:    0,
			MediumFocusBlocks:  0,
			LowFocusBlocks:     0,
		},
		HourlyBreakdown: []entity.HourlyDeepWorkData{},
	}
}

func (r *metricsRepository) buildDeepWorkSessionsResponse(filter entity.DeepWorkSessionsFilter, result deepWorkSessionsResult) *entity.DeepWorkSessionsResponse {
	var sessions []entity.DeepWorkSession
	if len(result.SessionsData) > 0 && string(result.SessionsData) != "[]" {
		if err := json.Unmarshal(result.SessionsData, &sessions); err != nil {
			sessions = []entity.DeepWorkSession{}
		}
	}

	var hourlyBreakdown []entity.HourlyDeepWorkData
	if len(result.HourlyBreakdownData) > 0 && string(result.HourlyBreakdownData) != "[]" {
		type hourlyRaw struct {
			Hour            int     `json:"hour"`
			Date            string  `json:"date"`
			Timestamp       string  `json:"timestamp"`
			TotalMins       int     `json:"total_mins"`
			Sessions        int     `json:"sessions"`
			DeepWorkMins    float64 `json:"deep_work_mins"`
			ContextSwitches int     `json:"context_switches"`
			SwitchesPerHour float64 `json:"switches_per_hour"`
			DeepWorkRate    float64 `json:"deep_work_rate"`
		}

		var rawData []hourlyRaw
		if err := json.Unmarshal(result.HourlyBreakdownData, &rawData); err == nil {
			hourlyBreakdown = make([]entity.HourlyDeepWorkData, len(rawData))
			for i, raw := range rawData {
				hourlyBreakdown[i] = entity.HourlyDeepWorkData{
					Hour:            raw.Hour,
					Date:            raw.Date,
					Timestamp:       raw.Timestamp,
					TotalMins:       raw.TotalMins,
					Sessions:        raw.Sessions,
					DeepWorkMins:    int(raw.DeepWorkMins),
					ContextSwitches: raw.ContextSwitches,
					SwitchesPerHour: utils.RoundToTwoDecimals(raw.SwitchesPerHour),
					DeepWorkRate:    utils.RoundToTwoDecimals(raw.DeepWorkRate),
				}
			}
		}
	}

	var deepWorkRate float64
	if result.TotalTrackedMinutes > 0 {
		deepWorkRate = utils.RoundToTwoDecimals((result.TotalMinutes / float64(result.TotalTrackedMinutes)) * 100)
	}

	return &entity.DeepWorkSessionsResponse{
		UserID:    filter.UserID,
		StartTime: filter.StartTime,
		EndTime:   filter.EndTime,
		Period:    utils.FormatPeriod(filter.StartTime, filter.EndTime),

		SessionsCount:   result.SessionsCount,
		TotalMinutes:    utils.RoundToTwoDecimals(result.TotalMinutes),
		TotalHours:      utils.RoundToTwoDecimals(result.TotalMinutes / 60),
		AverageMinutes:  utils.RoundToTwoDecimals(result.AverageMinutes),
		LongestMinutes:  utils.RoundToTwoDecimals(result.LongestMinutes),
		ShortestMinutes: utils.RoundToTwoDecimals(result.ShortestMinutes),
		UniqueDomains:   result.UniqueDomains,
		DeepWorkRate:    deepWorkRate,

		ContextSwitches: entity.ContextSwitchesStats{
			TotalSwitches:      result.TotalContextSwitches,
			AvgSwitchesPerHour: utils.RoundToTwoDecimals(result.AvgSwitchesPerHour),
			HighFocusBlocks:    result.HighFocusBlocks,
			MediumFocusBlocks:  result.MediumFocusBlocks,
			LowFocusBlocks:     result.LowFocusBlocks,
		},

		DeepWorkContextRatio: utils.RoundToTwoDecimals(result.DeepWorkContextRatio),
		Sessions:             sessions,
		HourlyBreakdown:      hourlyBreakdown,
	}
}

func (r *metricsRepository) buildEmptyEngagedTimeMetric(filter entity.EngagedTimeFilter) *entity.EngagedTimeMetric {
	return &entity.EngagedTimeMetric{
		UserID:             filter.UserID,
		ActiveMinutes:      0,
		ActiveHours:        0,
		ActiveEvents:       0,
		Sessions:           0,
		TrackedMinutes:     0,
		TrackedHours:       0,
		EngagementRate:     0,
		StartTime:          filter.StartTime,
		EndTime:            filter.EndTime,
		Period:             utils.FormatPeriod(filter.StartTime, filter.EndTime),
		UniqueDomainsCount: 0,
		DomainsList:        []string{},
		DeepWork: entity.DeepWorkData{
			SessionsCount:  0,
			TotalMinutes:   0,
			TotalHours:     0,
			AverageMinutes: 0,
			LongestMinutes: 0,
			DeepWorkRate:   0,
			TopDomains:     []entity.DeepWorkDomain{},
		},
		HourlyBreakdown: []entity.HourlyData{},
	}
}

func calculateEngagementRate(activeMinutes int, totalTrackedMinutes int) float64 {
	if totalTrackedMinutes <= 0 {
		return 0
	}
	return utils.RoundToTwoDecimals((float64(activeMinutes) / float64(totalTrackedMinutes)) * 100)
}
