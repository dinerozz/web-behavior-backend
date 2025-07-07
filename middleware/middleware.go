package middleware

import (
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/model/response/wrapper"
	service "github.com/dinerozz/web-behavior-backend/internal/service/extension_user"
	"github.com/dinerozz/web-behavior-backend/pkg/utils"
	"github.com/gin-gonic/gin"
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

// APIKeyMiddleware middleware для проверки API ключей расширения
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

		// Добавляем пользователя в контекст для использования в handlers
		c.Set("extension_user", user)
		c.Set("extension_user_id", user.ID.String())
		c.Set("extension_username", user.Username)

		c.Next()
	}
}

// OptionalAPIKeyMiddleware middleware для опциональной проверки API ключа
// Если ключ предоставлен - валидирует его, если нет - пропускает
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
