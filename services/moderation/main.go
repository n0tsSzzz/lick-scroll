package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/services/moderation/handlers"
	"lick-scroll/services/moderation/repository"

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
	if err := db.AutoMigrate(&models.Post{}); err != nil {
		log.Error("Failed to migrate database: %v", err)
		panic(err)
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		panic(err)
	}

	moderationRepo := repository.NewModerationRepository(db)
	moderationHandler := handlers.NewModerationHandler(moderationRepo, redisClient, log)

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/moderation/review/:post_id", moderationHandler.ReviewPost)
		api.GET("/moderation/pending", moderationHandler.GetPendingPosts)
		api.POST("/moderation/approve/:post_id", moderationHandler.ApprovePost)
		api.POST("/moderation/reject/:post_id", moderationHandler.RejectPost)
	}

	log.Info("Moderation service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Error("Failed to start server: %v", err)
		panic(err)
	}
}

