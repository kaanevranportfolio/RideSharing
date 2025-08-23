package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the matching service
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

	// Matching algorithm parameters
	MaxSearchRadius       float64 // km
	MaxMatchingTimeout    int     // seconds
	MaxDriversToConsider  int     // number of drivers
	DriverResponseTimeout int     // seconds
	PriorityBoostRadius   float64 // km
	PremiumPriorityBoost  float64 // multiplier
	MaxConcurrentMatches  int     // concurrent processing limit
	MatchingRetryAttempts int     // retry attempts
	MatchingRetryDelayMs  int     // ms between retries
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		HTTPPort:    getEnv("HTTP_PORT", "8084"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Database config
		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabasePort:     getEnvInt("DB_PORT", 5432),
		DatabaseName:     getEnv("DB_NAME", "rideshare"),
		DatabaseUser:     getEnv("DB_USER", "postgres"),
		DatabasePassword: getEnv("DB_PASSWORD", "postgres"),

		// MongoDB config
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DB", "rideshare"),

		// Redis config
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDatabase: getEnvInt("REDIS_DB", 0),

		// Matching parameters
		MaxSearchRadius:       getEnvFloat("MAX_SEARCH_RADIUS", 10.0),
		MaxMatchingTimeout:    getEnvInt("MAX_MATCHING_TIMEOUT", 30),
		MaxDriversToConsider:  getEnvInt("MAX_DRIVERS_TO_CONSIDER", 20),
		DriverResponseTimeout: getEnvInt("DRIVER_RESPONSE_TIMEOUT", 30),
		PriorityBoostRadius:   getEnvFloat("PRIORITY_BOOST_RADIUS", 2.0),
		PremiumPriorityBoost:  getEnvFloat("PREMIUM_PRIORITY_BOOST", 1.5),
		MaxConcurrentMatches:  getEnvInt("MAX_CONCURRENT_MATCHES", 100),
		MatchingRetryAttempts: getEnvInt("MATCHING_RETRY_ATTEMPTS", 3),
		MatchingRetryDelayMs:  getEnvInt("MATCHING_RETRY_DELAY_MS", 1000),
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

// getEnvFloat gets an environment variable as float64 with a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
