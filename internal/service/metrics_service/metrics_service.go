package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/dinerozz/web-behavior-backend/internal/service/ai_analytics"
	"time"
)

type MetricsService struct {
	repo      repository.UserMetricsRepository
	aiService *ai_analytics.AIAnalyticsService
}

func NewMetricsService(repo repository.UserMetricsRepository, aiService *ai_analytics.AIAnalyticsService) *MetricsService {
	return &MetricsService{repo: repo, aiService: aiService}
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

	metric, err := s.repo.GetTrackedTimeTotal(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total tracked time: %w", err)
	}

	return metric, nil
}

func (s *MetricsService) GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error) {
	if filter.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	//if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
	//	return nil, fmt.Errorf("start_time and end_time are required")
	//}
	//
	//if filter.EndTime.Before(filter.StartTime) {
	//	return nil, fmt.Errorf("end_time must be after start_time")
	//}

	if filter.EndTime.Sub(filter.StartTime) > 90*24*time.Hour {
		return nil, fmt.Errorf("period cannot exceed 90 days")
	}

	metric, err := s.repo.GetEngagedTime(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate engaged time: %w", err)
	}

	if s.aiService != nil {
		analysis, err := s.aiService.AnalyzeDomainUsage(
			ctx,
			metric.UniqueDomainsCount,
			metric.DomainsList,
			metric.DeepWork,
			metric.EngagementRate,
			metric.TrackedHours,
		)
		if err != nil {
			metric.FocusLevel = s.aiService.DetermineFocusLevelFallback(metric.UniqueDomainsCount)
		} else {
			metric.FocusLevel = analysis.FocusLevel
			metric.FocusInsight = analysis.FocusInsight
			metric.WorkPattern = analysis.WorkPattern
			metric.Recommendations = analysis.Recommendations
			metric.Analysis = analysis.Analysis
		}
	}

	return metric, nil
}

func (s *MetricsService) GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error) {
	if filter.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	return s.repo.GetTopDomains(ctx, filter)
}
