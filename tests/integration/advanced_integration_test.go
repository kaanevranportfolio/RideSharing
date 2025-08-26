//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rideshare-platform/tests/testutils"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestAdvancedIntegrationScenarios tests complex multi-service scenarios
func TestAdvancedIntegrationScenarios(t *testing.T) {
	testutils.SkipIfShort(t)
	
	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	t.Run("multi_user_trip_workflow", func(t *testing.T) {
		ctx := context.Background()
		
		// Create test schema if needed
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS test_trips (
				id VARCHAR(255) PRIMARY KEY,
				rider_id VARCHAR(255),
				driver_id VARCHAR(255),
				status VARCHAR(50),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test schema: %v", err)
		}
		defer db.ExecContext(ctx, "DROP TABLE IF EXISTS test_trips")
		
		// Simulate multi-user scenario
		const numRiders = 3
		const numDrivers = 2
		
		// Create riders
		riderIDs := make([]string, numRiders)
		for i := 0; i < numRiders; i++ {
			riderIDs[i] = fmt.Sprintf("rider-%d-%d", i, time.Now().UnixNano())
		}
		
		// Create drivers
		driverIDs := make([]string, numDrivers)
		for i := 0; i < numDrivers; i++ {
			driverIDs[i] = fmt.Sprintf("driver-%d-%d", i, time.Now().UnixNano())
		}
		
		// Create trips concurrently
		var wg sync.WaitGroup
		trips := make([]string, numRiders)
		
		for i := 0; i < numRiders; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				tripID := fmt.Sprintf("trip-%d-%d", index, time.Now().UnixNano())
				trips[index] = tripID
				
				// Assign driver round-robin
				driverID := driverIDs[index%numDrivers]
				
				_, err := db.ExecContext(ctx,
					"INSERT INTO test_trips (id, rider_id, driver_id, status) VALUES ($1, $2, $3, $4)",
					tripID, riderIDs[index], driverID, "active")
				if err != nil {
					t.Errorf("Failed to create trip %d: %v", index, err)
				}
			}(i)
		}
		
		wg.Wait()
		
		// Verify all trips were created
		var tripCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_trips").Scan(&tripCount)
		if err != nil {
			t.Fatalf("Failed to count trips: %v", err)
		}
		
		if tripCount != numRiders {
			t.Errorf("Expected %d trips, got %d", numRiders, tripCount)
		}
		
		// Verify driver distribution
		for i, driverID := range driverIDs {
			var driverTripCount int
			err = db.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM test_trips WHERE driver_id = $1",
				driverID).Scan(&driverTripCount)
			if err != nil {
				t.Errorf("Failed to count trips for driver %d: %v", i, err)
				continue
			}
			
			if driverTripCount == 0 {
				t.Errorf("Driver %s has no trips assigned", driverID)
			}
		}
	})

	t.Run("concurrent_database_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Create test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS concurrent_ops (
				id VARCHAR(255) PRIMARY KEY,
				operation_type VARCHAR(50),
				worker_id INT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		defer db.ExecContext(ctx, "DROP TABLE IF EXISTS concurrent_ops")
		
		const numWorkers = 5
		const opsPerWorker = 10
		
		var wg sync.WaitGroup
		errors := make(chan error, numWorkers)
		
		// Launch concurrent workers
		for workerID := 0; workerID < numWorkers; workerID++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				for op := 0; op < opsPerWorker; op++ {
					opID := fmt.Sprintf("worker-%d-op-%d-%d", id, op, time.Now().UnixNano())
					
					_, err := db.ExecContext(ctx,
						"INSERT INTO concurrent_ops (id, operation_type, worker_id) VALUES ($1, $2, $3)",
						opID, "insert", id)
					if err != nil {
						errors <- err
						return
					}
					
					// Small delay to increase concurrency
					time.Sleep(1 * time.Millisecond)
				}
				errors <- nil
			}(workerID)
		}
		
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			if err != nil {
				t.Errorf("Worker error: %v", err)
			}
		}
		
		// Verify all operations completed
		var totalOps int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concurrent_ops").Scan(&totalOps)
		if err != nil {
			t.Fatalf("Failed to count operations: %v", err)
		}
		
		expectedOps := numWorkers * opsPerWorker
		if totalOps != expectedOps {
			t.Errorf("Expected %d operations, got %d", expectedOps, totalOps)
		}
		
		// Verify each worker completed their operations
		for workerID := 0; workerID < numWorkers; workerID++ {
			var workerOps int
			err = db.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM concurrent_ops WHERE worker_id = $1",
				workerID).Scan(&workerOps)
			if err != nil {
				t.Errorf("Failed to count operations for worker %d: %v", workerID, err)
				continue
			}
			
			if workerOps != opsPerWorker {
				t.Errorf("Worker %d: expected %d operations, got %d", workerID, opsPerWorker, workerOps)
			}
		}
	})
}

// TestDatabasePerformanceIntegration tests database performance under various conditions
func TestDatabasePerformanceIntegration(t *testing.T) {
	testutils.SkipIfShort(t)
	
	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	t.Run("bulk_insert_performance", func(t *testing.T) {
		ctx := context.Background()
		
		// Create test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS bulk_test (
				id VARCHAR(255) PRIMARY KEY,
				data VARCHAR(1000),
				number_value INT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		defer db.ExecContext(ctx, "DROP TABLE IF EXISTS bulk_test")
		
		const batchSize = 100
		start := time.Now()
		
		// Use transaction for better performance
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		
		stmt, err := tx.PrepareContext(ctx,
			"INSERT INTO bulk_test (id, data, number_value) VALUES ($1, $2, $3)")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()
		
		// Insert batch data
		for i := 0; i < batchSize; i++ {
			id := fmt.Sprintf("bulk-%d-%d", i, time.Now().UnixNano())
			data := fmt.Sprintf("test data for record %d", i)
			
			_, err = stmt.ExecContext(ctx, id, data, i)
			if err != nil {
				tx.Rollback()
				t.Fatalf("Failed to insert record %d: %v", i, err)
			}
		}
		
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}
		
		duration := time.Since(start)
		t.Logf("Bulk insert of %d records took %v", batchSize, duration)
		
		// Verify all records were inserted
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bulk_test").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count records: %v", err)
		}
		
		if count != batchSize {
			t.Errorf("Expected %d records, got %d", batchSize, count)
		}
		
		// Performance assertion - should complete in reasonable time
		if duration > 5*time.Second {
			t.Errorf("Bulk insert took too long: %v (expected < 5s)", duration)
		}
	})

	t.Run("concurrent_read_write_stress", func(t *testing.T) {
		ctx := context.Background()
		
		// Create test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS stress_test (
				id VARCHAR(255) PRIMARY KEY,
				counter INT DEFAULT 0,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		defer db.ExecContext(ctx, "DROP TABLE IF EXISTS stress_test")
		
		// Insert initial record
		testID := fmt.Sprintf("stress-%d", time.Now().UnixNano())
		_, err = db.ExecContext(ctx,
			"INSERT INTO stress_test (id, counter) VALUES ($1, $2)",
			testID, 0)
		if err != nil {
			t.Fatalf("Failed to insert initial record: %v", err)
		}
		
		const numReaders = 3
		const numWriters = 2
		const testDuration = 2 * time.Second
		
		var wg sync.WaitGroup
		errors := make(chan error, numReaders+numWriters)
		
		// Start readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(readerID int) {
				defer wg.Done()
				
				deadline := time.Now().Add(testDuration)
				readCount := 0
				
				for time.Now().Before(deadline) {
					var counter int
					err := db.QueryRowContext(ctx,
						"SELECT counter FROM stress_test WHERE id = $1", testID).Scan(&counter)
					if err != nil && err != sql.ErrNoRows {
						errors <- fmt.Errorf("reader %d error: %v", readerID, err)
						return
					}
					readCount++
					time.Sleep(10 * time.Millisecond)
				}
				
				t.Logf("Reader %d completed %d reads", readerID, readCount)
				errors <- nil
			}(i)
		}
		
		// Start writers
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				
				deadline := time.Now().Add(testDuration)
				writeCount := 0
				
				for time.Now().Before(deadline) {
					_, err := db.ExecContext(ctx,
						"UPDATE stress_test SET counter = counter + 1, updated_at = $1 WHERE id = $2",
						time.Now(), testID)
					if err != nil {
						errors <- fmt.Errorf("writer %d error: %v", writerID, err)
						return
					}
					writeCount++
					time.Sleep(50 * time.Millisecond)
				}
				
				t.Logf("Writer %d completed %d writes", writerID, writeCount)
				errors <- nil
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			if err != nil {
				t.Errorf("Stress test error: %v", err)
			}
		}
		
		// Verify final state
		var finalCounter int
		err = db.QueryRowContext(ctx,
			"SELECT counter FROM stress_test WHERE id = $1", testID).Scan(&finalCounter)
		if err != nil {
			t.Fatalf("Failed to get final counter: %v", err)
		}
		
		if finalCounter <= 0 {
			t.Error("Counter should have been incremented by writers")
		}
		
		t.Logf("Final counter value: %d", finalCounter)
	})
}

	t.Run("bulk_operations_performance", func(t *testing.T) {
		ctx := context.Background()
		const batchSize = 50

		start := time.Now()

		// Create users in batches
		userIDs := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			userIDs[i] = testutils.GenerateTestID()
		}

		// Bulk insert users
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err, "Failed to begin transaction")

		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO users (id, email, first_name, last_name, user_type, status, created_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)`)
		require.NoError(t, err, "Failed to prepare statement")
		defer stmt.Close()

		for i, userID := range userIDs {
			email := "bulk-user-" + userID[0:8] + "@example.com"
			_, err = stmt.ExecContext(ctx, userID, email, "Bulk", "User", "rider", "active", time.Now())
			require.NoError(t, err, "Failed to insert user %d", i)
		}

		err = tx.Commit()
		require.NoError(t, err, "Failed to commit transaction")

		duration := time.Since(start)
		t.Logf("Bulk insert of %d users took %v", batchSize, duration)

		// Verify all users were inserted
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'bulk-user-%'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, batchSize, count, "All bulk users should be inserted")

		// Performance assertion - should complete within reasonable time
		assert.Less(t, duration, 5*time.Second, "Bulk operations should complete quickly")
	})

	t.Run("concurrent_read_write_operations", func(t *testing.T) {
		ctx := context.Background()
		const numReaders = 5
		const numWriters = 3
		const operationDuration = 2 * time.Second

		// Create base data
		baseUserID := testutils.GenerateTestID()
		err := createTestUserInDB(db, baseUserID, "concurrent-base@example.com", "rider")
		require.NoError(t, err, "Failed to create base user")

		// Start concurrent readers
		readerErrors := make(chan error, numReaders)
		for i := 0; i < numReaders; i++ {
			go func(readerID int) {
				deadline := time.Now().Add(operationDuration)
				for time.Now().Before(deadline) {
					var email string
					err := db.QueryRowContext(ctx,
						"SELECT email FROM users WHERE id = $1", baseUserID).Scan(&email)
					if err != nil && err != sql.ErrNoRows {
						readerErrors <- err
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
				readerErrors <- nil
			}(i)
		}

		// Start concurrent writers
		writerErrors := make(chan error, numWriters)
		for i := 0; i < numWriters; i++ {
			go func(writerID int) {
				deadline := time.Now().Add(operationDuration)
				counter := 0
				for time.Now().Before(deadline) {
					userID := testutils.GenerateTestID()
					email := "concurrent-writer-" + userID[0:8] + "@example.com"
					err := createTestUserInDB(db, userID, email, "driver")
					if err != nil {
						writerErrors <- err
						return
					}
					counter++
					time.Sleep(50 * time.Millisecond)
				}
				t.Logf("Writer %d created %d users", writerID, counter)
				writerErrors <- nil
			}(i)
		}

		// Collect results
		for i := 0; i < numReaders; i++ {
			err := <-readerErrors
			assert.NoError(t, err, "Reader %d should complete without error", i)
		}

		for i := 0; i < numWriters; i++ {
			err := <-writerErrors
			assert.NoError(t, err, "Writer %d should complete without error", i)
		}
	})
}

// Helper types and functions
type TestUserData struct {
	ID       string
	Email    string
	UserType string
}

type TestTripData struct {
	ID      string
	RiderID string
}

func createMultipleTestUsers(t *testing.T, db *sql.DB, count int, userType string) []TestUserData {
	users := make([]TestUserData, count)

	for i := 0; i < count; i++ {
		userID := testutils.GenerateTestID()
		email := userType + "-" + userID[0:8] + "@example.com"

		err := createTestUserInDB(db, userID, email, userType)
		require.NoError(t, err, "Failed to create test user %d", i)

		users[i] = TestUserData{
			ID:       userID,
			Email:    email,
			UserType: userType,
		}
	}

	return users
}

func assignDriverToTrip(ctx context.Context, db *sql.DB, tripID, driverID string) error {
	_, err := db.ExecContext(ctx,
		"UPDATE trips SET driver_id = $1, status = $2, updated_at = $3 WHERE id = $4",
		driverID, "accepted", time.Now(), tripID)
	return err
}
