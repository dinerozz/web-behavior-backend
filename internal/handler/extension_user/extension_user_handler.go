// internal/handler/extension_user_handler.go
package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/extension_user"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type ExtensionUserHandler struct {
	service service.ExtensionUserService
}

func NewExtensionUserHandler(service service.ExtensionUserService) *ExtensionUserHandler {
	return &ExtensionUserHandler{
		service: service,
	}
}

// CreateExtensionUser godoc
// @Summary      Create extension user
// @Description  Create a new extension user with API key
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        user  body      entity.CreateExtensionUserRequest  true  "User data"
// @Success      201   {object}  wrapper.ResponseWrapper{data=entity.ExtensionUser}
// @Failure      400   {object}  wrapper.ErrorWrapper
// @Failure      500   {object}  wrapper.ErrorWrapper
// @Router       /extension/users/generate [post]
func (h *ExtensionUserHandler) CreateExtensionUser(c *gin.Context) {
	var req entity.CreateExtensionUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid request body: " + err.Error(),
			Success: false,
		})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "username already exists" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: err.Error(),
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusCreated, wrapper.ResponseWrapper{
		Data:    user,
		Success: true,
	})
}

// GetExtensionUserByID godoc
// @Summary      Get extension user by ID
// @Description  Get a specific extension user by their ID
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  wrapper.ResponseWrapper{data=entity.ExtensionUserPublic}
// @Failure      400  {object}  wrapper.ErrorWrapper
// @Failure      404  {object}  wrapper.ErrorWrapper
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /extension/users/{id} [get]
func (h *ExtensionUserHandler) GetExtensionUserByID(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.FromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
			Success: false,
		})
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Extension user not found",
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    user,
		Success: true,
	})
}

// GetExtensionUserByUsername godoc
// @Summary      Get extension user by username
// @Description  Get a specific extension user by their username
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        username   path      string  true  "Username"
// @Success      200        {object}  wrapper.ResponseWrapper{data=entity.ExtensionUserPublic}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      404        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /extension/username/{username} [get]
func (h *ExtensionUserHandler) GetExtensionUserByUsername(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Username is required",
			Success: false,
		})
		return
	}

	user, err := h.service.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Extension user not found",
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    user,
		Success: true,
	})
}

// GetAllExtensionUsers godoc
// @Summary      Get all extension users
// @Description  Get list of extension users with optional filters
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        username   query     string  false  "Filter by username"
// @Param        isActive   query     bool    false  "Filter by active status"
// @Param        limit      query     int     false  "Limit (default: 50, max: 200)"
// @Param        offset     query     int     false  "Offset (default: 0)"
// @Success      200        {object}  wrapper.ResponseWrapper{data=[]entity.ExtensionUserPublic}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /extension/users [get]
func (h *ExtensionUserHandler) GetAllExtensionUsers(c *gin.Context) {
	var filter entity.ExtensionUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid query parameters: " + err.Error(),
			Success: false,
		})
		return
	}

	users, err := h.service.GetAllUsers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    users,
		Success: true,
	})
}

// UpdateExtensionUser godoc
// @Summary      Update extension user
// @Description  Update extension user information
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        id    path      string                              true  "User ID"
// @Param        user  body      entity.UpdateExtensionUserRequest  true  "Update user data"
// @Success      200   {object}  wrapper.ResponseWrapper{data=entity.ExtensionUserPublic}
// @Failure      400   {object}  wrapper.ErrorWrapper
// @Failure      404   {object}  wrapper.ErrorWrapper
// @Failure      500   {object}  wrapper.ErrorWrapper
// @Router       /extension/users/{id} [put]
func (h *ExtensionUserHandler) UpdateExtensionUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.FromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
			Success: false,
		})
		return
	}

	var req entity.UpdateExtensionUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid request body: " + err.Error(),
			Success: false,
		})
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), userID, req)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Extension user not found",
				Success: false,
			})
			return
		}
		// todo check api key for unique
		if err.Error() == "username already exists" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: err.Error(),
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    user,
		Success: true,
	})
}

// RegenerateAPIKey godoc
// @Summary      Regenerate API key
// @Description  Regenerate API key for extension user
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  wrapper.ResponseWrapper{data=entity.RegenerateAPIKeyResponse}
// @Failure      400  {object}  wrapper.ErrorWrapper
// @Failure      404  {object}  wrapper.ErrorWrapper
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /extension/users/{id}/regenerate-key [post]
func (h *ExtensionUserHandler) RegenerateAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.FromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
			Success: false,
		})
		return
	}

	response, err := h.service.RegenerateAPIKey(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Extension user not found",
				Success: false,
			})
			return
		}
		if err.Error() == "user is inactive" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
				Message: "Cannot regenerate API key for inactive user",
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    response,
		Success: true,
	})
}

// DeleteExtensionUser godoc
// @Summary      Delete extension user
// @Description  Delete an extension user
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  wrapper.ResponseWrapper{data=string}
// @Failure      400  {object}  wrapper.ErrorWrapper
// @Failure      404  {object}  wrapper.ErrorWrapper
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /extension/users/{id} [delete]
func (h *ExtensionUserHandler) DeleteExtensionUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.FromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "Invalid UUID format",
			Success: false,
		})
		return
	}

	err = h.service.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, wrapper.ErrorWrapper{
				Message: "Extension user not found",
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    "Extension user deleted successfully",
		Success: true,
	})
}

// ValidateAPIKey godoc
// @Summary      Validate API key
// @Description  Validate extension user API key
// @Tags         /api/v1/inayla/extension
// @Accept       json
// @Produce      json
// @Param        X-API-Key  header    string  true  "API Key"
// @Success      200        {object}  wrapper.ResponseWrapper{data=entity.ExtensionUserPublic}
// @Failure      400        {object}  wrapper.ErrorWrapper
// @Failure      401        {object}  wrapper.ErrorWrapper
// @Failure      500        {object}  wrapper.ErrorWrapper
// @Router       /extension/users/auth [post]
func (h *ExtensionUserHandler) ValidateAPIKey(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{
			Message: "X-API-Key header is required",
			Success: false,
		})
		return
	}

	user, err := h.service.ValidateAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		if err.Error() == "invalid or inactive API key" || err.Error() == "API key is required" {
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
				Message: "Invalid API key",
				Success: false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	publicUser := &entity.ExtensionUserPublic{
		ID:         user.ID,
		Username:   user.Username,
		IsActive:   user.IsActive,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
		LastUsedAt: user.LastUsedAt,
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    publicUser,
		Success: true,
	})
}

// GetExtensionUserStats godoc
// @Summary      Get extension users statistics
// @Description  Get statistics about extension users
// @Tags         /api/v1/admin/extension
// @Accept       json
// @Produce      json
// @Success      200  {object}  wrapper.ResponseWrapper{data=entity.ExtensionUserStats}
// @Failure      500  {object}  wrapper.ErrorWrapper
// @Router       /extension/users/stats [get]
func (h *ExtensionUserHandler) GetExtensionUserStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{
		Data:    stats,
		Success: true,
	})
}
