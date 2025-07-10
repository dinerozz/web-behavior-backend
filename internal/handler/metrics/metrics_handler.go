package metrics

import (
	"context"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type MetricsHandler struct {
	service MetricsService
}

type MetricsService interface {
	GetTrackedTime(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
	GetTrackedTimeTotal(ctx context.Context, filter entity.TrackedTimeFilter) (*entity.TrackedTimeMetric, error)
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
