package main

import (
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/services/auth/handlers"
	"lick-scroll/services/auth/repository"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "lick-scroll/services/auth/docs" // Swagger docs
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// @title           Auth Service API
// @version         1.0
// @description     Authentication and authorization service for Lick Scroll platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8001
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
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Error("Failed to migrate database: %v", err)
		panic(err)
	}

	jwtService := jwt.NewService(cfg.JWTSecret)
	userRepo := repository.NewUserRepository(db)
	authHandler := handlers.NewAuthHandler(userRepo, jwtService, log)

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API Documentation selector page (served from embedded or static)
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lick Scroll - API Documentation</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            max-width: 600px;
            width: 100%%;
        }
        h1 { color: #333; margin-bottom: 10px; font-size: 28px; }
        .subtitle { color: #666; margin-bottom: 30px; font-size: 14px; }
        .server-select { margin-bottom: 30px; }
        label { display: block; font-weight: 600; color: #333; margin-bottom: 12px; font-size: 14px; }
        select {
            width: 100%%;
            padding: 14px 16px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            background: white;
            cursor: pointer;
            transition: all 0.3s;
            appearance: none;
            background-image: url("data:image/svg+xml,%%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%%3E%%3Cpath fill='%%23333' d='M6 9L1 4h10z'/%%3E%%3C/svg%%3E");
            background-repeat: no-repeat;
            background-position: right 16px center;
            padding-right: 40px;
        }
        select:hover { border-color: #667eea; }
        select:focus { outline: none; border-color: #667eea; box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1); }
        .services-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 12px;
            margin-top: 20px;
        }
        .service-card {
            padding: 16px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            text-decoration: none;
            color: #333;
            transition: all 0.3s;
            display: block;
        }
        .service-card:hover {
            border-color: #667eea;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
        }
        .service-name { font-weight: 600; margin-bottom: 4px; }
        .service-port { color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Lick Scroll API</h1>
        <p class="subtitle">Выберите сервис для просмотра документации</p>
        <div class="server-select">
            <label for="serverSelect">Servers</label>
            <select id="serverSelect" onchange="window.location.href=this.value">
                <option value="">Выберите сервис...</option>
                <option value="http://localhost:8001/swagger/index.html">http://localhost:8001 - Auth Service</option>
                <option value="http://localhost:8002/swagger/index.html">http://localhost:8002 - Post Service</option>
                <option value="http://localhost:8003/swagger/index.html">http://localhost:8003 - Feed Service</option>
                <option value="http://localhost:8005/swagger/index.html">http://localhost:8005 - Wallet Service</option>
                <option value="http://localhost:8008/swagger/index.html">http://localhost:8008 - Analytics Service</option>
            </select>
        </div>
        <div class="services-grid">
            <a href="http://localhost:8001/swagger/index.html" class="service-card">
                <div class="service-name">🔐 Auth Service</div>
                <div class="service-port">Port: 8001</div>
            </a>
            <a href="http://localhost:8002/swagger/index.html" class="service-card">
                <div class="service-name">📝 Post Service</div>
                <div class="service-port">Port: 8002</div>
            </a>
            <a href="http://localhost:8003/swagger/index.html" class="service-card">
                <div class="service-name">📰 Feed Service</div>
                <div class="service-port">Port: 8003</div>
            </a>
            <a href="http://localhost:8005/swagger/index.html" class="service-card">
                <div class="service-name">💰 Wallet Service</div>
                <div class="service-port">Port: 8005</div>
            </a>
            <a href="http://localhost:8008/swagger/index.html" class="service-card">
                <div class="service-name">📊 Analytics Service</div>
                <div class="service-port">Port: 8008</div>
            </a>
        </div>
    </div>
</body>
</html>`)
	})

	// Swagger redirect
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
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
			
			claims, err := jwtService.ValidateToken(authHeader[7:]) // Remove "Bearer "
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
		}
	}

	log.Info("Auth service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Error("Failed to start server: %v", err)
		panic(err)
	}
}

