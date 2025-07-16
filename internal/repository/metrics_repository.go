package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	ActiveMinutes       int            `db:"active_minutes"`
	ActiveEventsCount   int            `db:"active_events_count"`
	TotalTrackedMinutes int            `db:"total_tracked_minutes"`
	IdleMinutes         int            `db:"idle_minutes"`
	SessionsCount       int            `db:"sessions_count"`
	PeriodStart         time.Time      `db:"period_start"`
	PeriodEnd           time.Time      `db:"period_end"`
	TotalMinutes        float64        `db:"total_minutes"`
	UniqueDomainsCount  int            `db:"unique_domains_count"`
	DomainsList         pq.StringArray `db:"domains_list"`

	DeepSessionsCount int             `db:"deep_sessions_count"`
	TotalDeepMinutes  float64         `db:"total_deep_minutes"`
	AvgDeepMinutes    float64         `db:"avg_deep_minutes"`
	MaxDeepMinutes    float64         `db:"max_deep_minutes"`
	TopDeepDomains    json.RawMessage `db:"top_deep_domains"`

	HourlyBreakdownData json.RawMessage `db:"hourly_breakdown_data"`
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

	HourlyBreakdownData json.RawMessage `db:"hourly_breakdown_data"`
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

func (qb *queryBuilder) buildQueryWithDeepWorkAndHourly() string {
	eventPlaceholders := qb.addActiveEventsPlaceholders()
	activeDataEventPlaceholders1 := qb.addActiveEventsPlaceholders()
	//activeDataEventPlaceholders2 := qb.addActiveEventsPlaceholders()
	hourlyEventPlaceholders1 := qb.addActiveEventsPlaceholders()
	hourlyEventPlaceholders2 := qb.addActiveEventsPlaceholders()

	return fmt.Sprintf(`
    %s,
    %s,
    %s,
    %s,
    %s
    SELECT 
        COALESCE(ad.active_minutes, 0) as active_minutes,
        COALESCE(ad.total_tracked_minutes, 0) as total_tracked_minutes,
        COALESCE(ad.idle_minutes, 0) as idle_minutes,
        COALESCE(ad.active_events_count, 0) as active_events_count,
        COALESCE(ad.sessions_count, 0) as sessions_count,
        COALESCE(tb.actual_start, $2::timestamp) as period_start,
        COALESCE(tb.actual_end, $3::timestamp) as period_end,
        0 as total_minutes, -- deprecated field
        COALESCE(dd.unique_domains_count, 0) as unique_domains_count,
        COALESCE(dd.domains_list, ARRAY[]::text[]) as domains_list,
        
        -- Deep Work данные
        (SELECT COALESCE(COUNT(*), 0) FROM deep_work_sessions) as deep_sessions_count,
        (SELECT COALESCE(SUM(duration_minutes), 0) FROM deep_work_sessions) as total_deep_minutes,
        (SELECT COALESCE(AVG(duration_minutes), 0) FROM deep_work_sessions) as avg_deep_minutes,
        (SELECT COALESCE(MAX(duration_minutes), 0) FROM deep_work_sessions) as max_deep_minutes,
        
        -- Top domains для deep work
        (SELECT COALESCE(
            json_agg(
                json_build_object(
                    'domain', domain,
                    'minutes', domain_minutes,
                    'sessions', domain_sessions
                ) ORDER BY domain_minutes DESC
            ),
            '[]'::json
        ) FROM (
            SELECT 
                domain,
                SUM(duration_minutes) as domain_minutes,
                COUNT(*) as domain_sessions
            FROM deep_work_sessions
            GROUP BY domain
            ORDER BY domain_minutes DESC
            LIMIT 3
        ) top_domains_subquery) as top_deep_domains,
        
        -- Hourly breakdown
        (SELECT COALESCE(
            json_agg(
                json_build_object(
                    'hour', hour,
                    'date', date,
                    'engaged_minutes', engaged_minutes,
                    'total_minutes', total_minutes,
                    'idle_minutes', idle_minutes,
                    'active_events', active_events,
                    'sessions_count', sessions_count
                ) ORDER BY date, hour
            ),
            '[]'::json
        ) FROM hourly_stats) as hourly_breakdown_data
        
    FROM time_bounds tb
    FULL OUTER JOIN active_data ad ON true
    FULL OUTER JOIN domains_data dd ON true`,

		fmt.Sprintf(timeBoundsQuery, qb.sessionFilter),
		fmt.Sprintf(activeDataQuery, eventPlaceholders, qb.sessionFilter, activeDataEventPlaceholders1, qb.sessionFilter, qb.sessionFilter),
		fmt.Sprintf(domainsDataQuery, qb.sessionFilter),
		fmt.Sprintf(deepWorkQuery, eventPlaceholders, qb.sessionFilter),
		fmt.Sprintf(hourlyBreakdownQuery, hourlyEventPlaceholders1, qb.sessionFilter, hourlyEventPlaceholders2))
}

func (qb *queryBuilder) buildQueryWithDeepWork() string {
	eventPlaceholders := qb.addActiveEventsPlaceholders()
	activeDataEventPlaceholders1 := qb.addActiveEventsPlaceholders()
	//activeDataEventPlaceholders2 := qb.addActiveEventsPlaceholders()

	return fmt.Sprintf(`
    %s,
    %s,
    %s,
    %s
    SELECT 
        COALESCE(ad.active_minutes, 0) as active_minutes,
        COALESCE(ad.total_tracked_minutes, 0) as total_tracked_minutes,
        COALESCE(ad.idle_minutes, 0) as idle_minutes,
        COALESCE(ad.active_events_count, 0) as active_events_count,
        COALESCE(ad.sessions_count, 0) as sessions_count,
        COALESCE(tb.actual_start, $2::timestamp) as period_start,
        COALESCE(tb.actual_end, $3::timestamp) as period_end,
        0 as total_minutes, -- deprecated field
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
    GROUP BY tb.actual_start, tb.actual_end, 
             ad.active_minutes, ad.total_tracked_minutes, ad.idle_minutes, 
             ad.active_events_count, ad.sessions_count,
             dd.unique_domains_count, dd.domains_list,
             dws.deep_sessions_count, dws.total_deep_minutes, dws.avg_deep_minutes, dws.max_deep_minutes`,

		fmt.Sprintf(timeBoundsQuery, qb.sessionFilter),
		fmt.Sprintf(activeDataQuery, eventPlaceholders, qb.sessionFilter, activeDataEventPlaceholders1, qb.sessionFilter, qb.sessionFilter),
		fmt.Sprintf(domainsDataQuery, qb.sessionFilter),
		fmt.Sprintf(deepWorkQuery, eventPlaceholders, qb.sessionFilter))
}

func calculateEngagementRate(activeMinutes int, totalTrackedMinutes int) float64 {
	if totalTrackedMinutes <= 0 {
		return 0
	}
	return utils.RoundToTwoDecimals((float64(activeMinutes) / float64(totalTrackedMinutes)) * 100)
}

func (r *metricsRepository) buildEngagedTimeMetric(filter entity.EngagedTimeFilter, result engagedTimeResult) *entity.EngagedTimeMetric {
	engagementRate := calculateEngagementRate(result.ActiveMinutes, result.TotalTrackedMinutes)

	//focusLevel := r.determineFocusLevel(result.UniqueDomainsCount)
	//focusInsight := r.generateFocusInsight(result.UniqueDomainsCount, result.DomainsList)

	var topDomains []entity.DeepWorkDomain
	if len(result.TopDeepDomains) > 0 && string(result.TopDeepDomains) != "[]" {
		if err := json.Unmarshal(result.TopDeepDomains, &topDomains); err != nil {
			topDomains = []entity.DeepWorkDomain{}
		}
	}

	var hourlyBreakdown []entity.HourlyData
	if len(result.HourlyBreakdownData) > 0 && string(result.HourlyBreakdownData) != "[]" {
		type hourlyRaw struct {
			Hour           int    `json:"hour"`
			Date           string `json:"date"`
			EngagedMinutes int    `json:"engaged_minutes"`
			TotalMinutes   int    `json:"total_minutes"` // изменено с float64 на int
			IdleMinutes    int    `json:"idle_minutes"`  // добавлено новое поле
			ActiveEvents   int    `json:"active_events"`
			SessionsCount  int    `json:"sessions_count"`
		}

		var rawData []hourlyRaw
		if err := json.Unmarshal(result.HourlyBreakdownData, &rawData); err == nil {
			hourlyBreakdown = make([]entity.HourlyData, len(rawData))
			for i, raw := range rawData {
				totalMins := raw.TotalMinutes
				idleMins := raw.IdleMinutes

				// Проверяем консистентность данных
				if totalMins != (raw.EngagedMinutes + idleMins) {
					// Если есть несоответствие, пересчитываем
					idleMins = totalMins - raw.EngagedMinutes
					if idleMins < 0 {
						idleMins = 0
						totalMins = raw.EngagedMinutes
					}
				}

				var productivity float64
				if totalMins > 0 {
					productivity = utils.RoundToTwoDecimals((float64(raw.EngagedMinutes) / float64(totalMins)) * 100)
				}

				hourlyBreakdown[i] = entity.HourlyData{
					Hour:         raw.Hour,
					Date:         raw.Date,
					Timestamp:    utils.FormatHourTimestamp(raw.Hour),
					EngagedMins:  raw.EngagedMinutes,
					IdleMins:     idleMins,
					TotalMins:    totalMins,
					Events:       raw.ActiveEvents,
					Sessions:     raw.SessionsCount,
					Productivity: productivity,
				}
			}
		}
	}

	// Рассчитываем deep work rate от отслеживаемого времени (не от общего периода)
	var deepWorkRate float64
	if result.TotalTrackedMinutes > 0 {
		deepWorkRate = utils.RoundToTwoDecimals((result.TotalDeepMinutes / float64(result.TotalTrackedMinutes)) * 100)
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
		FocusLevel:         "",
		FocusInsight:       "",
		WorkPattern:        "",
		Recommendations:    []string{},
		DeepWork: entity.DeepWorkData{
			SessionsCount:  result.DeepSessionsCount,
			TotalMinutes:   utils.RoundToTwoDecimals(result.TotalDeepMinutes),
			TotalHours:     utils.RoundToTwoDecimals(result.TotalDeepMinutes / 60),
			AverageMinutes: utils.RoundToTwoDecimals(result.AvgDeepMinutes),
			LongestMinutes: utils.RoundToTwoDecimals(result.MaxDeepMinutes),
			DeepWorkRate:   deepWorkRate,
			TopDomains:     topDomains,
		},
		HourlyBreakdown: hourlyBreakdown,
	}
}

const (
	timeBoundsQuery = `
    WITH time_bounds AS (
        SELECT 
            MIN(timestamp) as actual_start,
            MAX(timestamp) as actual_end
        FROM user_behaviors 
        WHERE user_id = $1 
            AND timestamp >= $2 
            AND timestamp <= $3 %s
    )`

	activeDataQuery = `
    minute_activity AS (
        SELECT
            DATE_TRUNC('minute', ub.timestamp) AS minute,
            MAX(CASE WHEN ub.event_type IN (%s) THEN 1 ELSE 0 END) AS is_active,
            1 AS is_tracked
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end %s
        GROUP BY DATE_TRUNC('minute', ub.timestamp)
    ),
    active_data AS (
        SELECT 
            SUM(is_active) as active_minutes,
            SUM(is_tracked) as total_tracked_minutes,
            SUM(CASE WHEN is_active = 0 THEN 1 ELSE 0 END) as idle_minutes,
            (SELECT COUNT(*) FROM user_behaviors ub2 
             CROSS JOIN time_bounds tb2
             WHERE ub2.user_id = $1 
               AND ub2.timestamp >= tb2.actual_start 
               AND ub2.timestamp <= tb2.actual_end
               AND ub2.event_type IN (%s) %s) as active_events_count,
            (SELECT COUNT(DISTINCT session_id) FROM user_behaviors ub3
             CROSS JOIN time_bounds tb3
             WHERE ub3.user_id = $1 
               AND ub3.timestamp >= tb3.actual_start 
               AND ub3.timestamp <= tb3.actual_end %s) as sessions_count
        FROM minute_activity
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

	hourlyBreakdownQuery = `
    hourly_minute_activity AS (
        SELECT 
            EXTRACT(HOUR FROM ub.timestamp) as hour,
            DATE(ub.timestamp) as date,
            DATE_TRUNC('minute', ub.timestamp) AS minute,
            MAX(CASE WHEN ub.event_type IN (%s) THEN 1 ELSE 0 END) AS is_active,
            ub.session_id
        FROM user_behaviors ub
        CROSS JOIN time_bounds tb
        WHERE ub.user_id = $1 
            AND ub.timestamp >= tb.actual_start 
            AND ub.timestamp <= tb.actual_end %s
        GROUP BY EXTRACT(HOUR FROM ub.timestamp), DATE(ub.timestamp), 
                 DATE_TRUNC('minute', ub.timestamp), ub.session_id
    ),
    hourly_stats AS (
        SELECT 
            hour,
            date,
            SUM(is_active) as engaged_minutes,
            COUNT(*) as total_minutes,
            SUM(CASE WHEN is_active = 0 THEN 1 ELSE 0 END) as idle_minutes,
            (SELECT COUNT(*) FROM user_behaviors ub4
             CROSS JOIN time_bounds tb4
             WHERE ub4.user_id = $1 
               AND EXTRACT(HOUR FROM ub4.timestamp) = hma.hour
               AND DATE(ub4.timestamp) = hma.date
               AND ub4.timestamp >= tb4.actual_start 
               AND ub4.timestamp <= tb4.actual_end
               AND ub4.event_type IN (%s)) as active_events,
            COUNT(DISTINCT session_id) as sessions_count
        FROM hourly_minute_activity hma
        GROUP BY hour, date
        ORDER BY date, hour
    )`
)

func (r *metricsRepository) GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error) {
	qb := newQueryBuilder(filter)
	query := qb.buildQueryWithDeepWorkAndHourly()

	var result engagedTimeResult
	err := r.db.GetContext(ctx, &result, query, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get engaged time: %w", err)
	}

	return r.buildEngagedTimeMetric(filter, result), nil
}

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

const deepWorkSessionsMainQuery = `
WITH active_events AS (
	SELECT
		ub.user_id,
		ub.timestamp,
		ub.event_type,
		ub.url,
		CASE 
			WHEN ub.url ~ '^https?://' THEN 
				split_part(split_part(ub.url, '://', 2), '/', 1)
			ELSE 
				split_part(ub.url, '/', 1)
		END as domain,
		LAG(ub.timestamp) OVER (PARTITION BY ub.user_id ORDER BY ub.timestamp) AS prev_timestamp
	FROM user_behaviors ub
	WHERE ub.user_id = $1 
		AND ub.timestamp >= $2 
		AND ub.timestamp <= $3
		AND ub.event_type IN ('click', 'keydown', 'scrollend') %s
),
gaps_marked AS (
	SELECT *,
		CASE 
			WHEN prev_timestamp IS NULL 
				OR EXTRACT(EPOCH FROM (timestamp - prev_timestamp)) > 300
			THEN 1 ELSE 0 
		END AS is_new_block
	FROM active_events
),
numbered_blocks AS (
	SELECT *,
		SUM(is_new_block) OVER (PARTITION BY user_id ORDER BY timestamp) AS block_id
	FROM gaps_marked
),
block_boundaries AS (
	SELECT
		user_id,
		block_id,
		MIN(timestamp) AS start_time,
		MAX(timestamp) AS last_active_time,
		COUNT(*) AS total_events
	FROM numbered_blocks
	GROUP BY user_id, block_id
),
final_blocks AS (
	SELECT *,
		last_active_time + INTERVAL '5 minutes' AS end_time,
		EXTRACT(EPOCH FROM (last_active_time + INTERVAL '5 minutes') - start_time) / 60.0 AS duration_minutes
	FROM block_boundaries
),
deep_work_blocks AS (
	SELECT *
	FROM final_blocks
	WHERE duration_minutes >= 25
),
context_switches_per_block AS (
	SELECT 
		nb.block_id,
		COUNT(CASE WHEN nb.domain IS DISTINCT FROM prev_domain THEN 1 END) AS context_switches
	FROM (
		SELECT 
			block_id,
			domain,
			LAG(domain) OVER (PARTITION BY block_id ORDER BY timestamp) AS prev_domain
		FROM numbered_blocks
		WHERE block_id IN (SELECT block_id FROM deep_work_blocks)
	) nb
	GROUP BY nb.block_id
),
deep_work_with_switches AS (
	SELECT 
		dwb.*,
		COALESCE(cs.context_switches, 0) AS context_switches,
		ROUND(COALESCE(cs.context_switches * 60.0 / dwb.duration_minutes, 0)::numeric, 2) AS switches_per_hour,
		CASE 
			WHEN COALESCE(cs.context_switches * 60.0 / dwb.duration_minutes, 0) <= 5 THEN 'high'
			WHEN COALESCE(cs.context_switches * 60.0 / dwb.duration_minutes, 0) <= 15 THEN 'medium'
			ELSE 'low'
		END AS focus_level
	FROM deep_work_blocks dwb
	LEFT JOIN context_switches_per_block cs ON dwb.block_id = cs.block_id
),
hourly_basic_stats AS (
	SELECT 
		EXTRACT(HOUR FROM ub.timestamp)::integer as hour,
		DATE(ub.timestamp) as date,
		COUNT(DISTINCT DATE_TRUNC('minute', ub.timestamp))::integer as total_minutes,
		COUNT(DISTINCT ub.session_id)::integer as sessions_count
	FROM user_behaviors ub
	WHERE ub.user_id = $1 
		AND ub.timestamp >= $2 
		AND ub.timestamp <= $3 %s
	GROUP BY EXTRACT(HOUR FROM ub.timestamp), DATE(ub.timestamp)
),
hourly_deep_work AS (
	SELECT 
		EXTRACT(HOUR FROM dwws.start_time)::integer as hour,
		DATE(dwws.start_time) as date,
		SUM(
			EXTRACT(EPOCH FROM 
				LEAST(DATE_TRUNC('hour', dwws.start_time) + INTERVAL '1 hour', dwws.end_time) -
				GREATEST(DATE_TRUNC('hour', dwws.start_time), dwws.start_time)
			) / 60.0
		)::numeric as deep_work_minutes
	FROM deep_work_with_switches dwws
	GROUP BY EXTRACT(HOUR FROM dwws.start_time), DATE(dwws.start_time)
),
hourly_context_switches AS (
	SELECT 
		EXTRACT(HOUR FROM timestamp)::integer as hour,
		DATE(timestamp) as date,
		COUNT(CASE WHEN domain IS DISTINCT FROM prev_domain THEN 1 END)::integer as context_switches
	FROM (
		SELECT 
			timestamp,
			domain,
			LAG(domain) OVER (PARTITION BY EXTRACT(HOUR FROM timestamp), DATE(timestamp) ORDER BY timestamp) as prev_domain
		FROM numbered_blocks
		WHERE block_id IN (SELECT block_id FROM deep_work_blocks)
	) hourly_domains
	GROUP BY EXTRACT(HOUR FROM timestamp), DATE(timestamp)
),
hourly_breakdown AS (
	SELECT 
		hbs.hour,
		hbs.date,
		hbs.total_minutes,
		hbs.sessions_count as sessions,
		COALESCE(hdw.deep_work_minutes, 0::numeric) as deep_work_minutes,
		COALESCE(hcs.context_switches, 0) as context_switches,
		CASE 
			WHEN COALESCE(hdw.deep_work_minutes, 0::numeric) > 0::numeric 
			THEN ROUND((COALESCE(hcs.context_switches, 0)::numeric * 60.0 / COALESCE(hdw.deep_work_minutes, 0::numeric)), 2)
			ELSE 0::numeric 
		END as switches_per_hour,
		CASE 
			WHEN hbs.total_minutes > 0 
			THEN ROUND((COALESCE(hdw.deep_work_minutes, 0::numeric) * 100.0 / hbs.total_minutes::numeric), 2)
			ELSE 0::numeric 
		END as deep_work_rate
	FROM hourly_basic_stats hbs
	LEFT JOIN hourly_deep_work hdw ON hbs.hour = hdw.hour AND hbs.date = hdw.date
	LEFT JOIN hourly_context_switches hcs ON hbs.hour = hcs.hour AND hbs.date = hcs.date
	ORDER BY hbs.date, hbs.hour
),
total_tracked_time AS (
	SELECT 
		COUNT(DISTINCT DATE_TRUNC('minute', ub.timestamp))::integer as total_tracked_minutes
	FROM user_behaviors ub
	WHERE ub.user_id = $1 
		AND ub.timestamp >= $2 
		AND ub.timestamp <= $3 %s
),
aggregated_stats AS (
	SELECT 
		COUNT(*)::integer as sessions_count,
		COALESCE(SUM(duration_minutes), 0::numeric) as total_minutes,
		COALESCE(AVG(duration_minutes), 0::numeric) as average_minutes,
		COALESCE(MAX(duration_minutes), 0::numeric) as longest_minutes,
		COALESCE(MIN(duration_minutes), 0::numeric) as shortest_minutes,
		(SELECT COUNT(DISTINCT domain) FROM numbered_blocks nb 
		 WHERE nb.block_id IN (SELECT block_id FROM deep_work_blocks))::integer as unique_domains,
		COALESCE(SUM(context_switches), 0)::integer as total_context_switches,
		COALESCE(AVG(switches_per_hour), 0::numeric) as avg_switches_per_hour,
		COUNT(CASE WHEN focus_level = 'high' THEN 1 END)::integer as high_focus_blocks,
		COUNT(CASE WHEN focus_level = 'medium' THEN 1 END)::integer as medium_focus_blocks,
		COUNT(CASE WHEN focus_level = 'low' THEN 1 END)::integer as low_focus_blocks,
		CASE 
			WHEN COALESCE(SUM(context_switches), 0) > 0 
			THEN COUNT(*)::numeric / COALESCE(SUM(context_switches), 0)::numeric
			ELSE COUNT(*)::numeric
		END as deep_work_context_ratio
	FROM deep_work_with_switches
),
sessions_detailed AS (
	SELECT 
		COALESCE(
			json_agg(
				json_build_object(
					'block_id', block_id,
					'start_time', start_time,
					'end_time', end_time,
					'duration_minutes', duration_minutes,
					'total_events', total_events,
					'context_switches', context_switches,
					'switches_per_hour', switches_per_hour,
					'focus_level', focus_level
				) ORDER BY start_time
			),
			'[]'::json
		) as sessions_data
	FROM deep_work_with_switches
),
hourly_breakdown_data AS (
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
					'total_mins', total_minutes,
					'sessions', sessions,
					'deep_work_mins', deep_work_minutes,
					'context_switches', context_switches,
					'switches_per_hour', switches_per_hour,
					'deep_work_rate', deep_work_rate
				) ORDER BY date, hour
			),
			'[]'::json
		) as hourly_data
	FROM hourly_breakdown
)
SELECT 
	COALESCE(ag.sessions_count, 0) as sessions_count,
	COALESCE(ag.total_minutes, 0::numeric) as total_minutes,
	COALESCE(ag.average_minutes, 0::numeric) as average_minutes,
	COALESCE(ag.longest_minutes, 0::numeric) as longest_minutes,
	COALESCE(ag.shortest_minutes, 0::numeric) as shortest_minutes,
	COALESCE(ag.unique_domains, 0) as unique_domains,
	COALESCE(ag.total_context_switches, 0) as total_context_switches,
	COALESCE(ag.avg_switches_per_hour, 0::numeric) as avg_switches_per_hour,
	COALESCE(ag.high_focus_blocks, 0) as high_focus_blocks,
	COALESCE(ag.medium_focus_blocks, 0) as medium_focus_blocks,
	COALESCE(ag.low_focus_blocks, 0) as low_focus_blocks,
	COALESCE(ag.deep_work_context_ratio, 0::numeric) as deep_work_context_ratio,
	sd.sessions_data,
	COALESCE(tt.total_tracked_minutes, 0) as total_tracked_minutes,
	hbd.hourly_data as hourly_breakdown_data
FROM aggregated_stats ag
CROSS JOIN sessions_detailed sd
CROSS JOIN total_tracked_time tt
CROSS JOIN hourly_breakdown_data hbd`

func (r *metricsRepository) GetDeepWorkSessions(ctx context.Context, filter entity.DeepWorkSessionsFilter) (*entity.DeepWorkSessionsResponse, error) {
	sessionFilter := ""

	if filter.SessionID != nil {
		sessionFilter = " AND session_id = $4"
	}

	query := fmt.Sprintf(deepWorkSessionsMainQuery, sessionFilter, sessionFilter, sessionFilter)

	args := []interface{}{filter.UserID, filter.StartTime, filter.EndTime}

	if filter.SessionID != nil {
		args = append(args, *filter.SessionID)
	}

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
