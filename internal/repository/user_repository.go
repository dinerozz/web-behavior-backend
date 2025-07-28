package repository

import (
	"database/sql"
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUserWithPassword(user *request.CreateUserWithPassword) (response.User, error) {
	query := `INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id, username`

	var userID uuid.UUID
	var username sql.NullString

	err := r.db.QueryRow(query, user.Username, user.Password).Scan(&userID, &username)
	if err != nil {
		return response.User{}, err
	}

	return response.User{
		ID:       userID,
		Username: username.String,
	}, nil
}

func (r *UserRepository) CreateOrAuthenticateUserWithPassword(user *request.CreateUserWithPassword) (response.User, error) {
	query := `SELECT id, username FROM users WHERE username = $1`

	var userID uuid.UUID
	var username sql.NullString
	err := r.db.QueryRow(query, user.Username).Scan(&userID, &username)
	if err != nil {
		if err == sql.ErrNoRows {
			query := `INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id, username`
			err := r.db.QueryRow(query, user.Username, user.Password).Scan(&userID, &username)
			if err != nil {
				return response.User{}, err
			}
		} else {
			return response.User{}, err
		}
	}

	return response.User{
		ID:       userID,
		Username: username.String,
	}, nil
}

func (r *UserRepository) GetUserById(userID uuid.UUID) (response.User, error) {
	query := `SELECT u.id, u.username, is_super_admin
    FROM users u WHERE u.id = $1`

	user := response.User{}

	err := r.db.QueryRow(query, userID).Scan(&user.ID, &user.Username, &user.IsSuperAdmin)
	if err != nil {
		return response.User{}, err
	}

	if user.IsSuperAdmin != nil && *user.IsSuperAdmin == false {
		return response.User{
			ID:       user.ID,
			Username: user.Username,
		}, nil
	}

	return user, nil
}

func (r *UserRepository) GetUserByUsername(username string) (response.User, error) {
	query := `SELECT id, username, is_super_admin, password FROM users WHERE username = $1`

	var user response.User
	var password sql.NullString
	err := r.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.IsSuperAdmin, &password)
	if err != nil {
		return response.User{}, err
	}

	if password.Valid {
		user.Password = &password.String
	}

	return user, nil
}

func (r *UserRepository) IsUserSuperAdmin(userID uuid.UUID) (bool, error) {
	query := `SELECT is_super_admin FROM users WHERE id = $1`

	var isSuperAdmin sql.NullBool
	err := r.db.QueryRow(query, userID).Scan(&isSuperAdmin)
	if err != nil {
		return false, err
	}

	return isSuperAdmin.Valid && isSuperAdmin.Bool, nil
}

func (r *UserRepository) GetAllUsers() ([]response.User, error) {
	query := `SELECT id, username, is_super_admin, created_at, updated_at FROM users ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []response.User
	for rows.Next() {
		var user response.User
		var isSuperAdmin sql.NullBool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(&user.ID, &user.Username, &isSuperAdmin, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		if isSuperAdmin.Valid {
			user.IsSuperAdmin = &isSuperAdmin.Bool
		}

		if createdAt.Valid {
			user.CreatedAt = &createdAt.Time
		}

		if updatedAt.Valid {
			user.UpdatedAt = &updatedAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
