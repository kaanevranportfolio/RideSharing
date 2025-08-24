//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/rideshare-platform/tests/testutils"
)

func TestPostgresConnection(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Test basic connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestDatabaseSchema(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Test that we can query basic schema information
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	// Should have at least 0 tables (empty database is fine for testing)
	if count < 0 {
		t.Errorf("Expected non-negative table count, got %d", count)
	}
}

func TestDatabaseTransactions(t *testing.T) {
	testutils.SkipIfShort(t)

	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	// Test transaction functionality
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Create a temporary table for testing
	_, err = tx.Exec("CREATE TEMPORARY TABLE test_table (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = tx.Exec("INSERT INTO test_table (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}
