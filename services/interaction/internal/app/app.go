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
	"lick-scroll/pkg/queue"
	interactionHTTP "lick-scroll/services/interaction/internal/controller/http"
	"lick-scroll/services/interaction/internal/repo/persistent"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Run(cfg *config.Config, log *logger.Logger, db *gorm.DB, redisClient *redis.Client, queueClient *queue.Client) {
	jwtService := jwt.NewService(cfg.JWTSecret)

	// Initialize repositories
	interactionRepo := persistent.NewInteractionRepository(db)

	// Initialize HTTP handlers
	interactionHandler := interactionHTTP.NewInteractionHandler(interactionRepo, db, redisClient, queueClient, log)

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
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	api.Use(middleware.RateLimitMiddleware(redisClient, 100, time.Minute))

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(jwtService))
	{
		protected.POST("/interactions/posts/:post_id/like", interactionHandler.LikePost)
		protected.GET("/interactions/posts/:post_id/liked", interactionHandler.IsLiked)
		protected.GET("/interactions/posts/liked", interactionHandler.GetLikedPosts)
		protected.POST("/interactions/posts/:post_id/view", interactionHandler.IncrementView)
	}

	// Public routes
	{
		api.GET("/interactions/posts/:post_id/likes", interactionHandler.GetLikeCount)
		api.GET("/interactions/posts/:post_id/views", interactionHandler.GetViewCount)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Interaction service starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down interaction service...")

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

	// Close Redis connection if it was initialized
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Error("Error closing Redis: %v", err)
		}
	}

	// Close RabbitMQ connection if it was initialized
	if queueClient != nil {
		queueClient.Close()
	}

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
		panic(err)
	}

	log.Info("Interaction service exited")
}
