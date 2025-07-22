package organization

import (
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/organization"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"net/http"
)

type OrganizationHandler struct {
	srv *organization.OrganizationService
}

func NewOrganizationHandler(srv *organization.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{srv: srv}
}

// CreateOrganization godoc
// @Summary Create new organization
// @Description Create a new organization with the authenticated user as admin
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param organization body request.CreateOrganization true "Organization object"
// @Success 201 {object} wrapper.ResponseWrapper{data=response.Organization}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations [post]
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	var orgRequest request.CreateOrganization
	if err := c.ShouldBindJSON(&orgRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	organization, err := h.srv.CreateOrganization(&orgRequest, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusCreated, wrapper.ResponseWrapper{Data: organization, Success: true})
}

// GetOrganization godoc
// @Summary Get organization by ID
// @Description Get organization details by ID (user must have access)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} wrapper.ResponseWrapper{data=response.Organization}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id} [get]
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	organization, err := h.srv.GetOrganizationByID(orgID, userUUID)
	if err != nil {
		if err.Error() == "access denied: user does not have access to this organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Access denied", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: organization, Success: true})
}

// GetOrganizationWithMembers godoc
// @Summary Get organization with members
// @Description Get organization details including member list (user must have access)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} wrapper.ResponseWrapper{data=response.OrganizationWithMembers}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id}/members [get]
func (h *OrganizationHandler) GetOrganizationWithMembers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	organization, err := h.srv.GetOrganizationWithMembers(orgID, userUUID)
	if err != nil {
		if err.Error() == "access denied: user does not have access to this organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Access denied", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: organization, Success: true})
}

// UpdateOrganization godoc
// @Summary Update organization
// @Description Update organization details (admin only)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param organization body request.UpdateOrganization true "Organization update object"
// @Success 200 {object} wrapper.ResponseWrapper{data=response.Organization}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id} [put]
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	var updateRequest request.UpdateOrganization
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	organization, err := h.srv.UpdateOrganization(orgID, &updateRequest, userUUID)
	if err != nil {
		if err.Error() == "only admins can update organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Admin access required", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: organization, Success: true})
}

// DeleteOrganization godoc
// @Summary Delete organization
// @Description Delete organization (admin only)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} wrapper.SuccessWrapper
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id} [delete]
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	err = h.srv.DeleteOrganization(orgID, userUUID)
	if err != nil {
		if err.Error() == "only admins can delete organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Admin access required", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.SuccessWrapper{Message: "Organization deleted successfully", Success: true})
}

// GetUserOrganizations godoc
// @Summary Get user's organizations
// @Description Get list of organizations user has access to
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=response.UserOrganizations}
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/my [get]
func (h *OrganizationHandler) GetUserOrganizations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	organizations, err := h.srv.GetUserOrganizations(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: organizations, Success: true})
}

// AddUserToOrganization godoc
// @Summary Add user to organization
// @Description Add user to organization (admin only)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param user body request.AddUserToOrganization true "User to add"
// @Success 200 {object} wrapper.SuccessWrapper
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id}/users [post]
func (h *OrganizationHandler) AddUserToOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	var addUserRequest request.AddUserToOrganization
	if err := c.ShouldBindJSON(&addUserRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	err = h.srv.AddUserToOrganization(orgID, &addUserRequest, userUUID)
	if err != nil {
		if err.Error() == "only admins can add users to organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Admin access required", Success: false})
			return
		}
		if err.Error() == "user is already in this organization" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "User is already in this organization", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.SuccessWrapper{Message: "User added to organization successfully", Success: true})
}

// RemoveUserFromOrganization godoc
// @Summary Remove user from organization
// @Description Remove user from organization (admin only)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param user_id path string true "User ID to remove"
// @Success 200 {object} wrapper.SuccessWrapper
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id}/users/{user_id} [delete]
func (h *OrganizationHandler) RemoveUserFromOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	userToRemoveIDStr := c.Param("user_id")
	userToRemoveID, err := uuid.FromString(userToRemoveIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid user ID", Success: false})
		return
	}

	err = h.srv.RemoveUserFromOrganization(orgID, userToRemoveID, userUUID)
	if err != nil {
		if err.Error() == "only admins can remove users from organization" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Admin access required", Success: false})
			return
		}
		if err.Error() == "cannot remove the only admin from organization" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Cannot remove the only admin from organization", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.SuccessWrapper{Message: "User removed from organization successfully", Success: true})
}

// UpdateUserRole godoc
// @Summary Update user role in organization
// @Description Update user role in organization (admin only)
// @Tags /api/v1/organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param user_id path string true "User ID"
// @Param role query string true "New role" Enums(admin, member, viewer)
// @Success 200 {object} wrapper.SuccessWrapper
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 403 {object} wrapper.ErrorWrapper
// @Failure 404 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /organizations/{id}/users/{user_id}/role [put]
func (h *OrganizationHandler) UpdateUserRole(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
		return
	}

	userUUID, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	orgIDStr := c.Param("id")
	orgID, err := uuid.FromString(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid organization ID", Success: false})
		return
	}

	userToUpdateIDStr := c.Param("user_id")
	userToUpdateID, err := uuid.FromString(userToUpdateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid user ID", Success: false})
		return
	}

	role := c.Query("role")
	if role == "" {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Role parameter is required", Success: false})
		return
	}

	err = h.srv.UpdateUserRole(orgID, userToUpdateID, role, userUUID)
	if err != nil {
		if err.Error() == "only admins can update user roles" {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Admin access required", Success: false})
			return
		}
		if err.Error() == "cannot demote the only admin from organization" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Cannot demote the only admin from organization", Success: false})
			return
		}
		if err.Error() == "invalid role: must be 'admin', 'member', or 'viewer'" {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid role: must be 'admin', 'member', or 'viewer'", Success: false})
			return
		}
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.SuccessWrapper{Message: "User role updated successfully", Success: true})
}
