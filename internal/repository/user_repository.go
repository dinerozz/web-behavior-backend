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
	query := `SELECT u.id, u.username 
    FROM users u WHERE u.id = $1`

	user := response.User{}

	err := r.db.QueryRow(query, userID).Scan(&user.ID, &user.Username)
	if err != nil {
		return response.User{}, err
	}

	return user, nil
}

func (r *UserRepository) GetUserByUsername(username string) (response.User, error) {
	query := `SELECT id, username, password FROM users WHERE username = $1`

	var user response.User
	var password sql.NullString
	err := r.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &password)
	if err != nil {
		return response.User{}, err
	}

	if password.Valid {
		user.Password = &password.String
	}

	return user, nil
}
