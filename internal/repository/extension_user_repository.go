// internal/repository/extension_user_repository.go
package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type ExtensionUserRepository interface {
	Create(ctx context.Context, user *entity.ExtensionUser) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ExtensionUser, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error)
	GetByUsername(ctx context.Context, username string) (*entity.ExtensionUser, error)
	GetAll(ctx context.Context, filter entity.ExtensionUserFilter) ([]entity.ExtensionUser, error)
	Update(ctx context.Context, id uuid.UUID, req entity.UpdateExtensionUserRequest) (*entity.ExtensionUser, error)
	RegenerateAPIKey(ctx context.Context, id uuid.UUID) (string, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, apiKey string) error
	GetStats(ctx context.Context) (*entity.ExtensionUserStats, error)
	IsAPIKeyValid(ctx context.Context, apiKey string) bool
}

type extensionUserRepository struct {
	db *sqlx.DB
}

func NewExtensionUserRepository(db *sqlx.DB) ExtensionUserRepository {
	return &extensionUserRepository{db: db}
}

func (r *extensionUserRepository) Create(ctx context.Context, user *entity.ExtensionUser) error {
	user.ID = uuid.Must(uuid.NewV4())
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	apiKey, err := r.generateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}
	user.APIKey = apiKey

	query := `
		INSERT INTO extension_users (id, username, api_key, is_active, created_at, updated_at)
		VALUES (:id, :username, :api_key, :is_active, :created_at, :updated_at)`

	_, err = r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to create extension user: %w", err)
	}

	return nil
}

func (r *extensionUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ExtensionUser, error) {
	var user entity.ExtensionUser
	query := `SELECT * FROM extension_users WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get extension user by ID: %w", err)
	}

	return &user, nil
}

func (r *extensionUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error) {
	var user entity.ExtensionUser
	query := `SELECT * FROM extension_users WHERE api_key = $1 AND is_active = true`

	err := r.db.GetContext(ctx, &user, query, apiKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get extension user by API key: %w", err)
	}

	return &user, nil
}

func (r *extensionUserRepository) GetByUsername(ctx context.Context, username string) (*entity.ExtensionUser, error) {
	var user entity.ExtensionUser
	query := `SELECT * FROM extension_users WHERE username = $1`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get extension user by username: %w", err)
	}

	return &user, nil
}

func (r *extensionUserRepository) GetAll(ctx context.Context, filter entity.ExtensionUserFilter) ([]entity.ExtensionUser, error) {
	var users []entity.ExtensionUser

	query := "SELECT * FROM extension_users WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.Username != "" {
		query += fmt.Sprintf(" AND username ILIKE $%d", argIndex)
		args = append(args, "%"+filter.Username+"%")
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

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

	err := r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension users: %w", err)
	}

	return users, nil
}

func (r *extensionUserRepository) Update(ctx context.Context, id uuid.UUID, req entity.UpdateExtensionUserRequest) (*entity.ExtensionUser, error) {
	var setParts []string
	var args []interface{}
	argIndex := 1

	if req.Username != nil {
		setParts = append(setParts, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, *req.Username)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE extension_users 
		SET %s
		WHERE id = $%d
		RETURNING *`, strings.Join(setParts, ", "), argIndex)

	var user entity.ExtensionUser
	err := r.db.GetContext(ctx, &user, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update extension user: %w", err)
	}

	return &user, nil
}

func (r *extensionUserRepository) RegenerateAPIKey(ctx context.Context, id uuid.UUID) (string, error) {
	newAPIKey, err := r.generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate new API key: %w", err)
	}

	query := `
		UPDATE extension_users 
		SET api_key = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_active = true`

	result, err := r.db.ExecContext(ctx, query, newAPIKey, id)
	if err != nil {
		return "", fmt.Errorf("failed to update API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return "", fmt.Errorf("user not found or inactive")
	}

	return newAPIKey, nil
}

func (r *extensionUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM extension_users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete extension user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *extensionUserRepository) UpdateLastUsed(ctx context.Context, apiKey string) error {
	query := `
		UPDATE extension_users 
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE api_key = $1 AND is_active = true`

	_, err := r.db.ExecContext(ctx, query, apiKey)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	return nil
}

func (r *extensionUserRepository) GetStats(ctx context.Context) (*entity.ExtensionUserStats, error) {
	var stats entity.ExtensionUserStats

	// Общая статистика
	generalQuery := `
		SELECT 
			COUNT(*) as total_users,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_users,
			COUNT(CASE WHEN is_active = false THEN 1 END) as inactive_users
		FROM extension_users`

	err := r.db.QueryRowContext(ctx, generalQuery).Scan(
		&stats.TotalUsers,
		&stats.ActiveUsers,
		&stats.InactiveUsers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get general stats: %w", err)
	}

	// Пользователи, использовавшие API сегодня
	todayQuery := `
		SELECT COUNT(*) 
		FROM extension_users 
		WHERE last_used_at >= CURRENT_DATE 
		AND is_active = true`

	err = r.db.QueryRowContext(ctx, todayQuery).Scan(&stats.UsersUsedToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get today stats: %w", err)
	}

	// Пользователи, использовавшие API на этой неделе
	weekQuery := `
		SELECT COUNT(*) 
		FROM extension_users 
		WHERE last_used_at >= CURRENT_DATE - INTERVAL '7 days'
		AND is_active = true`

	err = r.db.QueryRowContext(ctx, weekQuery).Scan(&stats.UsersUsedThisWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get week stats: %w", err)
	}

	return &stats, nil
}

func (r *extensionUserRepository) IsAPIKeyValid(ctx context.Context, apiKey string) bool {
	var count int
	query := `SELECT COUNT(*) FROM extension_users WHERE api_key = $1 AND is_active = true`

	err := r.db.QueryRowContext(ctx, query, apiKey).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

// generateAPIKey генерирует уникальный API ключ
func (r *extensionUserRepository) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	apiKey := "wb_" + hex.EncodeToString(bytes)

	var count int
	query := `SELECT COUNT(*) FROM extension_users WHERE api_key = $1`
	err := r.db.QueryRow(query, apiKey).Scan(&count)
	if err != nil {
		return "", err
	}

	// Если ключ уже существует, генерируем новый
	if count > 0 {
		return r.generateAPIKey()
	}

	return apiKey, nil
}
