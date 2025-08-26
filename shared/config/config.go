package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`

	// Database configurations
	Database DatabaseConfig `json:"database"`
	MongoDB  MongoConfig    `json:"mongodb"`
	Redis    RedisConfig    `json:"redis"`

	// External services
	JWT   JWTConfig   `json:"jwt"`
	Kafka KafkaConfig `json:"kafka"`

	// Monitoring
	Metrics MetricsConfig `json:"metrics"`

	// Environment
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DatabaseConfig represents PostgreSQL database configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Database        string        `json:"database"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// MongoConfig represents MongoDB configuration
type MongoConfig struct {
	URI                    string        `json:"uri"`
	Database               string        `json:"database"`
	MaxPoolSize            uint64        `json:"max_pool_size"`
	MinPoolSize            uint64        `json:"min_pool_size"`
	MaxConnIdleTime        time.Duration `json:"max_conn_idle_time"`
	MaxConnecting          uint64        `json:"max_connecting"`
	ConnectTimeout         time.Duration `json:"connect_timeout"`
	ServerSelectionTimeout time.Duration `json:"server_selection_timeout"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	Database     int           `json:"database"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey       string        `json:"secret_key"`
	ExpiryDuration  time.Duration `json:"expiry_duration"`
	RefreshDuration time.Duration `json:"refresh_duration"`
	Issuer          string        `json:"issuer"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
	GroupID string   `json:"group_id"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Path    string `json:"path"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "postgres"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			Database:        getEnv("DB_NAME", "rideshare_platform"),
			Username:        getEnv("DB_USER", "rideshare"),
			Password:        getEnv("DB_PASSWORD", ""),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", time.Hour),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 15*time.Minute),
		},
		MongoDB: MongoConfig{
			URI:                    getEnv("MONGO_URI", "mongodb://mongodb:27017"),
			Database:               getEnv("MONGO_DATABASE", "rideshare_geo"),
			MaxPoolSize:            uint64(getEnvAsInt("MONGO_MAX_POOL_SIZE", 100)),
			MinPoolSize:            uint64(getEnvAsInt("MONGO_MIN_POOL_SIZE", 10)),
			MaxConnIdleTime:        getEnvAsDuration("MONGO_MAX_CONN_IDLE_TIME", 30*time.Minute),
			MaxConnecting:          uint64(getEnvAsInt("MONGO_MAX_CONNECTING", 10)),
			ConnectTimeout:         getEnvAsDuration("MONGO_CONNECT_TIMEOUT", 10*time.Second),
			ServerSelectionTimeout: getEnvAsDuration("MONGO_SERVER_SELECTION_TIMEOUT", 5*time.Second),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "redis"),
			Port:         getEnvAsInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			Database:     getEnvAsInt("REDIS_DATABASE", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 100),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 10),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			IdleTimeout:  getEnvAsDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		},
		JWT: JWTConfig{
			SecretKey:       getEnv("JWT_SECRET_KEY", ""),
			ExpiryDuration:  getEnvAsDuration("JWT_EXPIRY", 24*time.Hour),
			RefreshDuration: getEnvAsDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "rideshare-platform"),
		},
		Kafka: KafkaConfig{
			Brokers: getEnvAsSlice("KAFKA_BROKERS", []string{"kafka:9092"}),
			GroupID: getEnv("KAFKA_GROUP_ID", "rideshare-platform"),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvAsBool("METRICS_ENABLED", true),
			Host:    getEnv("METRICS_HOST", "0.0.0.0"),
			Port:    getEnvAsInt("METRICS_PORT", 9090),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return config, nil
}

// GetDatabaseURL returns the PostgreSQL connection URL
func (c *DatabaseConfig) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// GetRedisURL returns the Redis connection URL
func (c *RedisConfig) GetRedisURL() string {
	if c.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d", c.Password, c.Host, c.Port, c.Database)
	}
	return fmt.Sprintf("redis://%s:%d/%d", c.Host, c.Port, c.Database)
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsStaging returns true if running in staging environment
func (c *Config) IsStaging() bool {
	return c.Environment == "staging"
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		// For more complex parsing, consider using a proper CSV parser
		result := []string{}
		for _, item := range []string{value} {
			if item != "" {
				result = append(result, item)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}

	if c.JWT.SecretKey == "" || c.JWT.SecretKey == "your-secret-key" {
		return fmt.Errorf("JWT secret key must be set and not use default value")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}

	return nil
}
