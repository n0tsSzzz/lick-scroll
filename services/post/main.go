package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/middleware"
	"lick-scroll/pkg/models"
	"lick-scroll/pkg/s3"
	"lick-scroll/services/post/handlers"
	"lick-scroll/services/post/repository"

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

	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Error("Failed to create S3 client: %v", err)
		panic(err)
	}

	jwtService := jwt.NewService(cfg.JWTSecret)
	postRepo := repository.NewPostRepository(db)
	postHandler := handlers.NewPostHandler(postRepo, s3Client, redisClient, log)

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	api.Use(middleware.RateLimitMiddleware(redisClient, 100, 60)) // 100 requests per minute

	{
		api.POST("/posts", postHandler.CreatePost)
		api.GET("/posts/:id", postHandler.GetPost)
		api.GET("/posts", postHandler.ListPosts)
		api.PUT("/posts/:id", postHandler.UpdatePost)
		api.DELETE("/posts/:id", postHandler.DeletePost)
		api.GET("/posts/creator/:creator_id", postHandler.GetCreatorPosts)
	}

	log.Info("Post service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Error("Failed to start server: %v", err)
		panic(err)
	}
}

