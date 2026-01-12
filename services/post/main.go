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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "lick-scroll/services/post/docs" // Swagger docs
)

// @title           Post Service API
// @version         1.0
// @description     Post management service for Lick Scroll platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8002
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

	// Validate JWT_SECRET for services that use JWT
	if cfg.JWTSecret == "your-secret-key-change-in-production" || cfg.JWTSecret == "" {
		panic("JWT_SECRET must be set in environment variables")
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

