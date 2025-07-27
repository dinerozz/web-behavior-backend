package middleware

import (
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	service "github.com/dinerozz/web-behavior-backend/internal/service/extension_user"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"net/http"
	"strings"
)

func AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "Missing authentication token", Success: false})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			fmt.Println("Error validating token", err)
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "Invalid authentication token", Success: false})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Next()
	}
}

func SwaggerHostMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			host := c.Request.Host
			if !strings.HasPrefix(host, "finansly.space") {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Access denied",
				})
				return
			}
		}
		c.Next()
	}
}

func APIKeyMiddleware(extensionUserService service.ExtensionUserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
				Message: "X-API-Key header is required",
				Success: false,
			})
			c.Abort()
			return
		}

		user, err := extensionUserService.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{
				Message: "Invalid or inactive API key",
				Success: false,
			})
			c.Abort()
			return
		}

		c.Set("extension_user", user)
		c.Set("extension_user_id", user.ID.String())
		c.Set("extension_username", user.Username)

		c.Next()
	}
}

func OptionalAPIKeyMiddleware(extensionUserService service.ExtensionUserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		if apiKey != "" {
			user, err := extensionUserService.ValidateAPIKey(c.Request.Context(), apiKey)
			if err == nil {
				c.Set("extension_user", user)
				c.Set("extension_user_id", user.ID.String())
				c.Set("extension_username", user.Username)
			}
		}

		c.Next()
	}
}

func SuperAdminMiddleware(userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, wrapper.ErrorWrapper{Message: "User ID not found", Success: false})
			c.Abort()
			return
		}

		userUUID, err := uuid.FromString(userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: "Invalid user ID", Success: false})
			c.Abort()
			return
		}

		isSuperAdmin, err := userRepo.IsUserSuperAdmin(userUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, wrapper.ErrorWrapper{Message: "Failed to check admin status", Success: false})
			c.Abort()
			return
		}

		if !isSuperAdmin {
			c.JSON(http.StatusForbidden, wrapper.ErrorWrapper{Message: "Super admin access required", Success: false})
			c.Abort()
			return
		}

		c.Set("is_super_admin", true)
		c.Next()
	}
}
