package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTExpiryHours   int
	RedisURL         string
	S3EndpointURL    string
	S3PublicURL      string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	S3Region         string
	S3VirtualHost    bool
	InternalAPIToken string
	CORSOrigins      string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	TokenEncryptionKey string
	DashboardURL       string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTExpiryHours:   expiryHours,
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379/0"),
		S3EndpointURL:    getEnv("S3_ENDPOINT_URL", "http://localhost:9000"),
		S3PublicURL:      os.Getenv("S3_PUBLIC_URL"),
		S3AccessKey:      getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:      getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:         getEnv("S3_BUCKET", "gradle-artifacts"),
		S3Region:         getEnv("S3_REGION", "us-east-1"),
		S3VirtualHost:    getEnv("S3_VIRTUAL_HOST_STYLE", "false") == "true",
		InternalAPIToken: os.Getenv("INTERNAL_API_TOKEN"),
		CORSOrigins:      getEnv("CORS_ORIGINS", "http://localhost:5173"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		TokenEncryptionKey: os.Getenv("GOOGLE_TOKEN_ENCRYPTION_KEY"),
		DashboardURL:       getEnv("DASHBOARD_URL", "http://localhost:5173"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.InternalAPIToken == "" {
		return nil, fmt.Errorf("INTERNAL_API_TOKEN is required")
	}
	// Google integration is optional at boot (local dev without Classroom
	// wired up yet); handlers that need it check these individually and
	// return a clear error instead of the server refusing to start.

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
