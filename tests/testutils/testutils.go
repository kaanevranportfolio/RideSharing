package testutils

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestConfig holds configuration for tests
type TestConfig struct {
	DatabaseURL    string
	APIGatewayURL  string
	UserServiceURL string
	TripServiceURL string
	TestTimeout    time.Duration
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	// Build PostgreSQL connection string from environment variables
	postgresHost := getEnv("TEST_POSTGRES_HOST", "localhost")
	postgresPort := getEnv("TEST_POSTGRES_PORT", "5433")
	postgresUser := getEnv("TEST_POSTGRES_USER", "postgres")
	postgresPassword := getEnv("TEST_POSTGRES_PASSWORD", "testpass123")
	postgresDB := getEnv("TEST_POSTGRES_DB", "rideshare_test")

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	return &TestConfig{
		DatabaseURL:    databaseURL,
		APIGatewayURL:  getEnv("API_GATEWAY_URL", "http://localhost:8080"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		TripServiceURL: getEnv("TRIP_SERVICE_URL", "http://localhost:8084"),
		TestTimeout:    30 * time.Second,
	}
}

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T, config *TestConfig) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// CleanupTestDB cleans up test database
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if db != nil {
		db.Close()
	}
}

// WaitForService waits for a service to be ready
func WaitForService(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/health")
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable) {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("Service at %s did not become ready within %v", url, timeout)
}

// HTTPGet performs a GET request with timeout
func HTTPGet(t *testing.T, url string) *http.Response {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("HTTP GET failed for %s: %v", url, err)
	}

	return resp
}

// HTTPPost performs a POST request with timeout
func HTTPPost(t *testing.T, url string, contentType string, body string) *http.Response {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, contentType, nil)
	if err != nil {
		t.Fatalf("HTTP POST failed for %s: %v", url, err)
	}

	return resp
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// SkipIfShort skips test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// SkipIfNoIntegration skips test if integration tag not provided
func SkipIfNoIntegration(t *testing.T) {
	t.Helper()
	// This will be automatically handled by build tags
}

// CreateTestUser creates a test user and returns user ID
func CreateTestUser(t *testing.T, baseURL string) string {
	t.Helper()

	// Mock implementation - in real scenario would create via API
	return fmt.Sprintf("test-user-%d", time.Now().Unix())
}

// CreateTestTrip creates a test trip and returns trip ID
func CreateTestTrip(t *testing.T, baseURL string, userID string) string {
	t.Helper()

	// Mock implementation - in real scenario would create via API
	return fmt.Sprintf("test-trip-%d", time.Now().Unix())
}
