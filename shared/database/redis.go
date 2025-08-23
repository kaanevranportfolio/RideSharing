package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rideshare-platform/shared/config"
	"github.com/rideshare-platform/shared/logger"
)

// RedisDB represents a Redis database connection
type RedisDB struct {
	Client *redis.Client
	config *config.DatabaseConfig
	logger *logger.Logger
}

// NewRedisDB creates a new Redis database connection
func NewRedisDB(cfg *config.DatabaseConfig, log *logger.Logger) (*RedisDB, error) {
	// Create Redis client options
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           0, // Use default DB
		PoolSize:     cfg.MaxOpenConns,
		MinIdleConns: cfg.MaxIdleConns,
		MaxConnAge:   time.Duration(cfg.ConnMaxLifetime) * time.Second,
		IdleTimeout:  time.Duration(cfg.ConnMaxIdleTime) * time.Second,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.WithFields(logger.Fields{
		"host": cfg.Host,
		"port": cfg.Port,
		"db":   0,
	}).Info("Connected to Redis database")

	return &RedisDB{
		Client: client,
		config: cfg,
		logger: log,
	}, nil
}

// Close closes the Redis connection
func (r *RedisDB) Close() error {
	if r.Client != nil {
		r.logger.Logger.Info("Closing Redis database connection")
		return r.Client.Close()
	}
	return nil
}

// Health checks the Redis health
func (r *RedisDB) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Stats returns Redis pool statistics
func (r *RedisDB) Stats() *redis.PoolStats {
	return r.Client.PoolStats()
}

// LogStats logs Redis connection pool statistics
func (r *RedisDB) LogStats(ctx context.Context) {
	stats := r.Client.PoolStats()
	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"hits":        stats.Hits,
		"misses":      stats.Misses,
		"timeouts":    stats.Timeouts,
		"total_conns": stats.TotalConns,
		"idle_conns":  stats.IdleConns,
		"stale_conns": stats.StaleConns,
	}).Info("Redis connection pool stats")
}

// RedisCache provides caching operations with logging
type RedisCache struct {
	client *redis.Client
	logger *logger.Logger
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(db *RedisDB, prefix string, logger *logger.Logger) *RedisCache {
	return &RedisCache{
		client: db.Client,
		logger: logger,
		prefix: prefix,
	}
}

// key adds prefix to the key
func (c *RedisCache) key(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + ":" + key
}

// Set sets a key-value pair with expiration
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	err := c.client.Set(ctx, c.key(key), value, expiration).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "SET", key, false, duration)
	return err
}

// Get gets a value by key
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.client.Get(ctx, c.key(key)).Result()
	duration := time.Since(start)

	hit := err == nil
	c.logger.LogCacheOperation(ctx, "GET", key, hit, duration)

	if err == redis.Nil {
		return "", nil // Key does not exist
	}
	return val, err
}

// GetBytes gets a value as bytes by key
func (c *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	val, err := c.client.Get(ctx, c.key(key)).Bytes()
	duration := time.Since(start)

	hit := err == nil
	c.logger.LogCacheOperation(ctx, "GET_BYTES", key, hit, duration)

	if err == redis.Nil {
		return nil, nil // Key does not exist
	}
	return val, err
}

// Del deletes keys
func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.key(key)
	}

	start := time.Now()
	err := c.client.Del(ctx, prefixedKeys...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "DEL", fmt.Sprintf("%v", keys), false, duration)
	return err
}

// Exists checks if keys exist
func (c *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.key(key)
	}

	start := time.Now()
	count, err := c.client.Exists(ctx, prefixedKeys...).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "EXISTS", fmt.Sprintf("%v", keys), count > 0, duration)
	return count, err
}

// Expire sets expiration for a key
func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	start := time.Now()
	err := c.client.Expire(ctx, c.key(key), expiration).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "EXPIRE", key, false, duration)
	return err
}

// TTL gets time to live for a key
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	ttl, err := c.client.TTL(ctx, c.key(key)).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "TTL", key, false, duration)
	return ttl, err
}

// Incr increments a key
func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	val, err := c.client.Incr(ctx, c.key(key)).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "INCR", key, false, duration)
	return val, err
}

// IncrBy increments a key by value
func (c *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	start := time.Now()
	val, err := c.client.IncrBy(ctx, c.key(key), value).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "INCRBY", key, false, duration)
	return val, err
}

// Decr decrements a key
func (c *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	val, err := c.client.Decr(ctx, c.key(key)).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "DECR", key, false, duration)
	return val, err
}

// DecrBy decrements a key by value
func (c *RedisCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	start := time.Now()
	val, err := c.client.DecrBy(ctx, c.key(key), value).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "DECRBY", key, false, duration)
	return val, err
}

// HSet sets hash field
func (c *RedisCache) HSet(ctx context.Context, key string, values ...interface{}) error {
	start := time.Now()
	err := c.client.HSet(ctx, c.key(key), values...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "HSET", key, false, duration)
	return err
}

// HGet gets hash field
func (c *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	start := time.Now()
	val, err := c.client.HGet(ctx, c.key(key), field).Result()
	duration := time.Since(start)

	hit := err == nil
	c.logger.LogCacheOperation(ctx, "HGET", key+":"+field, hit, duration)

	if err == redis.Nil {
		return "", nil // Field does not exist
	}
	return val, err
}

// HGetAll gets all hash fields
func (c *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	start := time.Now()
	val, err := c.client.HGetAll(ctx, c.key(key)).Result()
	duration := time.Since(start)

	hit := err == nil && len(val) > 0
	c.logger.LogCacheOperation(ctx, "HGETALL", key, hit, duration)
	return val, err
}

// HDel deletes hash fields
func (c *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	start := time.Now()
	err := c.client.HDel(ctx, c.key(key), fields...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "HDEL", key, false, duration)
	return err
}

// LPush pushes elements to the left of a list
func (c *RedisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	start := time.Now()
	err := c.client.LPush(ctx, c.key(key), values...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "LPUSH", key, false, duration)
	return err
}

// RPush pushes elements to the right of a list
func (c *RedisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
	start := time.Now()
	err := c.client.RPush(ctx, c.key(key), values...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "RPUSH", key, false, duration)
	return err
}

// LPop pops an element from the left of a list
func (c *RedisCache) LPop(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.client.LPop(ctx, c.key(key)).Result()
	duration := time.Since(start)

	hit := err == nil
	c.logger.LogCacheOperation(ctx, "LPOP", key, hit, duration)

	if err == redis.Nil {
		return "", nil // List is empty
	}
	return val, err
}

// RPop pops an element from the right of a list
func (c *RedisCache) RPop(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.client.RPop(ctx, c.key(key)).Result()
	duration := time.Since(start)

	hit := err == nil
	c.logger.LogCacheOperation(ctx, "RPOP", key, hit, duration)

	if err == redis.Nil {
		return "", nil // List is empty
	}
	return val, err
}

// LLen gets the length of a list
func (c *RedisCache) LLen(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	length, err := c.client.LLen(ctx, c.key(key)).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "LLEN", key, false, duration)
	return length, err
}

// SAdd adds members to a set
func (c *RedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	start := time.Now()
	err := c.client.SAdd(ctx, c.key(key), members...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "SADD", key, false, duration)
	return err
}

// SRem removes members from a set
func (c *RedisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	start := time.Now()
	err := c.client.SRem(ctx, c.key(key), members...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "SREM", key, false, duration)
	return err
}

// SMembers gets all members of a set
func (c *RedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	members, err := c.client.SMembers(ctx, c.key(key)).Result()
	duration := time.Since(start)

	hit := err == nil && len(members) > 0
	c.logger.LogCacheOperation(ctx, "SMEMBERS", key, hit, duration)
	return members, err
}

// SIsMember checks if a member exists in a set
func (c *RedisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	start := time.Now()
	exists, err := c.client.SIsMember(ctx, c.key(key), member).Result()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "SISMEMBER", key, exists, duration)
	return exists, err
}

// ZAdd adds members to a sorted set
func (c *RedisCache) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	start := time.Now()
	err := c.client.ZAdd(ctx, c.key(key), members...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "ZADD", key, false, duration)
	return err
}

// ZRange gets members from a sorted set by range
func (c *RedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	startTime := time.Now()
	members, err := c.client.ZRange(ctx, c.key(key), start, stop).Result()
	duration := time.Since(startTime)

	hit := err == nil && len(members) > 0
	c.logger.LogCacheOperation(ctx, "ZRANGE", key, hit, duration)
	return members, err
}

// ZRangeByScore gets members from a sorted set by score range
func (c *RedisCache) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	start := time.Now()
	members, err := c.client.ZRangeByScore(ctx, c.key(key), opt).Result()
	duration := time.Since(start)

	hit := err == nil && len(members) > 0
	c.logger.LogCacheOperation(ctx, "ZRANGEBYSCORE", key, hit, duration)
	return members, err
}

// ZRem removes members from a sorted set
func (c *RedisCache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	start := time.Now()
	err := c.client.ZRem(ctx, c.key(key), members...).Err()
	duration := time.Since(start)

	c.logger.LogCacheOperation(ctx, "ZREM", key, false, duration)
	return err
}

// Pipeline creates a Redis pipeline
func (c *RedisCache) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline creates a Redis transaction pipeline
func (c *RedisCache) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}
