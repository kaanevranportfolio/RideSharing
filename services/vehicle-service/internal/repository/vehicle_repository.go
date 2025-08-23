package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/database"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// VehicleRepository handles vehicle data persistence
type VehicleRepository struct {
	db     *database.PostgresDB
	logger *logger.Logger
}

// NewVehicleRepository creates a new vehicle repository
func NewVehicleRepository(db *database.PostgresDB, log *logger.Logger) *VehicleRepository {
	return &VehicleRepository{
		db:     db,
		logger: log,
	}
}

// Create creates a new vehicle
func (r *VehicleRepository) Create(ctx context.Context, vehicle *models.Vehicle) error {
	query := `
		INSERT INTO vehicles (id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		vehicle.ID, vehicle.DriverID, vehicle.Make, vehicle.Model, vehicle.Year,
		vehicle.Color, vehicle.LicensePlate, vehicle.VehicleType, vehicle.Status,
		vehicle.Capacity, vehicle.InsurancePolicyNumber,
		vehicle.InsuranceExpiry, vehicle.RegistrationExpiry,
		vehicle.CreatedAt, vehicle.UpdatedAt,
	)

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id":    vehicle.ID,
			"driver_id":     vehicle.DriverID,
			"license_plate": vehicle.LicensePlate,
		}).Error("Failed to create vehicle")
		return fmt.Errorf("failed to create vehicle: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id":    vehicle.ID,
		"driver_id":     vehicle.DriverID,
		"license_plate": vehicle.LicensePlate,
		"make":          vehicle.Make,
		"model":         vehicle.Model,
	}).Info("Vehicle created successfully")

	return nil
}

// GetByID retrieves a vehicle by ID
func (r *VehicleRepository) GetByID(ctx context.Context, id string) (*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE id = $1
	`

	vehicle := &models.Vehicle{}

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
		&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
		&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
		&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
		&vehicle.CreatedAt, &vehicle.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vehicle not found: %s", id)
		}
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": id,
		}).Error("Failed to get vehicle by ID")
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	return vehicle, nil
}

// GetByDriverID retrieves vehicles by driver ID
func (r *VehicleRepository) GetByDriverID(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE driver_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, driverID)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
		}).Error("Failed to get vehicles by driver ID")
		return nil, fmt.Errorf("failed to get vehicles by driver ID: %w", err)
	}
	defer rows.Close()

	var vehicles []*models.Vehicle
	for rows.Next() {
		vehicle := &models.Vehicle{}

		err := rows.Scan(
			&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
			&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
			&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
			&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
			&vehicle.CreatedAt, &vehicle.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error("Failed to scan vehicle row")
			return nil, fmt.Errorf("failed to scan vehicle: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vehicles: %w", err)
	}

	return vehicles, nil
}

// GetByLicensePlate retrieves a vehicle by license plate
func (r *VehicleRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE license_plate = $1
	`

	vehicle := &models.Vehicle{}

	err := r.db.QueryRowContext(ctx, query, licensePlate).Scan(
		&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
		&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
		&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
		&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
		&vehicle.CreatedAt, &vehicle.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vehicle not found: %s", licensePlate)
		}
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"license_plate": licensePlate,
		}).Error("Failed to get vehicle by license plate")
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	return vehicle, nil
}

// Update updates a vehicle
func (r *VehicleRepository) Update(ctx context.Context, vehicle *models.Vehicle) error {
	query := `
		UPDATE vehicles
		SET driver_id = $2, make = $3, model = $4, year = $5, color = $6,
			license_plate = $7, vehicle_type = $8, status = $9, capacity = $10,
			insurance_policy_number = $11, insurance_expiry = $12,
			registration_expiry = $13, updated_at = $14
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		vehicle.ID, vehicle.DriverID, vehicle.Make, vehicle.Model, vehicle.Year,
		vehicle.Color, vehicle.LicensePlate, vehicle.VehicleType, vehicle.Status,
		vehicle.Capacity, vehicle.InsurancePolicyNumber,
		vehicle.InsuranceExpiry, vehicle.RegistrationExpiry, vehicle.UpdatedAt,
	)

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": vehicle.ID,
		}).Error("Failed to update vehicle")
		return fmt.Errorf("failed to update vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found: %s", vehicle.ID)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicle.ID,
	}).Info("Vehicle updated successfully")

	return nil
}

// UpdateStatus updates vehicle status
func (r *VehicleRepository) UpdateStatus(ctx context.Context, id string, status models.VehicleStatus) error {
	query := `
		UPDATE vehicles
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": id,
			"status":     status,
		}).Error("Failed to update vehicle status")
		return fmt.Errorf("failed to update vehicle status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found: %s", id)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": id,
		"status":     status,
	}).Info("Vehicle status updated successfully")

	return nil
}

// Delete soft deletes a vehicle
func (r *VehicleRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE vehicles
		SET status = 'inactive', updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": id,
		}).Error("Failed to delete vehicle")
		return fmt.Errorf("failed to delete vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found: %s", id)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": id,
	}).Info("Vehicle deleted successfully")

	return nil
}

// List retrieves vehicles with pagination and filtering
func (r *VehicleRepository) List(ctx context.Context, limit, offset int, status string, vehicleType string) ([]*models.Vehicle, error) {
	var query string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE 1=1
	`

	conditions := ""
	if status != "" {
		conditions += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if vehicleType != "" {
		conditions += fmt.Sprintf(" AND vehicle_type = $%d", argIndex)
		args = append(args, vehicleType)
		argIndex++
	}

	query = baseQuery + conditions + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to list vehicles")
		return nil, fmt.Errorf("failed to list vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*models.Vehicle
	for rows.Next() {
		vehicle := &models.Vehicle{}

		err := rows.Scan(
			&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
			&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
			&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
			&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
			&vehicle.CreatedAt, &vehicle.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error("Failed to scan vehicle row")
			return nil, fmt.Errorf("failed to scan vehicle: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vehicles: %w", err)
	}

	return vehicles, nil
}

// Count counts total vehicles with filtering
func (r *VehicleRepository) Count(ctx context.Context, status string, vehicleType string) (int64, error) {
	var query string
	var args []interface{}
	argIndex := 1

	baseQuery := "SELECT COUNT(*) FROM vehicles WHERE 1=1"
	conditions := ""

	if status != "" {
		conditions += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if vehicleType != "" {
		conditions += fmt.Sprintf(" AND vehicle_type = $%d", argIndex)
		args = append(args, vehicleType)
		argIndex++
	}

	query = baseQuery + conditions

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count vehicles")
		return 0, fmt.Errorf("failed to count vehicles: %w", err)
	}

	return count, nil
}

// GetAvailableVehicles retrieves available vehicles for a driver
func (r *VehicleRepository) GetAvailableVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE driver_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, driverID)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id": driverID,
		}).Error("Failed to get available vehicles")
		return nil, fmt.Errorf("failed to get available vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*models.Vehicle
	for rows.Next() {
		vehicle := &models.Vehicle{}

		err := rows.Scan(
			&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
			&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
			&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
			&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
			&vehicle.CreatedAt, &vehicle.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error("Failed to scan available vehicle row")
			return nil, fmt.Errorf("failed to scan available vehicle: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating available vehicles: %w", err)
	}

	return vehicles, nil
}

// GetVehiclesWithExpiredInsurance retrieves vehicles with expired insurance
func (r *VehicleRepository) GetVehiclesWithExpiredInsurance(ctx context.Context) ([]*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE insurance_expiry IS NOT NULL 
			AND insurance_expiry <= $1
			AND status != 'inactive'
		ORDER BY insurance_expiry ASC
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get vehicles with expired insurance")
		return nil, fmt.Errorf("failed to get vehicles with expired insurance: %w", err)
	}
	defer rows.Close()

	var vehicles []*models.Vehicle
	for rows.Next() {
		vehicle := &models.Vehicle{}

		err := rows.Scan(
			&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
			&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
			&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
			&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
			&vehicle.CreatedAt, &vehicle.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error("Failed to scan expired insurance vehicle row")
			return nil, fmt.Errorf("failed to scan expired insurance vehicle: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired insurance vehicles: %w", err)
	}

	return vehicles, nil
}

// GetVehiclesWithExpiredRegistration retrieves vehicles with expired registration
func (r *VehicleRepository) GetVehiclesWithExpiredRegistration(ctx context.Context) ([]*models.Vehicle, error) {
	query := `
		SELECT id, driver_id, make, model, year, color, license_plate,
			vehicle_type, status, capacity, insurance_policy_number,
			insurance_expiry, registration_expiry, created_at, updated_at
		FROM vehicles
		WHERE registration_expiry IS NOT NULL 
			AND registration_expiry <= $1
			AND status != 'inactive'
		ORDER BY registration_expiry ASC
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get vehicles with expired registration")
		return nil, fmt.Errorf("failed to get vehicles with expired registration: %w", err)
	}
	defer rows.Close()

	var vehicles []*models.Vehicle
	for rows.Next() {
		vehicle := &models.Vehicle{}

		err := rows.Scan(
			&vehicle.ID, &vehicle.DriverID, &vehicle.Make, &vehicle.Model, &vehicle.Year,
			&vehicle.Color, &vehicle.LicensePlate, &vehicle.VehicleType, &vehicle.Status,
			&vehicle.Capacity, &vehicle.InsurancePolicyNumber,
			&vehicle.InsuranceExpiry, &vehicle.RegistrationExpiry,
			&vehicle.CreatedAt, &vehicle.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error("Failed to scan expired registration vehicle row")
			return nil, fmt.Errorf("failed to scan expired registration vehicle: %w", err)
		}

		vehicles = append(vehicles, vehicle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired registration vehicles: %w", err)
	}

	return vehicles, nil
}

// LicensePlateExists checks if a license plate already exists
func (r *VehicleRepository) LicensePlateExists(ctx context.Context, licensePlate string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM vehicles WHERE license_plate = $1)"

	var exists bool
	err := r.db.QueryRowContext(ctx, query, licensePlate).Scan(&exists)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"license_plate": licensePlate,
		}).Error("Failed to check license plate existence")
		return false, fmt.Errorf("failed to check license plate existence: %w", err)
	}

	return exists, nil
}

// UpdateInsuranceInfo updates vehicle insurance information
func (r *VehicleRepository) UpdateInsuranceInfo(ctx context.Context, id, policyNumber string, expiry time.Time) error {
	query := `
		UPDATE vehicles
		SET insurance_policy_number = $2, insurance_expiry = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, policyNumber, expiry, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": id,
		}).Error("Failed to update vehicle insurance info")
		return fmt.Errorf("failed to update insurance info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found: %s", id)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id":    id,
		"policy_number": policyNumber,
		"expiry":        expiry,
	}).Info("Vehicle insurance info updated successfully")

	return nil
}

// UpdateRegistrationExpiry updates vehicle registration expiry
func (r *VehicleRepository) UpdateRegistrationExpiry(ctx context.Context, id string, expiry time.Time) error {
	query := `
		UPDATE vehicles
		SET registration_expiry = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, expiry, time.Now())
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"vehicle_id": id,
		}).Error("Failed to update vehicle registration expiry")
		return fmt.Errorf("failed to update registration expiry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found: %s", id)
	}

	r.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": id,
		"expiry":     expiry,
	}).Info("Vehicle registration expiry updated successfully")

	return nil
}
