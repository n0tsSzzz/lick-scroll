package internal

import (
	"context"
	"fmt"
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
	notificationHTTP "lick-scroll/services/notification/internal/controller/http"
	"lick-scroll/services/notification/internal/repo/persistent"
	"lick-scroll/services/notification/internal/usecase"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "lick-scroll/services/notification/docs" // Swagger docs
)

func Run(cfg *config.Config, log *logger.Logger, db *gorm.DB, redisClient *redis.Client, queueClient *queue.Client) {
	jwtService := jwt.NewService(cfg.JWTSecret)

	// Initialize Repository
	notificationRepo := persistent.NewNotificationRepository(db)

	// Initialize UseCase
	notificationUseCase := usecase.NewNotificationUseCase(notificationRepo, redisClient, queueClient, log)

	// Initialize HTTP handlers
	notificationHandler := notificationHTTP.NewNotificationHandler(notificationUseCase, redisClient, log, jwtService)

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
	// Protected routes - require authentication
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(jwtService))
	{
		protected.GET("/notifications", notificationHandler.GetNotifications)
		protected.DELETE("/notifications/:post_id", notificationHandler.DeleteNotificationByPostID)
		protected.GET("/notifications/settings/:creator_id", notificationHandler.GetNotificationSettings)
		protected.POST("/notifications/settings/:creator_id", notificationHandler.EnableNotifications)
		protected.DELETE("/notifications/settings/:creator_id", notificationHandler.DisableNotifications)
	}
	// WebSocket endpoint - handles authentication internally via query parameter
	api.GET("/notifications/ws", notificationHandler.HandleWebSocket)
	// Admin routes - no auth required (for internal service calls)
	{
		api.POST("/notifications/send", notificationHandler.SendNotification)
		api.POST("/notifications/broadcast", notificationHandler.BroadcastNotification)
		api.POST("/notifications/process-queue", notificationHandler.ProcessNotificationQueue)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Start processing notification queue in a goroutine
	go func() {
		log.Info("Starting notification queue processor...")
		
		// Start consuming from RabbitMQ queue
		err := queueClient.ConsumeNotificationTasks(func(task map[string]interface{}) error {
			log.Info("[NOTIFICATION HANDLER] Received task from RabbitMQ queue: %+v", task)
			
			// Determine notification type from task
			notificationType, _ := task["type"].(string)
			if notificationType == "" {
				// Default to "new_post" for backward compatibility
				notificationType = "new_post"
			}

			log.Info("[NOTIFICATION HANDLER] Processing notification task: type=%s", notificationType)

			// Route to appropriate handler based on type
			switch notificationType {
			case "new_post":
				return notificationUseCase.HandleNewPostNotification(task)
			case "like":
				return notificationUseCase.HandleLikeNotification(task)
			case "subscription":
				return notificationUseCase.HandleSubscriptionNotification(task)
			default:
				log.Error("[NOTIFICATION HANDLER] Unknown notification type: %s, task=%+v", notificationType, task)
				return fmt.Errorf("unknown notification type: %s", notificationType)
			}
		})
		if err != nil {
			log.Error("Error starting notification queue consumer: %v", err)
		}
	}()

	// Start server in a goroutine
	go func() {
		log.Info("Notification service starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down notification service...")

	// The context is used to inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Error("Error closing Redis: %v", err)
	}

	// Close RabbitMQ connection
	if queueClient != nil {
		queueClient.Close()
	}

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
		panic(err)
	}

	log.Info("Notification service exited")
}
