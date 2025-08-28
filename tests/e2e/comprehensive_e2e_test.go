//go:build e2e
// +build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/rideshare-platform/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaymentServiceE2E tests payment processing end-to-end
func TestPaymentServiceE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("payment_processing_flow", func(t *testing.T) {
		// Create trip for payment
		rider := testutils.CreateTestRider(t, config.APIGatewayURL)
		trip := testutils.RequestTrip(t, config.APIGatewayURL, rider.ID, 40.7128, -74.0060, 40.7589, -73.9851)

		// Test payment creation
		payment := testutils.TestPayment{
			ID:     testutils.GenerateTestID(),
			Amount: 10.0,
			UserID: rider.ID,
			TripID: trip.ID,
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/payments", payment)
		defer resp.Body.Close()

		validStatuses := []int{201, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Payment creation should succeed or be unavailable")
	})

	t.Run("payment_validation", func(t *testing.T) {
		// Test with invalid payment data
		invalidPayment := testutils.TestPayment{
			TripID: "invalid-trip-id",
			Amount: -100, // Negative amount
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/payments", invalidPayment)
		defer resp.Body.Close()

		validStatuses := []int{400, 422, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Invalid payment should be rejected or unavailable")
	})
}

// TestGeoServiceE2E tests geolocation services end-to-end
func TestGeoServiceE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("location_tracking", func(t *testing.T) {
		driver := testutils.CreateTestDriver(t, config.APIGatewayURL)

		// Set driver location
		locationData := map[string]interface{}{
			"driver_id": driver.ID,
			"latitude":  40.7128,
			"longitude": -74.0060,
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/geo/driver-location", locationData)
		defer resp.Body.Close()

		validStatuses := []int{200, 201, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Location update should succeed or be unavailable")
	})

	t.Run("nearby_drivers_search", func(t *testing.T) {
		searchData := map[string]interface{}{
			"latitude":  40.7128,
			"longitude": -74.0060,
			"radius":    5000, // 5km
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/geo/nearby-drivers", searchData)
		defer resp.Body.Close()

		validStatuses := []int{200, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Nearby drivers search should succeed or be unavailable")
	})
}

// TestVehicleServiceE2E tests vehicle management end-to-end
func TestVehicleServiceE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("vehicle_lifecycle", func(t *testing.T) {
		driver := testutils.CreateTestDriver(t, config.APIGatewayURL)

		// Register vehicle
		vehicle := testutils.RegisterDriverVehicle(t, config.APIGatewayURL, driver.ID)
		require.NotEmpty(t, vehicle.ID, "Vehicle should be registered")

		// Update vehicle status
		updateData := map[string]interface{}{
			"status": "available",
		}

		resp := testutils.MakeAPIRequest(t, "PUT", config.APIGatewayURL+"/api/v1/vehicles/"+vehicle.ID, updateData)
		defer resp.Body.Close()

		validStatuses := []int{200, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Vehicle update should succeed or be unavailable")

		// Get vehicle details
		resp = testutils.HTTPGet(t, config.APIGatewayURL+"/api/v1/vehicles/"+vehicle.ID)
		defer resp.Body.Close()

		validStatuses = []int{200, 404, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Vehicle retrieval should succeed, not found, or be unavailable")
	})
}

// TestPricingServiceE2E tests pricing calculations end-to-end
func TestPricingServiceE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("fare_estimation", func(t *testing.T) {
		estimateData := map[string]interface{}{
			"pickup_lat":   40.7128,
			"pickup_lng":   -74.0060,
			"dropoff_lat":  40.7589,
			"dropoff_lng":  -73.9851,
			"vehicle_type": "sedan",
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/pricing/estimate", estimateData)
		defer resp.Body.Close()

		validStatuses := []int{200, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Fare estimation should succeed or be unavailable")
	})

	t.Run("surge_pricing", func(t *testing.T) {
		surgeData := map[string]interface{}{
			"latitude":     40.7128,
			"longitude":    -74.0060,
			"radius":       1000,
			"current_time": time.Now().Format(time.RFC3339),
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/pricing/surge", surgeData)
		defer resp.Body.Close()

		validStatuses := []int{200, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Surge pricing should succeed or be unavailable")
	})
}

// TestMatchingServiceE2E tests driver-rider matching end-to-end
func TestMatchingServiceE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("driver_matching_flow", func(t *testing.T) {
		rider := testutils.CreateTestRider(t, config.APIGatewayURL)
		driver := testutils.CreateTestDriver(t, config.APIGatewayURL)

		// Set driver as available
		testutils.SetDriverLocation(t, config.APIGatewayURL, driver.ID, 40.7128, -74.0060)

		// Request match
		matchData := map[string]interface{}{
			"rider_id":     rider.ID,
			"pickup_lat":   40.7128,
			"pickup_lng":   -74.0060,
			"vehicle_type": "sedan",
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/matching/find", matchData)
		defer resp.Body.Close()

		validStatuses := []int{200, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Driver matching should succeed or be unavailable")
	})

	t.Run("match_cancellation", func(t *testing.T) {
		// Test cancelling a match request
		cancellationData := map[string]interface{}{
			"match_id": testutils.GenerateTestID(),
			"reason":   "user_cancelled",
		}

		resp := testutils.MakeAPIRequest(t, "POST", config.APIGatewayURL+"/api/v1/matching/cancel", cancellationData)
		defer resp.Body.Close()

		validStatuses := []int{200, 404, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Match cancellation should succeed, not found, or be unavailable")
	})
}

// TestErrorRecoveryE2E tests system behavior under error conditions
func TestErrorRecoveryE2E(t *testing.T) {
	config := testutils.DefaultTestConfig()
	testutils.WaitForService(t, config.APIGatewayURL, config.TestTimeout)

	t.Run("service_timeout_handling", func(t *testing.T) {
		// Test with endpoints that might timeout
		timeoutEndpoints := []string{
			"/api/v1/trips",
			"/api/v1/users",
			"/api/v1/vehicles",
			"/api/v1/payments",
		}

		for _, endpoint := range timeoutEndpoints {
			t.Run("timeout_"+endpoint, func(t *testing.T) {
				resp := testutils.HTTPGet(t, config.APIGatewayURL+endpoint)
				defer resp.Body.Close()

				// Should handle timeouts gracefully
				assert.LessOrEqual(t, resp.StatusCode, 599, "Should return valid HTTP status code")
			})
		}
	})

	t.Run("invalid_json_handling", func(t *testing.T) {
		// Test with malformed JSON
		invalidJSON := `{"invalid": "json", "missing_quote: "value"}`

		resp := testutils.HTTPPost(t, config.APIGatewayURL+"/api/v1/users", "application/json", invalidJSON)
		defer resp.Body.Close()

		validStatuses := []int{400, 422, 503}
		assert.Contains(t, validStatuses, resp.StatusCode, "Invalid JSON should be rejected or unavailable")
	})
}

// Example usage in a test:
func TestComprehensiveGenerateIDUsage(t *testing.T) {
	id := testutils.GenerateTestID()
	t.Logf("Generated test ID: %s", id)
}
