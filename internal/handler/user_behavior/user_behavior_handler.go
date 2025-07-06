// internal/handler/user_behavior_handler.go
package handler

import (
	service "github.com/dinerozz/web-behavior-backend/internal/service/user_behavior"
	"net/http"
	"strconv"
	"time"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserBehaviorHandler struct {
	service service.UserBehaviorService
}

func NewUserBehaviorHandler(service service.UserBehaviorService) *UserBehaviorHandler {
	return &UserBehaviorHandler{
		service: service,
	}
}

// CreateBehavior godoc
// @Summary      Create user behavior event
// @Description  Create a single user behavior event
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        behavior  body      entity.CreateUserBehaviorRequest  true  "Behavior data"
// @Success      201       {object}  wrapper.SuccessResponse{data=entity.UserBehavior}
// @Failure      400       {object}  wrapper.ErrorWrapper
// @Failure      500       {object}  wrapper.ErrorWrapper
// @Router       /behaviors [post]
func (h *UserBehaviorHandler) CreateBehavior(c *gin.Context) {
	var req entity.CreateUserBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	behavior, err := h.service.CreateBehavior(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, wrapper.ResponseWrapper{
		Data: behavior,
	})
}

// BatchCreateBehaviors godoc
// @Summary      Batch create user behavior events
// @Description  Create multiple user behavior events in one request
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        behaviors  body      entity.BatchCreateUserBehaviorRequest  true  "Behaviors data"
// @Success      201        {object}  wrapper.SuccessResponse{data=string}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /behaviors/batch [post]
func (h *UserBehaviorHandler) BatchCreateBehaviors(c *gin.Context) {
	var req entity.BatchCreateUserBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	err := h.service.BatchCreateBehaviors(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, wrapper.ResponseWrapper{
		Data: "Successfully created " + strconv.Itoa(len(req.Events)) + " behavior events",
	})
}

// GetBehaviorByID godoc
// @Summary      Get behavior by ID
// @Description  Get a specific user behavior event by ID
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Behavior ID"
// @Success      200  {object}  wrapper.SuccessResponse{data=entity.UserBehavior}
// @Failure      400  {object}  wrapper.ErrorWrapper
// @Failure      404  {object}  wrapper.ErrorWrapper
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /behaviors/{id} [get]
func (h *UserBehaviorHandler) GetBehaviorByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
		})
		return
	}

	behavior, err := h.service.GetBehaviorByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "behavior not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Behavior not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: behavior,
	})
}

// GetBehaviors godoc
// @Summary      Get user behaviors
// @Description  Get user behavior events with optional filters
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        userId     query     string  false  "User ID"
// @Param        sessionId  query     string  false  "Session ID"
// @Param        eventType  query     string  false  "Event type"
// @Param        url        query     string  false  "URL (partial match)"
// @Param        startTime  query     string  false  "Start time (RFC3339 format)"
// @Param        endTime    query     string  false  "End time (RFC3339 format)"
// @Param        limit      query     int     false  "Limit (default: 100, max: 1000)"
// @Param        offset     query     int     false  "Offset (default: 0)"
// @Success      200        {object}  wrapper.ResponseWrapper{data=[]entity.UserBehavior}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /behaviors [get]
func (h *UserBehaviorHandler) GetBehaviors(c *gin.Context) {
	var filter entity.UserBehaviorFilter

	// Парсинг query параметров
	if userID := c.Query("userId"); userID != "" {
		filter.UserID = &userID
	}

	if sessionID := c.Query("sessionId"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	if eventType := c.Query("eventType"); eventType != "" {
		filter.EventType = &eventType
	}

	if url := c.Query("url"); url != "" {
		filter.URL = &url
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid startTime format, use RFC3339",
			})
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid endTime format, use RFC3339",
			})
			return
		}
		filter.EndTime = &endTime
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid limit value",
			})
			return
		}
		filter.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid offset value",
			})
			return
		}
		filter.Offset = offset
	}

	behaviors, err := h.service.GetBehaviors(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: behaviors,
	})
}

// GetStats godoc
// @Summary      Get behavior statistics
// @Description  Get statistics about user behaviors
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        userId     query     string  false  "User ID"
// @Param        sessionId  query     string  false  "Session ID"
// @Param        eventType  query     string  false  "Event type"
// @Param        url        query     string  false  "URL (partial match)"
// @Param        startTime  query     string  false  "Start time (RFC3339 format)"
// @Param        endTime    query     string  false  "End time (RFC3339 format)"
// @Success      200        {object}  wrapper.ResponseWrapper{data=entity.UserBehaviorStats}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /behaviors/stats [get]
func (h *UserBehaviorHandler) GetStats(c *gin.Context) {
	var filter entity.UserBehaviorFilter

	// Парсинг тех же параметров что и для GetBehaviors
	if userID := c.Query("userId"); userID != "" {
		filter.UserID = &userID
	}

	if sessionID := c.Query("sessionId"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	if eventType := c.Query("eventType"); eventType != "" {
		filter.EventType = &eventType
	}

	if url := c.Query("url"); url != "" {
		filter.URL = &url
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid startTime format, use RFC3339",
			})
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid endTime format, use RFC3339",
			})
			return
		}
		filter.EndTime = &endTime
	}

	stats, err := h.service.GetStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: stats,
	})
}

// GetSessionSummary godoc
// @Summary      Get session summary
// @Description  Get summary information about a specific session
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        sessionId  path      string  true  "Session ID"
// @Success      200        {object}  wrapper.ResponseWrapper{data=entity.SessionSummary}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      404        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /behaviors/sessions/{sessionId} [get]
func (h *UserBehaviorHandler) GetSessionSummary(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Session ID is required",
		})
		return
	}

	summary, err := h.service.GetSessionSummary(c.Request.Context(), sessionID)
	if err != nil {
		if err.Error() == "session not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: summary,
	})
}

// GetUserSessions godoc
// @Summary      Get user sessions
// @Description  Get all sessions for a specific user
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        userId  path      string  true   "User ID"
// @Param        limit   query     int     false  "Limit (default: 50, max: 200)"
// @Param        offset  query     int     false  "Offset (default: 0)"
// @Success      200     {object}  wrapper.ResponseWrapper{data=[]entity.SessionSummary}
// @Failure      400     {object}  wrapper.ErrorWrapper
// @Failure      500     {object}  wrapper.ErrorWrapper
// @Router       /behaviors/users/{userId}/sessions [get]
func (h *UserBehaviorHandler) GetUserSessions(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "User ID is required",
		})
		return
	}

	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid limit value",
			})
			return
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Invalid offset value",
			})
			return
		}
	}

	sessions, err := h.service.GetUserSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: sessions,
	})
}

// DeleteBehavior godoc
// @Summary      Delete behavior
// @Description  Delete a specific user behavior event
// @Tags         user-behaviors
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Behavior ID"
// @Success      200  {object}  wrapper.ResponseWrapper{data=string}
// @Failure      400  {object}  wrapper.ErrorWrapper
// @Failure      404  {object}  wrapper.ErrorWrapper
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /behaviors/{id} [delete]
func (h *UserBehaviorHandler) DeleteBehavior(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
		})
		return
	}

	err = h.service.DeleteBehavior(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "failed to delete behavior: sql: no rows in result set" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Behavior not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data: "Behavior deleted successfully",
	})
}

// RegisterRoutes регистрирует все маршруты для user behavior
func (h *UserBehaviorHandler) RegisterRoutes(router *gin.RouterGroup) {
	behaviors := router.Group("/behaviors")
	{
		behaviors.POST("", h.CreateBehavior)
		behaviors.POST("/batch", h.BatchCreateBehaviors)
		behaviors.GET("", h.GetBehaviors)
		behaviors.GET("/stats", h.GetStats)
		behaviors.GET("/:id", h.GetBehaviorByID)
		behaviors.DELETE("/:id", h.DeleteBehavior)

		// Session routes
		behaviors.GET("/sessions/:sessionId", h.GetSessionSummary)
		behaviors.GET("/users/:userId/sessions", h.GetUserSessions)
	}
}
