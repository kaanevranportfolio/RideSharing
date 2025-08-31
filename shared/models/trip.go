package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// TripStatus represents the current status of a trip
type TripStatus string

// TripError represents a trip-related error
type TripError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *TripError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

const (
	TripStatusRequested      TripStatus = "requested"
	TripStatusMatched        TripStatus = "matched"
	TripStatusDriverAssigned TripStatus = "driver_assigned"
	TripStatusDriverArriving TripStatus = "driver_arriving"
	TripStatusDriverArrived  TripStatus = "driver_arrived"
	TripStatusTripStarted    TripStatus = "trip_started"
	TripStatusInProgress     TripStatus = "in_progress"
	TripStatusCompleted      TripStatus = "completed"
	TripStatusCancelled      TripStatus = "cancelled"
	TripStatusFailed         TripStatus = "failed"
)

// Trip represents a trip in the rideshare platform
type Trip struct {
	ID                       string      `json:"id" db:"id"`
	RiderID                  string      `json:"rider_id" db:"rider_id"`
	DriverID                 *string     `json:"driver_id" db:"driver_id"`
	VehicleID                *string     `json:"vehicle_id" db:"vehicle_id"`
	PickupLocation           Location    `json:"pickup_location" db:"pickup_location"`
	Destination              Location    `json:"destination" db:"destination"`
	ActualRoute              *[]Location `json:"actual_route,omitempty" db:"actual_route"`
	Status                   TripStatus  `json:"status" db:"status"`
	EstimatedFareCents       *int64      `json:"estimated_fare_cents" db:"estimated_fare_cents"`
	ActualFareCents          *int64      `json:"actual_fare_cents" db:"actual_fare_cents"`
	Currency                 string      `json:"currency" db:"currency"`
	EstimatedDistanceKm      *float64    `json:"estimated_distance_km" db:"estimated_distance_km"`
	ActualDistanceKm         *float64    `json:"actual_distance_km" db:"actual_distance_km"`
	EstimatedDurationSeconds *int        `json:"estimated_duration_seconds" db:"estimated_duration_seconds"`
	ActualDurationSeconds    *int        `json:"actual_duration_seconds" db:"actual_duration_seconds"`
	RequestedAt              time.Time   `json:"requested_at" db:"requested_at"`
	MatchedAt                *time.Time  `json:"matched_at" db:"matched_at"`
	DriverAssignedAt         *time.Time  `json:"driver_assigned_at" db:"driver_assigned_at"`
	DriverArrivedAt          *time.Time  `json:"driver_arrived_at" db:"driver_arrived_at"`
	StartedAt                *time.Time  `json:"started_at" db:"started_at"`
	CompletedAt              *time.Time  `json:"completed_at" db:"completed_at"`
	CancelledBy              *string     `json:"cancelled_by" db:"cancelled_by"`
	CancellationReason       *string     `json:"cancellation_reason" db:"cancellation_reason"`
	PassengerCount           int         `json:"passenger_count" db:"passenger_count"`
	SpecialRequests          *string     `json:"special_requests" db:"special_requests"`
	PromoCode                *string     `json:"promo_code" db:"promo_code"`
	CreatedAt                time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at" db:"updated_at"`
}

// TripEvent represents an event in the trip lifecycle for event sourcing
type TripEvent struct {
	ID           string                 `json:"id" db:"id"`
	TripID       string                 `json:"trip_id" db:"trip_id"`
	EventType    string                 `json:"event_type" db:"event_type"`
	EventData    map[string]interface{} `json:"event_data" db:"event_data"`
	EventVersion int                    `json:"event_version" db:"event_version"`
	UserID       *string                `json:"user_id" db:"user_id"`
	Timestamp    time.Time              `json:"timestamp" db:"timestamp"`
	Metadata     map[string]string      `json:"metadata" db:"metadata"`
}

// NewTrip creates a new trip with default values
func NewTrip(riderID string, pickupLocation, destination Location, passengerCount int) *Trip {
	return &Trip{
		ID:             generateID(),
		RiderID:        riderID,
		PickupLocation: pickupLocation,
		Destination:    destination,
		Status:         TripStatusRequested,
		Currency:       "USD",
		PassengerCount: passengerCount,
		RequestedAt:    time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// NewTripEvent creates a new trip event
func NewTripEvent(tripID, eventType string, eventData map[string]interface{}, userID *string) *TripEvent {
	return &TripEvent{
		ID:        generateID(),
		TripID:    tripID,
		EventType: eventType,
		EventData: eventData,
		UserID:    userID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// IsActive returns true if the trip is in an active state
func (t *Trip) IsActive() bool {
	activeStatuses := []TripStatus{
		TripStatusRequested,
		TripStatusMatched,
		TripStatusDriverAssigned,
		TripStatusDriverArriving,
		TripStatusDriverArrived,
		TripStatusTripStarted,
		TripStatusInProgress,
	}

	for _, status := range activeStatuses {
		if t.Status == status {
			return true
		}
	}
	return false
}

// IsCompleted returns true if the trip is completed
func (t *Trip) IsCompleted() bool {
	return t.Status == TripStatusCompleted
}

// IsCancelled returns true if the trip is cancelled
func (t *Trip) IsCancelled() bool {
	return t.Status == TripStatusCancelled
}

// HasDriver returns true if the trip has a driver assigned
func (t *Trip) HasDriver() bool {
	return t.DriverID != nil
}

// HasVehicle returns true if the trip has a vehicle assigned
func (t *Trip) HasVehicle() bool {
	return t.VehicleID != nil
}

// UpdateStatus updates the trip status with state machine validation
func (t *Trip) UpdateStatus(status TripStatus, userID *string) (*TripEvent, error) {
	// Validate state transition
	if !t.isValidTransition(t.Status, status) {
		return nil, &TripError{
			Code:    "INVALID_STATE_TRANSITION",
			Message: fmt.Sprintf("Cannot transition from %s to %s", t.Status, status),
		}
	}

	oldStatus := t.Status
	t.Status = status
	t.UpdatedAt = time.Now()

	// Set appropriate timestamps based on status
	now := time.Now()
	switch status {
	case TripStatusMatched:
		t.MatchedAt = &now
	case TripStatusDriverAssigned:
		t.DriverAssignedAt = &now
	case TripStatusDriverArrived:
		t.DriverArrivedAt = &now
	case TripStatusTripStarted:
		t.StartedAt = &now
	case TripStatusCompleted:
		t.CompletedAt = &now
	}

	// Create event for event sourcing
	eventData := map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(status),
		"timestamp":  now,
	}

	return NewTripEvent(t.ID, "status_changed", eventData, userID), nil
}

// AssignDriver assigns a driver and vehicle to the trip
func (t *Trip) AssignDriver(driverID, vehicleID string, userID *string) *TripEvent {
	t.DriverID = &driverID
	t.VehicleID = &vehicleID
	t.UpdatedAt = time.Now()

	eventData := map[string]interface{}{
		"driver_id":  driverID,
		"vehicle_id": vehicleID,
		"timestamp":  time.Now(),
	}

	return NewTripEvent(t.ID, "driver_assigned", eventData, userID)
}

// SetEstimatedFare sets the estimated fare for the trip
func (t *Trip) SetEstimatedFare(fareCents int64) {
	t.EstimatedFareCents = &fareCents
	t.UpdatedAt = time.Now()
}

// SetActualFare sets the actual fare for the trip
func (t *Trip) SetActualFare(fareCents int64) {
	t.ActualFareCents = &fareCents
	t.UpdatedAt = time.Now()
}

// SetEstimatedDistance sets the estimated distance for the trip
func (t *Trip) SetEstimatedDistance(distanceKm float64) {
	t.EstimatedDistanceKm = &distanceKm
	t.UpdatedAt = time.Now()
}

// SetActualDistance sets the actual distance for the trip
func (t *Trip) SetActualDistance(distanceKm float64) {
	t.ActualDistanceKm = &distanceKm
	t.UpdatedAt = time.Now()
}

// SetEstimatedDuration sets the estimated duration for the trip
func (t *Trip) SetEstimatedDuration(durationSeconds int) {
	t.EstimatedDurationSeconds = &durationSeconds
	t.UpdatedAt = time.Now()
}

// SetActualDuration sets the actual duration for the trip
func (t *Trip) SetActualDuration(durationSeconds int) {
	t.ActualDurationSeconds = &durationSeconds
	t.UpdatedAt = time.Now()
}

// Cancel cancels the trip with a reason
func (t *Trip) Cancel(cancelledBy, reason string, userID *string) *TripEvent {
	t.Status = TripStatusCancelled
	t.CancelledBy = &cancelledBy
	t.CancellationReason = &reason
	t.UpdatedAt = time.Now()

	eventData := map[string]interface{}{
		"cancelled_by": cancelledBy,
		"reason":       reason,
		"timestamp":    time.Now(),
	}

	return NewTripEvent(t.ID, "trip_cancelled", eventData, userID)
}

// AddRoutePoint adds a point to the actual route
func (t *Trip) AddRoutePoint(location Location) {
	if t.ActualRoute == nil {
		t.ActualRoute = &[]Location{}
	}
	*t.ActualRoute = append(*t.ActualRoute, location)
	t.UpdatedAt = time.Now()
}

// GetDuration returns the trip duration in seconds
func (t *Trip) GetDuration() *int {
	if t.StartedAt != nil && t.CompletedAt != nil {
		duration := int(t.CompletedAt.Sub(*t.StartedAt).Seconds())
		return &duration
	}
	return nil
}

// GetDistance returns the distance between pickup and destination
func (t *Trip) GetDistance() float64 {
	return t.PickupLocation.DistanceTo(&t.Destination)
}

// SetPromoCode sets the promo code for the trip
func (t *Trip) SetPromoCode(promoCode string) {
	t.PromoCode = &promoCode
	t.UpdatedAt = time.Now()
}

// SetSpecialRequests sets special requests for the trip
func (t *Trip) SetSpecialRequests(requests string) {
	t.SpecialRequests = &requests
	t.UpdatedAt = time.Now()
}

// MarshalEventData marshals event data to JSON
func (te *TripEvent) MarshalEventData() ([]byte, error) {
	return json.Marshal(te.EventData)
}

// UnmarshalEventData unmarshals event data from JSON
func (te *TripEvent) UnmarshalEventData(data []byte) error {
	return json.Unmarshal(data, &te.EventData)
}

// IsValidTripStatus checks if a trip status is valid
func IsValidTripStatus(status string) bool {
	validStatuses := []TripStatus{
		TripStatusRequested,
		TripStatusMatched,
		TripStatusDriverAssigned,
		TripStatusDriverArriving,
		TripStatusDriverArrived,
		TripStatusTripStarted,
		TripStatusInProgress,
		TripStatusCompleted,
		TripStatusCancelled,
		TripStatusFailed,
	}

	for _, validStatus := range validStatuses {
		if TripStatus(status) == validStatus {
			return true
		}
	}
	return false
}

// GetTripStatuses returns all valid trip statuses
func GetTripStatuses() []TripStatus {
	return []TripStatus{
		TripStatusRequested,
		TripStatusMatched,
		TripStatusDriverAssigned,
		TripStatusDriverArriving,
		TripStatusDriverArrived,
		TripStatusTripStarted,
		TripStatusInProgress,
		TripStatusCompleted,
		TripStatusCancelled,
		TripStatusFailed,
	}
}

// isValidTransition validates state transitions according to business rules
func (t *Trip) isValidTransition(from, to TripStatus) bool {
	// Define valid state transitions
	validTransitions := map[TripStatus][]TripStatus{
		TripStatusRequested: {
			TripStatusMatched,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusMatched: {
			TripStatusDriverAssigned,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusDriverAssigned: {
			TripStatusDriverArriving,
			TripStatusDriverArrived,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusDriverArriving: {
			TripStatusDriverArrived,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusDriverArrived: {
			TripStatusTripStarted,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusTripStarted: {
			TripStatusInProgress,
			TripStatusCompleted,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusInProgress: {
			TripStatusCompleted,
			TripStatusCancelled,
			TripStatusFailed,
		},
		TripStatusCompleted: {}, // Terminal state
		TripStatusCancelled: {}, // Terminal state
		TripStatusFailed:    {}, // Terminal state
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if to == allowedState {
			return true
		}
	}

	return false
}

// ProcessStateTransition processes a state transition with business logic validation
func (t *Trip) ProcessStateTransition(newStatus TripStatus, context *TransitionContext) (*TripEvent, error) {
	// Validate state transition
	if !t.isValidTransition(t.Status, newStatus) {
		return nil, &TripError{
			Code:    "INVALID_STATE_TRANSITION",
			Message: fmt.Sprintf("Cannot transition from %s to %s", t.Status, newStatus),
		}
	}

	// Apply business rules based on transition
	switch newStatus {
	case TripStatusDriverAssigned:
		if context.DriverID == "" || context.VehicleID == "" {
			return nil, &TripError{
				Code:    "MISSING_DRIVER_INFO",
				Message: "Driver ID and Vehicle ID are required for driver assignment",
			}
		}
		return t.assignDriverInternal(context.DriverID, context.VehicleID, context.UserID)

	case TripStatusTripStarted:
		if context.StartLocation == nil {
			return nil, &TripError{
				Code:    "MISSING_START_LOCATION",
				Message: "Start location is required when starting trip",
			}
		}
		return t.startTripInternal(*context.StartLocation, context.UserID)

	case TripStatusCompleted:
		if context.EndLocation == nil {
			return nil, &TripError{
				Code:    "MISSING_END_LOCATION",
				Message: "End location is required when completing trip",
			}
		}
		return t.completeTripInternal(*context.EndLocation, context.FinalFare, context.UserID)

	default:
		// Default state transition
		return t.UpdateStatus(newStatus, context.UserID)
	}
}

// TransitionContext holds context data for state transitions
type TransitionContext struct {
	UserID        *string
	DriverID      string
	VehicleID     string
	StartLocation *Location
	EndLocation   *Location
	FinalFare     *int64
}

// assignDriverInternal handles driver assignment with validation
func (t *Trip) assignDriverInternal(driverID, vehicleID string, userID *string) (*TripEvent, error) {
	if t.DriverID != nil {
		return nil, &TripError{
			Code:    "DRIVER_ALREADY_ASSIGNED",
			Message: "Trip already has a driver assigned",
		}
	}

	t.DriverID = &driverID
	t.VehicleID = &vehicleID
	t.Status = TripStatusDriverAssigned
	t.DriverAssignedAt = &time.Time{}
	*t.DriverAssignedAt = time.Now()
	t.UpdatedAt = time.Now()

	eventData := map[string]interface{}{
		"driver_id":  driverID,
		"vehicle_id": vehicleID,
		"timestamp":  time.Now(),
	}

	return NewTripEvent(t.ID, "driver_assigned", eventData, userID), nil
}

// startTripInternal handles trip start with validation
func (t *Trip) startTripInternal(startLocation Location, userID *string) (*TripEvent, error) {
	if t.DriverID == nil {
		return nil, &TripError{
			Code:    "NO_DRIVER_ASSIGNED",
			Message: "Cannot start trip without driver assignment",
		}
	}

	t.Status = TripStatusTripStarted
	t.StartedAt = &time.Time{}
	*t.StartedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Initialize actual route with start location
	t.ActualRoute = &[]Location{startLocation}

	eventData := map[string]interface{}{
		"start_location": startLocation,
		"timestamp":      time.Now(),
	}

	return NewTripEvent(t.ID, "trip_started", eventData, userID), nil
}

// completeTripInternal handles trip completion with validation
func (t *Trip) completeTripInternal(endLocation Location, finalFare *int64, userID *string) (*TripEvent, error) {
	if t.StartedAt == nil {
		return nil, &TripError{
			Code:    "TRIP_NOT_STARTED",
			Message: "Cannot complete trip that hasn't been started",
		}
	}

	t.Status = TripStatusCompleted
	t.CompletedAt = &time.Time{}
	*t.CompletedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Add end location to route
	if t.ActualRoute != nil {
		*t.ActualRoute = append(*t.ActualRoute, endLocation)
	}

	// Set final fare if provided
	if finalFare != nil {
		t.ActualFareCents = finalFare
	}

	// Calculate actual duration
	if t.StartedAt != nil {
		duration := int(t.CompletedAt.Sub(*t.StartedAt).Seconds())
		t.ActualDurationSeconds = &duration
	}

	eventData := map[string]interface{}{
		"end_location": endLocation,
		"final_fare":   finalFare,
		"timestamp":    time.Now(),
	}

	return NewTripEvent(t.ID, "trip_completed", eventData, userID), nil
}

// ApplyEvent applies an event to reconstruct trip state (for event sourcing)
func (t *Trip) ApplyEvent(event *TripEvent) error {
	switch event.EventType {
	case "status_changed":
		if newStatus, ok := event.EventData["new_status"].(string); ok {
			t.Status = TripStatus(newStatus)
		}
	case "driver_assigned":
		if driverID, ok := event.EventData["driver_id"].(string); ok {
			t.DriverID = &driverID
		}
		if vehicleID, ok := event.EventData["vehicle_id"].(string); ok {
			t.VehicleID = &vehicleID
		}
	case "trip_started":
		t.Status = TripStatusTripStarted
		if timestamp, ok := event.EventData["timestamp"].(time.Time); ok {
			t.StartedAt = &timestamp
		}
	case "trip_completed":
		t.Status = TripStatusCompleted
		if timestamp, ok := event.EventData["timestamp"].(time.Time); ok {
			t.CompletedAt = &timestamp
		}
		if finalFare, ok := event.EventData["final_fare"].(int64); ok {
			t.ActualFareCents = &finalFare
		}
	case "trip_cancelled":
		t.Status = TripStatusCancelled
		if cancelledBy, ok := event.EventData["cancelled_by"].(string); ok {
			t.CancelledBy = &cancelledBy
		}
		if reason, ok := event.EventData["reason"].(string); ok {
			t.CancellationReason = &reason
		}
	}

	// Update timestamp to event timestamp
	t.UpdatedAt = event.Timestamp
	return nil
}

// ReplayEvents replays a sequence of events to reconstruct trip state
func (t *Trip) ReplayEvents(events []*TripEvent) error {
	for _, event := range events {
		if err := t.ApplyEvent(event); err != nil {
			return fmt.Errorf("failed to replay event %s: %w", event.ID, err)
		}
	}
	return nil
}
