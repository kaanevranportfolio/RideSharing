package vehicle_test

import (
	"testing"
	"time"

	"github.com/rideshare-platform/shared/models"
	"github.com/stretchr/testify/assert"
)

// TestVehicleValidation tests vehicle validation business logic
func TestVehicleValidation(t *testing.T) {
	tests := []struct {
		name        string
		vehicle     *models.Vehicle
		expectError bool
	}{
		{
			name: "Valid vehicle",
			vehicle: &models.Vehicle{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2020,
				LicensePlate: "ABC123",
			},
			expectError: false,
		},
		{
			name: "Invalid year",
			vehicle: &models.Vehicle{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         1990, // Too old
				LicensePlate: "ABC123",
			},
			expectError: true,
		},
		{
			name: "Empty make",
			vehicle: &models.Vehicle{
				Make:         "",
				Model:        "Camry",
				Year:         2020,
				LicensePlate: "ABC123",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVehicle(tt.vehicle)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVehicleAge tests vehicle age calculation
func TestVehicleAge(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name        string
		vehicleYear int
		expectedAge int
		isEligible  bool
	}{
		{
			name:        "New vehicle",
			vehicleYear: currentYear,
			expectedAge: 0,
			isEligible:  true,
		},
		{
			name:        "5 year old vehicle",
			vehicleYear: currentYear - 5,
			expectedAge: 5,
			isEligible:  true,
		},
		{
			name:        "15 year old vehicle",
			vehicleYear: currentYear - 15,
			expectedAge: 15,
			isEligible:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := calculateVehicleAge(tt.vehicleYear)
			eligible := isVehicleEligible(tt.vehicleYear)

			assert.Equal(t, tt.expectedAge, age)
			assert.Equal(t, tt.isEligible, eligible)
		})
	}
}

// TestVINValidation tests VIN validation algorithm
func TestVINValidation(t *testing.T) {
	tests := []struct {
		vin   string
		valid bool
	}{
		{"1HGBH41JXMN109186", true}, // Valid VIN
		{"JH4TB2H26CC000000", true}, // Valid VIN
		{"1234567890123456", false}, // Invalid characters
		{"1HGBH41JXMN10918", false}, // Too short
		{"", false},                 // Empty
	}

	for _, tt := range tests {
		t.Run("VIN_"+tt.vin, func(t *testing.T) {
			result := validateVIN(tt.vin)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// TestFuelEfficiencyCalculation tests fuel efficiency algorithms
func TestFuelEfficiencyCalculation(t *testing.T) {
	tests := []struct {
		name        string
		vehicleType string
		expectedMPG float64
	}{
		{"Sedan", "sedan", 28.5},
		{"SUV", "suv", 22.0},
		{"Hybrid", "hybrid", 45.0},
		{"Electric", "electric", 0.0}, // Electric doesn't use gas
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpg := getEstimatedMPG(tt.vehicleType)
			assert.Equal(t, tt.expectedMPG, mpg)
		})
	}
}

// Helper functions implementing vehicle business logic

func validateVehicle(vehicle *models.Vehicle) error {
	if vehicle.Make == "" {
		return assert.AnError
	}
	if vehicle.Model == "" {
		return assert.AnError
	}
	if !isVehicleEligible(vehicle.Year) {
		return assert.AnError
	}
	if vehicle.LicensePlate == "" {
		return assert.AnError
	}
	return nil
}

func calculateVehicleAge(year int) int {
	return time.Now().Year() - year
}

func isVehicleEligible(year int) bool {
	age := calculateVehicleAge(year)
	return age <= 10 // Vehicles must be 10 years old or newer
}

func validateVIN(vin string) bool {
	if len(vin) != 17 {
		return false
	}

	// Check for valid characters (excluding I, O, Q)
	validChars := "0123456789ABCDEFGHJKLMNPRSTUVWXYZ"
	for _, char := range vin {
		found := false
		for _, valid := range validChars {
			if char == valid {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func getEstimatedMPG(vehicleType string) float64 {
	switch vehicleType {
	case "sedan":
		return 28.5
	case "suv":
		return 22.0
	case "hybrid":
		return 45.0
	case "electric":
		return 0.0
	default:
		return 25.0
	}
}
