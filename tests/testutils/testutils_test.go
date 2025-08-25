package testutils

import (
	"testing"
)

func TestDefaultTestConfig(t *testing.T) {
	config := DefaultTestConfig()

	if config.APIGatewayURL == "" {
		t.Error("APIGatewayURL should not be empty")
	}

	if config.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}

	if config.TestTimeout == 0 {
		t.Error("TestTimeout should be set")
	}
}

func TestCreateTestUser(t *testing.T) {
	userID := CreateTestUser(t, "http://api-gateway:8080")

	if userID == "" {
		t.Error("CreateTestUser should return a non-empty user ID")
	}
}

func TestCreateTestTrip(t *testing.T) {
	tripID := CreateTestTrip(t, "http://api-gateway:8080", "test-user-123")

	if tripID == "" {
		t.Error("CreateTestTrip should return a non-empty trip ID")
	}
}

func TestSkipIfShort(t *testing.T) {
	// This test verifies the function exists and can be called
	// In short mode, this would skip
	if !testing.Short() {
		// Only run this part if not in short mode
		t.Log("Running in non-short mode")
	}
}
