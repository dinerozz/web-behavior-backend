// internal/repository/user_behavior_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	uuid2 "github.com/gofrs/uuid"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"strconv"
	"strings"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/jmoiron/sqlx"
)

type UserBehaviorRepository interface {
	Create(ctx context.Context, behavior *entity.UserBehavior) error
	BatchCreate(ctx context.Context, behaviors []entity.UserBehavior) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UserBehavior, error)
	GetByFilter(ctx context.Context, filter entity.UserBehaviorFilter) ([]entity.UserBehavior, error)
	GetStats(ctx context.Context, filter entity.UserBehaviorFilter) (*entity.UserBehaviorStats, error)
	GetSessionSummary(ctx context.Context, sessionID string) (*entity.SessionSummary, error)
	GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]entity.SessionSummary, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userBehaviorRepository struct {
	db *sqlx.DB
}

func NewUserBehaviorRepository(db *sqlx.DB) UserBehaviorRepository {
	return &userBehaviorRepository{db: db}
}

func (r *userBehaviorRepository) Create(ctx context.Context, behavior *entity.UserBehavior) error {
	behavior.ID = uuid2.UUID(uuid.New())
	behavior.CreatedAt = time.Now()
	behavior.UpdatedAt = time.Now()

	query := `
		INSERT INTO user_behaviors (id, session_id, timestamp, event_type, url, user_id, user_name, x, y, key, created_at, updated_at)
		VALUES (:id, :session_id, :timestamp, :event_type, :url, :user_id, :user_name, :x, :y, :key, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, behavior)
	return err
}

func (r *userBehaviorRepository) BatchCreate(ctx context.Context, behaviors []entity.UserBehavior) error {
	if len(behaviors) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO user_behaviors (id, session_id, timestamp, event_type, url, user_id, user_name, x, y, key, created_at, updated_at)
		VALUES (:id, :session_id, :timestamp, :event_type, :url, :user_id, :user_name, :x, :y, :key, :created_at, :updated_at)`

	for i := range behaviors {
		behaviors[i].ID = uuid2.UUID(uuid.New())
		behaviors[i].CreatedAt = time.Now()
		behaviors[i].UpdatedAt = time.Now()
	}

	_, err = tx.NamedExecContext(ctx, query, behaviors)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *userBehaviorRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserBehavior, error) {
	var behavior entity.UserBehavior
	query := `SELECT * FROM user_behaviors WHERE id = $1`

	err := r.db.GetContext(ctx, &behavior, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &behavior, nil
}

func (r *userBehaviorRepository) GetByFilter(ctx context.Context, filter entity.UserBehaviorFilter) ([]entity.UserBehavior, error) {
	var behaviors []entity.UserBehavior

	query := "SELECT * FROM user_behaviors WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.SessionID != nil {
		query += fmt.Sprintf(" AND session_id = $%d", argIndex)
		args = append(args, *filter.SessionID)
		argIndex++
	}

	if filter.EventType != nil {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, *filter.EventType)
		argIndex++
	}

	if filter.URL != nil {
		query += fmt.Sprintf(" AND url ILIKE $%d", argIndex)
		args = append(args, "%"+*filter.URL+"%")
		argIndex++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *filter.EndTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	err := r.db.SelectContext(ctx, &behaviors, query, args...)
	return behaviors, err
}

func (r *userBehaviorRepository) GetStats(ctx context.Context, filter entity.UserBehaviorFilter) (*entity.UserBehaviorStats, error) {
	stats := &entity.UserBehaviorStats{
		EventsByType: make(map[string]int64),
	}

	// Базовые условия для всех запросов
	whereClause, args := r.buildWhereClause(filter)

	// Общее количество событий
	totalQuery := "SELECT COUNT(*) FROM user_behaviors" + whereClause
	err := r.db.GetContext(ctx, &stats.TotalEvents, totalQuery, args...)
	if err != nil {
		return nil, err
	}

	// Уникальные пользователи
	uniqueUsersWhereClause, uniqueUsersArgs := r.buildWhereClauseWithExtra(filter, "user_id IS NOT NULL")
	uniqueUsersQuery := "SELECT COUNT(DISTINCT user_id) FROM user_behaviors" + uniqueUsersWhereClause
	err = r.db.GetContext(ctx, &stats.UniqueUsers, uniqueUsersQuery, uniqueUsersArgs...)
	if err != nil {
		return nil, err
	}

	// Уникальные сессии
	uniqueSessionsQuery := "SELECT COUNT(DISTINCT session_id) FROM user_behaviors" + whereClause
	err = r.db.GetContext(ctx, &stats.UniqueSessions, uniqueSessionsQuery, args...)
	if err != nil {
		return nil, err
	}

	// События по типам
	eventTypesQuery := "SELECT event_type, COUNT(*) FROM user_behaviors" + whereClause + " GROUP BY event_type"
	rows, err := r.db.QueryContext(ctx, eventTypesQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		stats.EventsByType[eventType] = count
	}

	// Популярные URL
	urlsQuery := "SELECT url, COUNT(*) as count FROM user_behaviors" + whereClause + " GROUP BY url ORDER BY count DESC LIMIT 10"
	err = r.db.SelectContext(ctx, &stats.PopularURLs, urlsQuery, args...)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *userBehaviorRepository) GetSessionSummary(ctx context.Context, sessionID string) (*entity.SessionSummary, error) {
	query := `
		SELECT 
			session_id,
			user_id,
			user_name,
			MIN(timestamp) as start_time,
			MAX(timestamp) as end_time,
			EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) as duration,
			COUNT(*) as events_count,
			array_agg(DISTINCT url) as urls
		FROM user_behaviors 
		WHERE session_id = $1 
		GROUP BY session_id, user_id, user_name`

	var summary entity.SessionSummary
	var urls []string

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&summary.SessionID,
		&summary.UserID,
		&summary.UserName,
		&summary.StartTime,
		&summary.EndTime,
		&summary.Duration,
		&summary.EventsCount,
		(*StringSlice)(&urls),
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	summary.URLs = urls
	return &summary, nil
}

func (r *userBehaviorRepository) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]entity.SessionSummary, error) {
	query := `
       SELECT 
          session_id,
          user_id,
          user_name,
          MIN(timestamp) as start_time,
          MAX(timestamp) as end_time,
          EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) as duration,
          COUNT(*) as events_count,
          array_agg(DISTINCT url) as urls
       FROM user_behaviors 
       WHERE user_id = $1
       GROUP BY session_id, user_id, user_name
       ORDER BY start_time DESC
       LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []entity.SessionSummary
	for rows.Next() {
		var session entity.SessionSummary
		var urls pq.StringArray
		var durationStr string

		err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.UserName,
			&session.StartTime,
			&session.EndTime,
			&durationStr,
			&session.EventsCount,
			&urls,
		)
		if err != nil {
			return nil, err
		}

		duration, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration: %v", err)
		}
		session.Duration = duration

		session.URLs = []string(urls)
		sessions = append(sessions, session)
	}

	return sessions, nil
}
func (r *userBehaviorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM user_behaviors WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *userBehaviorRepository) buildWhereClause(filter entity.UserBehaviorFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIndex))
		args = append(args, *filter.SessionID)
		argIndex++
	}

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, *filter.EventType)
		argIndex++
	}

	if filter.URL != nil {
		conditions = append(conditions, fmt.Sprintf("url ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.URL+"%")
		argIndex++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *userBehaviorRepository) buildWhereClauseWithExtra(filter entity.UserBehaviorFilter, extraConditions ...string) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Основные условия из фильтра
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIndex))
		args = append(args, *filter.SessionID)
		argIndex++
	}

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, *filter.EventType)
		argIndex++
	}

	if filter.URL != nil {
		conditions = append(conditions, fmt.Sprintf("url ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.URL+"%")
		argIndex++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	// Добавляем дополнительные условия
	conditions = append(conditions, extraConditions...)

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// StringSlice для работы с PostgreSQL массивами
type StringSlice []string

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into StringSlice", value)
	}

	// Обработка PostgreSQL array формата {item1,item2,item3}
	str = strings.Trim(str, "{}")
	if str == "" {
		*s = []string{}
		return nil
	}

	*s = strings.Split(str, ",")
	return nil
}
