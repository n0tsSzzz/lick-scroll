package internal

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lick-scroll/pkg/config"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/middleware"
	analyticsHTTP "lick-scroll/services/analytics/internal/controller/http"
	"lick-scroll/services/analytics/internal/repo/persistent"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "lick-scroll/services/analytics/docs" // Swagger docs
)

func Run(cfg *config.Config, log *logger.Logger, db *gorm.DB, redisClient *redis.Client) {
	jwtService := jwt.NewService(cfg.JWTSecret)

	// Initialize repositories
	analyticsRepo := persistent.NewAnalyticsRepository(db)

	// Initialize HTTP handlers
	analyticsHandler := analyticsHTTP.NewAnalyticsHandler(analyticsRepo, redisClient, log)

	// Setup router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	api.Use(middleware.RateLimitMiddleware(redisClient, 100, time.Minute))

	{
		api.GET("/analytics/creator/stats", analyticsHandler.GetCreatorStats)
		api.GET("/analytics/creator/posts/:post_id", analyticsHandler.GetPostStats)
		api.GET("/analytics/creator/revenue", analyticsHandler.GetRevenue)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Analytics service starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down analytics service...")

	// The context is used to inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close database connection
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Error("Error closing database: %v", err)
		}
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Error("Error closing Redis: %v", err)
	}

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
		panic(err)
	}

	log.Info("Analytics service exited")
}
