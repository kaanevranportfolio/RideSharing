package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache with TTL
type MemoryCache struct {
	items map[string]*cacheItem
	mutex sync.RWMutex
	ttl   time.Duration
}

type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*cacheItem),
		ttl:   defaultTTL,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from memory cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mutex.RLock()
	item, exists := c.items[key]
	c.mutex.RUnlock()

	if !exists {
		return ErrCacheMiss
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		c.mutex.Lock()
		delete(c.items, key)
		c.mutex.Unlock()
		return ErrCacheMiss
	}

	if err := json.Unmarshal(item.data, dest); err != nil {
		return fmt.Errorf("memory cache unmarshal error: %w", err)
	}

	return nil
}

// Set stores a value in memory cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("memory cache marshal error: %w", err)
	}

	if expiration == 0 {
		expiration = c.ttl
	}

	c.mutex.Lock()
	c.items[key] = &cacheItem{
		data:      data,
		expiresAt: time.Now().Add(expiration),
	}
	c.mutex.Unlock()

	return nil
}

// Delete removes a key from memory cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()
	return nil
}

// Exists checks if a key exists in memory cache
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mutex.RLock()
	item, exists := c.items[key]
	c.mutex.RUnlock()

	if !exists {
		return false, nil
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		c.mutex.Lock()
		delete(c.items, key)
		c.mutex.Unlock()
		return false, nil
	}

	return true, nil
}

// InvalidatePattern removes all keys matching a pattern (simple prefix matching)
func (c *MemoryCache) InvalidatePattern(ctx context.Context, pattern string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	keysToDelete := make([]string, 0)
	for key := range c.items {
		// Simple pattern matching - could be enhanced with regex
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.items, key)
	}

	return nil
}

// cleanup removes expired items periodically
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mutex.Unlock()
	}
}
