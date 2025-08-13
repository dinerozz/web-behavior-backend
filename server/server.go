package server

import (
	"context"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/config"
	"github.com/dinerozz/web-behavior-backend/docs"
	aiHandler "github.com/dinerozz/web-behavior-backend/internal/handler/ai-analytics"
	downloadExtensionHandler "github.com/dinerozz/web-behavior-backend/internal/handler/download_extension"
	userExtensionHandler "github.com/dinerozz/web-behavior-backend/internal/handler/extension_user"
	"github.com/dinerozz/web-behavior-backend/internal/handler/metrics"
	organizationHandler "github.com/dinerozz/web-behavior-backend/internal/handler/organization"
	userHandler "github.com/dinerozz/web-behavior-backend/internal/handler/user"
	handler "github.com/dinerozz/web-behavior-backend/internal/handler/user_behavior"
	userBehaviorHandler "github.com/dinerozz/web-behavior-backend/internal/handler/user_behavior"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	aiAnalyticsService "github.com/dinerozz/web-behavior-backend/internal/service/ai_analytics"
	extensionUserService "github.com/dinerozz/web-behavior-backend/internal/service/extension_user"
	metricsService "github.com/dinerozz/web-behavior-backend/internal/service/metrics_service"
	organizationService "github.com/dinerozz/web-behavior-backend/internal/service/organization"
	"github.com/dinerozz/web-behavior-backend/internal/service/redis"
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
	userHandler              *userHandler.UserHandler
	userBehaviorHandler      *userBehaviorHandler.UserBehaviorHandler
	userExtensionHandler     *userExtensionHandler.ExtensionUserHandler
	userExtensionService     extensionUserService.ExtensionUserService
	userMetricsHandler       *metrics.MetricsHandler
	aiAnalyticsHandler       *aiHandler.AIAnalyticsHandler
	organizationHandler      *organizationHandler.OrganizationHandler
	downloadExtensionHandler *downloadExtensionHandler.ExtensionHandler
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

	redisConfig := redis.RedisConfig{
		Host:     config.Redis.Host,
		Port:     config.Redis.Port,
		Password: config.Redis.Password,
	}

	redisService := redis.NewRedisService(redisConfig)
	if redisService == nil {
		log.Fatal("Failed to create Redis service")
	}
	defer redisService.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	userBehaviorRepo := repository.NewUserBehaviorRepository(db)
	userExtensionRepo := repository.NewExtensionUserRepository(db)
	userMetricsRepo := repository.NewMetricsRepository(db)
	organizationRepo := repository.NewOrganizationRepository(db)

	// Initialize services
	userSrv := user.NewUserService(userRepo)
	userBehaviorService := service.NewUserBehaviorService(userBehaviorRepo)
	userExtensionService := extensionUserService.NewExtensionUserService(userExtensionRepo, *organizationRepo)
	organizationSrv := organizationService.NewOrganizationService(organizationRepo, userRepo)

	aiService := aiAnalyticsService.NewAIAnalyticsService("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")

	userMetricsService := metricsService.NewMetricsService(userMetricsRepo, aiService)

	// Initialize handlers
	userHandler := userHandler.NewUserHandler(userSrv, organizationSrv)
	userBehaviorHandler := handler.NewUserBehaviorHandler(userBehaviorService)
	userExtensionHandler := userExtensionHandler.NewExtensionUserHandler(userExtensionService)
	userMetricsHandler := metrics.NewMetricsHandler(userMetricsService, redisService)
	aiAnalyticsHandler := aiHandler.NewAIAnalyticsHandler(aiService, redisService)
	organizationHandler := organizationHandler.NewOrganizationHandler(organizationSrv)
	downloadExtensionHandler := downloadExtensionHandler.NewExtensionHandler(userRepo)

	routerHandler := &RouterHandler{
		userHandler:              userHandler,
		userBehaviorHandler:      userBehaviorHandler,
		userExtensionHandler:     userExtensionHandler,
		userExtensionService:     userExtensionService,
		userMetricsHandler:       userMetricsHandler,
		aiAnalyticsHandler:       aiAnalyticsHandler,
		organizationHandler:      organizationHandler,
		downloadExtensionHandler: downloadExtensionHandler,
	}

	r := setupRouter(routerHandler, userRepo)

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

func setupRouter(routerHandler *RouterHandler, userRepo *repository.UserRepository) *gin.Engine {
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" && (strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:")) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "https://inayla.com" {
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

	// Public routes for data collection
	publicRoutes := r.Group("/api/v1/inayla")
	{
		publicRoutes.POST("/behaviors", routerHandler.userBehaviorHandler.CreateBehavior)
		publicRoutes.POST("/behaviors/batch", routerHandler.userBehaviorHandler.BatchCreateBehaviors)

		extensionRoutes := publicRoutes.Group("/extension")
		extensionRoutes.Use(middleware.APIKeyMiddleware(routerHandler.userExtensionService))
		{
			extensionRoutes.GET("/users/auth", routerHandler.userExtensionHandler.ValidateAPIKey)
		}
	}

	// Public admin authentication routes
	publicAdminRoutes := r.Group("/api/v1/admin")
	{
		publicAdminRoutes.POST("/users/auth", routerHandler.userHandler.AuthenticateUserWithPassword)
	}

	// Private authenticated routes
	privateRoutes := r.Group("/api/v1/admin")
	privateRoutes.Use(middleware.AuthenticationMiddleware())
	{
		// User routes
		privateRoutes.GET("/users/profile", routerHandler.userHandler.GetUserById)
		privateRoutes.GET("/users/profile/full", routerHandler.userHandler.GetUserWithOrganizations)
		privateRoutes.POST("/users/logout", routerHandler.userHandler.Logout)
		privateRoutes.POST("/users/register", routerHandler.userHandler.CreateUserWithPassword)

		superAdminRoutes := privateRoutes.Group("")
		superAdminRoutes.Use(middleware.SuperAdminMiddleware(userRepo))
		{
			superAdminRoutes.GET("/users", routerHandler.userHandler.GetAllUsers)
		}

		// Organization routes
		orgRoutes := privateRoutes.Group("/organizations")
		{
			// Organization CRUD
			orgRoutes.POST("", routerHandler.organizationHandler.CreateOrganization)
			orgRoutes.GET("", routerHandler.organizationHandler.GetAll)
			orgRoutes.GET("/my", routerHandler.organizationHandler.GetUserOrganizations)
			orgRoutes.GET("/:id", routerHandler.organizationHandler.GetOrganization)
			orgRoutes.GET("/:id/members", routerHandler.organizationHandler.GetOrganizationWithMembers)
			orgRoutes.PUT("/:id", routerHandler.organizationHandler.UpdateOrganization)
			orgRoutes.DELETE("/:id", routerHandler.organizationHandler.DeleteOrganization)

			// User management within organizations
			orgRoutes.POST("/:id/users", routerHandler.organizationHandler.AddUserToOrganization)
			orgRoutes.DELETE("/:id/users/:user_id", routerHandler.organizationHandler.RemoveUserFromOrganization)
			orgRoutes.PUT("/:id/users/:user_id/role", routerHandler.organizationHandler.UpdateUserRole)
		}

		// Behavior analytics routes
		privateRoutes.GET("/behaviors", routerHandler.userBehaviorHandler.GetBehaviors)
		privateRoutes.GET("/behaviors/periods", routerHandler.userBehaviorHandler.GetBehaviorsPeriods)
		privateRoutes.GET("/behaviors/stats", routerHandler.userBehaviorHandler.GetStats)
		privateRoutes.GET("/behaviors/sessions/:sessionId", routerHandler.userBehaviorHandler.GetSessionSummary)
		privateRoutes.GET("/behaviors/:id", routerHandler.userBehaviorHandler.GetBehaviorByID)
		privateRoutes.GET("/behaviors/users/:userId/sessions", routerHandler.userBehaviorHandler.GetUserSessions)
		privateRoutes.GET("/behaviors/user-events", routerHandler.userBehaviorHandler.GetUserEventsCount)
		privateRoutes.DELETE("/behaviors/:id", routerHandler.userBehaviorHandler.DeleteBehavior)

		// AI analytics routes
		privateRoutes.POST("/ai-analytics/domain-usage", routerHandler.aiAnalyticsHandler.AnalyzeDomainUsage)
		privateRoutes.GET("/ai-analytics/focus-level", routerHandler.aiAnalyticsHandler.GetFocusLevel)

		// Metrics routes
		privateRoutes.GET("/metrics/tracked-time", routerHandler.userMetricsHandler.GetTrackedTime)
		privateRoutes.GET("/metrics/tracked-time-total", routerHandler.userMetricsHandler.GetTrackedTimeTotal)
		privateRoutes.GET("/metrics/engaged-time", routerHandler.userMetricsHandler.GetEngagedTime)
		privateRoutes.GET("/metrics/top-domains", routerHandler.userMetricsHandler.GetTopDomains)
		privateRoutes.GET("/metrics/deep-work-sessions", routerHandler.userMetricsHandler.GetDeepWorkSessions)

		// Extension management routes
		extensionRoutes := privateRoutes.Group("/extension")
		{
			extensionRoutes.POST("/users/generate", routerHandler.userExtensionHandler.CreateExtensionUser)
			extensionRoutes.POST("/users/:id/regenerate-key", routerHandler.userExtensionHandler.RegenerateAPIKey)
			extensionRoutes.GET("/users", routerHandler.userExtensionHandler.GetAllExtensionUsers)
			extensionRoutes.GET("/users/:id", routerHandler.userExtensionHandler.GetExtensionUserByID)
			extensionRoutes.GET("/users/stats", routerHandler.userExtensionHandler.GetExtensionUserStats)
			extensionRoutes.DELETE("/users/:id", routerHandler.userExtensionHandler.DeleteExtensionUser)
			extensionRoutes.PUT("/users/:id", routerHandler.userExtensionHandler.UpdateExtensionUser)
		}

		// ===== CHROME EXTENSION AUTH ROUTES =====
		authGroup := r.Group("/api/auth")
		{
			authGroup.Any("/verify-admin", routerHandler.downloadExtensionHandler.VerifyAdmin)
		}

		// ===== CHROME EXTENSION PUBLIC ROUTES =====
		chromeExtensionGroup := r.Group("/api/download-extension")
		{
			chromeExtensionGroup.GET("/info", routerHandler.downloadExtensionHandler.GetExtensionInfo)

			chromeExtensionGroup.GET("/health", routerHandler.downloadExtensionHandler.GetExtensionHealth)
		}

		chromeExtensionAdminRoutes := privateRoutes.Group("/download-extension")
		chromeExtensionAdminRoutes.Use(middleware.SuperAdminMiddleware(userRepo))
		{
			chromeExtensionAdminRoutes.GET("/stats", routerHandler.downloadExtensionHandler.GetExtensionStats)
			//chromeExtensionAdminRoutes.POST("/deploy", routerHandler.downloadExtensionHandler.DeployExtension)

		}
	}

	return r
}
