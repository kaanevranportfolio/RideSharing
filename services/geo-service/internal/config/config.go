package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rideshare-platform/shared/config"
)

// Config holds all configuration for the geo service
type Config struct {
	// Service configuration
	ServiceName     string `json:"service_name"`
	Environment     string `json:"environment"`
	LogLevel        string `json:"log_level"`
	GRPCPort        int    `json:"grpc_port"`
	HTTPPort        int    `json:"http_port"`
	ShutdownTimeout int    `json:"shutdown_timeout"`

	// Database configuration
	Database config.DatabaseConfig `json:"database"`

	// Redis configuration
	Redis *config.RedisConfig `json:"redis"`

	// Geospatial configuration
	Geospatial GeospatialConfig `json:"geospatial"`

	// Cache configuration
	Cache CacheConfig `json:"cache"`
}

// GeospatialConfig holds geospatial-specific configuration
type GeospatialConfig struct {
	// Default calculation method for distance
	DefaultDistanceMethod string `json:"default_distance_method"`

	// Maximum search radius in kilometers
	MaxSearchRadiusKm float64 `json:"max_search_radius_km"`

	// Default geohash precision
	DefaultGeohashPrecision int `json:"default_geohash_precision"`

	// Maximum number of nearby drivers to return
	MaxNearbyDrivers int `json:"max_nearby_drivers"`

	// Location update frequency in seconds
	LocationUpdateFrequency int `json:"location_update_frequency"`

	// Driver location TTL in seconds (how long to keep location data)
	DriverLocationTTL int `json:"driver_location_ttl"`

	// Route optimization settings
	RouteOptimization RouteOptimizationConfig `json:"route_optimization"`
}

// RouteOptimizationConfig holds route optimization settings
type RouteOptimizationConfig struct {
	// Maximum waypoints allowed in a single optimization request
	MaxWaypoints int `json:"max_waypoints"`

	// Default vehicle speed in km/h for different vehicle types
	DefaultSpeeds map[string]float64 `json:"default_speeds"`

	// Traffic factor multipliers for different times of day
	TrafficFactors map[string]float64 `json:"traffic_factors"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	// Distance calculation cache TTL in seconds
	DistanceCacheTTL int `json:"distance_cache_ttl"`

	// ETA calculation cache TTL in seconds
	ETACacheTTL int `json:"eta_cache_ttl"`

	// Route cache TTL in seconds
	RouteCacheTTL int `json:"route_cache_ttl"`

	// Enable/disable caching
	EnableCaching bool `json:"enable_caching"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName:     getEnv("SERVICE_NAME", "geo-service"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		GRPCPort:        getEnvInt("GRPC_PORT", 50053),
		HTTPPort:        getEnvInt("HTTP_PORT", 8053),
		ShutdownTimeout: getEnvInt("SHUTDOWN_TIMEOUT", 30),
	}

	// Load database configuration
	cfg.Database = config.DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 27017),
		Database:        getEnv("DB_NAME", "rideshare_geo"),
		Username:        getEnv("DB_USERNAME", ""),
		Password:        getEnv("DB_PASSWORD", ""),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 3600)) * time.Second,
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME", 900)) * time.Second,
	}

	// Load Redis configuration
	cfg.Redis = &config.RedisConfig{
		Host:         getEnv("REDIS_HOST", "localhost"),
		Port:         getEnvInt("REDIS_PORT", 6379),
		Password:     getEnv("REDIS_PASSWORD", ""),
		Database:     getEnvInt("REDIS_DATABASE", 0),
		PoolSize:     getEnvInt("REDIS_POOL_SIZE", 100),
		MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 10),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  5 * time.Minute,
	}

	// Load geospatial configuration
	cfg.Geospatial = GeospatialConfig{
		DefaultDistanceMethod:   getEnv("GEO_DEFAULT_DISTANCE_METHOD", "haversine"),
		MaxSearchRadiusKm:       getEnvFloat("GEO_MAX_SEARCH_RADIUS_KM", 50.0),
		DefaultGeohashPrecision: getEnvInt("GEO_DEFAULT_GEOHASH_PRECISION", 7),
		MaxNearbyDrivers:        getEnvInt("GEO_MAX_NEARBY_DRIVERS", 100),
		LocationUpdateFrequency: getEnvInt("GEO_LOCATION_UPDATE_FREQUENCY", 30),
		DriverLocationTTL:       getEnvInt("GEO_DRIVER_LOCATION_TTL", 300),
		RouteOptimization: RouteOptimizationConfig{
			MaxWaypoints: getEnvInt("GEO_MAX_WAYPOINTS", 25),
			DefaultSpeeds: map[string]float64{
				"car":     50.0, // km/h
				"bike":    20.0,
				"walking": 5.0,
			},
			TrafficFactors: map[string]float64{
				"rush_hour":  1.5,
				"normal":     1.0,
				"late_night": 0.8,
			},
		},
	}

	// Load cache configuration
	cfg.Cache = CacheConfig{
		DistanceCacheTTL: getEnvInt("CACHE_DISTANCE_TTL", 3600),
		ETACacheTTL:      getEnvInt("CACHE_ETA_TTL", 300),
		RouteCacheTTL:    getEnvInt("CACHE_ROUTE_TTL", 1800),
		EnableCaching:    getEnvBool("CACHE_ENABLE", true),
	}

	return cfg, nil
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetMongoDBConnectionString returns the MongoDB connection string
func (c *Config) GetMongoDBConnectionString() string {
	if c.Database.Username != "" && c.Database.Password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Database)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.Database)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.GRPCPort <= 0 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid GRPC port: %d", c.GRPCPort)
	}

	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTPPort)
	}

	if c.Geospatial.MaxSearchRadiusKm <= 0 {
		return fmt.Errorf("invalid max search radius: %f", c.Geospatial.MaxSearchRadiusKm)
	}

	if c.Geospatial.DefaultGeohashPrecision < 1 || c.Geospatial.DefaultGeohashPrecision > 12 {
		return fmt.Errorf("invalid geohash precision: %d", c.Geospatial.DefaultGeohashPrecision)
	}

	return nil
}
