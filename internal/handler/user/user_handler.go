package user

import (
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/service/user"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	srv *user.UserService
}

func NewUserHandler(srv *user.UserService) *UserHandler {
	return &UserHandler{srv: srv}
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
