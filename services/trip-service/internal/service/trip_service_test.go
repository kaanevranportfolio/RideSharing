package service

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTripRepository is a mock implementation of TripRepositoryInterface
type MockTripRepository struct {
	mock.Mock
}

func (m *MockTripRepository) Create(ctx context.Context, trip *models.Trip) error {
	args := m.Called(ctx, trip)
	return args.Error(0)
}

func (m *MockTripRepository) GetByID(ctx context.Context, id string) (*models.Trip, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trip), args.Error(1)
}

func (m *MockTripRepository) Update(ctx context.Context, trip *models.Trip) error {
	args := m.Called(ctx, trip)
	return args.Error(0)
}

func (m *MockTripRepository) GetByRiderID(ctx context.Context, riderID string) ([]*models.Trip, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Trip), args.Error(1)
}

func (m *MockTripRepository) GetByDriverID(ctx context.Context, driverID string) ([]*models.Trip, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Trip), args.Error(1)
}

func TestTripService_CreateTrip(t *testing.T) {
	mockRepo := new(MockTripRepository)
	logger := logger.NewLogger("test", "info")
	service := NewTripService(mockRepo, logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *CreateTripRequest
		setupMock   func(*MockTripRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_trip_creation",
			request: &CreateTripRequest{
				RiderID: "rider123",
				PickupLocation: models.Location{
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
				DestinationLocation: models.Location{
					Latitude:  37.7849,
					Longitude: -122.4094,
				},
				RideType:      "standard",
				EstimatedFare: 15.50,
				RequestedAt:   time.Now(),
			},
			setupMock: func(m *MockTripRepository) {
				m.On("Create", ctx, mock.AnythingOfType("*models.Trip")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "empty_rider_id_error",
			request: &CreateTripRequest{
				RiderID: "",
				PickupLocation: models.Location{
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
				DestinationLocation: models.Location{
					Latitude:  37.7849,
					Longitude: -122.4094,
				},
				RideType:      "standard",
				EstimatedFare: 15.50,
				RequestedAt:   time.Now(),
			},
			setupMock:   func(m *MockTripRepository) {},
			expectError: true,
			errorMsg:    "rider ID is required",
		},
		{
			name: "invalid_ride_type",
			request: &CreateTripRequest{
				RiderID: "rider123",
				PickupLocation: models.Location{
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
				DestinationLocation: models.Location{
					Latitude:  37.7849,
					Longitude: -122.4094,
				},
				RideType:      "invalid",
				EstimatedFare: 15.50,
				RequestedAt:   time.Now(),
			},
			setupMock:   func(m *MockTripRepository) {},
			expectError: true,
			errorMsg:    "invalid ride type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock(mockRepo)

			result, err := service.CreateTrip(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.RiderID, result.RiderID)
				assert.Equal(t, models.TripStatusRequested, result.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTripService_GetTrip(t *testing.T) {
	mockRepo := new(MockTripRepository)
	logger := logger.NewLogger("test", "info")
	service := NewTripService(mockRepo, logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		tripID      string
		setupMock   func(*MockTripRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "successful_trip_retrieval",
			tripID: "trip123",
			setupMock: func(m *MockTripRepository) {
				trip := &models.Trip{
					ID:      "trip123",
					RiderID: "rider123",
					Status:  models.TripStatusRequested,
				}
				m.On("GetByID", ctx, "trip123").Return(trip, nil)
			},
			expectError: false,
		},
		{
			name:        "empty_trip_id",
			tripID:      "",
			setupMock:   func(m *MockTripRepository) {},
			expectError: true,
			errorMsg:    "trip ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock(mockRepo)

			result, err := service.GetTrip(ctx, tt.tripID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.tripID, result.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTripService_AcceptTrip(t *testing.T) {
	mockRepo := new(MockTripRepository)
	logger := logger.NewLogger("test", "info")
	service := NewTripService(mockRepo, logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		tripID      string
		driverID    string
		setupMock   func(*MockTripRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name:     "successful_trip_acceptance",
			tripID:   "trip123",
			driverID: "driver456",
			setupMock: func(m *MockTripRepository) {
				trip := &models.Trip{
					ID:      "trip123",
					RiderID: "rider123",
					Status:  models.TripStatusRequested,
				}
				m.On("GetByID", ctx, "trip123").Return(trip, nil)
				m.On("Update", ctx, mock.AnythingOfType("*models.Trip")).Return(nil)
			},
			expectError: false,
		},
		{
			name:        "empty_trip_id",
			tripID:      "",
			driverID:    "driver456",
			setupMock:   func(m *MockTripRepository) {},
			expectError: true,
			errorMsg:    "trip ID is required",
		},
		{
			name:        "empty_driver_id",
			tripID:      "trip123",
			driverID:    "",
			setupMock:   func(m *MockTripRepository) {},
			expectError: true,
			errorMsg:    "driver ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock(mockRepo)

			result, err := service.AcceptTrip(ctx, tt.tripID, tt.driverID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.driverID, *result.DriverID)
				assert.Equal(t, models.TripStatusMatched, result.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTripService_CalculateTripDuration(t *testing.T) {
	logger := logger.NewLogger("test", "info")
	service := NewTripService(nil, logger)

	tests := []struct {
		name        string
		trip        *models.Trip
		expectError bool
		errorMsg    string
		expected    time.Duration
	}{
		{
			name: "successful_duration_calculation",
			trip: &models.Trip{
				Status: models.TripStatusCompleted,
				StartedAt: func() *time.Time {
					t := time.Now().Add(-30 * time.Minute)
					return &t
				}(),
				CompletedAt: func() *time.Time {
					t := time.Now()
					return &t
				}(),
			},
			expectError: false,
			expected:    30 * time.Minute,
		},
		{
			name: "trip_not_completed",
			trip: &models.Trip{
				Status: models.TripStatusRequested,
			},
			expectError: true,
			errorMsg:    "trip is not completed",
		},
		{
			name: "invalid_timestamps",
			trip: &models.Trip{
				Status:      models.TripStatusCompleted,
				StartedAt:   nil,
				CompletedAt: nil,
			},
			expectError: true,
			errorMsg:    "trip timestamps are invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CalculateTripDuration(tt.trip)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Allow for small time differences in test execution
				assert.InDelta(t, tt.expected.Seconds(), result.Seconds(), 5.0)
			}
		})
	}
}

func TestTripService_EstimateTripTime(t *testing.T) {
	logger := logger.NewLogger("test", "info")
	service := NewTripService(nil, logger)

	tests := []struct {
		name     string
		distance float64
		expected time.Duration
	}{
		{
			name:     "15_km_distance",
			distance: 15.0,
			expected: 30 * time.Minute, // 15 km at 30 km/h = 0.5 hours
		},
		{
			name:     "30_km_distance",
			distance: 30.0,
			expected: 60 * time.Minute, // 30 km at 30 km/h = 1 hour
		},
		{
			name:     "zero_distance",
			distance: 0.0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.EstimateTripTime(tt.distance)
			assert.Equal(t, tt.expected, result)
		})
	}
}
