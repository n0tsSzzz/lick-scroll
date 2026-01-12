package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/middleware"
	"lick-scroll/services/feed/handlers"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "lick-scroll/services/feed/docs" // Swagger docs
)

// @title           Feed Service API
// @version         1.0
// @description     Feed service for Lick Scroll platform
// @host      localhost:8003
// @BasePath  /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Validate JWT_SECRET for services that use JWT
	if cfg.JWTSecret == "your-secret-key-change-in-production" || cfg.JWTSecret == "" {
		panic("JWT_SECRET must be set in environment variables")
	}

	log := logger.New()
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		panic(err)
	}

	jwtService := jwt.NewService(cfg.JWTSecret)
	feedHandler := handlers.NewFeedHandler(redisClient, log)

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation - catch-all must be registered last
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	api.Use(middleware.RateLimitMiddleware(redisClient, 200, 60)) // 200 requests per minute

	{
		api.GET("/feed", feedHandler.GetFeed)
		api.GET("/feed/category/:category", feedHandler.GetFeedByCategory)
	}

	log.Info("Feed service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Error("Failed to start server: %v", err)
		panic(err)
	}
}

