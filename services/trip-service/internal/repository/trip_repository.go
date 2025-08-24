package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type TripRepository struct {
	db *sql.DB
}

type Trip struct {
	ID                       string                 `json:"id"`
	RiderID                  string                 `json:"rider_id"`
	DriverID                 *string                `json:"driver_id,omitempty"`
	VehicleID                *string                `json:"vehicle_id,omitempty"`
	PickupLocation           map[string]interface{} `json:"pickup_location"`
	Destination              map[string]interface{} `json:"destination"`
	Status                   string                 `json:"status"`
	EstimatedFareCents       *int64                 `json:"estimated_fare_cents,omitempty"`
	ActualFareCents          *int64                 `json:"actual_fare_cents,omitempty"`
	Currency                 string                 `json:"currency"`
	EstimatedDistanceKm      *float64               `json:"estimated_distance_km,omitempty"`
	ActualDistanceKm         *float64               `json:"actual_distance_km,omitempty"`
	EstimatedDurationSeconds *int                   `json:"estimated_duration_seconds,omitempty"`
	ActualDurationSeconds    *int                   `json:"actual_duration_seconds,omitempty"`
	RequestedAt              time.Time              `json:"requested_at"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at"`
}

func NewTripRepository(db *sql.DB) *TripRepository {
	return &TripRepository{db: db}
}

func (r *TripRepository) CreateTrip(ctx context.Context, trip *Trip) (*Trip, error) {
	// Generate UUID if not provided
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	trip.RequestedAt = now
	trip.CreatedAt = now
	trip.UpdatedAt = now

	// Set default values
	if trip.Status == "" {
		trip.Status = "requested"
	}
	if trip.Currency == "" {
		trip.Currency = "USD"
	}

	// Create trips table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS trips (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		rider_id UUID NOT NULL,
		driver_id UUID,
		vehicle_id UUID,
		pickup_location JSONB NOT NULL,
		destination JSONB NOT NULL,
		actual_route JSONB,
		status VARCHAR(20) NOT NULL DEFAULT 'requested' CHECK (status IN (
			'requested', 'matched', 'driver_assigned', 'driver_arriving', 
			'driver_arrived', 'trip_started', 'in_progress', 'completed', 
			'cancelled', 'failed'
		)),
		estimated_fare_cents BIGINT,
		actual_fare_cents BIGINT,
		currency VARCHAR(3) DEFAULT 'USD',
		estimated_distance_km DECIMAL(8,2),
		actual_distance_km DECIMAL(8,2),
		estimated_duration_seconds INTEGER,
		actual_duration_seconds INTEGER,
		requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		matched_at TIMESTAMP WITH TIME ZONE,
		driver_assigned_at TIMESTAMP WITH TIME ZONE,
		driver_arrived_at TIMESTAMP WITH TIME ZONE,
		started_at TIMESTAMP WITH TIME ZONE,
		completed_at TIMESTAMP WITH TIME ZONE,
		cancelled_by VARCHAR(20),
		cancellation_reason TEXT,
		passenger_count INTEGER DEFAULT 1,
		special_requests TEXT,
		promo_code VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	_, err := r.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return nil, err
	}

	// Convert location maps to JSON
	pickupJSON, err := json.Marshal(trip.PickupLocation)
	if err != nil {
		return nil, err
	}

	destinationJSON, err := json.Marshal(trip.Destination)
	if err != nil {
		return nil, err
	}

	// Insert the trip
	insertSQL := `
	INSERT INTO trips (
		id, rider_id, driver_id, vehicle_id, pickup_location, destination,
		status, estimated_fare_cents, actual_fare_cents, currency,
		estimated_distance_km, actual_distance_km, estimated_duration_seconds,
		actual_duration_seconds, requested_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err = r.db.ExecContext(ctx, insertSQL,
		trip.ID, trip.RiderID, trip.DriverID, trip.VehicleID,
		string(pickupJSON), string(destinationJSON),
		trip.Status, trip.EstimatedFareCents, trip.ActualFareCents, trip.Currency,
		trip.EstimatedDistanceKm, trip.ActualDistanceKm,
		trip.EstimatedDurationSeconds, trip.ActualDurationSeconds,
		trip.RequestedAt, trip.CreatedAt, trip.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return trip, nil
}

func (r *TripRepository) GetTrip(ctx context.Context, id string) (*Trip, error) {
	var trip Trip
	var pickupLocationJSON, destinationJSON string

	query := `
	SELECT id, rider_id, driver_id, vehicle_id, pickup_location, destination,
		   status, estimated_fare_cents, actual_fare_cents, currency,
		   estimated_distance_km, actual_distance_km, estimated_duration_seconds,
		   actual_duration_seconds, requested_at, created_at, updated_at
	FROM trips WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&trip.ID, &trip.RiderID, &trip.DriverID, &trip.VehicleID,
		&pickupLocationJSON, &destinationJSON,
		&trip.Status, &trip.EstimatedFareCents, &trip.ActualFareCents, &trip.Currency,
		&trip.EstimatedDistanceKm, &trip.ActualDistanceKm,
		&trip.EstimatedDurationSeconds, &trip.ActualDurationSeconds,
		&trip.RequestedAt, &trip.CreatedAt, &trip.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse JSON locations
	err = json.Unmarshal([]byte(pickupLocationJSON), &trip.PickupLocation)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(destinationJSON), &trip.Destination)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}
