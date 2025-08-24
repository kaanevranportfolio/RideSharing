package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port        string
	RedisURL    string
	DatabaseURL string
	Environment string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", ":8005"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@localhost/rideshare_db?sslmode=disable"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
