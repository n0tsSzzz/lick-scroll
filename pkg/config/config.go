package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT
	JWTSecret string

	// AWS S3
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3BucketName       string

	// Services URLs
	AuthServiceURL        string
	PostServiceURL        string
	FeedServiceURL        string
	FanoutServiceURL      string
	WalletServiceURL      string
	NotificationServiceURL string
	ModerationServiceURL  string
	AnalyticsServiceURL   string
}

func Load() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load()

	config := &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "lickscroll"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,

		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),

		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3BucketName:       getEnv("S3_BUCKET_NAME", "lick-scroll-content"),

		AuthServiceURL:        getEnv("AUTH_SERVICE_URL", "http://localhost:8001"),
		PostServiceURL:        getEnv("POST_SERVICE_URL", "http://localhost:8002"),
		FeedServiceURL:        getEnv("FEED_SERVICE_URL", "http://localhost:8003"),
		FanoutServiceURL:      getEnv("FANOUT_SERVICE_URL", "http://localhost:8004"),
		WalletServiceURL:      getEnv("WALLET_SERVICE_URL", "http://localhost:8005"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8006"),
		ModerationServiceURL:   getEnv("MODERATION_SERVICE_URL", "http://localhost:8007"),
		AnalyticsServiceURL:    getEnv("ANALYTICS_SERVICE_URL", "http://localhost:8008"),
	}

	// JWT_SECRET validation is optional - only required for services that use JWT
	// If not set, it will use default value and services without JWT will work fine

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

