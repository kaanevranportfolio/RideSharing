package repository

import (
	"context"
	"time"

	"github.com/rideshare-platform/shared/models"
)

// Enhanced Trip with all the fields for the state machine
type Trip struct {
	ID                 string                 `json:"id" bson:"_id"`
	RiderID            string                 `json:"rider_id" bson:"rider_id"`
	DriverID           string                 `json:"driver_id,omitempty" bson:"driver_id,omitempty"`
	VehicleID          string                 `json:"vehicle_id,omitempty" bson:"vehicle_id,omitempty"`
	Status             string                 `json:"status" bson:"status"`
	RideType           string                 `json:"ride_type" bson:"ride_type"`
	PickupLocation     *models.Location       `json:"pickup_location" bson:"pickup_location"`
	Destination        *models.Location       `json:"destination" bson:"destination"`
	CurrentLocation    *models.Location       `json:"current_location,omitempty" bson:"current_location,omitempty"`
	EstimatedFare      float64                `json:"estimated_fare" bson:"estimated_fare"`
	ActualFare         float64                `json:"actual_fare" bson:"actual_fare"`
	Distance           float64                `json:"distance_km" bson:"distance_km"`
	EstimatedDuration  int                    `json:"estimated_duration_seconds" bson:"estimated_duration_seconds"`
	ActualDuration     int                    `json:"actual_duration_seconds" bson:"actual_duration_seconds"`
	RequestedAt        time.Time              `json:"requested_at" bson:"requested_at"`
	MatchedAt          *time.Time             `json:"matched_at,omitempty" bson:"matched_at,omitempty"`
	StartedAt          *time.Time             `json:"started_at,omitempty" bson:"started_at,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty" bson:"cancelled_at,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty" bson:"cancellation_reason,omitempty"`
	PaymentStatus      string                 `json:"payment_status" bson:"payment_status"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" bson:"updated_at"`
}

// TripRepository interface for trip data operations
type TripRepository interface {
	CreateTrip(ctx context.Context, trip *Trip) error
	GetTrip(ctx context.Context, tripID string) (*Trip, error)
	UpdateTrip(ctx context.Context, trip *Trip) error
	GetTripsByRider(ctx context.Context, riderID string, limit int, offset int) ([]*Trip, error)
	GetTripsByDriver(ctx context.Context, driverID string, limit int, offset int) ([]*Trip, error)
	GetTripsByStatus(ctx context.Context, status string, limit int, offset int) ([]*Trip, error)
	GetActiveTripByRider(ctx context.Context, riderID string) (*Trip, error)
	GetActiveTripByDriver(ctx context.Context, driverID string) (*Trip, error)
}

// TripEvent for event sourcing
type TripEvent struct {
	ID        string                 `json:"id" bson:"_id"`
	TripID    string                 `json:"trip_id" bson:"trip_id"`
	EventType string                 `json:"event_type" bson:"event_type"`
	Data      map[string]interface{} `json:"data" bson:"data"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
	UserID    string                 `json:"user_id,omitempty" bson:"user_id,omitempty"`
}

// EventRepository interface for event sourcing
type EventRepository interface {
	SaveEvent(ctx context.Context, event *TripEvent) error
	GetTripEvents(ctx context.Context, tripID string) ([]*TripEvent, error)
	GetEventsByType(ctx context.Context, eventType string, limit int, offset int) ([]*TripEvent, error)
	GetEventsByUser(ctx context.Context, userID string, limit int, offset int) ([]*TripEvent, error)
	GetEventsAfter(ctx context.Context, timestamp time.Time, limit int) ([]*TripEvent, error)
}
