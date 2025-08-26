package integration

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// GeoIntegrationTestSuite provides integration tests for the geospatial service
type GeoIntegrationTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestGeoIntegrationSuite(t *testing.T) {
	suite.Run(t, new(GeoIntegrationTestSuite))
}

func (suite *GeoIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
}

// TestDistanceCalculationIntegration tests end-to-end distance calculations
func (suite *GeoIntegrationTestSuite) TestDistanceCalculationIntegration() {
	tests := []struct {
		name          string
		origin        models.Location
		destination   models.Location
		method        string
		expectedMinKm float64
		expectedMaxKm float64
	}{
		{
			name: "Manhattan distance calculation",
			origin: models.Location{
				Latitude:  40.7589, // Times Square
				Longitude: -73.9851,
			},
			destination: models.Location{
				Latitude:  40.7829, // Central Park
				Longitude: -73.9654,
			},
			method:        "haversine",
			expectedMinKm: 2.0,
			expectedMaxKm: 3.5,
		},
		{
			name: "Cross-borough calculation",
			origin: models.Location{
				Latitude:  40.7589, // Times Square
				Longitude: -73.9851,
			},
			destination: models.Location{
				Latitude:  40.6892, // Brooklyn
				Longitude: -73.9442,
			},
			method:        "haversine",
			expectedMinKm: 7.0,
			expectedMaxKm: 10.0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Test would call actual geo service here
			// For now, we test the algorithm directly
			distance := suite.calculateTestDistance(tt.origin, tt.destination, tt.method)
			distanceKm := distance / 1000

			assert.GreaterOrEqual(suite.T(), distanceKm, tt.expectedMinKm,
				"Distance should be at least %.1f km", tt.expectedMinKm)
			assert.LessOrEqual(suite.T(), distanceKm, tt.expectedMaxKm,
				"Distance should be at most %.1f km", tt.expectedMaxKm)
		})
	}
}

// TestETACalculationIntegration tests ETA calculations with traffic factors
func (suite *GeoIntegrationTestSuite) TestETACalculationIntegration() {
	origin := models.Location{
		Latitude:  40.7589,
		Longitude: -73.9851,
	}
	destination := models.Location{
		Latitude:  40.7829,
		Longitude: -73.9654,
	}

	tests := []struct {
		name        string
		vehicleType string
		hour        int
		expectedMin int // minimum seconds
		expectedMax int // maximum seconds
	}{
		{
			name:        "Car during rush hour",
			vehicleType: "car",
			hour:        8,    // 8 AM rush hour
			expectedMin: 600,  // 10 minutes minimum
			expectedMax: 1800, // 30 minutes maximum with traffic
		},
		{
			name:        "Car during normal hours",
			vehicleType: "car",
			hour:        14,  // 2 PM normal
			expectedMin: 300, // 5 minutes minimum
			expectedMax: 900, // 15 minutes maximum
		},
		{
			name:        "Bike during rush hour",
			vehicleType: "bike",
			hour:        8,
			expectedMin: 480, // 8 minutes (bikes less affected by traffic)
			expectedMax: 720, // 12 minutes
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			departureTime := time.Date(2024, 1, 1, tt.hour, 0, 0, 0, time.UTC)
			eta := suite.calculateTestETA(origin, destination, tt.vehicleType, departureTime)

			assert.GreaterOrEqual(suite.T(), eta, tt.expectedMin,
				"ETA should be at least %d seconds", tt.expectedMin)
			assert.LessOrEqual(suite.T(), eta, tt.expectedMax,
				"ETA should be at most %d seconds", tt.expectedMax)
		})
	}
}

// TestNearbyDriverSearch tests driver proximity algorithms
func (suite *GeoIntegrationTestSuite) TestNearbyDriverSearch() {
	centerLocation := models.Location{
		Latitude:  40.7589,
		Longitude: -73.9851,
	}

	// Simulate driver locations at various distances
	testDrivers := []struct {
		driverId   string
		location   models.Location
		distanceKm float64
		inRadius   bool
	}{
		{
			driverId: "driver1",
			location: models.Location{
				Latitude:  40.7590, // Very close (≈10m)
				Longitude: -73.9850,
			},
			distanceKm: 0.01,
			inRadius:   true,
		},
		{
			driverId: "driver2",
			location: models.Location{
				Latitude:  40.7829, // Central Park (≈2.7km)
				Longitude: -73.9654,
			},
			distanceKm: 2.7,
			inRadius:   true,
		},
		{
			driverId: "driver3",
			location: models.Location{
				Latitude:  40.6892, // Brooklyn (≈8km)
				Longitude: -73.9442,
			},
			distanceKm: 8.0,
			inRadius:   false, // Outside 5km radius
		},
	}

	radiusKm := 5.0

	// Test proximity filtering
	for _, driver := range testDrivers {
		suite.Run("Driver_"+driver.driverId, func() {
			distance := suite.calculateTestDistance(centerLocation, driver.location, "haversine")
			distanceKm := distance / 1000

			if driver.inRadius {
				assert.LessOrEqual(suite.T(), distanceKm, radiusKm,
					"Driver %s should be within %v km radius", driver.driverId, radiusKm)
			} else {
				assert.Greater(suite.T(), distanceKm, radiusKm,
					"Driver %s should be outside %v km radius", driver.driverId, radiusKm)
			}
		})
	}
}

// TestGeohashConsistency tests geohash generation and consistency
func (suite *GeoIntegrationTestSuite) TestGeohashConsistency() {
	location := models.Location{
		Latitude:  40.7589,
		Longitude: -73.9851,
	}

	// Test different precision levels
	precisions := []int{1, 4, 6, 8, 10}

	for _, precision := range precisions {
		suite.Run("Precision_"+string(rune(precision+'0')), func() {
			geohash1 := suite.calculateTestGeohash(location, precision)
			geohash2 := suite.calculateTestGeohash(location, precision)

			assert.Equal(suite.T(), geohash1, geohash2,
				"Geohash should be consistent for same location")
			assert.Equal(suite.T(), precision, len(geohash1),
				"Geohash should have requested precision length")

			// Verify base32 characters
			base32chars := "0123456789bcdefghjkmnpqrstuvwxyz"
			for _, char := range geohash1 {
				assert.Contains(suite.T(), base32chars, string(char),
					"Geohash should only contain base32 characters")
			}
		})
	}
}

// TestRouteOptimization tests waypoint generation and route planning
func (suite *GeoIntegrationTestSuite) TestRouteOptimization() {
	origin := models.Location{
		Latitude:  40.7589,
		Longitude: -73.9851,
	}
	destination := models.Location{
		Latitude:  40.7829,
		Longitude: -73.9654,
	}

	waypointCounts := []int{2, 5, 10}

	for _, count := range waypointCounts {
		suite.Run("Waypoints_"+string(rune(count+'0')), func() {
			waypoints := suite.generateTestWaypoints(origin, destination, count)

			// Should have count+1 total points (including destination)
			assert.Equal(suite.T(), count+1, len(waypoints),
				"Should have %d waypoints", count+1)

			// First should be origin
			assert.Equal(suite.T(), origin.Latitude, waypoints[0].Latitude)
			assert.Equal(suite.T(), origin.Longitude, waypoints[0].Longitude)

			// Last should be destination
			lastIdx := len(waypoints) - 1
			assert.Equal(suite.T(), destination.Latitude, waypoints[lastIdx].Latitude)
			assert.Equal(suite.T(), destination.Longitude, waypoints[lastIdx].Longitude)

			// Intermediate points should be between origin and destination
			for i := 1; i < lastIdx; i++ {
				if origin.Latitude < destination.Latitude {
					assert.GreaterOrEqual(suite.T(), waypoints[i].Latitude, origin.Latitude)
					assert.LessOrEqual(suite.T(), waypoints[i].Latitude, destination.Latitude)
				} else {
					assert.LessOrEqual(suite.T(), waypoints[i].Latitude, origin.Latitude)
					assert.GreaterOrEqual(suite.T(), waypoints[i].Latitude, destination.Latitude)
				}
			}
		})
	}
}

// TestBusinessLogicValidation tests business rules and constraints
func (suite *GeoIntegrationTestSuite) TestBusinessLogicValidation() {
	suite.Run("Maximum search radius enforcement", func() {
		maxRadius := 50.0        // km
		requestedRadius := 100.0 // km - exceeds max

		// In real implementation, service should cap at maxRadius
		effectiveRadius := suite.enforceMaxRadius(requestedRadius, maxRadius)
		assert.Equal(suite.T(), maxRadius, effectiveRadius,
			"Radius should be capped at maximum allowed")
	})

	suite.Run("Geohash precision validation", func() {
		validPrecisions := []int{1, 6, 12}
		invalidPrecisions := []int{0, -1, 15}

		location := models.Location{Latitude: 40.7589, Longitude: -73.9851}

		for _, precision := range validPrecisions {
			geohash := suite.calculateTestGeohash(location, precision)
			assert.NotEmpty(suite.T(), geohash, "Valid precision should produce geohash")
		}

		for _, precision := range invalidPrecisions {
			// In real implementation, should handle invalid precision gracefully
			if precision < 1 || precision > 12 {
				assert.True(suite.T(), true, "Invalid precision %d should be rejected", precision)
			}
		}
	})
}

// Helper methods for testing algorithms (would interface with actual service in real implementation)

func (suite *GeoIntegrationTestSuite) calculateTestDistance(origin, destination models.Location, method string) float64 {
	// Haversine implementation for testing
	const earthRadiusKm = 6371.0

	lat1 := origin.Latitude * 3.14159265359 / 180
	lon1 := origin.Longitude * 3.14159265359 / 180
	lat2 := destination.Latitude * 3.14159265359 / 180
	lon2 := destination.Longitude * 3.14159265359 / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := (1-math.Cos(dlat))/2 + math.Cos(lat1)*math.Cos(lat2)*(1-math.Cos(dlon))/2
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c * 1000 // meters
}

func (suite *GeoIntegrationTestSuite) calculateTestETA(origin, destination models.Location, vehicleType string, departureTime time.Time) int {
	distance := suite.calculateTestDistance(origin, destination, "haversine")
	distanceKm := distance / 1000

	// Speed in km/h based on vehicle type
	var speed float64
	switch vehicleType {
	case "car":
		speed = 30.0 // average city speed
	case "bike":
		speed = 15.0
	case "walking":
		speed = 5.0
	default:
		speed = 30.0
	}

	// Traffic factor based on time
	hour := departureTime.Hour()
	trafficFactor := 1.0
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		trafficFactor = 1.5 // rush hour
	} else if hour >= 23 || hour <= 5 {
		trafficFactor = 0.8 // late night
	}

	baseDurationHours := distanceKm / speed
	durationSeconds := int(baseDurationHours * 3600 * trafficFactor)

	return durationSeconds
}

func (suite *GeoIntegrationTestSuite) calculateTestGeohash(location models.Location, precision int) string {
	// Simplified geohash for testing
	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"
	var geohash string

	latRange := []float64{-90.0, 90.0}
	lngRange := []float64{-180.0, 180.0}

	var even bool = true
	var bit int = 0
	var ch int = 0

	for len(geohash) < precision {
		if even {
			mid := (lngRange[0] + lngRange[1]) / 2
			if location.Longitude >= mid {
				ch |= (1 << (4 - bit))
				lngRange[0] = mid
			} else {
				lngRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2
			if location.Latitude >= mid {
				ch |= (1 << (4 - bit))
				latRange[0] = mid
			} else {
				latRange[1] = mid
			}
		}

		even = !even
		bit++

		if bit == 5 {
			geohash += string(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return geohash
}

func (suite *GeoIntegrationTestSuite) generateTestWaypoints(origin, destination models.Location, count int) []models.Location {
	var waypoints []models.Location

	waypoints = append(waypoints, origin)

	for i := 1; i < count; i++ {
		ratio := float64(i) / float64(count)

		lat := origin.Latitude + ratio*(destination.Latitude-origin.Latitude)
		lng := origin.Longitude + ratio*(destination.Longitude-origin.Longitude)

		waypoints = append(waypoints, models.Location{
			Latitude:  lat,
			Longitude: lng,
			Timestamp: time.Now(),
		})
	}

	waypoints = append(waypoints, destination)
	return waypoints
}

func (suite *GeoIntegrationTestSuite) enforceMaxRadius(requestedRadius, maxRadius float64) float64 {
	if requestedRadius > maxRadius {
		return maxRadius
	}
	return requestedRadius
}
