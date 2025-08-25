package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache interface defines caching operations
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	InvalidatePattern(ctx context.Context, pattern string) error
}

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.getFullKey(key)

	val, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

// Set stores a value in cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := c.getFullKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

// Delete removes a key from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.getFullKey(key)

	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.getFullKey(key)

	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists error: %w", err)
	}

	return count > 0, nil
}

// InvalidatePattern removes all keys matching a pattern
func (c *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := c.getFullKey(pattern)

	keys, err := c.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		return fmt.Errorf("cache keys error: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("cache invalidate error: %w", err)
		}
	}

	return nil
}

func (c *RedisCache) getFullKey(key string) string {
	if c.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// ErrCacheMiss is returned when a key is not found in cache
var ErrCacheMiss = fmt.Errorf("cache miss")

// Multi-level cache implementation
type MultiLevelCache struct {
	l1 Cache // In-memory cache
	l2 Cache // Redis cache
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(l1, l2 Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1: l1,
		l2: l2,
	}
}

// Get tries L1 first, then L2
func (c *MultiLevelCache) Get(ctx context.Context, key string, dest interface{}) error {
	// Try L1 first
	if err := c.l1.Get(ctx, key, dest); err == nil {
		return nil
	}

	// Try L2
	if err := c.l2.Get(ctx, key, dest); err != nil {
		return err
	}

	// Store in L1 for next time
	c.l1.Set(ctx, key, dest, time.Minute*5)
	return nil
}

// Set stores in both levels
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Set in L1 with shorter expiration
	l1Expiration := expiration
	if expiration > time.Minute*10 {
		l1Expiration = time.Minute * 10
	}
	c.l1.Set(ctx, key, value, l1Expiration)

	// Set in L2 with full expiration
	return c.l2.Set(ctx, key, value, expiration)
}

// Delete removes from both levels
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	c.l1.Delete(ctx, key)
	return c.l2.Delete(ctx, key)
}

// Exists checks L1 first, then L2
func (c *MultiLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	if exists, _ := c.l1.Exists(ctx, key); exists {
		return true, nil
	}
	return c.l2.Exists(ctx, key)
}

// InvalidatePattern removes from both levels
func (c *MultiLevelCache) InvalidatePattern(ctx context.Context, pattern string) error {
	c.l1.InvalidatePattern(ctx, pattern)
	return c.l2.InvalidatePattern(ctx, pattern)
}
