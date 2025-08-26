//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestDatabaseConnection tests basic database connectivity
func TestDatabaseConnection(t *testing.T) {
	db, err := getTestDB()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("✅ Database connection successful")
}

// TestBasicCRUDOperations tests basic database operations
func TestBasicCRUDOperations(t *testing.T) {
	db, err := getTestDB()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("create_temporary_table", func(t *testing.T) {
		// Create a temporary table for testing
		_, err := db.ExecContext(ctx, `
			CREATE TEMPORARY TABLE integration_test_users (
				id VARCHAR(255) PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				name VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create temporary table: %v", err)
		}
		t.Log("✅ Temporary table created")
	})

	t.Run("insert_and_retrieve_data", func(t *testing.T) {
		// Insert test data
		testID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())
		testEmail := fmt.Sprintf("test%d@example.com", time.Now().UnixNano())
		testName := "Integration Test User"

		_, err := db.ExecContext(ctx,
			"INSERT INTO integration_test_users (id, email, name) VALUES ($1, $2, $3)",
			testID, testEmail, testName)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		// Retrieve and verify data
		var retrievedID, retrievedEmail, retrievedName string
		err = db.QueryRowContext(ctx,
			"SELECT id, email, name FROM integration_test_users WHERE id = $1",
			testID).Scan(&retrievedID, &retrievedEmail, &retrievedName)
		if err != nil {
			t.Fatalf("Failed to retrieve test data: %v", err)
		}

		if retrievedID != testID {
			t.Errorf("Expected ID %s, got %s", testID, retrievedID)
		}
		if retrievedEmail != testEmail {
			t.Errorf("Expected email %s, got %s", testEmail, retrievedEmail)
		}
		if retrievedName != testName {
			t.Errorf("Expected name %s, got %s", testName, retrievedName)
		}

		t.Logf("✅ CRUD operations successful: ID=%s, Email=%s", testID, testEmail)
	})

	t.Run("concurrent_operations", func(t *testing.T) {
		const numOperations = 10
		done := make(chan error, numOperations)

		// Perform concurrent insertions
		for i := 0; i < numOperations; i++ {
			go func(index int) {
				testID := fmt.Sprintf("concurrent-user-%d-%d", index, time.Now().UnixNano())
				testEmail := fmt.Sprintf("concurrent%d@example.com", index)
				testName := fmt.Sprintf("Concurrent User %d", index)

				_, err := db.ExecContext(ctx,
					"INSERT INTO integration_test_users (id, email, name) VALUES ($1, $2, $3)",
					testID, testEmail, testName)
				done <- err
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < numOperations; i++ {
			err := <-done
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i, err)
			}
		}

		// Verify all records were inserted
		var count int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM integration_test_users WHERE name LIKE 'Concurrent User%'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count concurrent records: %v", err)
		}

		if count != numOperations {
			t.Errorf("Expected %d concurrent records, got %d", numOperations, count)
		}

		t.Logf("✅ Concurrent operations successful: %d records inserted", count)
	})
}

// TestTransactionBehavior tests database transaction handling
func TestTransactionBehavior(t *testing.T) {
	db, err := getTestDB()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("transaction_commit", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Create a test table within transaction
		_, err = tx.ExecContext(ctx, `
			CREATE TEMPORARY TABLE tx_commit_test (
				id SERIAL PRIMARY KEY,
				data TEXT
			)
		`)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create table in transaction: %v", err)
		}

		// Insert data
		_, err = tx.ExecContext(ctx, "INSERT INTO tx_commit_test (data) VALUES ('committed')")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert data in transaction: %v", err)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify data persisted
		var data string
		err = db.QueryRowContext(ctx, "SELECT data FROM tx_commit_test LIMIT 1").Scan(&data)
		if err != nil {
			t.Fatalf("Failed to verify committed data: %v", err)
		}

		if data != "committed" {
			t.Errorf("Expected 'committed', got '%s'", data)
		}

		t.Log("✅ Transaction commit successful")
	})

	t.Run("transaction_rollback", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Create a test table within transaction
		_, err = tx.ExecContext(ctx, `
			CREATE TEMPORARY TABLE tx_rollback_test (
				id SERIAL PRIMARY KEY,
				data TEXT
			)
		`)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create table in transaction: %v", err)
		}

		// Insert data
		_, err = tx.ExecContext(ctx, "INSERT INTO tx_rollback_test (data) VALUES ('to_be_rolled_back')")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert data in transaction: %v", err)
		}

		// Rollback transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify data was rolled back
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tx_rollback_test").Scan(&count)
		if err == nil {
			t.Error("Expected table to not exist after rollback, but query succeeded")
		}

		t.Log("✅ Transaction rollback successful")
	})
}

// Helper function to get test database connection
func getTestDB() (*sql.DB, error) {
	// Use test database configuration
	postgresHost := getEnv("TEST_POSTGRES_HOST", "localhost")
	postgresPort := getEnv("TEST_POSTGRES_PORT", "5433")
	postgresUser := getEnv("TEST_POSTGRES_USER", "postgres")
	postgresPassword := getEnv("TEST_POSTGRES_PASSWORD", "testpass_change_me")
	postgresDB := getEnv("TEST_POSTGRES_DB", "rideshare_test")

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	return sql.Open("postgres", databaseURL)
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
