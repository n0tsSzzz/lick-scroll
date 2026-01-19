package internal

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
	"lick-scroll/pkg/queue"
	"lick-scroll/pkg/s3"
	authHTTP "lick-scroll/services/auth/internal/controller/http"
	"lick-scroll/services/auth/internal/repo/persistent"
	"lick-scroll/services/auth/internal/usecase"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
	
	_ "lick-scroll/services/auth/docs" // Swagger docs
)

type App struct {
	cfg        *config.Config
	log        *logger.Logger
	db         *gorm.DB
	redisClient *redis.Client
	s3Client   *s3.Client
	jwtService *jwt.Service
	queueClient *queue.Client
	httpServer *http.Server
}

func NewApp(cfg *config.Config) (*App, error) {
	log := logger.New()

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("Failed to connect to database: %v", err)
		return nil, err
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		// Redis is optional for auth service
		redisClient = nil
	}

	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Error("Failed to create S3 client: %v", err)
		return nil, err
	}

	queueClient, err := queue.NewRabbitMQClient(cfg, log)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ: %v (continuing without queue)", err)
		queueClient = nil
	}

	jwtService := jwt.NewService(cfg.JWTSecret)

	return &App{
		cfg:         cfg,
		log:         log,
		db:          db,
		redisClient: redisClient,
		s3Client:    s3Client,
		jwtService:  jwtService,
		queueClient: queueClient,
	}, nil
}

func (a *App) Run() error {
	// Initialize repositories
	userRepo := persistent.NewUserRepository(a.db)

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		a.jwtService,
		a.s3Client,
		a.queueClient,
		a.log,
	)

	// Initialize HTTP handlers
	authHandler := authHTTP.NewAuthHandler(authUseCase)

	// Setup router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)

		// Protected routes
		protected := api.Group("")
		protected.Use(func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}

			claims, err := a.jwtService.ValidateToken(authHeader[7:]) // Remove "Bearer "
			if err != nil {
				c.JSON(401, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}

			c.Set("user_id", claims.UserID)
			c.Set("user_role", claims.Role)
			c.Next()
		})
		{
			protected.GET("/me", authHandler.Me)
			protected.GET("/user/:id", authHandler.GetUser)
			protected.POST("/avatar", authHandler.UploadAvatar)
			// Subscription endpoints
			protected.GET("/users/:user_id/subscriptions", authHandler.GetSubscriptions)
			protected.POST("/users/:user_id/subscriptions/:creator_id", authHandler.Subscribe)
			protected.DELETE("/users/:user_id/subscriptions/:creator_id", authHandler.Unsubscribe)
			protected.GET("/users/:user_id/subscriptions/:creator_id/status", authHandler.GetSubscriptionStatus)
		}
	}

	// Create HTTP server
	a.httpServer = &http.Server{
		Addr:    ":" + a.cfg.ServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		a.log.Info("Auth service starting on port %s", a.cfg.ServerPort)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	return nil
}

func (a *App) Wait() {
	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	a.log.Info("Shutting down auth service...")
}

func (a *App) Shutdown() error {
	// The context is used to inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close database connection
	sqlDB, err := a.db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			a.log.Error("Error closing database: %v", err)
		}
	}

	// Close Redis connection
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			a.log.Error("Error closing Redis: %v", err)
		}
	}

	// Shutdown server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.log.Error("Server forced to shutdown: %v", err)
		return err
	}

	a.log.Info("Auth service exited")
	return nil
}
