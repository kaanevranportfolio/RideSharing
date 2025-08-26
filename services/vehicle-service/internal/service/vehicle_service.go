package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rideshare-platform/services/vehicle-service/internal/repository"
	"github.com/rideshare-platform/shared/events"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// VehicleService handles vehicle business logic
type VehicleService struct {
	vehicleRepo    VehicleRepositoryInterface
	cacheRepo      *repository.CacheRepository
	eventPublisher *events.EventPublisher
	logger         *logger.Logger
}

// NewVehicleService creates a new vehicle service
func NewVehicleService(
	vehicleRepo VehicleRepositoryInterface,
	cacheRepo *repository.CacheRepository,
	eventPublisher *events.EventPublisher,
	logger *logger.Logger,
) *VehicleService {
	return &VehicleService{
		vehicleRepo:    vehicleRepo,
		cacheRepo:      cacheRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateVehicle creates a new vehicle
func (s *VehicleService) CreateVehicle(ctx context.Context, req *CreateVehicleRequest) (*models.Vehicle, error) {
	// Validate request
	if err := s.validateCreateVehicleRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if license plate already exists
	exists, err := s.vehicleRepo.LicensePlateExists(ctx, req.LicensePlate)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to check license plate existence")
		return nil, fmt.Errorf("failed to check license plate: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("license plate already exists: %s", req.LicensePlate)
	}

	// Create vehicle
	vehicle := models.NewVehicle(
		req.DriverID,
		req.Make,
		req.Model,
		req.Year,
		req.Color,
		req.LicensePlate,
		models.VehicleType(req.VehicleType),
		req.Capacity,
	)

	// Set insurance info if provided
	if req.InsurancePolicyNumber != "" && req.InsuranceExpiry != nil {
		vehicle.SetInsuranceInfo(req.InsurancePolicyNumber, *req.InsuranceExpiry)
	}

	// Set registration expiry if provided
	if req.RegistrationExpiry != nil {
		vehicle.SetRegistrationExpiry(*req.RegistrationExpiry)
	}

	// Save to database
	if err := s.vehicleRepo.Create(ctx, vehicle); err != nil {
		s.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"driver_id":     req.DriverID,
			"license_plate": req.LicensePlate,
		}).Error("Failed to create vehicle")
		return nil, fmt.Errorf("failed to create vehicle: %w", err)
	}

	// Cache the vehicle (skip if no cache available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.CacheVehicle(ctx, vehicle, 1*time.Hour); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to cache vehicle")
		}
	}

	// Invalidate driver vehicles cache (skip if no cache available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.InvalidateDriverVehicles(ctx, req.DriverID); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate driver vehicles cache")
		}
	}

	// Publish event (skip if no publisher available)
	if s.eventPublisher != nil {
		event := events.NewEvent(
			events.VehicleRegisteredEvent,
			vehicle.ID,
			1,
			map[string]interface{}{
				"vehicle_id":    vehicle.ID,
				"driver_id":     vehicle.DriverID,
				"license_plate": vehicle.LicensePlate,
				"make":          vehicle.Make,
				"model":         vehicle.Model,
				"vehicle_type":  vehicle.VehicleType,
			},
			"vehicle-service",
		)

		if err := s.eventPublisher.PublishEvent(ctx, event); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to publish vehicle registered event")
		}
	}

	if s.logger != nil {
		s.logger.WithContext(ctx).WithFields(logger.Fields{
			"vehicle_id":    vehicle.ID,
			"driver_id":     vehicle.DriverID,
			"license_plate": vehicle.LicensePlate,
		}).Info("Vehicle created successfully")
	}

	return vehicle, nil
}

// GetVehicle retrieves a vehicle by ID
func (s *VehicleService) GetVehicle(ctx context.Context, id string) (*models.Vehicle, error) {
	if id == "" {
		return nil, fmt.Errorf("vehicle ID is required")
	}

	// Try cache first (skip if no cache available)
	var vehicle *models.Vehicle
	if s.cacheRepo != nil {
		var err error
		vehicle, err = s.cacheRepo.GetCachedVehicle(ctx, id)
		if err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to get vehicle from cache")
		}

		if vehicle != nil {
			return vehicle, nil
		}
	}

	// Get from database
	vehicle, err := s.vehicleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Cache the result (skip if no cache available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.CacheVehicle(ctx, vehicle, 1*time.Hour); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to cache vehicle")
		}
	}

	return vehicle, nil
}

// GetVehiclesByDriver retrieves vehicles for a driver
func (s *VehicleService) GetVehiclesByDriver(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	if driverID == "" {
		return nil, fmt.Errorf("driver ID is required")
	}

	// Try cache first (skip if no cache available)
	var vehicles []*models.Vehicle
	if s.cacheRepo != nil {
		var err error
		vehicles, err = s.cacheRepo.GetCachedDriverVehicles(ctx, driverID)
		if err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to get driver vehicles from cache")
		}

		if vehicles != nil {
			return vehicles, nil
		}
	}

	// Get from database
	vehicles, err := s.vehicleRepo.GetByDriverID(ctx, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicles by driver: %w", err)
	}

	// Cache the result (skip if no cache available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.CacheDriverVehicles(ctx, driverID, vehicles, 30*time.Minute); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to cache driver vehicles")
		}
	}

	return vehicles, nil
}

// GetAvailableVehicles retrieves available vehicles for a driver
func (s *VehicleService) GetAvailableVehicles(ctx context.Context, driverID string) ([]*models.Vehicle, error) {
	if driverID == "" {
		return nil, fmt.Errorf("driver ID is required")
	}

	// Try cache first (skip if no cache available)
	var vehicles []*models.Vehicle
	if s.cacheRepo != nil {
		var err error
		vehicles, err = s.cacheRepo.GetCachedAvailableVehicles(ctx, driverID)
		if err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to get available vehicles from cache")
		}

		if vehicles != nil {
			return vehicles, nil
		}
	}

	// Get from database
	vehicles, err := s.vehicleRepo.GetByDriverID(ctx, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available vehicles: %w", err)
	}

	// Filter for only available vehicles
	var availableVehicles []*models.Vehicle
	for _, vehicle := range vehicles {
		if vehicle.Status == models.VehicleStatusActive {
			availableVehicles = append(availableVehicles, vehicle)
		}
	}

	// Cache the result (skip if no cache available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.CacheAvailableVehicles(ctx, driverID, availableVehicles, 15*time.Minute); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to cache available vehicles")
		}
	}

	return availableVehicles, nil
}

// UpdateVehicle updates a vehicle
func (s *VehicleService) UpdateVehicle(ctx context.Context, req *UpdateVehicleRequest) (*models.Vehicle, error) {
	// Validate request
	if err := s.validateUpdateVehicleRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get existing vehicle
	vehicle, err := s.GetVehicle(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Check if license plate is being changed and if it already exists
	if req.LicensePlate != "" && req.LicensePlate != vehicle.LicensePlate {
		exists, err := s.vehicleRepo.LicensePlateExists(ctx, req.LicensePlate)
		if err != nil {
			return nil, fmt.Errorf("failed to check license plate: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("license plate already exists: %s", req.LicensePlate)
		}
	}

	// Update fields
	if req.Make != "" {
		vehicle.Make = req.Make
	}
	if req.Model != "" {
		vehicle.Model = req.Model
	}
	if req.Year > 0 {
		vehicle.Year = req.Year
	}
	if req.Color != "" {
		vehicle.Color = req.Color
	}
	if req.LicensePlate != "" {
		vehicle.LicensePlate = req.LicensePlate
	}
	if req.VehicleType != "" {
		vehicle.VehicleType = models.VehicleType(req.VehicleType)
	}
	if req.Capacity > 0 {
		vehicle.Capacity = req.Capacity
	}

	// Update insurance info if provided
	if req.InsurancePolicyNumber != "" && req.InsuranceExpiry != nil {
		vehicle.SetInsuranceInfo(req.InsurancePolicyNumber, *req.InsuranceExpiry)
	}

	// Update registration expiry if provided
	if req.RegistrationExpiry != nil {
		vehicle.SetRegistrationExpiry(*req.RegistrationExpiry)
	}

	vehicle.UpdatedAt = time.Now()

	// Save to database
	if err := s.vehicleRepo.Update(ctx, vehicle); err != nil {
		return nil, fmt.Errorf("failed to update vehicle: %w", err)
	}

	// Invalidate caches
	if s.cacheRepo != nil {
		if err := s.cacheRepo.InvalidateVehicle(ctx, vehicle.ID); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate vehicle cache")
			}
		}
		if err := s.cacheRepo.InvalidateDriverVehicles(ctx, vehicle.DriverID); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate driver vehicles cache")
			}
		}
	}

	// Publish event
	event := events.NewEvent(
		events.VehicleUpdatedEvent,
		vehicle.ID,
		1,
		map[string]interface{}{
			"vehicle_id":    vehicle.ID,
			"driver_id":     vehicle.DriverID,
			"license_plate": vehicle.LicensePlate,
		},
		"vehicle-service",
	)

	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to publish vehicle updated event")
			}
		}
	}

	if s.logger != nil {
		s.logger.WithContext(ctx).WithFields(logger.Fields{
			"vehicle_id": vehicle.ID,
			"driver_id":  vehicle.DriverID,
		}).Info("Vehicle updated successfully")
	}

	return vehicle, nil
}

// UpdateVehicleStatus updates vehicle status
func (s *VehicleService) UpdateVehicleStatus(ctx context.Context, id string, status models.VehicleStatus) error {
	if id == "" {
		return fmt.Errorf("vehicle ID is required")
	}

	// Get vehicle to get driver ID for cache invalidation
	vehicle, err := s.GetVehicle(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Update status in database
	if err := s.vehicleRepo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update vehicle status: %w", err)
	}

	// Invalidate caches (only if cache is available)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.InvalidateVehicle(ctx, id); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate vehicle cache")
		}
		if err := s.cacheRepo.InvalidateDriverVehicles(ctx, vehicle.DriverID); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate driver vehicles cache")
		}
		if err := s.cacheRepo.InvalidateAvailableVehicles(ctx, vehicle.DriverID); err != nil && s.logger != nil {
			s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate available vehicles cache")
		}
	}

	// Log only if logger is available
	if s.logger != nil {
		s.logger.WithContext(ctx).WithFields(logger.Fields{
			"vehicle_id": id,
			"status":     status,
		}).Info("Vehicle status updated successfully")
	}

	return nil
}

// DeleteVehicle soft deletes a vehicle
func (s *VehicleService) DeleteVehicle(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("vehicle ID is required")
	}

	// Get vehicle to get driver ID for cache invalidation
	vehicle, err := s.GetVehicle(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Delete from database
	if err := s.vehicleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete vehicle: %w", err)
	}

	// Invalidate caches
	if s.cacheRepo != nil {
		if err := s.cacheRepo.InvalidateVehicle(ctx, id); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate vehicle cache")
			}
		}
		if err := s.cacheRepo.InvalidateDriverVehicles(ctx, vehicle.DriverID); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to invalidate driver vehicles cache")
			}
		}
	}

	// Publish event
	event := events.NewEvent(
		events.VehicleDeactivatedEvent,
		vehicle.ID,
		1,
		map[string]interface{}{
			"vehicle_id":    vehicle.ID,
			"driver_id":     vehicle.DriverID,
			"license_plate": vehicle.LicensePlate,
		},
		"vehicle-service",
	)

	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
			if s.logger != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to publish vehicle deactivated event")
			}
		}
	}

	if s.logger != nil {
		s.logger.WithContext(ctx).WithFields(logger.Fields{
			"vehicle_id": id,
			"driver_id":  vehicle.DriverID,
		}).Info("Vehicle deleted successfully")
	}

	return nil
}

// ListVehicles retrieves vehicles with pagination and filtering
func (s *VehicleService) ListVehicles(ctx context.Context, req *ListVehiclesRequest) (*ListVehiclesResponse, error) {
	// Validate request
	if err := s.validateListVehiclesRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Build filters map
	filters := make(map[string]interface{})
	if req.Status != "" {
		filters["status"] = req.Status
	}
	if req.VehicleType != "" {
		filters["vehicle_type"] = req.VehicleType
	}

	// Get vehicles from database
	vehicles, err := s.vehicleRepo.List(ctx, req.Limit, req.Offset, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list vehicles: %w", err)
	}

	// Get total count
	total, err := s.vehicleRepo.Count(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count vehicles: %w", err)
	}

	return &ListVehiclesResponse{
		Vehicles: vehicles,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

// GetVehicleStats retrieves vehicle statistics
func (s *VehicleService) GetVehicleStats(ctx context.Context) (*VehicleStatsResponse, error) {
	// Try cache first
	cachedStats, err := s.cacheRepo.GetCachedVehicleStats(ctx)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to get vehicle stats from cache")
	}

	if cachedStats != nil {
		return &VehicleStatsResponse{
			TotalVehicles:    int64(cachedStats["total_vehicles"].(float64)),
			ActiveVehicles:   int64(cachedStats["active_vehicles"].(float64)),
			InactiveVehicles: int64(cachedStats["inactive_vehicles"].(float64)),
			VehiclesByType:   cachedStats["vehicles_by_type"].(map[string]interface{}),
		}, nil
	}

	// Calculate stats from database
	totalVehicles, err := s.vehicleRepo.Count(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total vehicles: %w", err)
	}

	activeVehicles, err := s.vehicleRepo.Count(ctx, map[string]interface{}{"status": "active"})
	if err != nil {
		return nil, fmt.Errorf("failed to count active vehicles: %w", err)
	}

	inactiveVehicles, err := s.vehicleRepo.Count(ctx, map[string]interface{}{"status": "inactive"})
	if err != nil {
		return nil, fmt.Errorf("failed to count inactive vehicles: %w", err)
	}

	// Get vehicles by type
	vehiclesByType := make(map[string]interface{})
	for _, vehicleType := range models.GetVehicleTypes() {
		count, err := s.vehicleRepo.Count(ctx, map[string]interface{}{"vehicle_type": string(vehicleType)})
		if err != nil {
			s.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
				"vehicle_type": vehicleType,
			}).Warn("Failed to count vehicles by type")
			continue
		}
		vehiclesByType[string(vehicleType)] = count
	}

	stats := &VehicleStatsResponse{
		TotalVehicles:    totalVehicles,
		ActiveVehicles:   activeVehicles,
		InactiveVehicles: inactiveVehicles,
		VehiclesByType:   vehiclesByType,
	}

	// Cache the stats
	statsMap := map[string]interface{}{
		"total_vehicles":    totalVehicles,
		"active_vehicles":   activeVehicles,
		"inactive_vehicles": inactiveVehicles,
		"vehicles_by_type":  vehiclesByType,
	}

	if err := s.cacheRepo.CacheVehicleStats(ctx, statsMap, 5*time.Minute); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to cache vehicle stats")
	}

	return stats, nil
}

// GetVehiclesWithExpiredInsurance retrieves vehicles with expired insurance
func (s *VehicleService) GetVehiclesWithExpiredInsurance(ctx context.Context) ([]*models.Vehicle, error) {
	vehicles, err := s.vehicleRepo.GetVehiclesWithExpiredInsurance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicles with expired insurance: %w", err)
	}

	return vehicles, nil
}

// GetVehiclesWithExpiredRegistration retrieves vehicles with expired registration
func (s *VehicleService) GetVehiclesWithExpiredRegistration(ctx context.Context) ([]*models.Vehicle, error) {
	vehicles, err := s.vehicleRepo.GetVehiclesWithExpiredRegistration(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicles with expired registration: %w", err)
	}

	return vehicles, nil
}

// Validation methods

func (s *VehicleService) validateCreateVehicleRequest(req *CreateVehicleRequest) error {
	if req.DriverID == "" {
		return fmt.Errorf("driver ID is required")
	}
	if req.Make == "" {
		return fmt.Errorf("make is required")
	}
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if req.Year <= 0 {
		return fmt.Errorf("year must be positive")
	}
	if req.LicensePlate == "" {
		return fmt.Errorf("license plate is required")
	}
	if req.VehicleType == "" {
		return fmt.Errorf("vehicle type is required")
	}
	if !models.IsValidVehicleType(req.VehicleType) {
		return fmt.Errorf("invalid vehicle type: %s", req.VehicleType)
	}
	if req.Capacity <= 0 {
		return fmt.Errorf("capacity must be positive")
	}
	return nil
}

func (s *VehicleService) validateUpdateVehicleRequest(req *UpdateVehicleRequest) error {
	if req.ID == "" {
		return fmt.Errorf("vehicle ID is required")
	}
	if req.VehicleType != "" && !models.IsValidVehicleType(req.VehicleType) {
		return fmt.Errorf("invalid vehicle type: %s", req.VehicleType)
	}
	return nil
}

func (s *VehicleService) validateListVehiclesRequest(req *ListVehiclesRequest) error {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	return nil
}

// Request/Response types

type CreateVehicleRequest struct {
	DriverID              string     `json:"driver_id"`
	Make                  string     `json:"make"`
	Model                 string     `json:"model"`
	Year                  int        `json:"year"`
	Color                 string     `json:"color"`
	LicensePlate          string     `json:"license_plate"`
	VehicleType           string     `json:"vehicle_type"`
	Capacity              int        `json:"capacity"`
	InsurancePolicyNumber string     `json:"insurance_policy_number,omitempty"`
	InsuranceExpiry       *time.Time `json:"insurance_expiry,omitempty"`
	RegistrationExpiry    *time.Time `json:"registration_expiry,omitempty"`
}

type UpdateVehicleRequest struct {
	ID                    string     `json:"id"`
	Make                  string     `json:"make,omitempty"`
	Model                 string     `json:"model,omitempty"`
	Year                  int        `json:"year,omitempty"`
	Color                 string     `json:"color,omitempty"`
	LicensePlate          string     `json:"license_plate,omitempty"`
	VehicleType           string     `json:"vehicle_type,omitempty"`
	Capacity              int        `json:"capacity,omitempty"`
	InsurancePolicyNumber string     `json:"insurance_policy_number,omitempty"`
	InsuranceExpiry       *time.Time `json:"insurance_expiry,omitempty"`
	RegistrationExpiry    *time.Time `json:"registration_expiry,omitempty"`
}

type ListVehiclesRequest struct {
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Status      string `json:"status,omitempty"`
	VehicleType string `json:"vehicle_type,omitempty"`
}

type ListVehiclesResponse struct {
	Vehicles []*models.Vehicle `json:"vehicles"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

type VehicleStatsResponse struct {
	TotalVehicles    int64                  `json:"total_vehicles"`
	ActiveVehicles   int64                  `json:"active_vehicles"`
	InactiveVehicles int64                  `json:"inactive_vehicles"`
	VehiclesByType   map[string]interface{} `json:"vehicles_by_type"`
}

// Validation methods
func (s *VehicleService) isValidVehicleType(vehicleType string) bool {
	validTypes := map[string]bool{
		"sedan":       true,
		"suv":         true,
		"hatchback":   true,
		"coupe":       true,
		"convertible": true,
		"truck":       true,
		"motorcycle":  true,
		"luxury":      true,
	}
	return validTypes[strings.ToLower(vehicleType)]
}
