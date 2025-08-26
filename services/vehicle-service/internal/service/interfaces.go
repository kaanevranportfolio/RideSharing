package service

import (
	"context"

	"github.com/rideshare-platform/shared/models"
)

// VehicleRepositoryInterface defines the interface for vehicle repository operations
type VehicleRepositoryInterface interface {
	Create(ctx context.Context, vehicle *models.Vehicle) error
	GetByID(ctx context.Context, vehicleID string) (*models.Vehicle, error)
	GetByDriverID(ctx context.Context, driverID string) ([]*models.Vehicle, error)
	Update(ctx context.Context, vehicle *models.Vehicle) error
	Delete(ctx context.Context, vehicleID string) error
	LicensePlateExists(ctx context.Context, licensePlate string) (bool, error)
	GetAvailableVehicles(ctx context.Context, vehicleType string, lat, lng float64, radius float64) ([]*models.Vehicle, error)

	// Additional methods needed by the service
	UpdateStatus(ctx context.Context, vehicleID string, status models.VehicleStatus) error
	List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]*models.Vehicle, error)
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
	GetVehiclesWithExpiredInsurance(ctx context.Context) ([]*models.Vehicle, error)
	GetVehiclesWithExpiredRegistration(ctx context.Context) ([]*models.Vehicle, error)
}
