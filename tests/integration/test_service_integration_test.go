//go:build integration
// +build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/rideshare-platform/tests/testutils"
)

func TestAPIGatewayHealth(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Wait for API Gateway to be ready
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	// Test health endpoint
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/health")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestAPIGatewayStatus(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Test status endpoint
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/status")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAPIGatewayUserEndpoint(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Test user endpoint
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/api/v1/users/123")
	defer resp.Body.Close()

	// Should return 200 (mock response) or 503 (service unavailable)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}

func TestAPIGatewayTripEndpoint(t *testing.T) {
	config := testutils.DefaultTestConfig()

	// Test trip endpoint
	resp := testutils.HTTPGet(t, config.APIGatewayURL+"/api/v1/trips/456")
	defer resp.Body.Close()

	// Should return 200 (mock response) or 503 (service unavailable)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}
}
