package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/middleware"
	"lick-scroll/pkg/models"
	"lick-scroll/pkg/queue"
	"lick-scroll/services/fanout/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New()
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("Failed to connect to database: %v", err)
		panic(err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		log.Error("Failed to migrate database: %v", err)
		panic(err)
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		panic(err)
	}

	queueClient, err := queue.NewRabbitMQClient(cfg, log)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ: %v", err)
		panic(err)
	}

	fanoutHandler := handlers.NewFanoutHandler(db, redisClient, queueClient, log)
	jwtService := jwt.NewService(cfg.JWTSecret)

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
	{
		api.POST("/fanout/post/:post_id", fanoutHandler.FanoutPost)
		api.POST("/subscribe/:creator_id", fanoutHandler.Subscribe)
		api.GET("/subscribe/:creator_id", fanoutHandler.GetSubscriptionStatus)
		api.DELETE("/subscribe/:creator_id", fanoutHandler.Unsubscribe)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Fanout service starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down fanout service...")

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

	// Close RabbitMQ connection
	if err := queueClient.Close(); err != nil {
		log.Error("Error closing RabbitMQ: %v", err)
	}

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
		panic(err)
	}

	log.Info("Fanout service exited")
}

