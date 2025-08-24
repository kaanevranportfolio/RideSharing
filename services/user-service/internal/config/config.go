package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the user service
type Config struct {
	HTTPPort    string
	Environment string
	LogLevel    string

	// Database configuration
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		HTTPPort:    getEnv("HTTP_PORT", "8081"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Database configuration
		DatabaseHost:     getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:     getEnv("DATABASE_USER", "rideshare_user"),
		DatabasePassword: getEnv("DATABASE_PASSWORD", "rideshare_password"),
		DatabaseName:     getEnv("DATABASE_NAME", "rideshare"),
		DatabaseSSLMode:  getEnv("DATABASE_SSL_MODE", "disable"),
	}, nil
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

// getEnvAsBool gets an environment variable as boolean with a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
