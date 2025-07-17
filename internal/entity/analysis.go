package entity

import "time"

type DetailedAnalysis struct {
	DomainBreakdown   DomainBreakdown   `json:"domain_breakdown"`
	ProductivityScore ProductivityScore `json:"productivity_score"`
	BehaviorInsights  []string          `json:"behavior_insights"`
	KeyFindings       []string          `json:"key_findings"`
}

type DomainBreakdown struct {
	WorkTools     []string `json:"work_tools"`    // GitHub, Jira, etc.
	Development   []string `json:"development"`   // localhost, CodeSandbox, etc.
	Research      []string `json:"research"`      // Stack Overflow, docs, etc.
	Communication []string `json:"communication"` // Gmail, Telegram, etc.
	Distractions  []string `json:"distractions"`  // YouTube, social, etc.
}

type ProductivityScore struct {
	Overall     int    `json:"overall"`     // 0-100
	Focus       int    `json:"focus"`       // на основе deep work
	Efficiency  int    `json:"efficiency"`  // engagement rate
	Balance     int    `json:"balance"`     // work/life domains
	Explanation string `json:"explanation"` // краткое объяснение счета
}

type DomainAnalysis struct {
	FocusLevel      string           `json:"focus_level"`
	FocusInsight    string           `json:"focus_insight"`
	Recommendations []string         `json:"recommendations"`
	WorkPattern     string           `json:"work_pattern"`
	Analysis        DetailedAnalysis `json:"analysis"`
}

type AIAnalyticsRequest struct {
	DomainsCount   int          `json:"domains_count" binding:"required,min=1"`
	Domains        []string     `json:"domains" binding:"required,min=1"`
	DeepWork       DeepWorkData `json:"deep_work" binding:"required"`
	EngagementRate float64      `json:"engagement_rate" binding:"required,min=0,max=100"`
	TrackedHours   float64      `json:"tracked_hours" binding:"required,min=0"`
	UserID         string       `json:"user_id,omitempty"`
	Period         string       `json:"period,omitempty"`
	Timestamp      time.Time    `json:"timestamp,omitempty"`
}

// FocusLevelResponse представляет ответ с уровнем фокуса
type FocusLevelResponse struct {
	FocusLevel string    `json:"focus_level"`
	Insight    string    `json:"insight"`
	Method     string    `json:"method"` // "ai" или "fallback"
	Timestamp  time.Time `json:"timestamp"`
}

type AIAnalyticsResponse struct {
	Data    *DomainAnalysis `json:"data"`
	Success bool            `json:"success"`
	Meta    *AnalyticsMeta  `json:"meta,omitempty"`
}

type AnalyticsMeta struct {
	ProcessedAt     time.Time `json:"processed_at"`
	ProcessingTime  int64     `json:"processing_time_ms"`
	AIModel         string    `json:"ai_model,omitempty"`
	DataQuality     string    `json:"data_quality"` // "high", "medium", "low"
	ConfidenceScore float64   `json:"confidence_score,omitempty"`
}

type EnhancedDomainAnalysis struct {
	DomainAnalysis
	RequestData AIAnalyticsRequest `json:"request_data"`
	Meta        AnalyticsMeta      `json:"meta"`
}

type ProductivityInsight struct {
	Type       string  `json:"type"`     // "positive", "negative", "neutral"
	Category   string  `json:"category"` // "focus", "efficiency", "balance"
	Message    string  `json:"message"`
	Impact     string  `json:"impact"` // "high", "medium", "low"
	Confidence float64 `json:"confidence"`
	Actionable bool    `json:"actionable"`
}

// RecommendationDetail детальная рекомендация
type RecommendationDetail struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"` // "high", "medium", "low"
	Category    string   `json:"category"` // "focus", "tools", "habits"
	ActionItems []string `json:"action_items"`
	Expected    string   `json:"expected_outcome"`
}

// DomainCategorization улучшенная категоризация доменов
type DomainCategorization struct {
	Domain     string  `json:"domain"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// DetailedAnalysisV2 улучшенная версия детального анализа
type DetailedAnalysisV2 struct {
	DetailedAnalysis
	DomainCategorization    []DomainCategorization `json:"domain_categorization"`
	ProductivityInsights    []ProductivityInsight  `json:"productivity_insights"`
	DetailedRecommendations []RecommendationDetail `json:"detailed_recommendations"`
	TrendsAnalysis          *TrendsAnalysis        `json:"trends_analysis,omitempty"`
}

// TrendsAnalysis анализ трендов (если доступны исторические данные)
type TrendsAnalysis struct {
	FocusTrend        string  `json:"focus_trend"` // "improving", "declining", "stable"
	EfficiencyTrend   string  `json:"efficiency_trend"`
	BalanceTrend      string  `json:"balance_trend"`
	WeeklyComparison  float64 `json:"weekly_comparison"`
	MonthlyComparison float64 `json:"monthly_comparison"`
	Seasonality       string  `json:"seasonality,omitempty"`
}

// AIAnalyticsHealthCheck проверка состояния AI сервиса
type AIAnalyticsHealthCheck struct {
	Available     bool      `json:"available"`
	ResponseTime  int64     `json:"response_time_ms"`
	Model         string    `json:"model"`
	LastCheck     time.Time `json:"last_check"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	RequestsToday int       `json:"requests_today"`
	RequestsLimit int       `json:"requests_limit"`
}

// BatchAnalyticsRequest запрос для пакетного анализа
type BatchAnalyticsRequest struct {
	Requests []AIAnalyticsRequest `json:"requests" binding:"required,min=1,max=10"`
	Options  BatchOptions         `json:"options"`
}

// BatchOptions опции для пакетного анализа
type BatchOptions struct {
	Parallel        bool `json:"parallel"`
	FailOnError     bool `json:"fail_on_error"`
	IncludeMetadata bool `json:"include_metadata"`
	ComparisonMode  bool `json:"comparison_mode"`
}

// BatchAnalyticsResponse ответ пакетного анализа
type BatchAnalyticsResponse struct {
	Results   []EnhancedDomainAnalysis `json:"results"`
	Success   bool                     `json:"success"`
	Total     int                      `json:"total"`
	Processed int                      `json:"processed"`
	Failed    int                      `json:"failed"`
	Errors    []BatchError             `json:"errors,omitempty"`
	Meta      BatchMeta                `json:"meta"`
}

// BatchError ошибка в пакетном анализе
type BatchError struct {
	Index   int                `json:"index"`
	Error   string             `json:"error"`
	Request AIAnalyticsRequest `json:"request"`
}

// BatchMeta метаданные пакетного анализа
type BatchMeta struct {
	ProcessedAt  time.Time `json:"processed_at"`
	TotalTime    int64     `json:"total_time_ms"`
	AverageTime  int64     `json:"average_time_ms"`
	ParallelMode bool      `json:"parallel_mode"`
}

// AIAnalyticsConfig конфигурация для AI аналитики
type AIAnalyticsConfig struct {
	Enabled          bool    `json:"enabled"`
	Model            string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	TimeoutSeconds   int     `json:"timeout_seconds"`
	RetryAttempts    int     `json:"retry_attempts"`
	RateLimitPerHour int     `json:"rate_limit_per_hour"`
	FallbackMode     bool    `json:"fallback_mode"`
}
