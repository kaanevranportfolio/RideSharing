package types

import (
	"context"
	"time"

	"github.com/rideshare-platform/shared/models"
)

// TripState represents the current state of a trip
type TripState string

const (
	TripStateRequested  TripState = "requested"
	TripStateMatching   TripState = "matching"
	TripStateMatched    TripState = "matched"
	TripStateDriverEn   TripState = "driver_en_route"
	TripStateArrived    TripState = "driver_arrived"
	TripStatePickedUp   TripState = "picked_up"
	TripStateInProgress TripState = "in_progress"
	TripStateCompleted  TripState = "completed"
	TripStateCancelled  TripState = "cancelled"
	TripStateDisputed   TripState = "disputed"
)

// TripEventType represents different types of trip events
type TripEventType string

const (
	EventTripRequested    TripEventType = "trip_requested"
	EventMatchingStarted  TripEventType = "matching_started"
	EventDriverMatched    TripEventType = "driver_matched"
	EventDriverEnRoute    TripEventType = "driver_en_route"
	EventDriverArrived    TripEventType = "driver_arrived"
	EventTripStarted      TripEventType = "trip_started"
	EventTripCompleted    TripEventType = "trip_completed"
	EventTripCancelled    TripEventType = "trip_cancelled"
	EventPaymentProcessed TripEventType = "payment_processed"
	EventTripRated        TripEventType = "trip_rated"
	EventTripDisputed     TripEventType = "trip_disputed"
	EventLocationUpdate   TripEventType = "location_update"
	EventETAUpdate        TripEventType = "eta_update"
)

// TripEvent represents an event in the trip lifecycle
type TripEvent struct {
	ID        string                 `json:"id"`
	TripID    string                 `json:"trip_id"`
	Type      TripEventType          `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Version   int                    `json:"version"`
	UserID    string                 `json:"user_id,omitempty"`
}

// TripAggregate represents the current state of a trip built from events
type TripAggregate struct {
	ID                  string                 `json:"id"`
	RiderID             string                 `json:"rider_id"`
	DriverID            string                 `json:"driver_id,omitempty"`
	VehicleID           string                 `json:"vehicle_id,omitempty"`
	State               TripState              `json:"state"`
	PickupLocation      *models.Location       `json:"pickup_location"`
	DestinationLocation *models.Location       `json:"destination_location"`
	CurrentLocation     *models.Location       `json:"current_location,omitempty"`
	RequestedAt         time.Time              `json:"requested_at"`
	MatchedAt           *time.Time             `json:"matched_at,omitempty"`
	StartedAt           *time.Time             `json:"started_at,omitempty"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
	CancelledAt         *time.Time             `json:"cancelled_at,omitempty"`
	EstimatedFare       *float64               `json:"estimated_fare,omitempty"`
	ActualFare          *float64               `json:"actual_fare,omitempty"`
	Distance            *float64               `json:"distance,omitempty"`
	Duration            *time.Duration         `json:"duration,omitempty"`
	Rating              *float64               `json:"rating,omitempty"`
	VehicleType         string                 `json:"vehicle_type"`
	PaymentMethod       string                 `json:"payment_method"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Version             int                    `json:"version"`
	LastUpdated         time.Time              `json:"last_updated"`
}

// TripRequest represents a new trip request
type TripRequest struct {
	RiderID         string           `json:"rider_id" binding:"required"`
	PickupLocation  *models.Location `json:"pickup_location" binding:"required"`
	Destination     *models.Location `json:"destination" binding:"required"`
	VehicleType     string           `json:"vehicle_type" binding:"required"`
	PaymentMethod   string           `json:"payment_method" binding:"required"`
	ScheduledTime   *time.Time       `json:"scheduled_time,omitempty"`
	SpecialRequests []string         `json:"special_requests,omitempty"`
	PriorityLevel   int              `json:"priority_level"`
}

// TripMatchRequest represents a driver match for a trip
type TripMatchRequest struct {
	TripID    string  `json:"trip_id" binding:"required"`
	DriverID  string  `json:"driver_id" binding:"required"`
	VehicleID string  `json:"vehicle_id" binding:"required"`
	ETA       int     `json:"eta"`
	Fare      float64 `json:"fare"`
}

// TripLocationUpdate represents a location update during a trip
type TripLocationUpdate struct {
	TripID    string           `json:"trip_id" binding:"required"`
	Location  *models.Location `json:"location" binding:"required"`
	Heading   *float64         `json:"heading,omitempty"`
	Speed     *float64         `json:"speed,omitempty"`
	UpdatedBy string           `json:"updated_by"`
	Timestamp time.Time        `json:"timestamp"`
}

// TripEventStore interface for event storage
type TripEventStore interface {
	SaveEvent(ctx context.Context, event *TripEvent) error
	GetEvents(ctx context.Context, tripID string) ([]*TripEvent, error)
	GetEventsAfterVersion(ctx context.Context, tripID string, version int) ([]*TripEvent, error)
}

// TripReadModel interface for read-side projections
type TripReadModel interface {
	SaveTrip(ctx context.Context, trip *TripAggregate) error
	GetTrip(ctx context.Context, tripID string) (*TripAggregate, error)
	GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*TripAggregate, error)
	GetTripsByDriver(ctx context.Context, driverID string, limit, offset int) ([]*TripAggregate, error)
	GetActiveTrips(ctx context.Context) ([]*TripAggregate, error)
}
