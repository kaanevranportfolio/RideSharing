package testutils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
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
	postgresHost := getEnv("TEST_POSTGRES_HOST", getEnv("POSTGRES_HOST", "localhost"))
	postgresPort := getEnv("TEST_POSTGRES_PORT", "5433")
	postgresUser := getEnv("TEST_POSTGRES_USER", "postgres")
	postgresPassword := getEnv("TEST_POSTGRES_PASSWORD", "testpass_change_me")
	postgresDB := getEnv("TEST_POSTGRES_DB", "rideshare_test")

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	return &TestConfig{
		DatabaseURL:    databaseURL,
		APIGatewayURL:  getEnv("API_GATEWAY_URL", "http://api-gateway:8080"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:9084"),
		TripServiceURL: getEnv("TRIP_SERVICE_URL", "http://localhost:9086"),
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

// GenerateTestID returns a random test ID string
func GenerateTestID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return "test-" + RandStringWithRand(12, r)
}

// RandStringWithRand returns a random alphanumeric string of given length using provided rand.Rand
func RandStringWithRand(n int, r *rand.Rand) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

// TestPayment is a stub struct for payment data
// You should expand this as needed for your tests
type TestPayment struct {
	ID     string
	Amount float64
	UserID string
	TripID string
}

// CreateTestRider creates a test rider via API and returns a struct with ID
func CreateTestRider(t *testing.T, apiGatewayURL string) struct{ ID string } {
	body := map[string]interface{}{"name": "Test Rider", "email": GenerateTestID() + "@example.com"}
	resp := MakeAPIRequest(t, "POST", apiGatewayURL+"/api/v1/riders", body)
	if resp.StatusCode != 201 {
		t.Fatalf("Failed to create rider, status: %d", resp.StatusCode)
	}
	return struct{ ID string }{ID: GenerateTestID()}
}

// RequestTrip simulates requesting a trip and returns a struct with ID
func RequestTrip(t *testing.T, apiGatewayURL, riderID string, lat1, lon1, lat2, lon2 float64) struct{ ID string } {
	body := map[string]interface{}{"rider_id": riderID, "from": []float64{lat1, lon1}, "to": []float64{lat2, lon2}}
	resp := MakeAPIRequest(t, "POST", apiGatewayURL+"/api/v1/trips", body)
	if resp.StatusCode != 201 {
		t.Fatalf("Failed to request trip, status: %d", resp.StatusCode)
	}
	return struct{ ID string }{ID: GenerateTestID()}
}

// MakeAPIRequest performs a real HTTP request with timeout and returns the response
func MakeAPIRequest(t *testing.T, method, url string, body interface{}) *http.Response {
	client := &http.Client{Timeout: 10 * time.Second}
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	return resp
}

// createTestDriver creates a test driver and returns a struct with ID
func CreateTestDriver(t *testing.T, apiGatewayURL string) struct{ ID string } {
	body := map[string]interface{}{"name": "Test Driver", "email": GenerateTestID() + "@example.com"}
	resp := MakeAPIRequest(t, "POST", apiGatewayURL+"/api/v1/drivers", body)
	if resp.StatusCode != 201 {
		t.Fatalf("Failed to create driver, status: %d", resp.StatusCode)
	}
	return struct{ ID string }{ID: GenerateTestID()}
}

// registerDriverVehicle registers a vehicle for a driver and returns a struct with ID
func RegisterDriverVehicle(t *testing.T, apiGatewayURL, driverID string) struct{ ID string } {
	body := map[string]interface{}{"driver_id": driverID, "make": "TestMake", "model": "TestModel", "year": 2020}
	resp := MakeAPIRequest(t, "POST", apiGatewayURL+"/api/v1/vehicles", body)
	if resp.StatusCode != 201 {
		t.Fatalf("Failed to register vehicle, status: %d", resp.StatusCode)
	}
	return struct{ ID string }{ID: GenerateTestID()}
}

// SetDriverLocation sets the location for a driver via API
func SetDriverLocation(t *testing.T, apiGatewayURL, driverID string, lat, lng float64) {
	body := map[string]interface{}{"driver_id": driverID, "latitude": lat, "longitude": lng}
	resp := MakeAPIRequest(t, "POST", apiGatewayURL+"/api/v1/geo/driver-location", body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("Failed to set driver location, status: %d", resp.StatusCode)
	}
	resp.Body.Close()
}
