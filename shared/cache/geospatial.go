package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// GeospatialCache handles location-based caching for rideshare platform
type GeospatialCache struct {
	client *redis.Client
	prefix string
}

// NewGeospatialCache creates a new geospatial cache
func NewGeospatialCache(client *redis.Client, prefix string) *GeospatialCache {
	return &GeospatialCache{
		client: client,
		prefix: prefix,
	}
}

// Location represents a geographic location
type Location struct {
	ID        string                 `json:"id"`
	Latitude  float64                `json:"latitude"`
	Longitude float64                `json:"longitude"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AddLocation adds a location to the geospatial index
func (c *GeospatialCache) AddLocation(ctx context.Context, key string, location Location) error {
	geoKey := c.getGeoKey(key)

	// Add to geospatial index
	err := c.client.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      location.ID,
		Longitude: location.Longitude,
		Latitude:  location.Latitude,
	}).Err()

	if err != nil {
		return fmt.Errorf("geospatial cache add error: %w", err)
	}

	// Set expiration on the geo key
	c.client.Expire(ctx, geoKey, time.Hour*24)

	// Store additional metadata if provided
	if len(location.Metadata) > 0 {
		metaKey := c.getMetaKey(key, location.ID)
		err = c.client.HMSet(ctx, metaKey, location.Metadata).Err()
		if err != nil {
			return fmt.Errorf("geospatial metadata cache error: %w", err)
		}
		c.client.Expire(ctx, metaKey, time.Hour*24)
	}

	return nil
}

// FindNearby finds locations within a radius
func (c *GeospatialCache) FindNearby(ctx context.Context, key string, lat, lon, radiusKm float64, limit int) ([]Location, error) {
	geoKey := c.getGeoKey(key)

	// Query nearby locations
	result, err := c.client.GeoRadius(ctx, geoKey, lon, lat, &redis.GeoRadiusQuery{
		Radius:    radiusKm,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
		Count:     limit,
		Sort:      "ASC", // Closest first
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("geospatial nearby query error: %w", err)
	}

	locations := make([]Location, 0, len(result))
	for _, item := range result {
		location := Location{
			ID:        item.Name,
			Longitude: item.Longitude,
			Latitude:  item.Latitude,
		}

		// Get metadata if available
		metaKey := c.getMetaKey(key, item.Name)
		metadata, err := c.client.HGetAll(ctx, metaKey).Result()
		if err == nil && len(metadata) > 0 {
			location.Metadata = make(map[string]interface{})
			for k, v := range metadata {
				location.Metadata[k] = v
			}
		}

		locations = append(locations, location)
	}

	return locations, nil
}

// RemoveLocation removes a location from the geospatial index
func (c *GeospatialCache) RemoveLocation(ctx context.Context, key string, locationID string) error {
	geoKey := c.getGeoKey(key)

	// Remove from geospatial index
	err := c.client.ZRem(ctx, geoKey, locationID).Err()
	if err != nil {
		return fmt.Errorf("geospatial remove error: %w", err)
	}

	// Remove metadata
	metaKey := c.getMetaKey(key, locationID)
	c.client.Del(ctx, metaKey)

	return nil
}

// UpdateLocationMetadata updates metadata for a location
func (c *GeospatialCache) UpdateLocationMetadata(ctx context.Context, key string, locationID string, metadata map[string]interface{}) error {
	metaKey := c.getMetaKey(key, locationID)

	err := c.client.HMSet(ctx, metaKey, metadata).Err()
	if err != nil {
		return fmt.Errorf("geospatial metadata update error: %w", err)
	}

	c.client.Expire(ctx, metaKey, time.Hour*24)
	return nil
}

// GetDistance calculates distance between two locations in the cache
func (c *GeospatialCache) GetDistance(ctx context.Context, key string, location1, location2 string) (float64, error) {
	geoKey := c.getGeoKey(key)

	result, err := c.client.GeoDist(ctx, geoKey, location1, location2, "km").Result()
	if err != nil {
		return 0, fmt.Errorf("geospatial distance error: %w", err)
	}

	return result, nil
}

// ClearArea removes all locations in a geographic area
func (c *GeospatialCache) ClearArea(ctx context.Context, key string, lat, lon, radiusKm float64) error {
	geoKey := c.getGeoKey(key)

	// Find all locations in the area
	result, err := c.client.GeoRadius(ctx, geoKey, lon, lat, &redis.GeoRadiusQuery{
		Radius: radiusKm,
		Unit:   "km",
	}).Result()

	if err != nil {
		return fmt.Errorf("geospatial clear area query error: %w", err)
	}

	// Remove each location
	for _, item := range result {
		c.RemoveLocation(ctx, key, item.Name)
	}

	return nil
}

func (c *GeospatialCache) getGeoKey(key string) string {
	return fmt.Sprintf("%s:geo:%s", c.prefix, key)
}

func (c *GeospatialCache) getMetaKey(key, locationID string) string {
	return fmt.Sprintf("%s:meta:%s:%s", c.prefix, key, locationID)
}

// Cache invalidation patterns for rideshare platform
type CacheInvalidator struct {
	cache Cache
}

// NewCacheInvalidator creates a cache invalidator
func NewCacheInvalidator(cache Cache) *CacheInvalidator {
	return &CacheInvalidator{cache: cache}
}

// InvalidateUser invalidates all user-related cache entries
func (c *CacheInvalidator) InvalidateUser(ctx context.Context, userID string) error {
	patterns := []string{
		fmt.Sprintf("user:%s:*", userID),
		fmt.Sprintf("auth:%s:*", userID),
		fmt.Sprintf("profile:%s", userID),
	}

	for _, pattern := range patterns {
		if err := c.cache.InvalidatePattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}

// InvalidateVehicle invalidates all vehicle-related cache entries
func (c *CacheInvalidator) InvalidateVehicle(ctx context.Context, vehicleID string) error {
	patterns := []string{
		fmt.Sprintf("vehicle:%s:*", vehicleID),
		fmt.Sprintf("location:%s", vehicleID),
	}

	for _, pattern := range patterns {
		if err := c.cache.InvalidatePattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}

// InvalidateTrip invalidates all trip-related cache entries
func (c *CacheInvalidator) InvalidateTrip(ctx context.Context, tripID string) error {
	patterns := []string{
		fmt.Sprintf("trip:%s:*", tripID),
		fmt.Sprintf("pricing:%s", tripID),
		fmt.Sprintf("route:%s", tripID),
	}

	for _, pattern := range patterns {
		if err := c.cache.InvalidatePattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}
