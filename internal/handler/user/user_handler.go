package user

import (
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/organization"
	"github.com/dinerozz/web-behavior-backend/internal/service/user"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

type UserHandler struct {
	srv    *user.UserService
	orgSrv *organization.OrganizationService
}

func NewUserHandler(srv *user.UserService, orgSrv *organization.OrganizationService) *UserHandler {
	return &UserHandler{
		srv:    srv,
		orgSrv: orgSrv,
	}
}

// CreateUserWithPassword godoc
// @Summary Create new user with password (Admin only)
// @Description Create a new user with password - only available for admin users
// @Tags /api/v1/admin/users
// @Accept json
// @Produce json
// @Param user body request.CreateUserWithPassword true "User object"
// @Success 201 {object} wrapper.ResponseWrapper{data=response.User}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 409 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /admin/users/register [post]
func (h *UserHandler) CreateUserWithPassword(c *gin.Context) {
	var userRequest request.CreateUserWithPassword
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	userExists := h.srv.CheckIfUserExistsByUsername(userRequest.Username)
	if userExists {
		c.JSON(http.StatusConflict, wrapper.ErrorWrapper{Message: "User with this username already exists", Success: false})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: "Failed to hash password", Success: false})
		return
	}

	userRequest.Password = string(hashedPassword)

	userResponse, err := h.srv.CreateUserWithPassword(&userRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusCreated, wrapper.ResponseWrapper{Data: userResponse, Success: true})
}

// AuthenticateUserWithPassword godoc
// @Summary Authenticate user with password
// @Description Authenticate an existing user with username and password
// @Tags /api/v1/admin/users
// @Accept json
// @Produce json
// @Param user body request.CreateUserWithPassword true "Login credentials"
// @Success 200 {object} wrapper.ResponseWrapper{data=string}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /admin/users/login [post]
func (h *UserHandler) AuthenticateUserWithPassword(c *gin.Context) {
	var loginRequest request.CreateUserWithPassword
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	userExists := h.srv.CheckIfUserExistsByUsername(loginRequest.Username)
	if !userExists {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "Invalid username or password", Success: false})
		return
	}

	existingUser, err := h.srv.GetUserByUsername(loginRequest.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: "Internal server error", Success: false})
		return
	}

	if existingUser.Password == nil {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User doesn't have a password", Success: false})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(*existingUser.Password), []byte(loginRequest.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "Invalid username or password", Success: false})
		return
	}

	token, err := utils.GenerateToken(existingUser.ID, existingUser.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: "Failed to generate token", Success: false})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", false, true)
	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: token, Success: true})
}

// GetUserById godoc
// @Summary Get user by ID
// @Description Get user by ID
// @Tags /api/v1/admin/users
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=response.User}
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /users/profile [get]
func (h *UserHandler) GetUserById(c *gin.Context) {
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

	user, err := h.srv.GetUserById(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: user, Success: true})
}

// GetUserWithOrganizations godoc
// @Summary Get user profile with organizations
// @Description Get user profile including their organizations
// @Tags /api/v1/admin/users
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.ResponseWrapper{data=map[string]interface{}}
// @Failure 401 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /users/profile/full [get]
func (h *UserHandler) GetUserWithOrganizations(c *gin.Context) {
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

	user, err := h.srv.GetUserById(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	userOrganizations, err := h.orgSrv.GetUserOrganizations(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	response := map[string]interface{}{
		"user":          user,
		"organizations": userOrganizations.Organizations,
	}

	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: response, Success: true})
}

// Logout godoc
// @Summary Logout user
// @Description Logout user by clearing authentication cookie
// @Tags /api/v1/admin/users
// @Accept json
// @Produce json
// @Success 200 {object} wrapper.SuccessWrapper{message=string}
// @Router /users/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, wrapper.SuccessWrapper{
		Message: "Successfully logged out",
		Success: true,
	})
}
