package config

import (
	"os"
	"strconv"
	"time"

	"github.com/rideshare-platform/shared/config"
)

// Config holds the vehicle service configuration
type Config struct {
	// Service configuration
	Environment string
	LogLevel    string
	HTTPPort    int
	GRPCPort    int
	JWTSecret   string

	// Database configuration
	Database config.DatabaseConfig

	// Redis configuration
	Redis *config.RedisConfig
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		HTTPPort:    getEnvAsInt("HTTP_PORT", 8082),
		GRPCPort:    getEnvAsInt("GRPC_PORT", 50052),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
	}

	// Database configuration
	cfg.Database = config.DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		Username:        getEnv("DB_USERNAME", "rideshare_user"),
		Password:        getEnv("DB_PASSWORD", "rideshare_password"),
		Database:        getEnv("DB_NAME", "rideshare"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
		ConnMaxIdleTime: time.Duration(getEnvAsInt("DB_CONN_MAX_IDLE_TIME", 60)) * time.Second,
	}

	// Redis configuration
	cfg.Redis = &config.RedisConfig{
		Host:         getEnv("REDIS_HOST", "localhost"),
		Port:         getEnvAsInt("REDIS_PORT", 6379),
		Password:     getEnv("REDIS_PASSWORD", ""),
		Database:     getEnvAsInt("REDIS_DATABASE", 0),
		PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 100),
		MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 10),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  5 * time.Minute,
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
