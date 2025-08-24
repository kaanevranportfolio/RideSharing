package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rideshare-platform/services/trip-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

// AdvancedTripService implements sophisticated trip lifecycle management
type AdvancedTripService struct {
	eventStore   types.TripEventStore
	readModel    types.TripReadModel
	logger       logger.Logger
	stateMachine *TripStateMachine
}

// TripStateMachine manages valid state transitions
type TripStateMachine struct {
	transitions map[types.TripState][]types.TripState
}

// NewTripStateMachine creates a new trip state machine
func NewTripStateMachine() *TripStateMachine {
	transitions := map[types.TripState][]types.TripState{
		types.TripStateRequested:  {types.TripStateMatching, types.TripStateCancelled},
		types.TripStateMatching:   {types.TripStateMatched, types.TripStateCancelled},
		types.TripStateMatched:    {types.TripStateDriverEn, types.TripStateCancelled},
		types.TripStateDriverEn:   {types.TripStateArrived, types.TripStateCancelled},
		types.TripStateArrived:    {types.TripStatePickedUp, types.TripStateCancelled},
		types.TripStatePickedUp:   {types.TripStateInProgress, types.TripStateCancelled},
		types.TripStateInProgress: {types.TripStateCompleted, types.TripStateCancelled, types.TripStateDisputed},
		types.TripStateCompleted:  {types.TripStateDisputed},
		types.TripStateCancelled:  {},
		types.TripStateDisputed:   {types.TripStateCompleted},
	}

	return &TripStateMachine{transitions: transitions}
}

// CanTransition checks if a state transition is valid
func (sm *TripStateMachine) CanTransition(from, to types.TripState) bool {
	allowedStates, exists := sm.transitions[from]
	if !exists {
		return false
	}

	for _, state := range allowedStates {
		if state == to {
			return true
		}
	}
	return false
}

// NewAdvancedTripService creates a new advanced trip service
func NewAdvancedTripService(eventStore types.TripEventStore, readModel types.TripReadModel, logger logger.Logger) *AdvancedTripService {
	return &AdvancedTripService{
		eventStore:   eventStore,
		readModel:    readModel,
		logger:       logger,
		stateMachine: NewTripStateMachine(),
	}
}

// RequestTrip initiates a new trip request
func (s *AdvancedTripService) RequestTrip(ctx context.Context, request *types.TripRequest) (*types.TripAggregate, error) {
	tripID := uuid.New().String()
	now := time.Now()

	// Create trip requested event
	eventData := map[string]interface{}{
		"rider_id":         request.RiderID,
		"pickup_location":  request.PickupLocation,
		"destination":      request.Destination,
		"vehicle_type":     request.VehicleType,
		"payment_method":   request.PaymentMethod,
		"priority_level":   request.PriorityLevel,
		"special_requests": request.SpecialRequests,
	}

	if request.ScheduledTime != nil {
		eventData["scheduled_time"] = request.ScheduledTime
	}

	event := &types.TripEvent{
		ID:        uuid.New().String(),
		TripID:    tripID,
		Type:      types.EventTripRequested,
		Data:      eventData,
		Timestamp: now,
		Version:   1,
		UserID:    request.RiderID,
	}

	// Save event
	if err := s.eventStore.SaveEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to save trip requested event: %w", err)
	}

	// Build aggregate from events
	aggregate := &types.TripAggregate{
		ID:                  tripID,
		RiderID:             request.RiderID,
		State:               types.TripStateRequested,
		PickupLocation:      request.PickupLocation,
		DestinationLocation: request.Destination,
		RequestedAt:         now,
		VehicleType:         request.VehicleType,
		PaymentMethod:       request.PaymentMethod,
		Version:             1,
		LastUpdated:         now,
		Metadata: map[string]interface{}{
			"priority_level":   request.PriorityLevel,
			"special_requests": request.SpecialRequests,
		},
	}

	// Save to read model
	if err := s.readModel.SaveTrip(ctx, aggregate); err != nil {
		s.logger.WithError(err).Error("Failed to save trip to read model")
	}

	s.logger.WithFields(logger.Fields{
		"trip_id":  tripID,
		"rider_id": request.RiderID,
	}).Info("Trip requested successfully")

	return aggregate, nil
}

// MatchDriver assigns a driver to a trip
func (s *AdvancedTripService) MatchDriver(ctx context.Context, matchRequest *types.TripMatchRequest) error {
	aggregate, err := s.getTripAggregate(ctx, matchRequest.TripID)
	if err != nil {
		return err
	}

	if !s.stateMachine.CanTransition(aggregate.State, types.TripStateMatched) {
		return fmt.Errorf("cannot transition from %s to %s", aggregate.State, types.TripStateMatched)
	}

	now := time.Now()
	eventData := map[string]interface{}{
		"driver_id":  matchRequest.DriverID,
		"vehicle_id": matchRequest.VehicleID,
		"eta":        matchRequest.ETA,
		"fare":       matchRequest.Fare,
		"matched_at": now,
	}

	event := &types.TripEvent{
		ID:        uuid.New().String(),
		TripID:    matchRequest.TripID,
		Type:      types.EventDriverMatched,
		Data:      eventData,
		Timestamp: now,
		Version:   aggregate.Version + 1,
		UserID:    matchRequest.DriverID,
	}

	if err := s.eventStore.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save driver matched event: %w", err)
	}

	// Update aggregate
	aggregate.State = types.TripStateMatched
	aggregate.DriverID = matchRequest.DriverID
	aggregate.VehicleID = matchRequest.VehicleID
	aggregate.MatchedAt = &now
	aggregate.EstimatedFare = &matchRequest.Fare
	aggregate.Version++
	aggregate.LastUpdated = now

	if err := s.readModel.SaveTrip(ctx, aggregate); err != nil {
		s.logger.WithError(err).Error("Failed to update trip in read model")
	}

	return nil
}

// CompleteTrip marks a trip as completed
func (s *AdvancedTripService) CompleteTrip(ctx context.Context, tripID string, actualFare float64, distance float64, duration time.Duration) error {
	aggregate, err := s.getTripAggregate(ctx, tripID)
	if err != nil {
		return err
	}

	if !s.stateMachine.CanTransition(aggregate.State, types.TripStateCompleted) {
		return fmt.Errorf("cannot transition from %s to %s", aggregate.State, types.TripStateCompleted)
	}

	now := time.Now()
	eventData := map[string]interface{}{
		"completed_at": now,
		"actual_fare":  actualFare,
		"distance":     distance,
		"duration":     duration.Seconds(),
	}

	event := &types.TripEvent{
		ID:        uuid.New().String(),
		TripID:    tripID,
		Type:      types.EventTripCompleted,
		Data:      eventData,
		Timestamp: now,
		Version:   aggregate.Version + 1,
	}

	if err := s.eventStore.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save trip completed event: %w", err)
	}

	// Update aggregate
	aggregate.State = types.TripStateCompleted
	aggregate.CompletedAt = &now
	aggregate.ActualFare = &actualFare
	aggregate.Distance = &distance
	aggregate.Duration = &duration
	aggregate.Version++
	aggregate.LastUpdated = now

	if err := s.readModel.SaveTrip(ctx, aggregate); err != nil {
		s.logger.WithError(err).Error("Failed to update completed trip in read model")
	}

	return nil
}

// CancelTrip cancels a trip
func (s *AdvancedTripService) CancelTrip(ctx context.Context, tripID, userID, reason string) error {
	aggregate, err := s.getTripAggregate(ctx, tripID)
	if err != nil {
		return err
	}

	if !s.stateMachine.CanTransition(aggregate.State, types.TripStateCancelled) {
		return fmt.Errorf("cannot transition from %s to %s", aggregate.State, types.TripStateCancelled)
	}

	now := time.Now()
	eventData := map[string]interface{}{
		"cancelled_at":   now,
		"cancelled_by":   userID,
		"cancel_reason":  reason,
		"previous_state": string(aggregate.State),
	}

	event := &types.TripEvent{
		ID:        uuid.New().String(),
		TripID:    tripID,
		Type:      types.EventTripCancelled,
		Data:      eventData,
		Timestamp: now,
		Version:   aggregate.Version + 1,
		UserID:    userID,
	}

	if err := s.eventStore.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save trip cancelled event: %w", err)
	}

	// Update aggregate
	aggregate.State = types.TripStateCancelled
	aggregate.CancelledAt = &now
	aggregate.Version++
	aggregate.LastUpdated = now

	if aggregate.Metadata == nil {
		aggregate.Metadata = make(map[string]interface{})
	}
	aggregate.Metadata["cancel_reason"] = reason
	aggregate.Metadata["cancelled_by"] = userID

	if err := s.readModel.SaveTrip(ctx, aggregate); err != nil {
		s.logger.WithError(err).Error("Failed to update cancelled trip in read model")
	}

	return nil
}

// GetTrip retrieves a trip by ID
func (s *AdvancedTripService) GetTrip(ctx context.Context, tripID string) (*types.TripAggregate, error) {
	return s.readModel.GetTrip(ctx, tripID)
}

// GetTripsByRider retrieves trips for a specific rider
func (s *AdvancedTripService) GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*types.TripAggregate, error) {
	return s.readModel.GetTripsByRider(ctx, riderID, limit, offset)
}

// GetActiveTrips retrieves all currently active trips
func (s *AdvancedTripService) GetActiveTrips(ctx context.Context) ([]*types.TripAggregate, error) {
	return s.readModel.GetActiveTrips(ctx)
}

// getTripAggregate rebuilds a trip aggregate from events or reads from cache
func (s *AdvancedTripService) getTripAggregate(ctx context.Context, tripID string) (*types.TripAggregate, error) {
	// Try read model first for performance
	aggregate, err := s.readModel.GetTrip(ctx, tripID)
	if err == nil && aggregate != nil {
		return aggregate, nil
	}

	// Fallback to rebuilding from events
	events, err := s.eventStore.GetEvents(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for trip %s: %w", tripID, err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("trip not found: %s", tripID)
	}

	return s.buildAggregateFromEvents(events), nil
}

// buildAggregateFromEvents reconstructs trip state from events
func (s *AdvancedTripService) buildAggregateFromEvents(events []*types.TripEvent) *types.TripAggregate {
	if len(events) == 0 {
		return nil
	}

	aggregate := &types.TripAggregate{
		ID:       events[0].TripID,
		Metadata: make(map[string]interface{}),
	}

	for _, event := range events {
		s.applyEvent(aggregate, event)
	}

	return aggregate
}

// applyEvent applies a single event to the aggregate
func (s *AdvancedTripService) applyEvent(aggregate *types.TripAggregate, event *types.TripEvent) {
	aggregate.Version = event.Version
	aggregate.LastUpdated = event.Timestamp

	switch event.Type {
	case types.EventTripRequested:
		aggregate.RiderID = event.Data["rider_id"].(string)
		aggregate.State = types.TripStateRequested
		aggregate.RequestedAt = event.Timestamp
		aggregate.VehicleType = event.Data["vehicle_type"].(string)
		aggregate.PaymentMethod = event.Data["payment_method"].(string)

	case types.EventMatchingStarted:
		aggregate.State = types.TripStateMatching

	case types.EventDriverMatched:
		aggregate.State = types.TripStateMatched
		aggregate.DriverID = event.Data["driver_id"].(string)
		aggregate.VehicleID = event.Data["vehicle_id"].(string)

	case types.EventTripCompleted:
		aggregate.State = types.TripStateCompleted
		if completedAt, ok := event.Data["completed_at"].(time.Time); ok {
			aggregate.CompletedAt = &completedAt
		}

	case types.EventTripCancelled:
		aggregate.State = types.TripStateCancelled
		if cancelledAt, ok := event.Data["cancelled_at"].(time.Time); ok {
			aggregate.CancelledAt = &cancelledAt
		}
	}
}
