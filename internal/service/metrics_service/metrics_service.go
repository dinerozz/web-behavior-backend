package service

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"time"
)

type MetricsService struct {
	repo repository.UserMetricsRepository
}

func NewMetricsService(repo repository.UserMetricsRepository) *MetricsService {
	return &MetricsService{repo: repo}
}

func (s *MetricsService) GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error) {
	if filter.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time and end_time are required")
	}

	if filter.EndTime.Before(filter.StartTime) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	// Ограничение на максимальный период (например, 90 дней)
	if filter.EndTime.Sub(filter.StartTime) > 90*24*time.Hour {
		return nil, fmt.Errorf("period cannot exceed 90 days")
	}

	metric, err := s.repo.GetTrackedTime(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tracked time: %w", err)
	}

	return metric, nil
}

func (s *MetricsService) GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error) {
	if filter.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time and end_time are required")
	}

	metric, err := s.repo.GetTrackedTimeTotal(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total tracked time: %w", err)
	}

	return metric, nil
}
