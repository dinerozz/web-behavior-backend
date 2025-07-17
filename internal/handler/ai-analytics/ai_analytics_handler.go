package ai_analytics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/ai_analytics"
	"github.com/gin-gonic/gin"
)

type AIAnalyticsHandler struct {
	aiService *ai_analytics.AIAnalyticsService
}

type AIAnalyticsService interface {
	AnalyzeDomainUsage(ctx context.Context, domainsCount int, domains []string, deepWorkData entity.DeepWorkData, engagementRate float64, trackedHours float64) (*entity.DomainAnalysis, error)
	DetermineFocusLevelFallback(domainsCount int) string
}

func NewAIAnalyticsHandler(aiService *ai_analytics.AIAnalyticsService) *AIAnalyticsHandler {
	return &AIAnalyticsHandler{aiService: aiService}
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

	if req.DomainsCount <= 0 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "domains_count must be greater than 0",
			Success: false,
		})
		return
	}

	if len(req.Domains) == 0 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "domains list cannot be empty",
			Success: false,
		})
		return
	}

	if req.EngagementRate < 0 || req.EngagementRate > 100 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "engagement_rate must be between 0 and 100",
			Success: false,
		})
		return
	}

	if req.TrackedHours < 0 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "tracked_hours must be non-negative",
			Success: false,
		})
		return
	}

	if req.DeepWork.SessionsCount == 0 && req.EngagementRate == 0 {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "insufficient data for analysis - need either deep work sessions or engagement activity",
			Success: false,
		})
		return
	}

	analysis, err := h.aiService.AnalyzeDomainUsage(
		c.Request.Context(),
		req.DomainsCount,
		req.Domains,
		req.DeepWork,
		req.EngagementRate,
		req.TrackedHours,
	)

	if err != nil {
		// Fallback на простую логику если AI недоступен
		analysis = &entity.DomainAnalysis{
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

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    analysis,
		Success: true,
	})
}

// GetFocusLevel godoc
// @Summary      Get focus level without AI
// @Description  Get basic focus level assessment based on domain count (fallback method)
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

	focusLevel := h.aiService.DetermineFocusLevelFallback(domainsCount)
	insight := h.generateFallbackInsight(domainsCount)

	response := entity.FocusLevelResponse{
		FocusLevel: focusLevel,
		Insight:    insight,
		Method:     "fallback",
		Timestamp:  time.Now(),
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    response,
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

func (h *AIAnalyticsHandler) RegisterRoutes(router *gin.RouterGroup) {
	analytics := router.Group("/ai-analytics")
	{
		analytics.POST("/domain-usage", h.AnalyzeDomainUsage)
		analytics.GET("/focus-level", h.GetFocusLevel)
	}
}
