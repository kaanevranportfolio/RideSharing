package vehicle_test

import (
	"context"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVehicleRepository is a mock implementation of the vehicle repository
type MockVehicleRepository struct {
	mock.Mock
}

func (m *MockVehicleRepository) CreateVehicle(ctx context.Context, vehicle *models.Vehicle) (*models.Vehicle, error) {
	args := m.Called(ctx, vehicle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vehicle), args.Error(1)
}

func (m *MockVehicleRepository) GetVehicleByID(ctx context.Context, id string) (*models.Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vehicle), args.Error(1)
}

func (m *MockVehicleRepository) GetVehiclesByDriverID(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Vehicle), args.Error(1)
}

// VehicleService interface for testing
type VehicleService interface {
	RegisterVehicle(ctx context.Context, vehicle *models.Vehicle) (*models.Vehicle, error)
	GetVehicleByID(ctx context.Context, id string) (*models.Vehicle, error)
	GetDriverVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error)
}

// MockVehicleService for testing business logic
type MockVehicleService struct {
	repo *MockVehicleRepository
}

func NewMockVehicleService(repo *MockVehicleRepository) *MockVehicleService {
	return &MockVehicleService{repo: repo}
}

func (s *MockVehicleService) RegisterVehicle(ctx context.Context, vehicle *models.Vehicle) (*models.Vehicle, error) {
	// Validate vehicle input
	if vehicle.DriverID == "" {
		return nil, assert.AnError
	}
	if vehicle.LicensePlate == "" {
		return nil, assert.AnError
	}
	if vehicle.VehicleType == "" {
		return nil, assert.AnError
	}

	// Set default values
	vehicle.ID = "generated-vehicle-id"
	vehicle.Status = models.VehicleStatusActive
	vehicle.CreatedAt = time.Now()
	vehicle.UpdatedAt = time.Now()

	return s.repo.CreateVehicle(ctx, vehicle)
}

func (s *MockVehicleService) GetVehicleByID(ctx context.Context, id string) (*models.Vehicle, error) {
	if id == "" {
		return nil, assert.AnError
	}
	return s.repo.GetVehicleByID(ctx, id)
}

func (s *MockVehicleService) GetDriverVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	if driverID == "" {
		return nil, assert.AnError
	}
	return s.repo.GetVehiclesByDriverID(ctx, driverID)
}

func TestVehicleService_RegisterVehicle(t *testing.T) {
	tests := []struct {
		name          string
		vehicle       *models.Vehicle
		setupMock     func(*MockVehicleRepository)
		expectedError bool
	}{
		{
			name: "successful vehicle registration",
			vehicle: &models.Vehicle{
				DriverID:     "driver123",
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2020,
				Color:        "Blue",
				LicensePlate: "ABC123",
				VehicleType:  models.VehicleTypeSedan,
				Capacity:     4,
			},
			setupMock: func(m *MockVehicleRepository) {
				expectedVehicle := &models.Vehicle{
					ID:           "vehicle123",
					DriverID:     "driver123",
					Make:         "Toyota",
					Model:        "Camry",
					Year:         2020,
					Color:        "Blue",
					LicensePlate: "ABC123",
					VehicleType:  models.VehicleTypeSedan,
					Capacity:     4,
					Status:       models.VehicleStatusActive,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*models.Vehicle")).Return(expectedVehicle, nil)
			},
			expectedError: false,
		},
		{
			name: "empty driver ID error",
			vehicle: &models.Vehicle{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2020,
				Color:        "Blue",
				LicensePlate: "ABC123",
				VehicleType:  models.VehicleTypeSedan,
				Capacity:     4,
			},
			setupMock: func(m *MockVehicleRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
		},
		{
			name: "empty license plate error",
			vehicle: &models.Vehicle{
				DriverID:    "driver123",
				Make:        "Toyota",
				Model:       "Camry",
				Year:        2020,
				Color:       "Blue",
				VehicleType: models.VehicleTypeSedan,
				Capacity:    4,
			},
			setupMock: func(m *MockVehicleRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockVehicleRepository)
			tt.setupMock(mockRepo)

			vehicleService := NewMockVehicleService(mockRepo)
			ctx := context.Background()

			// Execute
			result, err := vehicleService.RegisterVehicle(ctx, tt.vehicle)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.vehicle.DriverID, result.DriverID)
				assert.Equal(t, tt.vehicle.Make, result.Make)
				assert.Equal(t, tt.vehicle.Model, result.Model)
				assert.Equal(t, tt.vehicle.LicensePlate, result.LicensePlate)
				assert.NotEmpty(t, result.ID)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestVehicleService_GetDriverVehicles(t *testing.T) {
	tests := []struct {
		name          string
		driverID      string
		setupMock     func(*MockVehicleRepository)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "successful vehicle retrieval",
			driverID: "driver123",
			setupMock: func(m *MockVehicleRepository) {
				vehicles := []*models.Vehicle{
					{
						ID:          "vehicle1",
						DriverID:    "driver123",
						Make:        "Toyota",
						Model:       "Camry",
						VehicleType: models.VehicleTypeSedan,
						Status:      models.VehicleStatusActive,
					},
					{
						ID:          "vehicle2",
						DriverID:    "driver123",
						Make:        "Honda",
						Model:       "Accord",
						VehicleType: models.VehicleTypeSedan,
						Status:      models.VehicleStatusActive,
					},
				}
				m.On("GetVehiclesByDriverID", mock.Anything, "driver123").Return(vehicles, nil)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "empty driver ID",
			driverID: "",
			setupMock: func(m *MockVehicleRepository) {
				// No mock setup needed as validation should fail before repository call
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:     "no vehicles found",
			driverID: "driver456",
			setupMock: func(m *MockVehicleRepository) {
				m.On("GetVehiclesByDriverID", mock.Anything, "driver456").Return([]*models.Vehicle{}, nil)
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockVehicleRepository)
			tt.setupMock(mockRepo)

			vehicleService := NewMockVehicleService(mockRepo)
			ctx := context.Background()

			// Execute
			result, err := vehicleService.GetDriverVehicles(ctx, tt.driverID)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkVehicleService_RegisterVehicle(b *testing.B) {
	mockRepo := new(MockVehicleRepository)
	vehicle := &models.Vehicle{
		DriverID:     "bench-driver",
		Make:         "Benchmark",
		Model:        "Test",
		Year:         2023,
		Color:        "Red",
		LicensePlate: "BENCH123",
		VehicleType:  models.VehicleTypeSedan,
		Capacity:     4,
	}

	expectedVehicle := &models.Vehicle{
		ID:           "vehicle123",
		DriverID:     "bench-driver",
		Make:         "Benchmark",
		Model:        "Test",
		Year:         2023,
		Color:        "Red",
		LicensePlate: "BENCH123",
		VehicleType:  models.VehicleTypeSedan,
		Capacity:     4,
		Status:       models.VehicleStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*models.Vehicle")).Return(expectedVehicle, nil)

	vehicleService := NewMockVehicleService(mockRepo)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vehicleService.RegisterVehicle(ctx, vehicle)
	}
}
