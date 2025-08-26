package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// TripRepositoryInterface defines the repository interface for trips
type TripRepositoryInterface interface {
	Create(ctx context.Context, trip *models.Trip) error
	GetByID(ctx context.Context, id string) (*models.Trip, error)
	Update(ctx context.Context, trip *models.Trip) error
	GetByRiderID(ctx context.Context, riderID string) ([]*models.Trip, error)
	GetByDriverID(ctx context.Context, driverID string) ([]*models.Trip, error)
}

// TripService handles trip business logic
type TripService struct {
	tripRepo TripRepositoryInterface
	logger   *logger.Logger
}

// NewTripService creates a new trip service
func NewTripService(tripRepo TripRepositoryInterface, logger *logger.Logger) *TripService {
	return &TripService{
		tripRepo: tripRepo,
		logger:   logger,
	}
}

// CreateTripRequest represents a trip creation request
type CreateTripRequest struct {
	RiderID             string          `json:"rider_id"`
	PickupLocation      models.Location `json:"pickup_location"`
	DestinationLocation models.Location `json:"destination_location"`
	RideType            string          `json:"ride_type"`
	EstimatedFare       float64         `json:"estimated_fare"`
	RequestedAt         time.Time       `json:"requested_at"`
}

// Location represents a geographic location with address
type TripLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// CreateTrip creates a new trip request
func (s *TripService) CreateTrip(ctx context.Context, req *CreateTripRequest) (*models.Trip, error) {
	// Validate request
	if err := s.validateCreateTripRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create trip
	trip := &models.Trip{
		ID:      generateTripID(),
		RiderID: req.RiderID,
		Status:  models.TripStatusRequested,
		PickupLocation: models.Location{
			Latitude:  req.PickupLocation.Latitude,
			Longitude: req.PickupLocation.Longitude,
			Timestamp: time.Now(),
		},
		Destination: models.Location{
			Latitude:  req.DestinationLocation.Latitude,
			Longitude: req.DestinationLocation.Longitude,
			Timestamp: time.Now(),
		},
		EstimatedFareCents: func() *int64 {
			cents := int64(req.EstimatedFare * 100)
			return &cents
		}(),
		Currency:       "USD",
		PassengerCount: 1,
		RequestedAt:    req.RequestedAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save to database
	if err := s.tripRepo.Create(ctx, trip); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to create trip")
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id":  trip.ID,
		"rider_id": trip.RiderID,
	}).Info("Trip created successfully")

	return trip, nil
}

// GetTrip retrieves a trip by ID
func (s *TripService) GetTrip(ctx context.Context, id string) (*models.Trip, error) {
	if id == "" {
		return nil, fmt.Errorf("trip ID is required")
	}

	trip, err := s.tripRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"trip_id": id,
		}).Error("Failed to get trip")
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	return trip, nil
}

// AcceptTrip allows a driver to accept a trip request
func (s *TripService) AcceptTrip(ctx context.Context, tripID, driverID string) (*models.Trip, error) {
	if tripID == "" {
		return nil, fmt.Errorf("trip ID is required")
	}
	if driverID == "" {
		return nil, fmt.Errorf("driver ID is required")
	}

	// Get trip
	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate trip can be accepted
	if trip.Status != models.TripStatusRequested {
		return nil, fmt.Errorf("trip cannot be accepted, current status: %s", trip.Status)
	}

	// Update trip
	trip.DriverID = &driverID
	trip.Status = models.TripStatusMatched
	now := time.Now()
	trip.DriverAssignedAt = &now
	trip.UpdatedAt = now

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to accept trip")
		return nil, fmt.Errorf("failed to accept trip: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id":   trip.ID,
		"driver_id": driverID,
	}).Info("Trip accepted successfully")

	return trip, nil
}

// StartTrip marks a trip as started
func (s *TripService) StartTrip(ctx context.Context, tripID string) (*models.Trip, error) {
	if tripID == "" {
		return nil, fmt.Errorf("trip ID is required")
	}

	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	if trip.Status != models.TripStatusMatched {
		return nil, fmt.Errorf("trip cannot be started, current status: %s", trip.Status)
	}

	trip.Status = models.TripStatusTripStarted
	now := time.Now()
	trip.StartedAt = &now
	trip.UpdatedAt = now

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to start trip")
		return nil, fmt.Errorf("failed to start trip: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id": trip.ID,
	}).Info("Trip started successfully")

	return trip, nil
}

// CompleteTrip marks a trip as completed
func (s *TripService) CompleteTrip(ctx context.Context, tripID string, finalFare float64) (*models.Trip, error) {
	if tripID == "" {
		return nil, fmt.Errorf("trip ID is required")
	}
	if finalFare < 0 {
		return nil, fmt.Errorf("final fare must be non-negative")
	}

	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	if trip.Status != models.TripStatusTripStarted {
		return nil, fmt.Errorf("trip cannot be completed, current status: %s", trip.Status)
	}

	trip.Status = models.TripStatusCompleted
	finalFareCents := int64(finalFare * 100)
	trip.ActualFareCents = &finalFareCents
	now := time.Now()
	trip.CompletedAt = &now
	trip.UpdatedAt = now

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to complete trip")
		return nil, fmt.Errorf("failed to complete trip: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id":    trip.ID,
		"final_fare": finalFare,
	}).Info("Trip completed successfully")

	return trip, nil
}

// CancelTrip cancels a trip
func (s *TripService) CancelTrip(ctx context.Context, tripID, reason string) (*models.Trip, error) {
	if tripID == "" {
		return nil, fmt.Errorf("trip ID is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("cancellation reason is required")
	}

	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	if trip.Status == models.TripStatusCompleted || trip.Status == models.TripStatusCancelled {
		return nil, fmt.Errorf("trip cannot be cancelled, current status: %s", trip.Status)
	}

	trip.Status = models.TripStatusCancelled
	trip.CancellationReason = &reason
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to cancel trip")
		return nil, fmt.Errorf("failed to cancel trip: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id": trip.ID,
		"reason":  reason,
	}).Info("Trip cancelled successfully")

	return trip, nil
}

// GetRiderTrips retrieves all trips for a rider
func (s *TripService) GetRiderTrips(ctx context.Context, riderID string) ([]*models.Trip, error) {
	if riderID == "" {
		return nil, fmt.Errorf("rider ID is required")
	}

	trips, err := s.tripRepo.GetByRiderID(ctx, riderID)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get rider trips")
		return nil, fmt.Errorf("failed to get rider trips: %w", err)
	}

	return trips, nil
}

// GetDriverTrips retrieves all trips for a driver
func (s *TripService) GetDriverTrips(ctx context.Context, driverID string) ([]*models.Trip, error) {
	if driverID == "" {
		return nil, fmt.Errorf("driver ID is required")
	}

	trips, err := s.tripRepo.GetByDriverID(ctx, driverID)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get driver trips")
		return nil, fmt.Errorf("failed to get driver trips: %w", err)
	}

	return trips, nil
}

// CalculateTripDuration calculates the duration of a completed trip
func (s *TripService) CalculateTripDuration(trip *models.Trip) (time.Duration, error) {
	if trip.Status != models.TripStatusCompleted {
		return 0, fmt.Errorf("trip is not completed")
	}

	if trip.StartedAt == nil || trip.CompletedAt == nil {
		return 0, fmt.Errorf("trip timestamps are invalid")
	}

	return trip.CompletedAt.Sub(*trip.StartedAt), nil
}

// EstimateTripTime estimates trip time based on distance
func (s *TripService) EstimateTripTime(distance float64) time.Duration {
	// Simple estimation: assume average speed of 30 km/h in city
	avgSpeedKmh := 30.0
	hours := distance / avgSpeedKmh
	return time.Duration(hours * float64(time.Hour))
}

// validateCreateTripRequest validates a trip creation request
func (s *TripService) validateCreateTripRequest(req *CreateTripRequest) error {
	if req.RiderID == "" {
		return fmt.Errorf("rider ID is required")
	}

	if req.PickupLocation.Latitude == 0 || req.PickupLocation.Longitude == 0 {
		return fmt.Errorf("pickup location coordinates are required")
	}

	if req.DestinationLocation.Latitude == 0 || req.DestinationLocation.Longitude == 0 {
		return fmt.Errorf("destination location coordinates are required")
	}

	if req.RideType == "" {
		return fmt.Errorf("ride type is required")
	}

	validRideTypes := map[string]bool{
		"standard": true,
		"premium":  true,
		"xl":       true,
		"pool":     true,
	}

	if !validRideTypes[req.RideType] {
		return fmt.Errorf("invalid ride type: %s", req.RideType)
	}

	if req.EstimatedFare < 0 {
		return fmt.Errorf("estimated fare must be non-negative")
	}

	return nil
}

// generateTripID generates a unique trip ID
func generateTripID() string {
	return fmt.Sprintf("trip_%d", time.Now().UnixNano())
}
