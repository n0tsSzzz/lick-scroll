package main

import (
	"lick-scroll/pkg/config"
	app "lick-scroll/services/auth/internal/app"

	_ "lick-scroll/services/auth/docs" // Swagger docs
)

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

func init() {
	// Gin is set to release mode in app.go
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Validate JWT_SECRET for services that use JWT
	if cfg.JWTSecret == "your-secret-key-change-in-production" || cfg.JWTSecret == "" {
		panic("JWT_SECRET must be set in environment variables")
	}

	application, err := app.NewApp(cfg)
	if err != nil {
		panic(err)
	}

	if err := application.Run(); err != nil {
		panic(err)
	}

	application.Wait()

	if err := application.Shutdown(); err != nil {
		panic(err)
	}
}
