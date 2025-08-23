package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
)

// CacheRepository handles caching operations using Redis
type CacheRepository struct {
	cache  *database.RedisCache
	logger *logger.Logger
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(redis *database.RedisDB, log *logger.Logger) *CacheRepository {
	cache := database.NewRedisCache(redis, "geo-service", log)
	return &CacheRepository{
		cache:  cache,
		logger: log,
	}
}

// Set stores a value in cache with expiration
func (r *CacheRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to marshal value for cache")
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.cache.Set(ctx, key, string(data), expiration)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set value in cache")
		return fmt.Errorf("failed to set cache value: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key":        key,
		"expiration": expiration,
	}).Debug("Value cached successfully")

	return nil
}

// Get retrieves a value from cache
func (r *CacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := r.cache.Get(ctx, key)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Debug("Cache miss")
		return nil, fmt.Errorf("cache miss: %w", err)
	}

	if data == "" {
		return nil, fmt.Errorf("cache miss: key not found")
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key": key,
	}).Debug("Cache hit")

	return []byte(data), nil
}

// GetAndUnmarshal retrieves and unmarshals a value from cache
func (r *CacheRepository) GetAndUnmarshal(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to unmarshal cached value")
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	err := r.cache.Del(ctx, key)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to delete cache key")
		return fmt.Errorf("failed to delete cache key: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"key": key,
	}).Debug("Cache key deleted")

	return nil
}

// Exists checks if a key exists in cache
func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.cache.Exists(ctx, key)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to check cache key existence")
		return false, fmt.Errorf("failed to check cache key existence: %w", err)
	}

	return count > 0, nil
}

// SetExpiration sets expiration for an existing key
func (r *CacheRepository) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	err := r.cache.Expire(ctx, key, expiration)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set expiration for cache key")
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// GetTTL gets the time-to-live for a key
func (r *CacheRepository) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.cache.TTL(ctx, key)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"key": key,
		}).Error("Failed to get TTL for cache key")
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}
