package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	notificationApp "lick-scroll/services/notification/internal/app"

	"github.com/gin-gonic/gin"

	_ "lick-scroll/services/notification/docs" // Swagger docs
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// @title           Notification Service API
// @version         1.0
// @description     Notification service for real-time notifications via WebSocket and RabbitMQ
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8006
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	notificationApp.Run(cfg, log, db, redisClient, queueClient)
}
