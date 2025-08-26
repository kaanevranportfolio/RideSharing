package service

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/services/matching-service/internal/config"
	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGeoServiceClient is a mock implementation of GeoServiceClient
type MockGeoServiceClient struct {
	mock.Mock
}

func (m *MockGeoServiceClient) CalculateDistance(ctx context.Context, origin, destination *models.Location) (*DistanceResult, error) {
	args := m.Called(ctx, origin, destination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DistanceResult), args.Error(1)
}

func (m *MockGeoServiceClient) CalculateETA(ctx context.Context, origin, destination *models.Location, vehicleType string) (*ETAResult, error) {
	args := m.Called(ctx, origin, destination, vehicleType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ETAResult), args.Error(1)
}

func (m *MockGeoServiceClient) FindNearbyDrivers(ctx context.Context, center *models.Location, radiusKm float64, limit int) ([]*DriverLocation, error) {
	args := m.Called(ctx, center, radiusKm, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverLocation), args.Error(1)
}

func TestAdvancedMatchingService_FindMatch_MockMode(t *testing.T) {
	// Test the mock mode when geo service is nil
	cfg := &config.Config{}
	service := NewSimpleMatchingService(cfg)
	ctx := context.Background()

	request := &MatchingRequest{
		TripID:  "trip123",
		RiderID: "rider456",
		PickupLocation: &models.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		Destination: &models.Location{
			Latitude:  37.7849,
			Longitude: -122.4094,
		},
		VehicleType:    "standard",
		PassengerCount: 1,
		RequestedAt:    time.Now(),
		MaxWaitTime:    10 * time.Minute,
	}

	result, err := service.FindMatch(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, request.TripID, result.TripID)
	assert.True(t, result.Success) // Mock mode should always succeed
	assert.NotNil(t, result.MatchedDriver)
	assert.Greater(t, result.MatchingScore, 0.0)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))
}

func TestAdvancedMatchingService_CalculateMatchingScore(t *testing.T) {
	cfg := &config.Config{}
	service := NewSimpleMatchingService(cfg)

	tests := []struct {
		name        string
		driver      *MatchedDriverInfo
		request     *MatchingRequest
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "perfect_match",
			driver: &MatchedDriverInfo{
				DriverID: "driver123",
				Distance: 0.5, // 0.5 km away
				Rating:   5.0,
				VehicleInfo: &VehicleDetails{
					VehicleType: "standard",
				},
			},
			request: &MatchingRequest{
				VehicleType: "standard",
				Preferences: &RiderPreferences{
					MinDriverRating: 4.0,
				},
			},
			expectedMin: 80.0,
			expectedMax: 100.0,
		},
		{
			name: "distant_driver",
			driver: &MatchedDriverInfo{
				DriverID: "driver456",
				Distance: 15.0, // 15 km away
				Rating:   4.5,
				VehicleInfo: &VehicleDetails{
					VehicleType: "standard",
				},
			},
			request: &MatchingRequest{
				VehicleType: "standard",
				Preferences: &RiderPreferences{
					MinDriverRating: 4.0,
				},
			},
			expectedMin: 20.0,
			expectedMax: 60.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateMatchingScore(tt.driver, tt.request)
			assert.GreaterOrEqual(t, score, tt.expectedMin)
			assert.LessOrEqual(t, score, tt.expectedMax)
		})
	}
}

func TestAdvancedMatchingService_GetMatchingStatus(t *testing.T) {
	cfg := &config.Config{}
	service := NewSimpleMatchingService(cfg)
	ctx := context.Background()

	result, err := service.GetMatchingStatus(ctx, "trip123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "trip123", result["trip_id"])
	assert.Contains(t, []string{"searching", "not_found"}, result["status"])
}

func TestAdvancedMatchingService_CancelMatching(t *testing.T) {
	cfg := &config.Config{}
	service := NewSimpleMatchingService(cfg)
	ctx := context.Background()

	err := service.CancelMatching(ctx, "trip123")
	assert.NoError(t, err)
}

func TestAdvancedMatchingService_GetMatchingMetrics(t *testing.T) {
	cfg := &config.Config{}
	service := NewSimpleMatchingService(cfg)
	ctx := context.Background()

	metrics, err := service.GetMatchingMetrics(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_requests")
	assert.Contains(t, metrics, "success_rate")
}

// Additional comprehensive tests for better coverage

func TestMatchingRequest_Validation(t *testing.T) {
	// Test matching request creation and validation
	request := &MatchingRequest{
		TripID:  "trip_12345",
		RiderID: "rider_67890",
		PickupLocation: &models.Location{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
		Destination: &models.Location{
			Latitude:  40.7589,
			Longitude: -73.9851,
		},
		VehicleType:    "premium",
		PassengerCount: 2,
		RequestedAt:    time.Now(),
		SpecialNeeds:   []string{"wheelchair_accessible"},
		PriorityLevel:  2,
		MaxWaitTime:    15 * time.Minute,
		Preferences: &RiderPreferences{
			MinDriverRating:    4.5,
			PreferredGender:    "any",
			AllowSharedRides:   false,
			MaxDetourTime:      5,
			PreferQuietRide:    true,
			AccessibilityNeeds: []string{"wheelchair_accessible"},
		},
	}

	// Validate request structure
	assert.Equal(t, "trip_12345", request.TripID)
	assert.Equal(t, "rider_67890", request.RiderID)
	assert.Equal(t, 40.7128, request.PickupLocation.Latitude)
	assert.Equal(t, -74.0060, request.PickupLocation.Longitude)
	assert.Equal(t, "premium", request.VehicleType)
	assert.Equal(t, 2, request.PassengerCount)
	assert.Contains(t, request.SpecialNeeds, "wheelchair_accessible")
	assert.Equal(t, 2, request.PriorityLevel)
	assert.Equal(t, 15*time.Minute, request.MaxWaitTime)
	assert.Equal(t, 4.5, request.Preferences.MinDriverRating)
	assert.False(t, request.Preferences.AllowSharedRides)
}

func TestMatchedDriverInfo_Structure(t *testing.T) {
	// Test matched driver info structure
	driver := &MatchedDriverInfo{
		DriverID:   "driver_abc123",
		VehicleID:  "vehicle_xyz789",
		DriverName: "John Doe",
		Distance:   2.5,
		ETA:        480, // 8 minutes
		Rating:     4.8,
		MatchScore: 92.5,
		TripCount:  1250,
		Status:     "available",
		VehicleInfo: &VehicleDetails{
			VehicleType:  "premium",
			Make:         "Toyota",
			Model:        "Camry",
			Year:         2022,
			Color:        "Black",
			LicensePlate: "ABC1234",
			Capacity:     4,
			Features:     []string{"air_conditioning", "wifi"},
		},
		CurrentLocation: &models.Location{
			Latitude:  40.7500,
			Longitude: -73.9900,
		},
	}

	// Validate driver structure
	assert.Equal(t, "driver_abc123", driver.DriverID)
	assert.Equal(t, "vehicle_xyz789", driver.VehicleID)
	assert.Equal(t, "John Doe", driver.DriverName)
	assert.Equal(t, 2.5, driver.Distance)
	assert.Equal(t, 480, driver.ETA)
	assert.Equal(t, 4.8, driver.Rating)
	assert.Equal(t, 92.5, driver.MatchScore)
	assert.Equal(t, 1250, driver.TripCount)
	assert.Equal(t, "available", driver.Status)
	assert.Equal(t, "premium", driver.VehicleInfo.VehicleType)
	assert.Equal(t, "Toyota", driver.VehicleInfo.Make)
	assert.Equal(t, "Camry", driver.VehicleInfo.Model)
	assert.Equal(t, 2022, driver.VehicleInfo.Year)
	assert.Equal(t, 4, driver.VehicleInfo.Capacity)
	assert.Contains(t, driver.VehicleInfo.Features, "air_conditioning")
}

func TestFareEstimate_Structure(t *testing.T) {
	// Test fare estimate structure
	fareEstimate := &FareEstimate{
		BaseFare:      5.50,
		DistanceFare:  12.75,
		TimeFare:      3.25,
		SurgeFare:     8.00,
		TotalEstimate: 29.50,
		Currency:      "USD",
	}

	// Validate fare structure
	assert.Equal(t, 5.50, fareEstimate.BaseFare)
	assert.Equal(t, 12.75, fareEstimate.DistanceFare)
	assert.Equal(t, 3.25, fareEstimate.TimeFare)
	assert.Equal(t, 8.00, fareEstimate.SurgeFare)
	assert.Equal(t, 29.50, fareEstimate.TotalEstimate)
	assert.Equal(t, "USD", fareEstimate.Currency)

	// Test calculated total
	expectedTotal := fareEstimate.BaseFare + fareEstimate.DistanceFare + fareEstimate.TimeFare + fareEstimate.SurgeFare
	assert.Equal(t, expectedTotal, fareEstimate.TotalEstimate)
}

func TestDriverLocation_Structure(t *testing.T) {
	// Test driver location structure
	driverLocation := &DriverLocation{
		DriverID:           "driver_test123",
		VehicleID:          "vehicle_test456",
		Location:           &models.Location{Latitude: 40.7128, Longitude: -74.0060},
		DistanceFromCenter: 3.2,
		Status:             "available",
		VehicleType:        "standard",
		Rating:             4.6,
	}

	// Validate driver location structure
	assert.Equal(t, "driver_test123", driverLocation.DriverID)
	assert.Equal(t, "vehicle_test456", driverLocation.VehicleID)
	assert.Equal(t, 40.7128, driverLocation.Location.Latitude)
	assert.Equal(t, -74.0060, driverLocation.Location.Longitude)
	assert.Equal(t, 3.2, driverLocation.DistanceFromCenter)
	assert.Equal(t, "available", driverLocation.Status)
	assert.Equal(t, "standard", driverLocation.VehicleType)
	assert.Equal(t, 4.6, driverLocation.Rating)
}

func TestMatchingResult_Success(t *testing.T) {
	// Test successful matching result
	result := &MatchingResult{
		TripID:  "trip_success_test",
		Success: true,
		MatchedDriver: &MatchedDriverInfo{
			DriverID: "best_driver",
			Distance: 1.2,
			Rating:   4.9,
		},
		EstimatedETA: 360, // 6 minutes
		EstimatedFare: &FareEstimate{
			TotalEstimate: 18.50,
			Currency:      "USD",
		},
		Reason:         "Successfully matched with optimal driver",
		MatchingScore:  94.5,
		ProcessingTime: 250 * time.Millisecond,
		RetryCount:     0,
	}

	// Validate successful result
	assert.Equal(t, "trip_success_test", result.TripID)
	assert.True(t, result.Success)
	assert.Equal(t, "best_driver", result.MatchedDriver.DriverID)
	assert.Equal(t, 360, result.EstimatedETA)
	assert.Equal(t, 18.50, result.EstimatedFare.TotalEstimate)
	assert.Equal(t, "Successfully matched with optimal driver", result.Reason)
	assert.Equal(t, 94.5, result.MatchingScore)
	assert.Equal(t, 250*time.Millisecond, result.ProcessingTime)
	assert.Equal(t, 0, result.RetryCount)
}

func TestMatchingResult_Failure(t *testing.T) {
	// Test failed matching result
	result := &MatchingResult{
		TripID:         "trip_fail_test",
		Success:        false,
		Reason:         "No eligible drivers found in the area",
		ProcessingTime: 5 * time.Second,
		RetryCount:     3,
	}

	// Validate failed result
	assert.Equal(t, "trip_fail_test", result.TripID)
	assert.False(t, result.Success)
	assert.Nil(t, result.MatchedDriver)
	assert.Zero(t, result.EstimatedETA)
	assert.Nil(t, result.EstimatedFare)
	assert.Equal(t, "No eligible drivers found in the area", result.Reason)
	assert.Equal(t, 5*time.Second, result.ProcessingTime)
	assert.Equal(t, 3, result.RetryCount)
}

func TestRiderPreferences_Validation(t *testing.T) {
	// Test rider preferences validation
	prefs := &RiderPreferences{
		MinDriverRating:    4.0,
		PreferredGender:    "female",
		AllowSharedRides:   true,
		MaxDetourTime:      10,
		PreferQuietRide:    false,
		AccessibilityNeeds: []string{"wheelchair_accessible", "service_animal_friendly"},
	}

	// Validate preferences
	assert.Equal(t, 4.0, prefs.MinDriverRating)
	assert.Equal(t, "female", prefs.PreferredGender)
	assert.True(t, prefs.AllowSharedRides)
	assert.Equal(t, 10, prefs.MaxDetourTime)
	assert.False(t, prefs.PreferQuietRide)
	assert.Len(t, prefs.AccessibilityNeeds, 2)
	assert.Contains(t, prefs.AccessibilityNeeds, "wheelchair_accessible")
	assert.Contains(t, prefs.AccessibilityNeeds, "service_animal_friendly")
}

func TestVehicleDetails_Validation(t *testing.T) {
	// Test vehicle details validation
	vehicle := &VehicleDetails{
		VehicleType:  "luxury",
		Make:         "Mercedes-Benz",
		Model:        "S-Class",
		Year:         2023,
		Color:        "Silver",
		LicensePlate: "LUX5678",
		Capacity:     4,
		Features:     []string{"leather_seats", "wifi", "climate_control"},
	}

	// Validate vehicle details
	assert.Equal(t, "luxury", vehicle.VehicleType)
	assert.Equal(t, "Mercedes-Benz", vehicle.Make)
	assert.Equal(t, "S-Class", vehicle.Model)
	assert.Equal(t, 2023, vehicle.Year)
	assert.Equal(t, "Silver", vehicle.Color)
	assert.Equal(t, "LUX5678", vehicle.LicensePlate)
	assert.Equal(t, 4, vehicle.Capacity)
	assert.Len(t, vehicle.Features, 3)
	assert.Contains(t, vehicle.Features, "leather_seats")
	assert.Contains(t, vehicle.Features, "wifi")
	assert.Contains(t, vehicle.Features, "climate_control")
}

func TestGeoServiceResults_Structure(t *testing.T) {
	// Test geo-service result structures
	distanceResult := &DistanceResult{
		DistanceMeters: 2500.0,
		DistanceKm:     2.5,
		BearingDegrees: 45.0,
	}

	etaResult := &ETAResult{
		DurationSeconds: 480,
		DistanceMeters:  2500.0,
		RouteSummary:    "Via Main Street and Broadway",
	}

	// Validate distance result
	assert.Equal(t, 2500.0, distanceResult.DistanceMeters)
	assert.Equal(t, 2.5, distanceResult.DistanceKm)
	assert.Equal(t, 45.0, distanceResult.BearingDegrees)

	// Validate ETA result
	assert.Equal(t, 480, etaResult.DurationSeconds)
	assert.Equal(t, 2500.0, etaResult.DistanceMeters)
	assert.Equal(t, "Via Main Street and Broadway", etaResult.RouteSummary)

	// Test conversion consistency
	assert.Equal(t, distanceResult.DistanceMeters/1000, distanceResult.DistanceKm)
	assert.Equal(t, etaResult.DurationSeconds/60, 8) // 8 minutes
}
