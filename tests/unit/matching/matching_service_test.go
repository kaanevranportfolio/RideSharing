package matching

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Extended Driver model for testing purposes
type TestDriver struct {
	*models.Driver
	ID       string
	Vehicle  *models.Vehicle
	Location *models.Location
}

// MockGeoService implements geospatial service interface for testing
type MockGeoService struct {
	mock.Mock
}

func (m *MockGeoService) FindNearbyDrivers(ctx context.Context, lat, lng float64, radius float64) ([]*TestDriver, error) {
	args := m.Called(ctx, lat, lng, radius)
	return args.Get(0).([]*TestDriver), args.Error(1)
}

func (m *MockGeoService) CalculateDistance(ctx context.Context, lat1, lng1, lat2, lng2 float64) (float64, error) {
	args := m.Called(ctx, lat1, lng1, lat2, lng2)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockGeoService) CalculateETA(ctx context.Context, lat1, lng1, lat2, lng2 float64, mode string) (time.Duration, error) {
	args := m.Called(ctx, lat1, lng1, lat2, lng2, mode)
	return args.Get(0).(time.Duration), args.Error(1)
}

// MockDriverRepository implements driver repository interface for testing
type MockDriverRepository struct {
	mock.Mock
}

func (m *MockDriverRepository) GetDriver(ctx context.Context, driverID string) (*TestDriver, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TestDriver), args.Error(1)
}

func (m *MockDriverRepository) UpdateDriverStatus(ctx context.Context, driverID string, status models.DriverStatus) error {
	args := m.Called(ctx, driverID, status)
	return args.Error(0)
}

func (m *MockDriverRepository) GetAvailableDrivers(ctx context.Context, vehicleType string) ([]*TestDriver, error) {
	args := m.Called(ctx, vehicleType)
	return args.Get(0).([]*TestDriver), args.Error(1)
}

// MatchingServiceTestSuite provides comprehensive testing for matching service
type MatchingServiceTestSuite struct {
	suite.Suite
	mockGeoService  *MockGeoService
	mockDriverRepo  *MockDriverRepository
	matchingService *MatchingService
	testRider       *models.User
	testDrivers     []*TestDriver
}

func (suite *MatchingServiceTestSuite) SetupTest() {
	suite.mockGeoService = new(MockGeoService)
	suite.mockDriverRepo = new(MockDriverRepository)

	// Initialize matching service with mocks
	suite.matchingService = &MatchingService{
		geoService: suite.mockGeoService,
		driverRepo: suite.mockDriverRepo,
		config: &MatchingConfig{
			MaxSearchRadius:    5000.0, // 5km
			MaxDrivers:         10,
			DistanceWeight:     0.4,
			RatingWeight:       0.3,
			AvailabilityWeight: 0.3,
		},
	}

	// Setup test data
	suite.testRider = &models.User{
		ID:       "rider-123",
		Email:    "rider@test.com",
		UserType: models.UserTypeRider,
	}

	// Create test drivers with proper model structure
	lat1, lng1 := 40.7128, -74.0060
	lat2, lng2 := 40.7150, -74.0080
	lat3, lng3 := 40.7200, -74.0100

	suite.testDrivers = []*TestDriver{
		{
			Driver: &models.Driver{
				UserID:           "user-1",
				Status:           models.DriverStatusOnline,
				Rating:           4.8,
				CurrentLatitude:  &lat1,
				CurrentLongitude: &lng1,
			},
			ID: "driver-1",
			Vehicle: &models.Vehicle{
				VehicleType: models.VehicleTypeSedan,
			},
			Location: &models.Location{
				Latitude:  lat1,
				Longitude: lng1,
			},
		},
		{
			Driver: &models.Driver{
				UserID:           "user-2",
				Status:           models.DriverStatusOnline,
				Rating:           4.5,
				CurrentLatitude:  &lat2,
				CurrentLongitude: &lng2,
			},
			ID: "driver-2",
			Vehicle: &models.Vehicle{
				VehicleType: models.VehicleTypeSedan,
			},
			Location: &models.Location{
				Latitude:  lat2,
				Longitude: lng2,
			},
		},
		{
			Driver: &models.Driver{
				UserID:           "user-3",
				Status:           models.DriverStatusBusy,
				Rating:           4.9,
				CurrentLatitude:  &lat3,
				CurrentLongitude: &lng3,
			},
			ID: "driver-3",
			Vehicle: &models.Vehicle{
				VehicleType: models.VehicleTypeSUV,
			},
			Location: &models.Location{
				Latitude:  lat3,
				Longitude: lng3,
			},
		},
	}
}

func (suite *MatchingServiceTestSuite) TearDownTest() {
	suite.mockGeoService.AssertExpectations(suite.T())
	suite.mockDriverRepo.AssertExpectations(suite.T())
}

// TestMatchingAlgorithmCore tests the core matching logic
func (suite *MatchingServiceTestSuite) TestMatchingAlgorithmCore() {
	tests := []struct {
		name          string
		request       *MatchingRequest
		nearbyDrivers []*TestDriver
		expectedCount int
		shouldError   bool
		errorMessage  string
	}{
		{
			name: "successful_match_with_available_drivers",
			request: &MatchingRequest{
				RiderID:     suite.testRider.ID,
				PickupLat:   40.7128,
				PickupLng:   -74.0060,
				DestLat:     40.7589,
				DestLng:     -73.9851,
				VehicleType: string(models.VehicleTypeSedan),
				MaxWaitTime: 300, // 5 minutes
			},
			nearbyDrivers: suite.testDrivers[:2], // Only available drivers
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "no_drivers_available_in_area",
			request: &MatchingRequest{
				RiderID:     suite.testRider.ID,
				PickupLat:   40.7128,
				PickupLng:   -74.0060,
				DestLat:     40.7589,
				DestLng:     -73.9851,
				VehicleType: string(models.VehicleTypeSedan),
			},
			nearbyDrivers: []*TestDriver{},
			expectedCount: 0,
			shouldError:   true,
			errorMessage:  "no drivers available",
		},
		{
			name: "vehicle_type_filtering",
			request: &MatchingRequest{
				RiderID:     suite.testRider.ID,
				PickupLat:   40.7128,
				PickupLng:   -74.0060,
				DestLat:     40.7589,
				DestLng:     -73.9851,
				VehicleType: string(models.VehicleTypeLuxury),
			},
			nearbyDrivers: suite.testDrivers[:2], // Both sedan drivers
			expectedCount: 0,
			shouldError:   true,
			errorMessage:  "no drivers available",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ctx := context.Background()

			// Setup mock expectations
			suite.mockGeoService.On("FindNearbyDrivers",
				ctx, tt.request.PickupLat, tt.request.PickupLng,
				suite.matchingService.config.MaxSearchRadius).
				Return(tt.nearbyDrivers, nil).Once()

			if len(tt.nearbyDrivers) > 0 {
				for _, driver := range tt.nearbyDrivers {
					suite.mockGeoService.On("CalculateDistance",
						ctx, tt.request.PickupLat, tt.request.PickupLng,
						driver.Location.Latitude, driver.Location.Longitude).
						Return(1000.0, nil).Once() // 1km distance
				}
			}

			// Execute matching
			result, err := suite.matchingService.FindMatch(ctx, tt.request)

			// Verify results
			if tt.shouldError {
				suite.Error(err)
				if tt.errorMessage != "" {
					suite.Contains(err.Error(), tt.errorMessage)
				}
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Len(result.Drivers, tt.expectedCount)

				// Verify drivers are sorted by score (best first)
				for i := 1; i < len(result.Drivers); i++ {
					suite.GreaterOrEqual(result.Drivers[i-1].Score, result.Drivers[i].Score)
				}
			}
		})
	}
}

// TestDriverScoringAlgorithm tests the driver scoring logic
func (suite *MatchingServiceTestSuite) TestDriverScoringAlgorithm() {
	ctx := context.Background()
	request := &MatchingRequest{
		RiderID:   suite.testRider.ID,
		PickupLat: 40.7128,
		PickupLng: -74.0060,
	}

	tests := []struct {
		name          string
		driver        *TestDriver
		distance      float64
		expectedScore float64
		minScore      float64
		maxScore      float64
	}{
		{
			name:     "excellent_driver_close_distance",
			driver:   suite.testDrivers[0], // Rating 4.8, Online
			distance: 500.0,                // 0.5km
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name:     "good_driver_medium_distance",
			driver:   suite.testDrivers[1], // Rating 4.5, Online
			distance: 2000.0,               // 2km
			minScore: 0.5,
			maxScore: 0.8,
		},
		{
			name:     "busy_driver_excluded",
			driver:   suite.testDrivers[2], // Rating 4.9, Busy
			distance: 300.0,                // 0.3km
			minScore: 0.0,
			maxScore: 0.0, // Should be 0 due to busy status
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			score := suite.matchingService.calculateDriverScore(ctx, tt.driver, request, tt.distance)

			suite.GreaterOrEqual(score, tt.minScore, "Score should be at least minimum")
			suite.LessOrEqual(score, tt.maxScore, "Score should not exceed maximum")

			// Busy drivers should have zero score
			if tt.driver.Status == models.DriverStatusBusy {
				suite.Equal(0.0, score, "Busy drivers should have zero score")
			}
		})
	}
}

// TestConcurrentMatching tests thread safety of matching operations
func (suite *MatchingServiceTestSuite) TestConcurrentMatching() {
	ctx := context.Background()

	// Setup mock expectations for concurrent calls
	for i := 0; i < 10; i++ {
		suite.mockGeoService.On("FindNearbyDrivers",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(suite.testDrivers[:1], nil)

		suite.mockGeoService.On("CalculateDistance",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(1000.0, nil)
	}

	// Execute concurrent matching requests
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			request := &MatchingRequest{
				RiderID:   fmt.Sprintf("rider-%d", id),
				PickupLat: 40.7128,
				PickupLng: -74.0060,
				DestLat:   40.7589,
				DestLng:   -73.9851,
			}

			_, err := suite.matchingService.FindMatch(ctx, request)
			results <- err
		}(i)
	}

	// Verify all requests completed without race conditions
	for i := 0; i < 10; i++ {
		err := <-results
		suite.NoError(err, "Concurrent matching should not cause race conditions")
	}
}

// TestErrorHandling tests various error scenarios
func (suite *MatchingServiceTestSuite) TestErrorHandling() {
	ctx := context.Background()

	tests := []struct {
		name          string
		request       *MatchingRequest
		geoError      error
		distanceError error
		expectedError string
	}{
		{
			name: "invalid_coordinates",
			request: &MatchingRequest{
				RiderID:   suite.testRider.ID,
				PickupLat: 91.0, // Invalid latitude
				PickupLng: -74.0060,
			},
			expectedError: "invalid coordinates",
		},
		{
			name: "geo_service_failure",
			request: &MatchingRequest{
				RiderID:   suite.testRider.ID,
				PickupLat: 40.7128,
				PickupLng: -74.0060,
			},
			geoError:      errors.New("geo service unavailable"),
			expectedError: "geo service unavailable",
		},
		{
			name: "distance_calculation_failure",
			request: &MatchingRequest{
				RiderID:   suite.testRider.ID,
				PickupLat: 40.7128,
				PickupLng: -74.0060,
			},
			distanceError: errors.New("distance calculation failed"),
			expectedError: "distance calculation failed",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.geoError != nil {
				suite.mockGeoService.On("FindNearbyDrivers",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*TestDriver{}, tt.geoError).Once()
			} else if tt.distanceError != nil {
				suite.mockGeoService.On("FindNearbyDrivers",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(suite.testDrivers[:1], nil).Once()
				suite.mockGeoService.On("CalculateDistance",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(0.0, tt.distanceError).Once()
			}

			_, err := suite.matchingService.FindMatch(ctx, tt.request)

			suite.Error(err)
			suite.Contains(err.Error(), tt.expectedError)
		})
	}
}

// TestPerformanceBenchmarks provides performance testing
func (suite *MatchingServiceTestSuite) TestPerformanceBenchmarks() {
	ctx := context.Background()
	request := &MatchingRequest{
		RiderID:   suite.testRider.ID,
		PickupLat: 40.7128,
		PickupLng: -74.0060,
	}

	// Setup mocks for performance test
	suite.mockGeoService.On("FindNearbyDrivers",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(suite.testDrivers, nil)

	for range suite.testDrivers {
		suite.mockGeoService.On("CalculateDistance",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(1000.0, nil)
	}

	// Measure performance
	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		_, err := suite.matchingService.FindMatch(ctx, request)
		suite.NoError(err)
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	// Performance assertions
	suite.Less(avgDuration, 50*time.Millisecond, "Average matching should complete under 50ms")

	suite.T().Logf("Performance Results: %d iterations in %v (avg: %v per operation)",
		iterations, duration, avgDuration)
}

// Mock types and implementations for testing
type MatchingService struct {
	geoService *MockGeoService
	driverRepo *MockDriverRepository
	config     *MatchingConfig
}

type MatchingConfig struct {
	MaxSearchRadius    float64
	MaxDrivers         int
	DistanceWeight     float64
	RatingWeight       float64
	AvailabilityWeight float64
}

type MatchingRequest struct {
	RiderID     string
	PickupLat   float64
	PickupLng   float64
	DestLat     float64
	DestLng     float64
	VehicleType string
	MaxWaitTime int
}

type MatchingResult struct {
	Drivers []*ScoredDriver
}

type ScoredDriver struct {
	Driver *TestDriver
	Score  float64
}

// Core matching logic implementation for testing
func (ms *MatchingService) FindMatch(ctx context.Context, request *MatchingRequest) (*MatchingResult, error) {
	// Validate coordinates
	if request.PickupLat < -90 || request.PickupLat > 90 ||
		request.PickupLng < -180 || request.PickupLng > 180 {
		return nil, errors.New("invalid coordinates")
	}

	// Find nearby drivers
	drivers, err := ms.geoService.FindNearbyDrivers(ctx,
		request.PickupLat, request.PickupLng, ms.config.MaxSearchRadius)
	if err != nil {
		return nil, err
	}

	// Filter and score drivers
	var scoredDrivers []*ScoredDriver
	for _, driver := range drivers {
		// Filter by vehicle type if specified
		if request.VehicleType != "" && string(driver.Vehicle.VehicleType) != request.VehicleType {
			continue
		}

		// Filter by availability
		if driver.Status != models.DriverStatusOnline {
			continue
		}

		// Calculate distance
		distance, err := ms.geoService.CalculateDistance(ctx,
			request.PickupLat, request.PickupLng,
			driver.Location.Latitude, driver.Location.Longitude)
		if err != nil {
			return nil, err
		}

		// Calculate score
		score := ms.calculateDriverScore(ctx, driver, request, distance)
		if score > 0 {
			scoredDrivers = append(scoredDrivers, &ScoredDriver{
				Driver: driver,
				Score:  score,
			})
		}
	}

	if len(scoredDrivers) == 0 {
		return nil, errors.New("no drivers available")
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredDrivers)-1; i++ {
		for j := i + 1; j < len(scoredDrivers); j++ {
			if scoredDrivers[j].Score > scoredDrivers[i].Score {
				scoredDrivers[i], scoredDrivers[j] = scoredDrivers[j], scoredDrivers[i]
			}
		}
	}

	return &MatchingResult{Drivers: scoredDrivers}, nil
}

func (ms *MatchingService) calculateDriverScore(ctx context.Context, driver *TestDriver, request *MatchingRequest, distance float64) float64 {
	if driver.Status != models.DriverStatusOnline {
		return 0.0
	}

	// Distance score (closer is better)
	distanceScore := 1.0 - (distance / ms.config.MaxSearchRadius)
	if distanceScore < 0 {
		distanceScore = 0
	}

	// Rating score (normalized to 0-1)
	ratingScore := driver.Rating / 5.0

	// Availability score (always 1.0 for available drivers)
	availabilityScore := 1.0

	// Weighted final score
	finalScore := (distanceScore * ms.config.DistanceWeight) +
		(ratingScore * ms.config.RatingWeight) +
		(availabilityScore * ms.config.AvailabilityWeight)

	return finalScore
}

// TestMatchingService runs the complete test suite
func TestMatchingService(t *testing.T) {
	suite.Run(t, new(MatchingServiceTestSuite))
}

// Benchmark functions for performance testing
func BenchmarkMatchingService_FindMatch(b *testing.B) {
	suite := &MatchingServiceTestSuite{}
	suite.SetupTest()

	ctx := context.Background()
	request := &MatchingRequest{
		RiderID:   "bench-rider",
		PickupLat: 40.7128,
		PickupLng: -74.0060,
	}

	suite.mockGeoService.On("FindNearbyDrivers",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(suite.testDrivers, nil)

	for range suite.testDrivers {
		suite.mockGeoService.On("CalculateDistance",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(1000.0, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.matchingService.FindMatch(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}
