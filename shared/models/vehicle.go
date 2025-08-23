package models

import (
	"time"
)

// VehicleType represents the type of vehicle
type VehicleType string

const (
	VehicleTypeSedan     VehicleType = "sedan"
	VehicleTypeSUV       VehicleType = "suv"
	VehicleTypeHatchback VehicleType = "hatchback"
	VehicleTypeLuxury    VehicleType = "luxury"
	VehicleTypeVan       VehicleType = "van"
)

// VehicleStatus represents the current status of a vehicle
type VehicleStatus string

const (
	VehicleStatusInactive    VehicleStatus = "inactive"
	VehicleStatusActive      VehicleStatus = "active"
	VehicleStatusMaintenance VehicleStatus = "maintenance"
	VehicleStatusRetired     VehicleStatus = "retired"
)

// Vehicle represents a vehicle in the rideshare platform
type Vehicle struct {
	ID                    string        `json:"id" db:"id"`
	DriverID              string        `json:"driver_id" db:"driver_id"`
	Make                  string        `json:"make" db:"make"`
	Model                 string        `json:"model" db:"model"`
	Year                  int           `json:"year" db:"year"`
	Color                 string        `json:"color" db:"color"`
	LicensePlate          string        `json:"license_plate" db:"license_plate"`
	VehicleType           VehicleType   `json:"vehicle_type" db:"vehicle_type"`
	Status                VehicleStatus `json:"status" db:"status"`
	Capacity              int           `json:"capacity" db:"capacity"`
	InsurancePolicyNumber string        `json:"insurance_policy_number" db:"insurance_policy_number"`
	InsuranceExpiry       *time.Time    `json:"insurance_expiry" db:"insurance_expiry"`
	RegistrationExpiry    *time.Time    `json:"registration_expiry" db:"registration_expiry"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at" db:"updated_at"`
}

// NewVehicle creates a new vehicle with default values
func NewVehicle(driverID, make, model string, year int, color, licensePlate string, vehicleType VehicleType, capacity int) *Vehicle {
	return &Vehicle{
		ID:           generateID(),
		DriverID:     driverID,
		Make:         make,
		Model:        model,
		Year:         year,
		Color:        color,
		LicensePlate: licensePlate,
		VehicleType:  vehicleType,
		Status:       VehicleStatusActive,
		Capacity:     capacity,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// IsActive returns true if the vehicle is active
func (v *Vehicle) IsActive() bool {
	return v.Status == VehicleStatusActive
}

// IsAvailable returns true if the vehicle is available for trips
func (v *Vehicle) IsAvailable() bool {
	return v.Status == VehicleStatusActive
}

// UpdateStatus updates the vehicle's status
func (v *Vehicle) UpdateStatus(status VehicleStatus) {
	v.Status = status
	v.UpdatedAt = time.Now()
}

// SetInsuranceInfo sets the vehicle's insurance information
func (v *Vehicle) SetInsuranceInfo(policyNumber string, expiry time.Time) {
	v.InsurancePolicyNumber = policyNumber
	v.InsuranceExpiry = &expiry
	v.UpdatedAt = time.Now()
}

// SetRegistrationExpiry sets the vehicle's registration expiry
func (v *Vehicle) SetRegistrationExpiry(expiry time.Time) {
	v.RegistrationExpiry = &expiry
	v.UpdatedAt = time.Now()
}

// IsInsuranceValid checks if the vehicle's insurance is valid
func (v *Vehicle) IsInsuranceValid() bool {
	if v.InsuranceExpiry == nil {
		return false
	}
	return v.InsuranceExpiry.After(time.Now())
}

// IsRegistrationValid checks if the vehicle's registration is valid
func (v *Vehicle) IsRegistrationValid() bool {
	if v.RegistrationExpiry == nil {
		return false
	}
	return v.RegistrationExpiry.After(time.Now())
}

// IsValidForService checks if the vehicle is valid for service
func (v *Vehicle) IsValidForService() bool {
	return v.IsActive() && v.IsInsuranceValid() && v.IsRegistrationValid()
}

// GetDisplayName returns a display name for the vehicle
func (v *Vehicle) GetDisplayName() string {
	return v.Color + " " + v.Make + " " + v.Model
}

// GetVehicleTypeCapacity returns the default capacity for a vehicle type
func GetVehicleTypeCapacity(vehicleType VehicleType) int {
	switch vehicleType {
	case VehicleTypeSedan:
		return 4
	case VehicleTypeSUV:
		return 6
	case VehicleTypeHatchback:
		return 4
	case VehicleTypeLuxury:
		return 4
	case VehicleTypeVan:
		return 8
	default:
		return 4
	}
}

// IsValidVehicleType checks if a vehicle type is valid
func IsValidVehicleType(vehicleType string) bool {
	switch VehicleType(vehicleType) {
	case VehicleTypeSedan, VehicleTypeSUV, VehicleTypeHatchback, VehicleTypeLuxury, VehicleTypeVan:
		return true
	default:
		return false
	}
}

// GetVehicleTypes returns all valid vehicle types
func GetVehicleTypes() []VehicleType {
	return []VehicleType{
		VehicleTypeSedan,
		VehicleTypeSUV,
		VehicleTypeHatchback,
		VehicleTypeLuxury,
		VehicleTypeVan,
	}
}
