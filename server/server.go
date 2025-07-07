package server

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/config"
	"github.com/dinerozz/web-behavior-backend/docs"
	userExtensionHandler "github.com/dinerozz/web-behavior-backend/internal/handler/extension_user"
	userHandler "github.com/dinerozz/web-behavior-backend/internal/handler/user"
	handler "github.com/dinerozz/web-behavior-backend/internal/handler/user_behavior"
	userBehaviorHandler "github.com/dinerozz/web-behavior-backend/internal/handler/user_behavior"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	extensionUserService "github.com/dinerozz/web-behavior-backend/internal/service/extension_user"
	"github.com/dinerozz/web-behavior-backend/internal/service/user"
	service "github.com/dinerozz/web-behavior-backend/internal/service/user_behavior"
	"github.com/dinerozz/web-behavior-backend/middleware"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterHandler struct {
	userHandler          *userHandler.UserHandler
	userBehaviorHandler  *userBehaviorHandler.UserBehaviorHandler
	userExtensionHandler *userExtensionHandler.ExtensionUserHandler
	userExtensionService extensionUserService.ExtensionUserService
}

func RunServer(config *config.Config) {
	env := config.Env
	switch env {
	case "prod", "production":
		gin.SetMode(gin.ReleaseMode)
		log.Println("🚀 Starting server in PRODUCTION mode")
	case "dev", "development":
		gin.SetMode(gin.DebugMode)
		log.Println("🔧 Starting server in DEVELOPMENT mode")
	default:
		gin.SetMode(gin.DebugMode)
		log.Println("🔧 Starting server in DEVELOPMENT mode (default)")
	}

	fmt.Println("ENVs: ", config.DB.Host, config.DB.DBName, config.DB.User, config.Env)

	db, err := repository.NewRepository(config.DB)
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	userBehaviorRepo := repository.NewUserBehaviorRepository(db)
	userExtensionRepo := repository.NewExtensionUserRepository(db)

	userSrv := user.NewUserService(userRepo)
	userBehaviorService := service.NewUserBehaviorService(userBehaviorRepo)
	userExtensionService := extensionUserService.NewExtensionUserService(userExtensionRepo)

	userHandler := userHandler.NewUserHandler(userSrv)
	userBehaviorHandler := handler.NewUserBehaviorHandler(userBehaviorService)
	userExtensionHandler := userExtensionHandler.NewExtensionUserHandler(userExtensionService)

	routerHandler := &RouterHandler{
		userHandler:          userHandler,
		userBehaviorHandler:  userBehaviorHandler,
		userExtensionHandler: userExtensionHandler,
		userExtensionService: userExtensionService,
	}

	r := setupRouter(routerHandler)

	srv := &http.Server{
		Addr:    ":" + config.Server.Port,
		Handler: r,
	}

	go func() {
		log.Printf("✅ Server starting on port %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	gracefulShutdown(srv)
}

func gracefulShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("🔄 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
		return
	}

	select {
	case <-ctx.Done():
		log.Println("⚠️ Server shutdown timeout exceeded")
	default:
		log.Println("✅ Server gracefully stopped")
	}
}

func setupRouter(routerHandler *RouterHandler) *gin.Engine {
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" && (strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:")) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "https://web-behavior.space" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"service":   "web-behavior-app",
		})
	})

	docs.SwaggerInfo.Host = "127.0.0.1:8080"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.Title = "Web behavior app API"
	docs.SwaggerInfo.Description = "Web behavior app API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api/v1"

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	publicRoutes := r.Group("/api/v1/inayla")
	{
		publicRoutes.POST("/behaviors", routerHandler.userBehaviorHandler.CreateBehavior)

		extensionRoutes := publicRoutes.Group("/extension")
		extensionRoutes.Use(middleware.APIKeyMiddleware(routerHandler.userExtensionService))
		{
			//extensionRoutes.GET("/behaviors", routerHandler.userBehaviorHandler.GetBehaviors)
			//extensionRoutes.GET("/behaviors/stats", routerHandler.userBehaviorHandler.GetStats)
			extensionRoutes.GET("/users/auth", routerHandler.userExtensionHandler.ValidateAPIKey)
			//extensionRoutes.GET("/behaviors/:id", routerHandler.userBehaviorHandler.GetBehaviorByID)
			//extensionRoutes.GET("/behaviors/sessions/:sessionId", routerHandler.userBehaviorHandler.GetSessionSummary)
			//extensionRoutes.GET("/behaviors/users/:userId/sessions", routerHandler.userBehaviorHandler.GetUserSessions)
		}
	}

	publicAdminRoutes := r.Group("/api/v1/admin")
	{
		publicAdminRoutes.POST("/users/auth", routerHandler.userHandler.CreateOrAuthUserWithPassword)
	}

	privateRoutes := r.Group("/api/v1/admin")
	privateRoutes.Use(middleware.AuthenticationMiddleware())
	{
		extensionRoutes := privateRoutes.Group("/extension")

		privateRoutes.GET("/users/profile", routerHandler.userHandler.GetUserById)
		privateRoutes.GET("/behaviors", routerHandler.userBehaviorHandler.GetBehaviors)
		extensionRoutes.POST("/users/generate", routerHandler.userExtensionHandler.CreateExtensionUser)
		extensionRoutes.GET("/users/stats", routerHandler.userExtensionHandler.GetAllExtensionUsers)
	}

	return r
}
