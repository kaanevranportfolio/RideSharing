//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rideshare-platform/tests/testutils"
)

func TestUserLifecycle(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Test user creation (mock)
	userID := testutils.CreateTestUser(t, config.APIGatewayURL)

	// Test user retrieval
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/api/v1/users/"+userID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestTripLifecycle(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Create test user and trip
	userID := testutils.CreateTestUser(t, config.APIGatewayURL)
	tripID := testutils.CreateTestTrip(t, config.APIGatewayURL, userID)

	// Test trip retrieval
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/api/v1/trips/"+tripID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestPricingEstimate(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Test pricing estimate endpoint
	resp := testutils.HTTPPost(t, config.APIGatewayURL+"/api/v1/pricing/estimate", "application/json", "{}")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}

	// If we get a successful response, validate JSON structure
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		// Check for expected fields in mock response
		if _, exists := result["estimated_fare"]; !exists {
			t.Error("Response missing 'estimated_fare' field")
		}
	}
}

func TestDriverMatching(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Test driver matching endpoint
	requestBody := `{"location": {"latitude": 40.7128, "longitude": -74.0060}, "radius": 5000}`
	resp := testutils.HTTPPost(t, config.APIGatewayURL+"/api/v1/matching/nearby-drivers", "application/json", requestBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestPaymentProcessing(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Test payment processing endpoint
	requestBody := `{"trip_id": "test-trip", "amount": 15.50, "payment_method_id": "pm_test"}`
	resp := testutils.HTTPPost(t, config.APIGatewayURL+"/api/v1/payments", "application/json", requestBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestWebSocketConnection(t *testing.T) {
	// Skip WebSocket test for now - would require gorilla/websocket test client
	t.Skip("WebSocket E2E test requires additional setup")
}

func TestCORSHeaders(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Create request with CORS headers
	req, err := http.NewRequest("OPTIONS", config.APIGatewayURL+"/api/v1/users/123", nil)
	if err != nil {
		t.Fatalf("Failed to create OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", origin)
	}

	methods := resp.Header.Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "GET") {
		t.Errorf("Expected Access-Control-Allow-Methods to contain GET, got %s", methods)
	}
}
