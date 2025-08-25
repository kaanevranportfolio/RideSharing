package tests

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/cache"
	"github.com/rideshare-platform/shared/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuite provides a comprehensive testing framework for the rideshare platform
type TestSuite struct {
	PostgresDB *database.PostgresDB
	MongoDB    *database.MongoDB
	RedisCache *cache.RedisCache
	MemCache   *cache.MemoryCache
}

// SetupTestSuite initializes test infrastructure
func SetupTestSuite(t *testing.T) *TestSuite {
	// Setup test databases
	postgresDB, err := database.NewPostgresDB(&database.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		Database: "rideshare_test",
		SSLMode:  "disable",
	})
	require.NoError(t, err)

	mongoDB, err := database.NewMongoDB(&database.MongoConfig{
		URI:      "mongodb://localhost:27017",
		Database: "rideshare_test",
	})
	require.NoError(t, err)

	redisClient, err := database.NewRedisClient(&database.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for tests
	})
	require.NoError(t, err)

	// Setup caches
	redisCache := cache.NewRedisCache(redisClient, "test")
	memCache := cache.NewMemoryCache(time.Minute * 5)

	return &TestSuite{
		PostgresDB: postgresDB,
		MongoDB:    mongoDB,
		RedisCache: redisCache,
		MemCache:   memCache,
	}
}

// TeardownTestSuite cleans up test infrastructure
func (ts *TestSuite) TeardownTestSuite(t *testing.T) {
	// Clean up test data
	ctx := context.Background()

	// Clear Redis test data
	ts.RedisCache.InvalidatePattern(ctx, "test:*")

	// Clean up test databases
	// Note: In a real scenario, you might want to use transactions or separate test schemas
	ts.PostgresDB.Close()
	ts.MongoDB.Close()
}

// TestCachePerformance tests cache performance characteristics
func TestCachePerformance(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)

	ctx := context.Background()

	// Test data
	testData := map[string]interface{}{
		"small":  "test value",
		"medium": make([]byte, 1024),      // 1KB
		"large":  make([]byte, 1024*1024), // 1MB
	}

	t.Run("Memory Cache Performance", func(t *testing.T) {
		for size, data := range testData {
			t.Run(size, func(t *testing.T) {
				key := "perf_test_" + size

				// Measure set performance
				start := time.Now()
				err := ts.MemCache.Set(ctx, key, data, time.Minute)
				setDuration := time.Since(start)
				require.NoError(t, err)

				// Measure get performance
				start = time.Now()
				var result interface{}
				err = ts.MemCache.Get(ctx, key, &result)
				getDuration := time.Since(start)
				require.NoError(t, err)

				t.Logf("Memory Cache %s - Set: %v, Get: %v", size, setDuration, getDuration)

				// Performance assertions
				assert.Less(t, setDuration, time.Millisecond*10, "Set should be fast")
				assert.Less(t, getDuration, time.Millisecond*5, "Get should be very fast")
			})
		}
	})

	t.Run("Redis Cache Performance", func(t *testing.T) {
		for size, data := range testData {
			t.Run(size, func(t *testing.T) {
				key := "perf_test_" + size

				// Measure set performance
				start := time.Now()
				err := ts.RedisCache.Set(ctx, key, data, time.Minute)
				setDuration := time.Since(start)
				require.NoError(t, err)

				// Measure get performance
				start = time.Now()
				var result interface{}
				err = ts.RedisCache.Get(ctx, key, &result)
				getDuration := time.Since(start)
				require.NoError(t, err)

				t.Logf("Redis Cache %s - Set: %v, Get: %v", size, setDuration, getDuration)

				// Performance assertions (Redis is network-based, so higher thresholds)
				assert.Less(t, setDuration, time.Millisecond*50, "Set should be reasonably fast")
				assert.Less(t, getDuration, time.Millisecond*20, "Get should be fast")
			})
		}
	})
}

// TestDatabasePerformance tests database performance
func TestDatabasePerformance(t *testing.T) {
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)

	t.Run("PostgreSQL Connection Pool", func(t *testing.T) {
		ctx := context.Background()

		// Test concurrent connections
		concurrency := 10
		queries := 100

		start := time.Now()

		errChan := make(chan error, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				for j := 0; j < queries; j++ {
					err := ts.PostgresDB.Ping(ctx)
					if err != nil {
						errChan <- err
						return
					}
				}
				errChan <- nil
			}()
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			err := <-errChan
			require.NoError(t, err)
		}

		duration := time.Since(start)
		avgTimePerQuery := duration / time.Duration(concurrency*queries)

		t.Logf("PostgreSQL: %d concurrent connections, %d queries each, avg: %v per query",
			concurrency, queries, avgTimePerQuery)

		assert.Less(t, avgTimePerQuery, time.Millisecond*10, "Database queries should be fast")
	})
}

// BenchmarkCache provides benchmarks for caching operations
func BenchmarkCache(b *testing.B) {
	ts := SetupTestSuite(&testing.T{})
	defer ts.TeardownTestSuite(&testing.T{})

	ctx := context.Background()
	testData := "benchmark test data"

	b.Run("MemoryCache", func(b *testing.B) {
		b.Run("Set", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key := "bench_mem_set_" + string(rune(i))
				ts.MemCache.Set(ctx, key, testData, time.Minute)
			}
		})

		b.Run("Get", func(b *testing.B) {
			// Pre-populate cache
			for i := 0; i < 1000; i++ {
				key := "bench_mem_get_" + string(rune(i))
				ts.MemCache.Set(ctx, key, testData, time.Minute)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := "bench_mem_get_" + string(rune(i%1000))
				var result string
				ts.MemCache.Get(ctx, key, &result)
			}
		})
	})

	b.Run("RedisCache", func(b *testing.B) {
		b.Run("Set", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key := "bench_redis_set_" + string(rune(i))
				ts.RedisCache.Set(ctx, key, testData, time.Minute)
			}
		})

		b.Run("Get", func(b *testing.B) {
			// Pre-populate cache
			for i := 0; i < 1000; i++ {
				key := "bench_redis_get_" + string(rune(i))
				ts.RedisCache.Set(ctx, key, testData, time.Minute)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := "bench_redis_get_" + string(rune(i%1000))
				var result string
				ts.RedisCache.Get(ctx, key, &result)
			}
		})
	})
}
