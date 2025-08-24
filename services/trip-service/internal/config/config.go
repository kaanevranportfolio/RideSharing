package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the trip service
type Config struct {
	HTTPPort    string
	Environment string
	LogLevel    string

	// Database config
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string

	// MongoDB config
	MongoURI      string
	MongoDatabase string

	// Redis config
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDatabase int

	// Trip service parameters
	MaxActiveTripDuration int    // hours
	TripTimeoutMinutes    int    // minutes
	CancellationWindow    int    // minutes after booking
	MaxPassengerCount     int    // maximum passengers per trip
	DefaultCurrency       string // default currency code
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		HTTPPort:    getEnv("HTTP_PORT", "8085"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Database config
		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabasePort:     getEnvInt("DB_PORT", 5432),
		DatabaseName:     getEnv("DB_NAME", "rideshare"),
		DatabaseUser:     getEnv("DB_USER", "rideshare_user"),
		DatabasePassword: getEnv("DB_PASSWORD", "rideshare_password"),

		// MongoDB config
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DB", "rideshare"),

		// Redis config
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDatabase: getEnvInt("REDIS_DB", 0),

		// Trip parameters
		MaxActiveTripDuration: getEnvInt("MAX_ACTIVE_TRIP_DURATION", 24),
		TripTimeoutMinutes:    getEnvInt("TRIP_TIMEOUT_MINUTES", 30),
		CancellationWindow:    getEnvInt("CANCELLATION_WINDOW", 5),
		MaxPassengerCount:     getEnvInt("MAX_PASSENGER_COUNT", 4),
		DefaultCurrency:       getEnv("DEFAULT_CURRENCY", "USD"),
	}, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Add validation logic here if needed
	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
