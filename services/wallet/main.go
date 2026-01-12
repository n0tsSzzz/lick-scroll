package main

import (
	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/middleware"
	"lick-scroll/pkg/models"
	"lick-scroll/services/wallet/handlers"
	"lick-scroll/services/wallet/repository"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "lick-scroll/services/wallet/docs" // Swagger docs
)

// @title           Wallet Service API
// @version         1.0
// @description     Wallet and transaction service for Lick Scroll platform
// @host      localhost:8005
// @BasePath  /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

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
	if err := db.AutoMigrate(&models.Wallet{}, &models.Transaction{}); err != nil {
		log.Error("Failed to migrate database: %v", err)
		panic(err)
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		panic(err)
	}

	jwtService := jwt.NewService(cfg.JWTSecret)
	walletRepo := repository.NewWalletRepository(db)
	walletHandler := handlers.NewWalletHandler(walletRepo, redisClient, log)

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation - catch-all must be registered last
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	api.Use(middleware.RateLimitMiddleware(redisClient, 100, 60))

	{
		api.GET("/wallet", walletHandler.GetWallet)
		api.POST("/wallet/topup", walletHandler.TopUp)
		api.POST("/wallet/purchase/:post_id", walletHandler.PurchasePost)
		api.GET("/wallet/transactions", walletHandler.GetTransactions)
	}

	log.Info("Wallet service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Error("Failed to start server: %v", err)
		panic(err)
	}
}

