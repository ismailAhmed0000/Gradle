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
	InternalAPIToken string
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
		InternalAPIToken: os.Getenv("INTERNAL_API_TOKEN"),
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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
