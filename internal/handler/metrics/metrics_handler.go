package metrics

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

type MetricsHandler struct {
	service MetricsService
}

type MetricsService interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetEngagedTime(ctx context.Context, filter entity.EngagedTimeFilter) (*entity.EngagedTimeMetric, error)
	GetTopDomains(ctx context.Context, filter entity.TopDomainsFilter) (*entity.TopDomainsResponse, error)
}

func NewMetricsHandler(service MetricsService) *MetricsHandler {
	return &MetricsHandler{service: service}
}

// GetTrackedTime godoc
// @Summary      Get tracked time metric
// @Description  Calculate tracked time (sum of session durations) for a user
// @Tags         /api/v1/admin/metrics
// @Accept       json
// @Produce      json
// @Param        user_id      query     string  true   "User ID"
// @Param        start_time   query     string  true   "Start time (RFC3339 format)"
// @Param        end_time     query     string  true   "End time (RFC3339 format)"
// @Param        session_id   query     string  false  "Specific session ID"
// @Success      200          {object}  entity.TrackedTimeResponse
// @Failure      400          {object}  wrapper.ErrorWrapper
// @Failure      500          {object}  wrapper.ErrorWrapper
// @Router       /metrics/tracked-time [get]
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

// GetTrackedTimeTotal godoc
// @Summary      Get total tracked time metric
// @Description  Calculate total time from first to last event for a user
// @Tags         /api/v1/admin/metrics
// @Accept       json
// @Produce      json
// @Param        user_id      query     string  true   "User ID"
// @Param        session_id   query     string  false  "Specific session ID"
// @Success      200          {object}  entity.TrackedTimeResponse
// @Failure      400          {object}  wrapper.ErrorWrapper
// @Failure      500          {object}  wrapper.ErrorWrapper
// @Router       /metrics/tracked-time-total [get]
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

// GetEngagedTime godoc
// @Summary      Get engaged time metric with tracked time data
// @Description  Calculate engaged time (active minutes) and tracked time with engagement rate for a user
// @Tags         /api/v1/admin/metrics
// @Accept       json
// @Produce      json
// @Param        user_id      query     string  true   "User ID"
// @Param        start_time   query     string  true   "Start time (RFC3339 format)"
// @Param        end_time     query     string  true   "End time (RFC3339 format)"
// @Param        session_id   query     string  false  "Specific session ID"
// @Success      200          {object}  entity.EngagedTimeResponse
// @Failure      400          {object}  wrapper.ErrorWrapper
// @Failure      500          {object}  wrapper.ErrorWrapper
// @Router       /metrics/engaged-time [get]
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

	metric, err := h.service.GetEngagedTime(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, entity.EngagedTimeResponse{
		Data:    metric,
		Success: true,
	})
}

// @Summary Get top domains for user (all time)
// @Description Retrieve top visited domains for a specific user for all time
// @Tags metrics
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param limit query int false "Number of top domains to return (max 50, default 10)"
// @Param session_id query string false "Optional session ID filter"
// @Success 200 {object} wrapper.SuccessWrapper
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /api/v1/admin/metrics/top-domains [get]
func (h *MetricsHandler) GetTopDomains(c *gin.Context) {
	var filter entity.TopDomainsFilter

	// Парсинг обязательных параметров
	filter.UserID = c.Query("user_id")
	if filter.UserID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "user_id is required",
			Success: false,
		})
		return
	}

	// Парсинг опциональных параметров
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

	// Получение данных
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
