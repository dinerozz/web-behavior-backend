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

// CreateOrAuthUserWithPassword godoc
// @Summary Create or authenticate user with password
// @Description Create a new user or authenticate an existing user with password
// @Tags users
// @Accept json
// @Produce json
// @Param user body request.CreateUserWithPassword true "User object"
// @Success 200 {object} wrapper.ResponseWrapper{data=response.User}
// @Failure 400 {object} wrapper.ErrorWrapper
// @Failure 500 {object} wrapper.ErrorWrapper
// @Router /users/auth [post]
func (h *UserHandler) CreateOrAuthUserWithPassword(c *gin.Context) {
	var userRequest request.CreateUserWithPassword
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	originalPassword := userRequest.Password

	userExists := h.srv.CheckIfUserExistsByUsername(userRequest.Username)
	if userExists {
		existingUser, err := h.srv.GetUserByUsername(userRequest.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
			return
		}

		if existingUser.Password == nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "User doesn't have a password", Success: false})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(*existingUser.Password), []byte(originalPassword))
		if err != nil {
			c.JSON(http.StatusBadRequest, wrapper.ErrorWrapper{Message: "Invalid password", Success: false})
			return
		}

		token, err := utils.GenerateToken(existingUser.ID, existingUser.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
			return
		}

		c.SetCookie("token", token, 3600*24, "/", "", false, true)
		c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: token, Success: true})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(originalPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	userRequest.Password = string(hashedPassword)

	userResponse, err := h.srv.CreateOrAuthenticateUserWithPassword(&userRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	token, err := utils.GenerateToken(userResponse.ID, userResponse.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: err.Error(), Success: false})
		return
	}

	c.SetCookie("token", token, 3600*24, "/", "", false, true)
	c.JSON(http.StatusOK, wrapper.ResponseWrapper{Data: userResponse, Success: true})
}

// GetUserById godoc
// @Summary Get user by ID
// @Description Get user by ID
// @Tags users
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
// @Tags users
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
// @Tags users
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
