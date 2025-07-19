package ai_analytics

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/service/redis"
	"net/http"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/ai_analytics"
	"github.com/gin-gonic/gin"
)

type AIAnalyticsHandler struct {
	aiService    *ai_analytics.AIAnalyticsService
	redisService redis.ServiceInterface
}

type AIAnalyticsService interface {
	AnalyzeDomainUsage(ctx context.Context, domainsCount int, domains []string, deepWorkData entity.DeepWorkData, engagementRate float64, trackedHours float64) (*entity.DomainAnalysis, error)
	DetermineFocusLevelFallback(domainsCount int) string
}

func NewAIAnalyticsHandler(aiService *ai_analytics.AIAnalyticsService, redisService redis.ServiceInterface) *AIAnalyticsHandler {
	return &AIAnalyticsHandler{aiService: aiService, redisService: redisService}
}

func (h *AIAnalyticsHandler) generateCacheKey(req entity.AIAnalyticsRequest) string {
	params := fmt.Sprintf("domains_count:%d|domains:%v|deep_work:%+v|engagement_rate:%.2f|tracked_hours:%.2f",
		req.DomainsCount,
		req.Domains,
		req.DeepWork,
		req.EngagementRate,
		req.TrackedHours,
	)

	hash := md5.Sum([]byte(params))
	return fmt.Sprintf("ai_analytics:domain_usage:%x", hash)
}

// AnalyzeDomainUsage godoc
// @Summary      Analyze domain usage with AI
// @Description  Get AI-powered analysis of user's domain usage patterns, productivity insights, and recommendations
// @Tags         /api/v1/admin/ai-analytics
// @Accept       json
// @Produce      json
// @Param        request  body      entity.AIAnalyticsRequest  true  "Analytics request data"
// @Success      200      {object}  wrapper.ResponseWrapper{data=entity.DomainAnalysis}
// @Failure      400      {object}  wrapper.ErrorWrapper
// @Failure      500      {object}  wrapper.ErrorWrapper
// @Router       /ai-analytics/domain-usage [post]
func (h *AIAnalyticsHandler) AnalyzeDomainUsage(c *gin.Context) {
	var req entity.AIAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid request body: " + err.Error(),
			Success: false,
		})
		return
	}

	if err := h.validateRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := h.generateCacheKey(req)

	var cachedAnalysis entity.DomainAnalysis
	err := h.redisService.Get(ctx, cacheKey, &cachedAnalysis)
	if err == nil {
		c.Header("X-Cache", "HIT")
		c.JSON(http.StatusOK, wrapper.ResponseWrapper{
			Data:    &cachedAnalysis,
			Success: true,
		})
		return
	}

	c.Header("X-Cache", "MISS")

	analysis, err := h.aiService.AnalyzeDomainUsage(
		ctx,
		req.DomainsCount,
		req.Domains,
		req.DeepWork,
		req.EngagementRate,
		req.TrackedHours,
	)

	if err != nil {
		analysis = h.generateFallbackAnalysis(req)
	}

	cacheErr := h.redisService.Set(ctx, cacheKey, analysis, time.Hour)
	if cacheErr != nil {
		fmt.Printf("Failed to cache AI analysis result: %v\n", cacheErr)
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    analysis,
		Success: true,
	})
}

func (h *AIAnalyticsHandler) validateRequest(req entity.AIAnalyticsRequest) error {
	if req.DomainsCount <= 0 {
		return fmt.Errorf("domains_count must be greater than 0")
	}

	if len(req.Domains) == 0 {
		return fmt.Errorf("domains list cannot be empty")
	}

	if req.EngagementRate < 0 || req.EngagementRate > 100 {
		return fmt.Errorf("engagement_rate must be between 0 and 100")
	}

	if req.TrackedHours < 0 {
		return fmt.Errorf("tracked_hours must be non-negative")
	}

	if req.DeepWork.SessionsCount == 0 && req.EngagementRate == 0 {
		return fmt.Errorf("insufficient data for analysis - need either deep work sessions or engagement activity")
	}

	return nil
}

func (h *AIAnalyticsHandler) generateFocusLevelCacheKey(domainsCount int) string {
	return fmt.Sprintf("ai_analytics:focus_level:%d", domainsCount)
}

// GetFocusLevel godoc
// @Summary      Get focus level without AI (with Redis caching)
// @Description  Get basic focus level assessment based on domain count (fallback method). Results are cached in Redis for 6 hours.
// @Tags         /api/v1/admin/ai-analytics
// @Accept       json
// @Produce      json
// @Param        domains_count  query     int  true  "Number of unique domains"
// @Success      200            {object}  wrapper.ResponseWrapper{data=entity.FocusLevelResponse}
// @Failure      400            {object}  wrapper.ErrorWrapper
// @Router       /ai-analytics/focus-level [get]
func (h *AIAnalyticsHandler) GetFocusLevel(c *gin.Context) {
	domainsCountStr := c.Query("domains_count")
	if domainsCountStr == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "domains_count parameter is required",
			Success: false,
		})
		return
	}

	domainsCount := 0
	if _, err := fmt.Sscanf(domainsCountStr, "%d", &domainsCount); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "domains_count must be a valid integer",
			Success: false,
		})
		return
	}

	if domainsCount < 0 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "domains_count must be non-negative",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := h.generateFocusLevelCacheKey(domainsCount)

	var cachedResponse entity.FocusLevelResponse
	err := h.redisService.Get(ctx, cacheKey, &cachedResponse)
	if err == nil {
		c.Header("X-Cache", "HIT")
		c.Header("X-Cache-Key", cacheKey)
		c.JSON(http.StatusOK, wrapper.ResponseWrapper{
			Data:    &cachedResponse,
			Success: true,
		})
		return
	}

	c.Header("X-Cache", "MISS")
	c.Header("X-Cache-Key", cacheKey)
	focusLevel, _ := h.aiService.AnalyzeFocusWithAI(ctx, domainsCount)

	response := entity.FocusLevelResponse{
		FocusLevel: focusLevel.FocusLevel,
		Insight:    focusLevel.Insight,
		Method:     focusLevel.Method,
		Timestamp:  focusLevel.Timestamp,
	}

	cacheErr := h.redisService.Set(ctx, cacheKey, &response, 6*time.Hour)
	if cacheErr != nil {
		fmt.Printf("Failed to cache focus level result: %v\n", cacheErr)
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    &response,
		Success: true,
	})
}

func (h *AIAnalyticsHandler) generateFallbackInsight(domainsCount int) string {
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

func (h *AIAnalyticsHandler) generateFallbackAnalysis(req entity.AIAnalyticsRequest) *entity.DomainAnalysis {
	return &entity.DomainAnalysis{
		FocusLevel:      h.aiService.DetermineFocusLevelFallback(req.DomainsCount),
		FocusInsight:    h.generateFallbackInsight(req.DomainsCount),
		WorkPattern:     "unknown",
		Recommendations: []string{"AI анализ временно недоступен"},
		Analysis: entity.DetailedAnalysis{
			DomainBreakdown: entity.DomainBreakdown{
				WorkTools:     []string{},
				Development:   []string{},
				Research:      []string{},
				Communication: []string{},
				Distractions:  []string{},
			},
			ProductivityScore: entity.ProductivityScore{
				Overall:     0,
				Focus:       0,
				Efficiency:  0,
				Balance:     0,
				Explanation: "AI анализ недоступен, используется базовая оценка",
			},
			BehaviorInsights: []string{"Базовая оценка без AI анализа"},
			KeyFindings:      []string{"Детальный анализ требует AI сервиса"},
		},
	}
}

func (h *AIAnalyticsHandler) RegisterRoutes(router *gin.RouterGroup) {
	analytics := router.Group("/ai-analytics")
	{
		analytics.POST("/domain-usage", h.AnalyzeDomainUsage)
		analytics.GET("/focus-level", h.GetFocusLevel)
	}
}
