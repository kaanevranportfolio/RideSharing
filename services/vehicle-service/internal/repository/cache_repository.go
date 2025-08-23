package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// CacheRepository handles caching operations for vehicles
type CacheRepository struct {
	cache  *database.RedisCache
	logger *logger.Logger
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(redisDB *database.RedisDB, log *logger.Logger) *CacheRepository {
	cache := database.NewRedisCache(redisDB, "vehicle-service", log)
	return &CacheRepository{
		cache:  cache,
		logger: log,
	}
}

// CacheVehicle caches a vehicle object
func (r *CacheRepository) CacheVehicle(ctx context.Context, vehicle *models.Vehicle, ttl time.Duration) error {
	key := fmt.Sprintf("vehicle:%s", vehicle.ID)

	data, err := json.Marshal(vehicle)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": vehicle.ID,
		}).Error("Failed to marshal vehicle for caching")
		return fmt.Errorf("failed to marshal vehicle: %w", err)
	}

	if err := r.cache.Set(ctx, key, data, ttl); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": vehicle.ID,
			"key":        key,
		}).Error("Failed to cache vehicle")
		return fmt.Errorf("failed to cache vehicle: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicle.ID,
		"key":        key,
		"ttl":        ttl,
	}).Debug("Vehicle cached successfully")

	return nil
}

// GetCachedVehicle retrieves a cached vehicle
func (r *CacheRepository) GetCachedVehicle(ctx context.Context, vehicleID string) (*models.Vehicle, error) {
	key := fmt.Sprintf("vehicle:%s", vehicleID)

	data, err := r.cache.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached vehicle: %w", err)
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	var vehicle models.Vehicle
	if err := json.Unmarshal(data, &vehicle); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": vehicleID,
			"key":        key,
		}).Error("Failed to unmarshal cached vehicle")
		return nil, fmt.Errorf("failed to unmarshal cached vehicle: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicleID,
		"key":        key,
	}).Debug("Vehicle retrieved from cache")

	return &vehicle, nil
}

// InvalidateVehicle removes a vehicle from cache
func (r *CacheRepository) InvalidateVehicle(ctx context.Context, vehicleID string) error {
	key := fmt.Sprintf("vehicle:%s", vehicleID)

	if err := r.cache.Del(ctx, key); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": vehicleID,
			"key":        key,
		}).Error("Failed to invalidate vehicle cache")
		return fmt.Errorf("failed to invalidate vehicle cache: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicleID,
		"key":        key,
	}).Debug("Vehicle cache invalidated")

	return nil
}

// CacheDriverVehicles caches vehicles for a driver
func (r *CacheRepository) CacheDriverVehicles(ctx context.Context, driverID string, vehicles []*models.Vehicle, ttl time.Duration) error {
	key := fmt.Sprintf("driver_vehicles:%s", driverID)

	data, err := json.Marshal(vehicles)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
		}).Error("Failed to marshal driver vehicles for caching")
		return fmt.Errorf("failed to marshal driver vehicles: %w", err)
	}

	if err := r.cache.Set(ctx, key, data, ttl); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to cache driver vehicles")
		return fmt.Errorf("failed to cache driver vehicles: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":     driverID,
		"key":           key,
		"vehicle_count": len(vehicles),
		"ttl":           ttl,
	}).Debug("Driver vehicles cached successfully")

	return nil
}

// GetCachedDriverVehicles retrieves cached vehicles for a driver
func (r *CacheRepository) GetCachedDriverVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	key := fmt.Sprintf("driver_vehicles:%s", driverID)

	data, err := r.cache.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached driver vehicles: %w", err)
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	var vehicles []*models.Vehicle
	if err := json.Unmarshal(data, &vehicles); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to unmarshal cached driver vehicles")
		return nil, fmt.Errorf("failed to unmarshal cached driver vehicles: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":     driverID,
		"key":           key,
		"vehicle_count": len(vehicles),
	}).Debug("Driver vehicles retrieved from cache")

	return vehicles, nil
}

// InvalidateDriverVehicles removes driver vehicles from cache
func (r *CacheRepository) InvalidateDriverVehicles(ctx context.Context, driverID string) error {
	key := fmt.Sprintf("driver_vehicles:%s", driverID)

	if err := r.cache.Del(ctx, key); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to invalidate driver vehicles cache")
		return fmt.Errorf("failed to invalidate driver vehicles cache: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id": driverID,
		"key":       key,
	}).Debug("Driver vehicles cache invalidated")

	return nil
}

// CacheAvailableVehicles caches available vehicles for a driver
func (r *CacheRepository) CacheAvailableVehicles(ctx context.Context, driverID string, vehicles []*models.Vehicle, ttl time.Duration) error {
	key := fmt.Sprintf("available_vehicles:%s", driverID)

	data, err := json.Marshal(vehicles)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
		}).Error("Failed to marshal available vehicles for caching")
		return fmt.Errorf("failed to marshal available vehicles: %w", err)
	}

	if err := r.cache.Set(ctx, key, data, ttl); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to cache available vehicles")
		return fmt.Errorf("failed to cache available vehicles: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":     driverID,
		"key":           key,
		"vehicle_count": len(vehicles),
		"ttl":           ttl,
	}).Debug("Available vehicles cached successfully")

	return nil
}

// GetCachedAvailableVehicles retrieves cached available vehicles for a driver
func (r *CacheRepository) GetCachedAvailableVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	key := fmt.Sprintf("available_vehicles:%s", driverID)

	data, err := r.cache.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached available vehicles: %w", err)
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	var vehicles []*models.Vehicle
	if err := json.Unmarshal(data, &vehicles); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to unmarshal cached available vehicles")
		return nil, fmt.Errorf("failed to unmarshal cached available vehicles: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":     driverID,
		"key":           key,
		"vehicle_count": len(vehicles),
	}).Debug("Available vehicles retrieved from cache")

	return vehicles, nil
}

// InvalidateAvailableVehicles removes available vehicles from cache
func (r *CacheRepository) InvalidateAvailableVehicles(ctx context.Context, driverID string) error {
	key := fmt.Sprintf("available_vehicles:%s", driverID)

	if err := r.cache.Del(ctx, key); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
			"key":       key,
		}).Error("Failed to invalidate available vehicles cache")
		return fmt.Errorf("failed to invalidate available vehicles cache: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id": driverID,
		"key":       key,
	}).Debug("Available vehicles cache invalidated")

	return nil
}

// CacheVehiclesByType caches vehicles by type
func (r *CacheRepository) CacheVehiclesByType(ctx context.Context, vehicleType string, vehicles []*models.Vehicle, ttl time.Duration) error {
	key := fmt.Sprintf("vehicles_by_type:%s", vehicleType)

	data, err := json.Marshal(vehicles)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_type": vehicleType,
		}).Error("Failed to marshal vehicles by type for caching")
		return fmt.Errorf("failed to marshal vehicles by type: %w", err)
	}

	if err := r.cache.Set(ctx, key, data, ttl); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_type": vehicleType,
			"key":          key,
		}).Error("Failed to cache vehicles by type")
		return fmt.Errorf("failed to cache vehicles by type: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_type":  vehicleType,
		"key":           key,
		"vehicle_count": len(vehicles),
		"ttl":           ttl,
	}).Debug("Vehicles by type cached successfully")

	return nil
}

// GetCachedVehiclesByType retrieves cached vehicles by type
func (r *CacheRepository) GetCachedVehiclesByType(ctx context.Context, vehicleType string) ([]*models.Vehicle, error) {
	key := fmt.Sprintf("vehicles_by_type:%s", vehicleType)

	data, err := r.cache.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached vehicles by type: %w", err)
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	var vehicles []*models.Vehicle
	if err := json.Unmarshal(data, &vehicles); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_type": vehicleType,
			"key":          key,
		}).Error("Failed to unmarshal cached vehicles by type")
		return nil, fmt.Errorf("failed to unmarshal cached vehicles by type: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_type":  vehicleType,
		"key":           key,
		"vehicle_count": len(vehicles),
	}).Debug("Vehicles by type retrieved from cache")

	return vehicles, nil
}

// InvalidateVehiclesByType removes vehicles by type from cache
func (r *CacheRepository) InvalidateVehiclesByType(ctx context.Context, vehicleType string) error {
	key := fmt.Sprintf("vehicles_by_type:%s", vehicleType)

	if err := r.cache.Del(ctx, key); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_type": vehicleType,
			"key":          key,
		}).Error("Failed to invalidate vehicles by type cache")
		return fmt.Errorf("failed to invalidate vehicles by type cache: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_type": vehicleType,
		"key":          key,
	}).Debug("Vehicles by type cache invalidated")

	return nil
}

// CacheVehicleStats caches vehicle statistics
func (r *CacheRepository) CacheVehicleStats(ctx context.Context, stats map[string]interface{}, ttl time.Duration) error {
	key := "vehicle_stats"

	data, err := json.Marshal(stats)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to marshal vehicle stats for caching")
		return fmt.Errorf("failed to marshal vehicle stats: %w", err)
	}

	if err := r.cache.Set(ctx, key, data, ttl); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to cache vehicle stats")
		return fmt.Errorf("failed to cache vehicle stats: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key": key,
		"ttl": ttl,
	}).Debug("Vehicle stats cached successfully")

	return nil
}

// GetCachedVehicleStats retrieves cached vehicle statistics
func (r *CacheRepository) GetCachedVehicleStats(ctx context.Context) (map[string]interface{}, error) {
	key := "vehicle_stats"

	data, err := r.cache.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached vehicle stats: %w", err)
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(data, &stats); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to unmarshal cached vehicle stats")
		return nil, fmt.Errorf("failed to unmarshal cached vehicle stats: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key": key,
	}).Debug("Vehicle stats retrieved from cache")

	return stats, nil
}

// InvalidateVehicleStats removes vehicle statistics from cache
func (r *CacheRepository) InvalidateVehicleStats(ctx context.Context) error {
	key := "vehicle_stats"

	if err := r.cache.Del(ctx, key); err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to invalidate vehicle stats cache")
		return fmt.Errorf("failed to invalidate vehicle stats cache: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key": key,
	}).Debug("Vehicle stats cache invalidated")

	return nil
}

// InvalidateAllVehicleCaches removes all vehicle-related caches
func (r *CacheRepository) InvalidateAllVehicleCaches(ctx context.Context) error {
	// This is a simplified implementation - in production, you might want to use Redis SCAN
	// to find all keys with vehicle-related prefixes and delete them

	r.logger.WithContext(ctx).Info("Invalidating all vehicle caches")

	// For now, we'll just invalidate the main stats cache
	// Individual vehicle and driver caches will expire naturally
	return r.InvalidateVehicleStats(ctx)
}
