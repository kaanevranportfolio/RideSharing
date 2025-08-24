package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// Enhanced Trip Service with State Machine and Event Sourcing
type EnhancedTripService struct {
	tripRepo     repository.TripRepository
	eventRepo    repository.EventRepository
	logger       logger.Logger
	geoService   GeoServiceClient
	priceService PricingServiceClient
}

// Service interfaces for external dependencies
type GeoServiceClient interface {
	CalculateDistance(ctx context.Context, origin, destination *models.Location) (*DistanceResult, error)
	CalculateETA(ctx context.Context, origin, destination *models.Location, vehicleType string) (*ETAResult, error)
}

type PricingServiceClient interface {
	CalculatePrice(ctx context.Context, req *PricingRequest) (*PricingResponse, error)
}

// Trip States - Complete State Machine
const (
	TripStateRequested     = "requested"
	TripStateMatching      = "matching"
	TripStateMatched       = "matched"
	TripStateDriverEnRoute = "driver_en_route"
	TripStateDriverArrived = "driver_arrived"
	TripStateStarted       = "started"
	TripStateInProgress    = "in_progress"
	TripStateCompleted     = "completed"
	TripStateCancelled     = "cancelled"
	TripStateFailed        = "failed"
)

// Trip Events for Event Sourcing
const (
	EventTripRequested    = "trip.requested"
	EventMatchingStarted  = "trip.matching_started"
	EventDriverMatched    = "trip.driver_matched"
	EventDriverEnRoute    = "trip.driver_en_route"
	EventDriverArrived    = "trip.driver_arrived"
	EventTripStarted      = "trip.started"
	EventTripCompleted    = "trip.completed"
	EventTripCancelled    = "trip.cancelled"
	EventLocationUpdated  = "trip.location_updated"
	EventPaymentProcessed = "trip.payment_processed"
)

// Enhanced Data Structures
type Trip struct {
	ID                 string                 `json:"id"`
	RiderID            string                 `json:"rider_id"`
	DriverID           string                 `json:"driver_id,omitempty"`
	VehicleID          string                 `json:"vehicle_id,omitempty"`
	Status             string                 `json:"status"`
	RideType           string                 `json:"ride_type"`
	PickupLocation     *models.Location       `json:"pickup_location"`
	Destination        *models.Location       `json:"destination"`
	CurrentLocation    *models.Location       `json:"current_location,omitempty"`
	EstimatedFare      float64                `json:"estimated_fare"`
	ActualFare         float64                `json:"actual_fare"`
	Distance           float64                `json:"distance_km"`
	EstimatedDuration  int                    `json:"estimated_duration_seconds"`
	ActualDuration     int                    `json:"actual_duration_seconds"`
	RequestedAt        time.Time              `json:"requested_at"`
	MatchedAt          *time.Time             `json:"matched_at,omitempty"`
	StartedAt          *time.Time             `json:"started_at,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
	Rating             *TripRating            `json:"rating,omitempty"`
	PaymentStatus      string                 `json:"payment_status"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type TripRating struct {
	RiderRating   int    `json:"rider_rating"`
	DriverRating  int    `json:"driver_rating"`
	RiderComment  string `json:"rider_comment,omitempty"`
	DriverComment string `json:"driver_comment,omitempty"`
}

type TripEvent struct {
	ID        string                 `json:"id"`
	TripID    string                 `json:"trip_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
}

// Request/Response Types
type CreateTripRequest struct {
	RiderID         string           `json:"rider_id"`
	PickupLocation  *models.Location `json:"pickup_location"`
	Destination     *models.Location `json:"destination"`
	RideType        string           `json:"ride_type"`
	ScheduledTime   *time.Time       `json:"scheduled_time,omitempty"`
	PaymentMethod   string           `json:"payment_method"`
	SpecialRequests []string         `json:"special_requests,omitempty"`
}

type UpdateTripStatusRequest struct {
	TripID    string                 `json:"trip_id"`
	NewStatus string                 `json:"new_status"`
	UserID    string                 `json:"user_id"`
	Location  *models.Location       `json:"location,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TripLocationUpdate struct {
	TripID    string           `json:"trip_id"`
	Location  *models.Location `json:"location"`
	Timestamp time.Time        `json:"timestamp"`
}

// External service types
type DistanceResult struct {
	DistanceMeters float64 `json:"distance_meters"`
	DistanceKm     float64 `json:"distance_km"`
	BearingDegrees float64 `json:"bearing_degrees"`
}

type ETAResult struct {
	DurationSeconds int     `json:"duration_seconds"`
	DistanceMeters  float64 `json:"distance_meters"`
	RouteSummary    string  `json:"route_summary"`
}

type PricingRequest struct {
	Distance      float64 `json:"distance"`
	EstimatedTime int     `json:"estimated_time"`
	VehicleType   string  `json:"vehicle_type"`
	PickupArea    string  `json:"pickup_area"`
	RequestTime   int64   `json:"request_time"`
}

type PricingResponse struct {
	TotalFare       float64 `json:"total_fare"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Currency        string  `json:"currency"`
}

// State Machine Validation
var validTransitions = map[string][]string{
	TripStateRequested:     {TripStateMatching, TripStateCancelled},
	TripStateMatching:      {TripStateMatched, TripStateCancelled, TripStateFailed},
	TripStateMatched:       {TripStateDriverEnRoute, TripStateCancelled},
	TripStateDriverEnRoute: {TripStateDriverArrived, TripStateCancelled},
	TripStateDriverArrived: {TripStateStarted, TripStateCancelled},
	TripStateStarted:       {TripStateInProgress, TripStateCancelled},
	TripStateInProgress:    {TripStateCompleted, TripStateCancelled},
	TripStateCompleted:     {}, // Terminal state
	TripStateCancelled:     {}, // Terminal state
	TripStateFailed:        {}, // Terminal state
}

func NewEnhancedTripService(
	tripRepo repository.TripRepository,
	eventRepo repository.EventRepository,
	logger logger.Logger,
	geoService GeoServiceClient,
	priceService PricingServiceClient,
) *EnhancedTripService {
	return &EnhancedTripService{
		tripRepo:     tripRepo,
		eventRepo:    eventRepo,
		logger:       logger,
		geoService:   geoService,
		priceService: priceService,
	}
}

// CreateTrip - Enhanced trip creation with state machine and event sourcing
func (s *EnhancedTripService) CreateTrip(ctx context.Context, req *CreateTripRequest) (*Trip, error) {
	s.logger.WithFields(logger.Fields{
		"rider_id":  req.RiderID,
		"ride_type": req.RideType,
	}).Info("Creating new trip")

	// Validate request
	if err := s.validateCreateTripRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Calculate distance and ETA
	distance, err := s.geoService.CalculateDistance(ctx, req.PickupLocation, req.Destination)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate distance")
		return nil, fmt.Errorf("failed to calculate distance: %w", err)
	}

	eta, err := s.geoService.CalculateETA(ctx, req.PickupLocation, req.Destination, req.RideType)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate ETA")
		return nil, fmt.Errorf("failed to calculate ETA: %w", err)
	}

	// Calculate estimated fare
	pricing, err := s.priceService.CalculatePrice(ctx, &PricingRequest{
		Distance:      distance.DistanceKm,
		EstimatedTime: eta.DurationSeconds,
		VehicleType:   req.RideType,
		PickupArea:    s.getAreaFromLocation(req.PickupLocation),
		RequestTime:   time.Now().Unix(),
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate pricing")
		return nil, fmt.Errorf("failed to calculate pricing: %w", err)
	}

	// Create trip
	now := time.Now()
	trip := &Trip{
		ID:                s.generateTripID(),
		RiderID:           req.RiderID,
		Status:            TripStateRequested,
		RideType:          req.RideType,
		PickupLocation:    req.PickupLocation,
		Destination:       req.Destination,
		EstimatedFare:     pricing.TotalFare,
		Distance:          distance.DistanceKm,
		EstimatedDuration: eta.DurationSeconds,
		PaymentStatus:     "pending",
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Save trip
	if err := s.tripRepo.CreateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to save trip: %w", err)
	}

	// Record event
	event := &TripEvent{
		ID:        s.generateEventID(),
		TripID:    trip.ID,
		EventType: EventTripRequested,
		Data: map[string]interface{}{
			"rider_id":        req.RiderID,
			"pickup_location": req.PickupLocation,
			"destination":     req.Destination,
			"ride_type":       req.RideType,
			"estimated_fare":  pricing.TotalFare,
		},
		Timestamp: now,
		UserID:    req.RiderID,
	}

	if err := s.eventRepo.SaveEvent(ctx, event); err != nil {
		s.logger.WithError(err).Error("Failed to save trip event")
		// Don't fail the entire operation for event logging
	}

	s.logger.WithFields(logger.Fields{
		"trip_id": trip.ID,
		"status":  trip.Status,
	}).Info("Trip created successfully")

	return trip, nil
}

// UpdateTripStatus - Enhanced status updates with state machine validation
func (s *EnhancedTripService) UpdateTripStatus(ctx context.Context, req *UpdateTripStatusRequest) (*Trip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id":    req.TripID,
		"new_status": req.NewStatus,
		"user_id":    req.UserID,
	}).Info("Updating trip status")

	// Get current trip
	trip, err := s.tripRepo.GetTrip(ctx, req.TripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Validate state transition
	if !s.isValidTransition(trip.Status, req.NewStatus) {
		return nil, fmt.Errorf("invalid state transition from %s to %s", trip.Status, req.NewStatus)
	}

	// Update trip based on new status
	now := time.Now()
	oldStatus := trip.Status
	trip.Status = req.NewStatus
	trip.UpdatedAt = now

	// Handle status-specific updates
	switch req.NewStatus {
	case TripStateMatched:
		if driverID, ok := req.Metadata["driver_id"].(string); ok {
			trip.DriverID = driverID
		}
		if vehicleID, ok := req.Metadata["vehicle_id"].(string); ok {
			trip.VehicleID = vehicleID
		}
		trip.MatchedAt = &now

	case TripStateStarted:
		trip.StartedAt = &now

	case TripStateCompleted:
		trip.CompletedAt = &now
		if req.Location != nil {
			trip.CurrentLocation = req.Location
		}
		// Calculate actual duration
		if trip.StartedAt != nil {
			trip.ActualDuration = int(now.Sub(*trip.StartedAt).Seconds())
		}

	case TripStateCancelled:
		trip.CancelledAt = &now
		if reason, ok := req.Metadata["cancellation_reason"].(string); ok {
			trip.CancellationReason = reason
		}
	}

	// Update location if provided
	if req.Location != nil {
		trip.CurrentLocation = req.Location
	}

	// Save updated trip
	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to update trip: %w", err)
	}

	// Record event
	event := &TripEvent{
		ID:        s.generateEventID(),
		TripID:    trip.ID,
		EventType: s.getEventTypeForStatus(req.NewStatus),
		Data: map[string]interface{}{
			"old_status": oldStatus,
			"new_status": req.NewStatus,
			"metadata":   req.Metadata,
		},
		Timestamp: now,
		UserID:    req.UserID,
	}

	if err := s.eventRepo.SaveEvent(ctx, event); err != nil {
		s.logger.WithError(err).Error("Failed to save status update event")
	}

	s.logger.WithFields(logger.Fields{
		"trip_id":    trip.ID,
		"old_status": oldStatus,
		"new_status": trip.Status,
	}).Info("Trip status updated successfully")

	return trip, nil
}

// UpdateTripLocation - Real-time location tracking
func (s *EnhancedTripService) UpdateTripLocation(ctx context.Context, req *TripLocationUpdate) error {
	s.logger.WithFields(logger.Fields{
		"trip_id": req.TripID,
		"lat":     req.Location.Latitude,
		"lng":     req.Location.Longitude,
	}).Debug("Updating trip location")

	// Get current trip
	trip, err := s.tripRepo.GetTrip(ctx, req.TripID)
	if err != nil {
		return fmt.Errorf("failed to get trip: %w", err)
	}

	// Only update location for active trips
	if !s.isTripActive(trip.Status) {
		return errors.New("cannot update location for inactive trip")
	}

	// Update location
	trip.CurrentLocation = req.Location
	trip.UpdatedAt = time.Now()

	if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
		return fmt.Errorf("failed to update trip location: %w", err)
	}

	// Record location event
	event := &TripEvent{
		ID:        s.generateEventID(),
		TripID:    trip.ID,
		EventType: EventLocationUpdated,
		Data: map[string]interface{}{
			"location": req.Location,
		},
		Timestamp: req.Timestamp,
	}

	if err := s.eventRepo.SaveEvent(ctx, event); err != nil {
		s.logger.WithError(err).Error("Failed to save location update event")
	}

	return nil
}

// GetTrip - Retrieve trip with full details
func (s *EnhancedTripService) GetTrip(ctx context.Context, tripID string) (*Trip, error) {
	return s.tripRepo.GetTrip(ctx, tripID)
}

// GetTripHistory - Get trip events for history/analytics
func (s *EnhancedTripService) GetTripHistory(ctx context.Context, tripID string) ([]*TripEvent, error) {
	return s.eventRepo.GetTripEvents(ctx, tripID)
}

// Helper methods

func (s *EnhancedTripService) validateCreateTripRequest(req *CreateTripRequest) error {
	if req.RiderID == "" {
		return errors.New("rider_id is required")
	}
	if req.PickupLocation == nil {
		return errors.New("pickup_location is required")
	}
	if req.Destination == nil {
		return errors.New("destination is required")
	}
	if req.RideType == "" {
		return errors.New("ride_type is required")
	}
	return nil
}

func (s *EnhancedTripService) isValidTransition(currentStatus, newStatus string) bool {
	validStates, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, validState := range validStates {
		if validState == newStatus {
			return true
		}
	}
	return false
}

func (s *EnhancedTripService) isTripActive(status string) bool {
	activeStates := []string{
		TripStateMatched,
		TripStateDriverEnRoute,
		TripStateDriverArrived,
		TripStateStarted,
		TripStateInProgress,
	}

	for _, activeState := range activeStates {
		if status == activeState {
			return true
		}
	}
	return false
}

func (s *EnhancedTripService) getEventTypeForStatus(status string) string {
	eventMap := map[string]string{
		TripStateRequested:     EventTripRequested,
		TripStateMatching:      EventMatchingStarted,
		TripStateMatched:       EventDriverMatched,
		TripStateDriverEnRoute: EventDriverEnRoute,
		TripStateDriverArrived: EventDriverArrived,
		TripStateStarted:       EventTripStarted,
		TripStateCompleted:     EventTripCompleted,
		TripStateCancelled:     EventTripCancelled,
	}

	if eventType, exists := eventMap[status]; exists {
		return eventType
	}
	return "trip.status_updated"
}

func (s *EnhancedTripService) getAreaFromLocation(location *models.Location) string {
	// Mock implementation - in real system would use geofencing
	return "downtown"
}

func (s *EnhancedTripService) generateTripID() string {
	return fmt.Sprintf("trip_%d", time.Now().UnixNano())
}

func (s *EnhancedTripService) generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}
