package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
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

	if filter.EndTime.Sub(filter.StartTime) > 90*24*time.Hour {
		return nil, fmt.Errorf("period cannot exceed 90 days")
	}

	metric, err := s.repo.GetEngagedTime(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate engaged time: %w", err)
	}

	metric.FocusLevel = s.determineFocusLevelBasic(metric.UniqueDomainsCount)
	metric.FocusInsight = s.generateBasicFocusInsight(metric.UniqueDomainsCount, metric.DomainsList)
	metric.WorkPattern = s.determineWorkPatternBasic(metric)
	metric.Recommendations = s.generateBasicRecommendations(metric)

	return metric, nil
}

func (s *MetricsService) GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error) {
	if filter.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	return s.repo.GetTopDomains(ctx, filter)
}

func (s *MetricsService) GetDeepWorkSessions(ctx context.Context, filter entity.DeepWorkSessionsFilter) (*entity.DeepWorkSessionsResponse, error) {
	return s.repo.GetDeepWorkSessions(ctx, filter)
}

func (s *MetricsService) determineFocusLevelBasic(domainsCount int) string {
	switch {
	case domainsCount <= 5:
		return "high"
	case domainsCount <= 15:
		return "medium"
	default:
		return "low"
	}
}

func (s *MetricsService) generateBasicFocusInsight(domainsCount int, domains []string) string {
	switch {
	case domainsCount <= 5:
		return fmt.Sprintf("Высокая концентрация: работа в %d доменах указывает на фокусированную деятельность", domainsCount)
	case domainsCount <= 15:
		return fmt.Sprintf("Средняя концентрация: %d доменов говорит о сбалансированной многозадачности", domainsCount)
	case domainsCount <= 25:
		return fmt.Sprintf("Низкая концентрация: %d доменов может указывать на частые переключения контекста", domainsCount)
	default:
		return fmt.Sprintf("Очень низкая концентрация: %d доменов указывает на высокую фрагментацию внимания", domainsCount)
	}
}

func (s *MetricsService) determineWorkPatternBasic(metric *entity.EngagedTimeMetric) string {
	deepWorkRate := 0.0
	if metric.DeepWork.TotalMinutes > 0 && metric.TrackedMinutes > 0 {
		deepWorkRate = (metric.DeepWork.TotalMinutes / metric.TrackedMinutes) * 100
	}

	switch {
	case deepWorkRate >= 60:
		return "deep_focused"
	case deepWorkRate >= 30:
		return "balanced"
	case metric.UniqueDomainsCount > 20:
		return "high_switching"
	case metric.EngagementRate < 50:
		return "low_engagement"
	default:
		return "mixed"
	}
}

func (s *MetricsService) generateBasicRecommendations(metric *entity.EngagedTimeMetric) []string {
	var recommendations []string

	if metric.UniqueDomainsCount > 20 {
		recommendations = append(recommendations, "Рассмотрите возможность сокращения количества используемых инструментов для повышения концентрации")
	}

	if metric.EngagementRate < 50 {
		recommendations = append(recommendations, "Попробуйте техники активного планирования для повышения вовлеченности в работу")
	}

	if metric.DeepWork.SessionsCount == 0 {
		recommendations = append(recommendations, "Планируйте блоки времени для глубокой работы продолжительностью не менее 25 минут")
	} else if metric.DeepWork.AverageMinutes < 45 {
		recommendations = append(recommendations, "Увеличьте продолжительность сессий глубокой работы для лучшей концентрации")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Продолжайте поддерживать текущий уровень продуктивности")
	}

	return recommendations
}

func (s *MetricsService) PrepareAIAnalyticsData(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.AIAnalyticsRequest, error) {
	metric, err := s.GetEngagedTime(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get engaged time data: %w", err)
	}

	// Проверяем достаточность данных для AI анализа
	if metric.ActiveMinutes == 0 && metric.DeepWork.SessionsCount == 0 {
		return nil, fmt.Errorf("insufficient data for AI analysis")
	}

	return &entity.AIAnalyticsRequest{
		DomainsCount:   metric.UniqueDomainsCount,
		Domains:        metric.DomainsList,
		DeepWork:       metric.DeepWork,
		EngagementRate: metric.EngagementRate,
		TrackedHours:   metric.TrackedHours,
		UserID:         filter.UserID,
		Period:         metric.Period,
		Timestamp:      time.Now(),
	}, nil
}
