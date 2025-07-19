package metrics

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/service/redis"
	"net/http"
	"strconv"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	service      MetricsService
	redisService redis.ServiceInterface
}

type MetricsService interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error)
	GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error)
	GetDeepWorkSessions(ctx context.Context, filter entity.DeepWorkSessionsFilter) (*entity.DeepWorkSessionsResponse, error)
	//PrepareAIAnalyticsData(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.AIAnalyticsRequest, error)
}

func NewMetricsHandler(service MetricsService, redisService redis.ServiceInterface) *MetricsHandler {
	return &MetricsHandler{service: service, redisService: redisService}
}

func (h *MetricsHandler) GetTrackedTime(c *gin.Context) {
	var filter entity.TrackedTimeFilter

	filter.UserID = c.Query("user_id")
	if filter.UserID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "user_id is required",
			Success: false,
		})
		return
	}

	startTimeStr := c.Query("start_time")
	if startTimeStr == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "start_time is required (RFC3339 format)",
			Success: false,
		})
		return
	}

	endTimeStr := c.Query("end_time")
	if endTimeStr == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "end_time is required (RFC3339 format)",
			Success: false,
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid start_time format, use RFC3339",
			Success: false,
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid end_time format, use RFC3339",
			Success: false,
		})
		return
	}

	filter.StartTime = startTime
	filter.EndTime = endTime

	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	metric, err := h.service.GetTrackedTime(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, entity.TrackedTimeResponse{
		Data:    metric,
		Success: true,
	})
}

func (h *MetricsHandler) GetTrackedTimeTotal(c *gin.Context) {
	var filter entity.TrackedTimeFilter

	filter.UserID = c.Query("user_id")
	if filter.UserID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "user_id is required",
			Success: false,
		})
		return
	}

	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	metric, err := h.service.GetTrackedTimeTotal(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, entity.TrackedTimeResponse{
		Data:    metric,
		Success: true,
	})
}

func (h *MetricsHandler) generateEngagedTimeCacheKey(filter entity.EngagedTimeFilter) string {
	params := fmt.Sprintf("user_id:%s|start_time:%s|end_time:%s|session_id:%v",
		filter.UserID,
		filter.StartTime.Format(time.RFC3339),
		filter.EndTime.Format(time.RFC3339),
		filter.SessionID,
	)

	hash := md5.Sum([]byte(params))
	return fmt.Sprintf("metrics:engaged_time:%x", hash)
}

func (h *MetricsHandler) GetEngagedTime(c *gin.Context) {
	var filter entity.EngagedTimeFilter

	filter.UserID = c.Query("user_id")
	if filter.UserID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "user_id is required",
			Success: false,
		})
		return
	}

	startTimeStr := c.Query("start_time")
	if startTimeStr == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "start_time is required (RFC3339 format)",
			Success: false,
		})
		return
	}

	endTimeStr := c.Query("end_time")
	if endTimeStr == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "end_time is required (RFC3339 format)",
			Success: false,
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid start_time format, use RFC3339",
			Success: false,
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid end_time format, use RFC3339",
			Success: false,
		})
		return
	}

	filter.StartTime = startTime
	filter.EndTime = endTime

	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	ctx := c.Request.Context()
	cacheKey := h.generateEngagedTimeCacheKey(filter)

	// Попытка получить данные из кэша
	var cachedMetric entity.EngagedTimeMetric
	err = h.redisService.Get(ctx, cacheKey, &cachedMetric)
	if err == nil {
		// Данные найдены в кэше
		c.Header("X-Cache", "HIT")
		c.Header("X-Cache-Key", cacheKey) // Для отладки
		c.JSON(http.StatusOK, entity.EngagedTimeResponse{
			Data:    &cachedMetric,
			Success: true,
		})
		return
	}

	// Данные не найдены в кэше, выполняем запрос к базе данных
	c.Header("X-Cache", "MISS")
	c.Header("X-Cache-Key", cacheKey) // Для отладки

	metric, err := h.service.GetEngagedTime(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	// Кэшируем результат на 1 час
	cacheErr := h.redisService.Set(ctx, cacheKey, metric, time.Hour)
	if cacheErr != nil {
		// Логируем ошибку кэширования, но не прерываем выполнение
		fmt.Printf("Failed to cache engaged time result: %v\n", cacheErr)
	}

	c.JSON(http.StatusOK, entity.EngagedTimeResponse{
		Data:    metric,
		Success: true,
	})
}

//// @Summary      Prepare data for AI analytics
//// @Description  Get prepared data for AI analytics based on engaged time metrics
//// @Tags         /api/v1/admin/metrics
//// @Accept       json
//// @Produce      json
//// @Param        user_id      query     string  true   "User ID"
//// @Param        start_time   query     string  true   "Start time (RFC3339 format)"
//// @Param        end_time     query     string  true   "End time (RFC3339 format)"
//// @Param        session_id   query     string  false  "Specific session ID"
//// @Success      200          {object}  wrapper.ResponseWrapper{data=entity.AIAnalyticsRequest}
//// @Failure      400          {object}  wrapper.ErrorWrapper
//// @Failure      500          {object}  wrapper.ErrorWrapper
//// @Router       /metrics/ai-analytics-data [get]
//func (h *MetricsHandler) PrepareAIAnalyticsData(c *gin.Context) {
//	var filter entity.EngagedTimeFilter
//
//	filter.UserID = c.Query("user_id")
//	if filter.UserID == "" {
//		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
//			Message: "user_id is required",
//			Success: false,
//		})
//		return
//	}
//
//	startTimeStr := c.Query("start_time")
//	if startTimeStr == "" {
//		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
//			Message: "start_time is required (RFC3339 format)",
//			Success: false,
//		})
//		return
//	}
//
//	endTimeStr := c.Query("end_time")
//	if endTimeStr == "" {
//		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
//			Message: "end_time is required (RFC3339 format)",
//			Success: false,
//		})
//		return
//	}
//
//	startTime, err := time.Parse(time.RFC3339, startTimeStr)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
//			Message: "Invalid start_time format, use RFC3339",
//			Success: false,
//		})
//		return
//	}
//
//	endTime, err := time.Parse(time.RFC3339, endTimeStr)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
//			Message: "Invalid end_time format, use RFC3339",
//			Success: false,
//		})
//		return
//	}
//
//	filter.StartTime = startTime
//	filter.EndTime = endTime
//
//	if sessionID := c.Query("session_id"); sessionID != "" {
//		filter.SessionID = &sessionID
//	}
//
//	analyticsData, err := h.service.PrepareAIAnalyticsData(c.Request.Context(), filter)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
//			Message: err.Error(),
//			Success: false,
//		})
//		return
//	}
//
//	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
//		Data:    analyticsData,
//		Success: true,
//	})
//}

// GetTopDomains остается без изменений
func (h *MetricsHandler) GetTopDomains(c *gin.Context) {
	var filter entity.TopDomainsFilter

	filter.UserID = c.Query("user_id")
	if filter.UserID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "user_id is required",
			Success: false,
		})
		return
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "limit must be a positive integer",
				Success: false,
			})
			return
		}
		filter.Limit = limit
	}

	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	result, err := h.service.GetTopDomains(c.Request.Context(), filter)
	if err != nil {
		fmt.Println("Failed to get top domains", "error", err)
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: "Failed to retrieve top domains",
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    result,
		Success: true,
	})
}

// GetDeepWorkSessions остается без изменений
func (h *MetricsHandler) GetDeepWorkSessions(c *gin.Context) {
	userID := c.Query("user_id")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	sessionID := c.Query("session_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "user_id is required",
		})
		return
	}

	if startTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "start_time is required",
		})
		return
	}

	if endTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "end_time is required",
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid start_time format, use RFC3339 (e.g., 2025-07-10T08:00:00Z)",
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid end_time format, use RFC3339 (e.g., 2025-07-11T19:59:59Z)",
		})
		return
	}

	if endTime.Before(startTime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "end_time must be after start_time",
		})
		return
	}

	if endTime.Sub(startTime) > 30*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Time range cannot exceed 30 days",
		})
		return
	}

	filter := entity.DeepWorkSessionsFilter{
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	if sessionID != "" {
		filter.SessionID = &sessionID
	}

	result, err := h.service.GetDeepWorkSessions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get deep work sessions",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// RegisterRoutes регистрирует маршруты для метрик
func (h *MetricsHandler) RegisterRoutes(router *gin.RouterGroup) {
	metrics := router.Group("/metrics")
	{
		metrics.GET("/tracked-time", h.GetTrackedTime)
		metrics.GET("/tracked-time-total", h.GetTrackedTimeTotal)
		metrics.GET("/engaged-time", h.GetEngagedTime)
		//metrics.GET("/ai-analytics-data", h.PrepareAIAnalyticsData) // Новый эндпоинт
		metrics.GET("/top-domains", h.GetTopDomains)
		metrics.GET("/deep-work-sessions", h.GetDeepWorkSessions)
	}
}
