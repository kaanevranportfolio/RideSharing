package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
)

var ErrVehicleNotFound = errors.New("vehicle not found")

// MockVehicleRepository provides a complete test implementation
type MockVehicleRepository struct {
	vehicles map[string]*models.Vehicle
	drivers  map[string][]*models.Vehicle
}

func NewMockVehicleRepository() *MockVehicleRepository {
	return &MockVehicleRepository{
		vehicles: make(map[string]*models.Vehicle),
		drivers:  make(map[string][]*models.Vehicle),
	}
}

func (m *MockVehicleRepository) Create(ctx context.Context, vehicle *models.Vehicle) error {
	m.vehicles[vehicle.ID] = vehicle
	m.drivers[vehicle.DriverID] = append(m.drivers[vehicle.DriverID], vehicle)
	return nil
}

func (m *MockVehicleRepository) GetByID(ctx context.Context, id string) (*models.Vehicle, error) {
	vehicle, exists := m.vehicles[id]
	if !exists {
		return nil, ErrVehicleNotFound
	}
	return vehicle, nil
}

func (m *MockVehicleRepository) GetByDriverID(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	return m.drivers[driverID], nil
}

func (m *MockVehicleRepository) UpdateStatus(ctx context.Context, id string, status models.VehicleStatus) error {
	if vehicle, exists := m.vehicles[id]; exists {
		vehicle.Status = status
		return nil
	}
	return ErrVehicleNotFound
}

func (m *MockVehicleRepository) Update(ctx context.Context, vehicle *models.Vehicle) error {
	m.vehicles[vehicle.ID] = vehicle
	return nil
}

func (m *MockVehicleRepository) Delete(ctx context.Context, id string) error {
	delete(m.vehicles, id)
	return nil
}

func (m *MockVehicleRepository) List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]*models.Vehicle, error) {
	var result []*models.Vehicle
	for _, vehicle := range m.vehicles {
		result = append(result, vehicle)
	}
	return result, nil
}

func (m *MockVehicleRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	return int64(len(m.vehicles)), nil
}

func (m *MockVehicleRepository) LicensePlateExists(ctx context.Context, licensePlate string) (bool, error) {
	for _, vehicle := range m.vehicles {
		if vehicle.LicensePlate == licensePlate {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockVehicleRepository) GetAvailableVehicles(ctx context.Context, vehicleType string, lat, lng float64, radius float64) ([]*models.Vehicle, error) {
	var result []*models.Vehicle
	for _, vehicle := range m.vehicles {
		if vehicle.Status == models.VehicleStatusActive {
			if vehicleType == "" || string(vehicle.VehicleType) == vehicleType {
				result = append(result, vehicle)
			}
		}
	}
	return result, nil
}

func (m *MockVehicleRepository) GetVehiclesWithExpiredInsurance(ctx context.Context) ([]*models.Vehicle, error) {
	var result []*models.Vehicle
	now := time.Now()
	for _, vehicle := range m.vehicles {
		if vehicle.InsuranceExpiry != nil && vehicle.InsuranceExpiry.Before(now) {
			result = append(result, vehicle)
		}
	}
	return result, nil
}

func (m *MockVehicleRepository) GetVehiclesWithExpiredRegistration(ctx context.Context) ([]*models.Vehicle, error) {
	var result []*models.Vehicle
	now := time.Now()
	for _, vehicle := range m.vehicles {
		if vehicle.RegistrationExpiry != nil && vehicle.RegistrationExpiry.Before(now) {
			result = append(result, vehicle)
		}
	}
	return result, nil
}

// Test functions with comprehensive coverage
func TestVehicleService_CreateVehicle(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo: repo,
		// Use nil for cache and event publisher to avoid null pointer issues
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	tests := []struct {
		name    string
		request *CreateVehicleRequest
		wantErr bool
	}{
		{
			name: "successful vehicle creation",
			request: &CreateVehicleRequest{
				DriverID:     "driver-1",
				Make:         "Toyota",
				Model:        "Prius",
				Year:         2022,
				Color:        "White",
				LicensePlate: "ABC123",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: false,
		},
		{
			name: "empty driver ID error",
			request: &CreateVehicleRequest{
				DriverID:     "",
				Make:         "Toyota",
				Model:        "Prius",
				Year:         2022,
				Color:        "White",
				LicensePlate: "ABC124",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: true,
		},
		{
			name: "empty license plate error",
			request: &CreateVehicleRequest{
				DriverID:     "driver-1",
				Make:         "Toyota",
				Model:        "Prius",
				Year:         2022,
				Color:        "White",
				LicensePlate: "",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateVehicle(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVehicle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_GetVehicle(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create a test vehicle
	vehicle := models.NewVehicle(
		"driver-1",
		"Toyota",
		"Prius",
		2022,
		"White",
		"ABC123",
		models.VehicleTypeSedan,
		4,
	)
	repo.Create(context.Background(), vehicle)

	tests := []struct {
		name      string
		vehicleID string
		wantErr   bool
	}{
		{
			name:      "successful vehicle retrieval",
			vehicleID: vehicle.ID,
			wantErr:   false,
		},
		{
			name:      "vehicle not found",
			vehicleID: "non-existent",
			wantErr:   true,
		},
		{
			name:      "empty vehicle ID",
			vehicleID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetVehicle(context.Background(), tt.vehicleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVehicle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_GetVehiclesByDriver(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create test vehicles
	vehicle1 := models.NewVehicle("driver-1", "Toyota", "Prius", 2022, "White", "ABC123", models.VehicleTypeSedan, 4)
	vehicle2 := models.NewVehicle("driver-1", "Honda", "Civic", 2021, "Blue", "DEF456", models.VehicleTypeSedan, 4)
	repo.Create(context.Background(), vehicle1)
	repo.Create(context.Background(), vehicle2)

	tests := []struct {
		name     string
		driverID string
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "successful driver vehicles retrieval",
			driverID: "driver-1",
			wantErr:  false,
			wantLen:  2,
		},
		{
			name:     "empty driver ID",
			driverID: "",
			wantErr:  true,
			wantLen:  0,
		},
		{
			name:     "no vehicles found",
			driverID: "driver-2",
			wantErr:  false,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vehicles, err := service.GetVehiclesByDriver(context.Background(), tt.driverID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVehiclesByDriver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(vehicles) != tt.wantLen {
				t.Errorf("GetVehiclesByDriver() len = %v, want %v", len(vehicles), tt.wantLen)
			}
		})
	}
}

func TestVehicleService_GetAvailableVehicles(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create test vehicles with different statuses
	vehicle1 := models.NewVehicle("driver-1", "Toyota", "Prius", 2022, "White", "ABC123", models.VehicleTypeSedan, 4)
	vehicle1.Status = models.VehicleStatusActive
	vehicle2 := models.NewVehicle("driver-1", "Honda", "Civic", 2021, "Blue", "DEF456", models.VehicleTypeSedan, 4)
	vehicle2.Status = models.VehicleStatusMaintenance
	repo.Create(context.Background(), vehicle1)
	repo.Create(context.Background(), vehicle2)

	tests := []struct {
		name     string
		driverID string
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "successful available vehicles retrieval",
			driverID: "driver-1",
			wantErr:  false,
			wantLen:  1, // Only vehicle1 is active
		},
		{
			name:     "empty driver ID",
			driverID: "",
			wantErr:  true,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vehicles, err := service.GetAvailableVehicles(context.Background(), tt.driverID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvailableVehicles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(vehicles) != tt.wantLen {
				t.Errorf("GetAvailableVehicles() len = %v, want %v", len(vehicles), tt.wantLen)
			}
		})
	}
}

func TestVehicleService_UpdateVehicleStatus(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create a test vehicle
	vehicle := models.NewVehicle("driver-1", "Toyota", "Prius", 2022, "White", "ABC123", models.VehicleTypeSedan, 4)
	repo.Create(context.Background(), vehicle)

	tests := []struct {
		name      string
		vehicleID string
		status    models.VehicleStatus
		wantErr   bool
	}{
		{
			name:      "successful status update",
			vehicleID: vehicle.ID,
			status:    models.VehicleStatusMaintenance,
			wantErr:   false,
		},
		{
			name:      "empty vehicle ID",
			vehicleID: "",
			status:    models.VehicleStatusActive,
			wantErr:   true,
		},
		{
			name:      "non-existent vehicle",
			vehicleID: "non-existent",
			status:    models.VehicleStatusActive,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateVehicleStatus(context.Background(), tt.vehicleID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateVehicleStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_DeleteVehicle(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create a test vehicle
	vehicle := models.NewVehicle("driver-1", "Toyota", "Prius", 2022, "White", "ABC123", models.VehicleTypeSedan, 4)
	repo.Create(context.Background(), vehicle)

	tests := []struct {
		name      string
		vehicleID string
		wantErr   bool
	}{
		{
			name:      "successful vehicle deletion",
			vehicleID: vehicle.ID,
			wantErr:   false,
		},
		{
			name:      "empty vehicle ID",
			vehicleID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteVehicle(context.Background(), tt.vehicleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteVehicle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_LicensePlateValidation(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	// Create a vehicle with a license plate
	vehicle := models.NewVehicle("driver-1", "Toyota", "Prius", 2022, "White", "ABC123", models.VehicleTypeSedan, 4)
	repo.Create(context.Background(), vehicle)

	tests := []struct {
		name    string
		request *CreateVehicleRequest
		wantErr bool
	}{
		{
			name: "duplicate license plate",
			request: &CreateVehicleRequest{
				DriverID:     "driver-2",
				Make:         "Honda",
				Model:        "Civic",
				Year:         2021,
				Color:        "Blue",
				LicensePlate: "ABC123", // Same as existing vehicle
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: true,
		},
		{
			name: "unique license plate",
			request: &CreateVehicleRequest{
				DriverID:     "driver-2",
				Make:         "Honda",
				Model:        "Civic",
				Year:         2021,
				Color:        "Blue",
				LicensePlate: "XYZ789",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateVehicle(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVehicle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_VehicleTypeValidation(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo:    repo,
		cacheRepo:      nil,
		eventPublisher: nil,
		logger:         nil,
	}

	tests := []struct {
		name        string
		vehicleType string
		wantErr     bool
	}{
		{
			name:        "valid sedan type",
			vehicleType: string(models.VehicleTypeSedan),
			wantErr:     false,
		},
		{
			name:        "valid SUV type",
			vehicleType: string(models.VehicleTypeSUV),
			wantErr:     false,
		},
		{
			name:        "valid luxury type",
			vehicleType: string(models.VehicleTypeLuxury),
			wantErr:     false,
		},
		{
			name:        "empty vehicle type",
			vehicleType: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &CreateVehicleRequest{
				DriverID:     "driver-1",
				Make:         "Toyota",
				Model:        "Prius",
				Year:         2022,
				Color:        "White",
				LicensePlate: "TEST" + tt.vehicleType, // Make unique
				VehicleType:  tt.vehicleType,
				Capacity:     4,
			}
			_, err := service.CreateVehicle(context.Background(), request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVehicle() with vehicleType %s, error = %v, wantErr %v", tt.vehicleType, err, tt.wantErr)
			}
		})
	}
}

func TestVehicleService_ValidateCreateVehicleRequest(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo: repo,
	}

	tests := []struct {
		name    string
		req     *CreateVehicleRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid Request",
			req: &CreateVehicleRequest{
				DriverID:     "driver123",
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				LicensePlate: "ABC123",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: false,
		},
		{
			name: "Missing Driver ID",
			req: &CreateVehicleRequest{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				LicensePlate: "ABC123",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: true,
			errMsg:  "driver ID is required",
		},
		{
			name: "Missing Make",
			req: &CreateVehicleRequest{
				DriverID:     "driver123",
				Model:        "Camry",
				Year:         2022,
				LicensePlate: "ABC123",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     4,
			},
			wantErr: true,
			errMsg:  "make is required",
		},
		{
			name: "Invalid Vehicle Type",
			req: &CreateVehicleRequest{
				DriverID:     "driver123",
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				LicensePlate: "ABC123",
				VehicleType:  "invalid",
				Capacity:     4,
			},
			wantErr: true,
			errMsg:  "invalid vehicle type",
		},
		{
			name: "Invalid Capacity - Zero",
			req: &CreateVehicleRequest{
				DriverID:     "driver123",
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				LicensePlate: "ABC123",
				VehicleType:  string(models.VehicleTypeSedan),
				Capacity:     0,
			},
			wantErr: true,
			errMsg:  "capacity must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCreateVehicleRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreateVehicleRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateCreateVehicleRequest() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestVehicleService_ValidateUpdateVehicleRequest(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo: repo,
	}

	tests := []struct {
		name    string
		req     *UpdateVehicleRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid Request",
			req: &UpdateVehicleRequest{
				ID:          "vehicle123",
				Make:        "Honda",
				Model:       "Civic",
				Year:        2023,
				VehicleType: "sedan",
				Capacity:    4,
			},
			wantErr: false,
		},
		{
			name: "Missing Vehicle ID",
			req: &UpdateVehicleRequest{
				Make:        "Honda",
				Model:       "Civic",
				Year:        2023,
				VehicleType: string(models.VehicleTypeSedan),
				Capacity:    4,
			},
			wantErr: true,
			errMsg:  "vehicle ID is required",
		},
		{
			name: "Invalid Vehicle Type",
			req: &UpdateVehicleRequest{
				ID:          "vehicle123",
				VehicleType: "spaceship",
			},
			wantErr: true,
			errMsg:  "invalid vehicle type: spaceship",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateUpdateVehicleRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUpdateVehicleRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || err.Error() != tt.errMsg {
					t.Errorf("validateUpdateVehicleRequest() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestVehicleService_ValidateListVehiclesRequest(t *testing.T) {
	repo := NewMockVehicleRepository()
	service := &VehicleService{
		vehicleRepo: repo,
	}

	tests := []struct {
		name        string
		req         *ListVehiclesRequest
		expectedReq *ListVehiclesRequest
	}{
		{
			name: "Valid Request",
			req: &ListVehiclesRequest{
				Limit:  10,
				Offset: 0,
			},
			expectedReq: &ListVehiclesRequest{
				Limit:  10,
				Offset: 0,
			},
		},
		{
			name: "Limit Too High - Should Cap at 100",
			req: &ListVehiclesRequest{
				Limit:  200,
				Offset: 0,
			},
			expectedReq: &ListVehiclesRequest{
				Limit:  100,
				Offset: 0,
			},
		},
		{
			name: "Negative Offset - Should Reset to 0",
			req: &ListVehiclesRequest{
				Limit:  10,
				Offset: -5,
			},
			expectedReq: &ListVehiclesRequest{
				Limit:  10,
				Offset: 0,
			},
		},
		{
			name: "Zero Limit - Should Set Default",
			req: &ListVehiclesRequest{
				Limit:  0,
				Offset: 10,
			},
			expectedReq: &ListVehiclesRequest{
				Limit:  20,
				Offset: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateListVehiclesRequest(tt.req)
			if err != nil {
				t.Errorf("validateListVehiclesRequest() unexpected error = %v", err)
				return
			}
			if tt.req.Limit != tt.expectedReq.Limit {
				t.Errorf("validateListVehiclesRequest() limit = %v, want %v", tt.req.Limit, tt.expectedReq.Limit)
			}
			if tt.req.Offset != tt.expectedReq.Offset {
				t.Errorf("validateListVehiclesRequest() offset = %v, want %v", tt.req.Offset, tt.expectedReq.Offset)
			}
		})
	}
}
