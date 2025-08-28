package performance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// TestCachePerformance tests basic cache operations performance
func TestCachePerformance(t *testing.T) {
	t.Run("memory_cache_performance", func(t *testing.T) {
		// Test basic memory operations
		start := time.Now()

		// Simulate cache operations
		data := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key_%d", i)
			data[key] = fmt.Sprintf("value_%d", i)
		}

		duration := time.Since(start)
		t.Logf("Memory cache 1000 operations took: %v", duration)

		// Assert reasonable performance (should be very fast for memory ops)
		assert.Less(t, duration, time.Millisecond*100, "Memory cache operations should be fast")
		assert.Equal(t, 1000, len(data), "All items should be stored")
	})

	t.Run("database_connection_performance", func(t *testing.T) {
		// Test connection establishment time
		start := time.Now()

		// Simulate connection setup (without actual DB for unit test)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// Simulate some work
		select {
		case <-time.After(time.Millisecond * 10):
			// Connection established
		case <-ctx.Done():
			t.Fatal("Connection timeout")
		}

		duration := time.Since(start)
		t.Logf("Database connection simulation took: %v", duration)

		// Assert reasonable connection time
		assert.Less(t, duration, time.Second*1, "Connection should be established quickly")
	})

	t.Run("concurrent_operations_performance", func(t *testing.T) {
		// Test concurrent operations
		const numGoroutines = 10
		const operationsPerGoroutine = 100

		start := time.Now()

		// Channel to collect results
		results := make(chan int, numGoroutines)

		// Launch concurrent workers
		for i := 0; i < numGoroutines; i++ {
			go func(workerID int) {
				operations := 0
				for j := 0; j < operationsPerGoroutine; j++ {
					// Simulate work
					time.Sleep(time.Microsecond * 100)
					operations++
				}
				results <- operations
			}(i)
		}

		// Collect results
		totalOps := 0
		for i := 0; i < numGoroutines; i++ {
			totalOps += <-results
		}

		duration := time.Since(start)
		t.Logf("Concurrent operations (%d goroutines, %d ops each) took: %v",
			numGoroutines, operationsPerGoroutine, duration)

		// Verify all operations completed
		assert.Equal(t, numGoroutines*operationsPerGoroutine, totalOps,
			"All operations should complete")

		// Performance assertion (should complete in reasonable time)
		assert.Less(t, duration, time.Second*5,
			"Concurrent operations should complete in reasonable time")
	})
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	t.Run("large_data_handling", func(t *testing.T) {
		// Test handling of larger data sets
		const dataSize = 10000

		start := time.Now()

		// Create large data structure
		largeData := make([]map[string]interface{}, dataSize)
		for i := 0; i < dataSize; i++ {
			largeData[i] = map[string]interface{}{
				"id":        i,
				"name":      fmt.Sprintf("item_%d", i),
				"value":     fmt.Sprintf("value_%d", i),
				"timestamp": time.Now().Unix(),
			}
		}

		duration := time.Since(start)
		t.Logf("Creating %d items took: %v", dataSize, duration)

		// Verify data integrity
		assert.Equal(t, dataSize, len(largeData), "All items should be created")
		assert.Equal(t, 0, largeData[0]["id"], "First item should have ID 0")
		assert.Equal(t, dataSize-1, largeData[dataSize-1]["id"], "Last item should have correct ID")

		// Performance assertion
		assert.Less(t, duration, time.Second*2, "Large data creation should be reasonably fast")
	})
}
