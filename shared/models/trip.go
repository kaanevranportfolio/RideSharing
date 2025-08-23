package models

import (
	"encoding/json"
	"time"
)

// TripStatus represents the current status of a trip
type TripStatus string

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

// UpdateStatus updates the trip status and sets appropriate timestamps
func (t *Trip) UpdateStatus(status TripStatus, userID *string) *TripEvent {
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

	// Create event
	eventData := map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(status),
		"timestamp":  now,
	}

	return NewTripEvent(t.ID, "status_changed", eventData, userID)
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
