package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
	"github.com/rideshare-platform/shared/utils"
)

// TripService interface defines the contract for trip operations
type TripService interface {
	CreateTrip(ctx context.Context, riderID string, pickupLoc, dropoffLoc models.Location, vehicleType string, preferences repository.TripPreferences) (*repository.Trip, error)
	GetTrip(ctx context.Context, tripID string) (*repository.Trip, error)
	CancelTrip(ctx context.Context, tripID, userID, reason string) (*repository.Trip, error)
	AcceptTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error)
	StartTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error)
	CompleteTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error)
	UpdateLocation(ctx context.Context, tripID, userID string, location models.Location) (*repository.Trip, error)

	// Query methods
	GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*repository.Trip, error)
	GetTripsByDriver(ctx context.Context, driverID string, limit, offset int) ([]*repository.Trip, error)
	GetTripsByStatus(ctx context.Context, status string, limit, offset int) ([]*repository.Trip, error)
	GetActiveTripByRider(ctx context.Context, riderID string) (*repository.Trip, error)
	GetActiveTripByDriver(ctx context.Context, driverID string) (*repository.Trip, error)

	// Event methods
	GetTripEvents(ctx context.Context, tripID string) ([]*repository.TripEvent, error)
	GetEventsByType(ctx context.Context, eventType string, limit, offset int) ([]*repository.TripEvent, error)
	GetEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*repository.TripEvent, error)
}

// EnhancedTripService implements the TripService interface with state machine logic
type EnhancedTripService struct {
	tripRepo  repository.TripRepository
	eventRepo repository.EventRepository
	logger    logger.Logger
}

func NewEnhancedTripService(tripRepo repository.TripRepository, eventRepo repository.EventRepository, logger logger.Logger) *EnhancedTripService {
	return &EnhancedTripService{
		tripRepo:  tripRepo,
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// Trip state definitions
const (
	StatusRequested     = "requested"
	StatusMatching      = "matching"
	StatusMatched       = "matched"
	StatusDriverEnRoute = "driver_en_route"
	StatusDriverArrived = "driver_arrived"
	StatusStarted       = "started"
	StatusInProgress    = "in_progress"
	StatusCompleted     = "completed"
	StatusCancelled     = "cancelled"
	StatusFailed        = "failed"
)

// Event types
const (
	EventTripCreated     = "trip_created"
	EventTripMatching    = "trip_matching"
	EventTripMatched     = "trip_matched"
	EventDriverEnRoute   = "driver_en_route"
	EventDriverArrived   = "driver_arrived"
	EventTripStarted     = "trip_started"
	EventTripInProgress  = "trip_in_progress"
	EventTripCompleted   = "trip_completed"
	EventTripCancelled   = "trip_cancelled"
	EventTripFailed      = "trip_failed"
	EventLocationUpdated = "location_updated"
)

// State transition map
var validTransitions = map[string][]string{
	StatusRequested:     {StatusMatching, StatusCancelled},
	StatusMatching:      {StatusMatched, StatusCancelled, StatusFailed},
	StatusMatched:       {StatusDriverEnRoute, StatusCancelled},
	StatusDriverEnRoute: {StatusDriverArrived, StatusCancelled},
	StatusDriverArrived: {StatusStarted, StatusCancelled},
	StatusStarted:       {StatusInProgress, StatusCancelled},
	StatusInProgress:    {StatusCompleted, StatusCancelled, StatusFailed},
	StatusCompleted:     {}, // Terminal state
	StatusCancelled:     {}, // Terminal state
	StatusFailed:        {}, // Terminal state
}

func (s *EnhancedTripService) CreateTrip(ctx context.Context, riderID string, pickupLoc, dropoffLoc models.Location, vehicleType string, preferences repository.TripPreferences) (*repository.Trip, error) {
	s.logger.WithFields(logger.Fields{
		"rider_id":     riderID,
		"vehicle_type": vehicleType,
		"pickup_lat":   pickupLoc.Latitude,
		"pickup_lng":   pickupLoc.Longitude,
		"dropoff_lat":  dropoffLoc.Latitude,
		"dropoff_lng":  dropoffLoc.Longitude,
	}).Info("Creating new trip")

	// Check if rider has an active trip
	activeTrip, err := s.tripRepo.GetActiveTripByRider(ctx, riderID)
	if err == nil && activeTrip != nil {
		return nil, fmt.Errorf("rider already has an active trip: %s", activeTrip.ID)
	}

	// Create new trip
	trip := &repository.Trip{
		ID:          utils.GenerateID("trip"),
		RiderID:     riderID,
		Status:      StatusRequested,
		PickupLoc:   pickupLoc,
		DropoffLoc:  dropoffLoc,
		VehicleType: vehicleType,
		Preferences: preferences,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Calculate estimated distance and fare (placeholder logic)
	trip.EstimatedDistance = s.calculateDistance(pickupLoc, dropoffLoc)
	trip.EstimatedFare = s.calculateFare(trip.EstimatedDistance, vehicleType)

	// Save trip
	if err := s.tripRepo.CreateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	// Log event
	if err := s.logEvent(ctx, trip.ID, riderID, EventTripCreated, "Trip created successfully", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip created event")
	}

	return trip, nil
}

func (s *EnhancedTripService) GetTrip(ctx context.Context, tripID string) (*repository.Trip, error) {
	return s.tripRepo.GetTrip(ctx, tripID)
}

func (s *EnhancedTripService) CancelTrip(ctx context.Context, tripID, userID, reason string) (*repository.Trip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id": tripID,
		"user_id": userID,
		"reason":  reason,
	}).Info("Cancelling trip")

	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate state transition
	if !s.isValidTransition(trip.Status, StatusCancelled) {
		return nil, fmt.Errorf("cannot cancel trip in status: %s", trip.Status)
	}

	// Validate user permission
	if trip.RiderID != userID && trip.DriverID != userID {
		return nil, fmt.Errorf("user not authorized to cancel this trip")
	}

	// Update trip status
	trip.Status = StatusCancelled
	trip.CancellationReason = reason
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Log event
	if err := s.logEvent(ctx, tripID, userID, EventTripCancelled, reason, nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip cancelled event")
	}

	return trip, nil
}

func (s *EnhancedTripService) AcceptTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Driver accepting trip")

	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Check if driver has an active trip
	activeTrip, err := s.tripRepo.GetActiveTripByDriver(ctx, driverID)
	if err == nil && activeTrip != nil {
		return nil, fmt.Errorf("driver already has an active trip: %s", activeTrip.ID)
	}

	// Validate state transition
	if !s.isValidTransition(trip.Status, StatusMatched) {
		return nil, fmt.Errorf("cannot accept trip in status: %s", trip.Status)
	}

	// Update trip
	trip.Status = StatusMatched
	trip.DriverID = driverID
	trip.MatchedAt = time.Now()
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Log event
	if err := s.logEvent(ctx, tripID, driverID, EventTripMatched, "Trip matched with driver", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip matched event")
	}

	// Automatically transition to driver en route
	return s.transitionToDriverEnRoute(ctx, trip, driverID)
}

func (s *EnhancedTripService) StartTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Starting trip")

	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate driver
	if trip.DriverID != driverID {
		return nil, fmt.Errorf("driver not authorized for this trip")
	}

	// Validate state transition
	if !s.isValidTransition(trip.Status, StatusStarted) {
		return nil, fmt.Errorf("cannot start trip in status: %s", trip.Status)
	}

	// Update trip
	trip.Status = StatusStarted
	trip.StartedAt = time.Now()
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Log event
	if err := s.logEvent(ctx, tripID, driverID, EventTripStarted, "Trip started", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip started event")
	}

	// Automatically transition to in progress
	return s.transitionToInProgress(ctx, trip, driverID)
}

func (s *EnhancedTripService) CompleteTrip(ctx context.Context, tripID, driverID string) (*repository.Trip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Completing trip")

	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate driver
	if trip.DriverID != driverID {
		return nil, fmt.Errorf("driver not authorized for this trip")
	}

	// Validate state transition
	if !s.isValidTransition(trip.Status, StatusCompleted) {
		return nil, fmt.Errorf("cannot complete trip in status: %s", trip.Status)
	}

	// Update trip
	trip.Status = StatusCompleted
	trip.CompletedAt = time.Now()
	trip.UpdatedAt = time.Now()

	// Calculate actual fare based on completed trip
	trip.ActualFare = s.calculateActualFare(trip)

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Log event
	if err := s.logEvent(ctx, tripID, driverID, EventTripCompleted, "Trip completed successfully", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip completed event")
	}

	return trip, nil
}

func (s *EnhancedTripService) UpdateLocation(ctx context.Context, tripID, userID string, location models.Location) (*repository.Trip, error) {
	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate user
	if trip.RiderID != userID && trip.DriverID != userID {
		return nil, fmt.Errorf("user not authorized for this trip")
	}

	// Update appropriate location
	if trip.RiderID == userID {
		trip.RiderLocation = &location
	} else if trip.DriverID == userID {
		trip.DriverLocation = &location
	}

	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Log event
	locationData := map[string]interface{}{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"user_type": func() string {
			if trip.RiderID == userID {
				return "rider"
			}
			return "driver"
		}(),
	}

	if err := s.logEvent(ctx, tripID, userID, EventLocationUpdated, "Location updated", locationData); err != nil {
		s.logger.WithError(err).Error("Failed to log location update event")
	}

	return trip, nil
}

// Query methods implementation

func (s *EnhancedTripService) GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*repository.Trip, error) {
	return s.tripRepo.GetTripsByRider(ctx, riderID, limit, offset)
}

func (s *EnhancedTripService) GetTripsByDriver(ctx context.Context, driverID string, limit, offset int) ([]*repository.Trip, error) {
	return s.tripRepo.GetTripsByDriver(ctx, driverID, limit, offset)
}

func (s *EnhancedTripService) GetTripsByStatus(ctx context.Context, status string, limit, offset int) ([]*repository.Trip, error) {
	return s.tripRepo.GetTripsByStatus(ctx, status, limit, offset)
}

func (s *EnhancedTripService) GetActiveTripByRider(ctx context.Context, riderID string) (*repository.Trip, error) {
	return s.tripRepo.GetActiveTripByRider(ctx, riderID)
}

func (s *EnhancedTripService) GetActiveTripByDriver(ctx context.Context, driverID string) (*repository.Trip, error) {
	return s.tripRepo.GetActiveTripByDriver(ctx, driverID)
}

// Event methods implementation

func (s *EnhancedTripService) GetTripEvents(ctx context.Context, tripID string) ([]*repository.TripEvent, error) {
	return s.eventRepo.GetTripEvents(ctx, tripID)
}

func (s *EnhancedTripService) GetEventsByType(ctx context.Context, eventType string, limit, offset int) ([]*repository.TripEvent, error) {
	return s.eventRepo.GetEventsByType(ctx, eventType, limit, offset)
}

func (s *EnhancedTripService) GetEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*repository.TripEvent, error) {
	return s.eventRepo.GetEventsByUser(ctx, userID, limit, offset)
}

// Helper methods

func (s *EnhancedTripService) transitionToDriverEnRoute(ctx context.Context, trip *repository.Trip, driverID string) (*repository.Trip, error) {
	trip.Status = StatusDriverEnRoute
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip to driver en route: %w", err)
	}

	if err := s.logEvent(ctx, trip.ID, driverID, EventDriverEnRoute, "Driver is en route to pickup", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log driver en route event")
	}

	return trip, nil
}

func (s *EnhancedTripService) transitionToInProgress(ctx context.Context, trip *repository.Trip, driverID string) (*repository.Trip, error) {
	trip.Status = StatusInProgress
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip to in progress: %w", err)
	}

	if err := s.logEvent(ctx, trip.ID, driverID, EventTripInProgress, "Trip is now in progress", nil); err != nil {
		s.logger.WithError(err).Error("Failed to log trip in progress event")
	}

	return trip, nil
}

func (s *EnhancedTripService) isValidTransition(currentStatus, newStatus string) bool {
	validNextStates, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, validStatus := range validNextStates {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

func (s *EnhancedTripService) logEvent(ctx context.Context, tripID, userID, eventType, description string, data map[string]interface{}) error {
	event := &repository.TripEvent{
		ID:          utils.GenerateID("event"),
		TripID:      tripID,
		UserID:      userID,
		EventType:   eventType,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
	}

	return s.eventRepo.SaveEvent(ctx, event)
}

func (s *EnhancedTripService) calculateDistance(pickup, dropoff models.Location) float64 {
	// Haversine formula for distance calculation
	const earthRadius = 6371 // km

	lat1 := pickup.Latitude * (3.14159 / 180)
	lat2 := dropoff.Latitude * (3.14159 / 180)
	deltaLat := (dropoff.Latitude - pickup.Latitude) * (3.14159 / 180)
	deltaLng := (dropoff.Longitude - pickup.Longitude) * (3.14159 / 180)

	a := 0.5 - 0.5*((lat2-lat1)/2) + (lat1*lat2)*((deltaLng)/2)*((deltaLng)/2)
	return earthRadius * 2 * (a + (1 - a))
}

func (s *EnhancedTripService) calculateFare(distance float64, vehicleType string) float64 {
	baseFare := 2.5
	perKmRate := map[string]float64{
		"economy":  1.2,
		"standard": 1.5,
		"premium":  2.0,
		"luxury":   3.0,
	}

	rate, exists := perKmRate[vehicleType]
	if !exists {
		rate = perKmRate["standard"]
	}

	return baseFare + (distance * rate)
}

func (s *EnhancedTripService) calculateActualFare(trip *repository.Trip) float64 {
	// For now, use estimated fare. In real implementation, this would
	// calculate based on actual distance, time, surge pricing, etc.
	return trip.EstimatedFare
}
