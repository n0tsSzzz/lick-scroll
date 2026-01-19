package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	notificationApp "lick-scroll/services/notification/internal/app"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

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
