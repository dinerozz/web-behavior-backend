// internal/service/user_behavior_service.go
package service

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/google/uuid"
)

type UserBehaviorService interface {
	CreateBehavior(ctx context.Context, req entity.CreateUserBehaviorRequest) (*entity.UserBehavior, error)
	BatchCreateBehaviors(ctx context.Context, req entity.BatchCreateUserBehaviorRequest) error
	GetBehaviorByID(ctx context.Context, id uuid.UUID) (*entity.UserBehavior, error)
	GetBehaviors(ctx context.Context, filter entity.UserBehaviorFilter) ([]entity.UserBehavior, *entity.PaginationInfo, error)
	GetStats(ctx context.Context, filter entity.UserBehaviorFilter) (*entity.UserBehaviorStats, error)
	GetSessionSummary(ctx context.Context, sessionID string) (*entity.SessionSummary, error)
	GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]entity.SessionSummary, error)
	DeleteBehavior(ctx context.Context, id uuid.UUID) error
	ValidateEventType(eventType string) bool
	ValidateCoordinates(x, y *int, eventType string) error
}

type userBehaviorService struct {
	repo repository.UserBehaviorRepository
}

func NewUserBehaviorService(repo repository.UserBehaviorRepository) UserBehaviorService {
	return &userBehaviorService{
		repo: repo,
	}
}

// event types
var validEventTypes = map[string]bool{
	"pageshow":           true,
	"click":              true,
	"focus":              true,
	"blur":               true,
	"keydown":            true,
	"visibility_hidden":  true,
	"visibility_visible": true,
	"idle":               true,
	"scrollend":          true,
	"pagehide":           true,
}

func (s *userBehaviorService) CreateBehavior(ctx context.Context, req entity.CreateUserBehaviorRequest) (*entity.UserBehavior, error) {
	if !s.ValidateEventType(req.Type) {
		return nil, fmt.Errorf("invalid event type: %s", req.Type)
	}

	if err := s.ValidateCoordinates(req.X, req.Y, req.Type); err != nil {
		return nil, err
	}

	behavior := &entity.UserBehavior{
		SessionID: req.SessionID,
		Timestamp: req.Timestamp,
		Type:      req.Type,
		URL:       req.URL,
		UserID:    req.UserID,
		UserName:  req.UserName,
		X:         req.X,
		Y:         req.Y,
		Key:       req.Key,
	}

	if err := s.repo.Create(ctx, behavior); err != nil {
		return nil, fmt.Errorf("failed to create behavior: %w", err)
	}

	return behavior, nil
}

func (s *userBehaviorService) BatchCreateBehaviors(ctx context.Context, req entity.BatchCreateUserBehaviorRequest) error {
	if len(req.Events) == 0 {
		return fmt.Errorf("no events provided")
	}

	if len(req.Events) > 1000 {
		return fmt.Errorf("too many events, maximum is 1000")
	}

	var behaviors []entity.UserBehavior

	for i, event := range req.Events {
		// Валидация каждого события
		if !s.ValidateEventType(event.Type) {
			return fmt.Errorf("invalid event type at index %d: %s", i, event.Type)
		}

		if err := s.ValidateCoordinates(event.X, event.Y, event.Type); err != nil {
			return fmt.Errorf("validation error at index %d: %w", i, err)
		}

		behavior := entity.UserBehavior{
			SessionID: event.SessionID,
			Timestamp: event.Timestamp,
			Type:      event.Type,
			URL:       event.URL,
			UserID:    event.UserID,
			UserName:  event.UserName,
			X:         event.X,
			Y:         event.Y,
			Key:       event.Key,
		}

		behaviors = append(behaviors, behavior)
	}

	// Массовое сохранение
	if err := s.repo.BatchCreate(ctx, behaviors); err != nil {
		return fmt.Errorf("failed to batch create behaviors: %w", err)
	}

	return nil
}

func (s *userBehaviorService) GetBehaviorByID(ctx context.Context, id uuid.UUID) (*entity.UserBehavior, error) {
	behavior, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get behavior: %w", err)
	}

	if behavior == nil {
		return nil, fmt.Errorf("behavior not found")
	}

	return behavior, nil
}

func (s *userBehaviorService) GetBehaviors(ctx context.Context, filter entity.UserBehaviorFilter) ([]entity.UserBehavior, *entity.PaginationInfo, error) {
	if filter.Page > 0 && filter.PerPage > 0 {
		if filter.PerPage > 1000 {
			filter.PerPage = 1000
		}
		if filter.Page < 1 {
			filter.Page = 1
		}
	} else {
		// Старая логика для совместимости
		if filter.Limit <= 0 {
			filter.Limit = 100
		}
		if filter.Limit > 1000 {
			filter.Limit = 1000
		}
	}

	behaviors, err := s.repo.GetByFilter(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get behaviors: %w", err)
	}

	var paginationInfo *entity.PaginationInfo
	if filter.Page > 0 && filter.PerPage > 0 {
		total, err := s.repo.CountByFilter(ctx, filter)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to count behaviors: %w", err)
		}

		totalPages := (total + filter.PerPage - 1) / filter.PerPage
		paginationInfo = &entity.PaginationInfo{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      total,
			TotalPages: totalPages,
		}
	}

	return behaviors, paginationInfo, nil
}

func (s *userBehaviorService) GetStats(ctx context.Context, filter entity.UserBehaviorFilter) (*entity.UserBehaviorStats, error) {
	stats, err := s.repo.GetStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

func (s *userBehaviorService) GetSessionSummary(ctx context.Context, sessionID string) (*entity.SessionSummary, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	summary, err := s.repo.GetSessionSummary(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session summary: %w", err)
	}

	if summary == nil {
		return nil, fmt.Errorf("session not found")
	}

	return summary, nil
}

func (s *userBehaviorService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]entity.SessionSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	sessions, err := s.repo.GetUserSessions(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return sessions, nil
}

func (s *userBehaviorService) DeleteBehavior(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete behavior: %w", err)
	}

	return nil
}

func (s *userBehaviorService) ValidateEventType(eventType string) bool {
	return validEventTypes[eventType]
}

func (s *userBehaviorService) ValidateCoordinates(x, y *int, eventType string) error {
	// Для событий click координаты обязательны
	if eventType == "click" {
		if x == nil || y == nil {
			return fmt.Errorf("coordinates (x, y) are required for click events")
		}

		if *x > 10000 || *y > 10000 {
			return fmt.Errorf("coordinates seem too large")
		}
	}

	// Для других типов событий координаты необязательны, но если есть - валидируем
	if (x != nil || y != nil) && eventType != "click" {
		if x != nil && (*x < 0 || *x > 10000) {
			return fmt.Errorf("invalid x coordinate")
		}
		if y != nil && (*y < 0 || *y > 10000) {
			return fmt.Errorf("invalid y coordinate")
		}
	}

	return nil
}
