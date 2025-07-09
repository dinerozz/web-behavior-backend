// internal/service/extension_user_service.go
package service

import (
	"context"
	"fmt"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/gofrs/uuid"
)

type ExtensionUserService interface {
	CreateUser(ctx context.Context, req entity.CreateExtensionUserRequest) (*entity.ExtensionUser, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.ExtensionUser, error)
	GetUserByAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error)
	GetUserByUsername(ctx context.Context, username string) (*entity.ExtensionUserPublic, error)
	GetAllUsers(ctx context.Context, filter entity.ExtensionUserFilter) ([]entity.ExtensionUserPublic, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req entity.UpdateExtensionUserRequest) (*entity.ExtensionUserPublic, error)
	RegenerateAPIKey(ctx context.Context, id uuid.UUID) (*entity.RegenerateAPIKeyResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ValidateAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error)
	GetStats(ctx context.Context) (*entity.ExtensionUserStats, error)
}

type extensionUserService struct {
	repo repository.ExtensionUserRepository
}

func NewExtensionUserService(repo repository.ExtensionUserRepository) ExtensionUserService {
	return &extensionUserService{
		repo: repo,
	}
}

func (s *extensionUserService) CreateUser(ctx context.Context, req entity.CreateExtensionUserRequest) (*entity.ExtensionUser, error) {
	existingUser, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("username already exists")
	}

	user := &entity.ExtensionUser{
		Username: req.Username,
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create extension user: %w", err)
	}

	return user, nil
}

func (s *extensionUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.ExtensionUser, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *extensionUserService) GetUserByAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	user, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by API key: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid API key")
	}

	go func() {
		s.repo.UpdateLastUsed(context.Background(), apiKey)
	}()

	return user, nil
}

func (s *extensionUserService) GetUserByUsername(ctx context.Context, username string) (*entity.ExtensionUserPublic, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.toPublicUser(user), nil
}

func (s *extensionUserService) GetAllUsers(ctx context.Context, filter entity.ExtensionUserFilter) ([]entity.ExtensionUserPublic, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	users, err := s.repo.GetAll(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	publicUsers := make([]entity.ExtensionUserPublic, len(users))
	for i, user := range users {
		publicUsers[i] = *s.toPublicUser(&user)
	}

	return publicUsers, nil
}

func (s *extensionUserService) UpdateUser(ctx context.Context, id uuid.UUID, req entity.UpdateExtensionUserRequest) (*entity.ExtensionUserPublic, error) {
	// Проверяем существование пользователя
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Проверяем уникальность username если он обновляется
	if req.Username != nil && *req.Username != existingUser.Username {
		userWithSameUsername, err := s.repo.GetByUsername(ctx, *req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
		}
		if userWithSameUsername != nil {
			return nil, fmt.Errorf("username already exists")
		}
	}

	updatedUser, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	if updatedUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.toPublicUser(updatedUser), nil
}

func (s *extensionUserService) RegenerateAPIKey(ctx context.Context, id uuid.UUID) (*entity.RegenerateAPIKeyResponse, error) {
	// Проверяем существование пользователя
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	if !existingUser.IsActive {
		return nil, fmt.Errorf("user is inactive")
	}

	newAPIKey, err := s.repo.RegenerateAPIKey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate API key: %w", err)
	}

	return &entity.RegenerateAPIKeyResponse{
		ID:     id,
		APIKey: newAPIKey,
	}, nil
}

func (s *extensionUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *extensionUserService) ValidateAPIKey(ctx context.Context, apiKey string) (*entity.ExtensionUser, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	user, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid or inactive API key")
	}

	go func() {
		s.repo.UpdateLastUsed(context.Background(), apiKey)
	}()

	return user, nil
}

func (s *extensionUserService) GetStats(ctx context.Context) (*entity.ExtensionUserStats, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// toPublicUser конвертирует ExtensionUser в ExtensionUserPublic (скрывает API ключ)
func (s *extensionUserService) toPublicUser(user *entity.ExtensionUser) *entity.ExtensionUserPublic {
	return &entity.ExtensionUserPublic{
		ID:         user.ID,
		Username:   user.Username,
		IsActive:   user.IsActive,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
		LastUsedAt: user.LastUsedAt,
	}
}
