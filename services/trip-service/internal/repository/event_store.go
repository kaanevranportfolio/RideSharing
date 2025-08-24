package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rideshare-platform/services/trip-service/internal/types"
	"github.com/rideshare-platform/shared/logger"
)

// PostgreSQLEventStore implements TripEventStore using PostgreSQL
type PostgreSQLEventStore struct {
	db     *sql.DB
	logger logger.Logger
}

// NewPostgreSQLEventStore creates a new PostgreSQL event store
func NewPostgreSQLEventStore(db *sql.DB, logger logger.Logger) *PostgreSQLEventStore {
	return &PostgreSQLEventStore{
		db:     db,
		logger: logger,
	}
}

// SaveEvent saves a trip event to the event store
func (s *PostgreSQLEventStore) SaveEvent(ctx context.Context, event *types.TripEvent) error {
	query := `
		INSERT INTO trip_events (id, trip_id, event_type, event_data, timestamp, version, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	eventData, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.TripID,
		string(event.Type),
		eventData,
		event.Timestamp,
		event.Version,
		event.UserID,
	)

	if err != nil {
		s.logger.WithError(err).WithFields(logger.Fields{
			"event_id": event.ID,
			"trip_id":  event.TripID,
			"type":     event.Type,
		}).Error("Failed to save trip event")
		return fmt.Errorf("failed to save event: %w", err)
	}

	s.logger.WithFields(logger.Fields{
		"event_id": event.ID,
		"trip_id":  event.TripID,
		"type":     event.Type,
		"version":  event.Version,
	}).Debug("Trip event saved successfully")

	return nil
}

// GetEvents retrieves all events for a trip
func (s *PostgreSQLEventStore) GetEvents(ctx context.Context, tripID string) ([]*types.TripEvent, error) {
	query := `
		SELECT id, trip_id, event_type, event_data, timestamp, version, user_id
		FROM trip_events
		WHERE trip_id = $1
		ORDER BY version ASC
	`

	rows, err := s.db.QueryContext(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*types.TripEvent

	for rows.Next() {
		var event types.TripEvent
		var eventDataJSON []byte
		var userID sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.TripID,
			&event.Type,
			&eventDataJSON,
			&event.Timestamp,
			&event.Version,
			&userID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(eventDataJSON, &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		if userID.Valid {
			event.UserID = userID.String
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetEventsAfterVersion retrieves events for a trip after a specific version
func (s *PostgreSQLEventStore) GetEventsAfterVersion(ctx context.Context, tripID string, version int) ([]*types.TripEvent, error) {
	query := `
		SELECT id, trip_id, event_type, event_data, timestamp, version, user_id
		FROM trip_events
		WHERE trip_id = $1 AND version > $2
		ORDER BY version ASC
	`

	rows, err := s.db.QueryContext(ctx, query, tripID, version)
	if err != nil {
		return nil, fmt.Errorf("failed to query events after version: %w", err)
	}
	defer rows.Close()

	var events []*types.TripEvent

	for rows.Next() {
		var event types.TripEvent
		var eventDataJSON []byte
		var userID sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.TripID,
			&event.Type,
			&eventDataJSON,
			&event.Timestamp,
			&event.Version,
			&userID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(eventDataJSON, &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		if userID.Valid {
			event.UserID = userID.String
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events after version: %w", err)
	}

	return events, nil
}

// PostgreSQLTripReadModel implements TripReadModel using PostgreSQL
type PostgreSQLTripReadModel struct {
	db     *sql.DB
	logger logger.Logger
}

// NewPostgreSQLTripReadModel creates a new PostgreSQL read model
func NewPostgreSQLTripReadModel(db *sql.DB, logger logger.Logger) *PostgreSQLTripReadModel {
	return &PostgreSQLTripReadModel{
		db:     db,
		logger: logger,
	}
}

// SaveTrip saves a trip aggregate to the read model
func (r *PostgreSQLTripReadModel) SaveTrip(ctx context.Context, trip *types.TripAggregate) error {
	query := `
		INSERT INTO trips (
			id, rider_id, driver_id, vehicle_id, state, pickup_location, destination_location,
			current_location, requested_at, matched_at, started_at, completed_at, cancelled_at,
			estimated_fare, actual_fare, distance, duration, rating, vehicle_type, payment_method,
			metadata, version, last_updated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		ON CONFLICT (id) DO UPDATE SET
			driver_id = EXCLUDED.driver_id,
			vehicle_id = EXCLUDED.vehicle_id,
			state = EXCLUDED.state,
			current_location = EXCLUDED.current_location,
			matched_at = EXCLUDED.matched_at,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			cancelled_at = EXCLUDED.cancelled_at,
			estimated_fare = EXCLUDED.estimated_fare,
			actual_fare = EXCLUDED.actual_fare,
			distance = EXCLUDED.distance,
			duration = EXCLUDED.duration,
			rating = EXCLUDED.rating,
			metadata = EXCLUDED.metadata,
			version = EXCLUDED.version,
			last_updated = EXCLUDED.last_updated
		WHERE trips.version < EXCLUDED.version
	`

	pickupLocationJSON, _ := json.Marshal(trip.PickupLocation)
	destinationLocationJSON, _ := json.Marshal(trip.DestinationLocation)
	currentLocationJSON, _ := json.Marshal(trip.CurrentLocation)
	metadataJSON, _ := json.Marshal(trip.Metadata)

	var durationSeconds sql.NullFloat64
	if trip.Duration != nil {
		durationSeconds.Valid = true
		durationSeconds.Float64 = trip.Duration.Seconds()
	}

	_, err := r.db.ExecContext(ctx, query,
		trip.ID,
		trip.RiderID,
		stringToNullString(trip.DriverID),
		stringToNullString(trip.VehicleID),
		string(trip.State),
		pickupLocationJSON,
		destinationLocationJSON,
		nullableJSON(currentLocationJSON),
		trip.RequestedAt,
		timeToNullTime(trip.MatchedAt),
		timeToNullTime(trip.StartedAt),
		timeToNullTime(trip.CompletedAt),
		timeToNullTime(trip.CancelledAt),
		float64ToNullFloat64(trip.EstimatedFare),
		float64ToNullFloat64(trip.ActualFare),
		float64ToNullFloat64(trip.Distance),
		durationSeconds,
		float64ToNullFloat64(trip.Rating),
		trip.VehicleType,
		trip.PaymentMethod,
		metadataJSON,
		trip.Version,
		trip.LastUpdated,
	)

	if err != nil {
		r.logger.WithError(err).WithField("trip_id", trip.ID).Error("Failed to save trip to read model")
		return fmt.Errorf("failed to save trip: %w", err)
	}

	return nil
}

// GetTrip retrieves a trip by ID from the read model
func (r *PostgreSQLTripReadModel) GetTrip(ctx context.Context, tripID string) (*types.TripAggregate, error) {
	query := `
		SELECT id, rider_id, driver_id, vehicle_id, state, pickup_location, destination_location,
			current_location, requested_at, matched_at, started_at, completed_at, cancelled_at,
			estimated_fare, actual_fare, distance, duration, rating, vehicle_type, payment_method,
			metadata, version, last_updated
		FROM trips
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, tripID)

	trip, err := r.scanTrip(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	return trip, nil
}

// GetTripsByRider retrieves trips for a specific rider
func (r *PostgreSQLTripReadModel) GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*types.TripAggregate, error) {
	query := `
		SELECT id, rider_id, driver_id, vehicle_id, state, pickup_location, destination_location,
			current_location, requested_at, matched_at, started_at, completed_at, cancelled_at,
			estimated_fare, actual_fare, distance, duration, rating, vehicle_type, payment_method,
			metadata, version, last_updated
		FROM trips
		WHERE rider_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, riderID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query trips by rider: %w", err)
	}
	defer rows.Close()

	return r.scanTrips(rows)
}

// GetTripsByDriver retrieves trips for a specific driver
func (r *PostgreSQLTripReadModel) GetTripsByDriver(ctx context.Context, driverID string, limit, offset int) ([]*types.TripAggregate, error) {
	query := `
		SELECT id, rider_id, driver_id, vehicle_id, state, pickup_location, destination_location,
			current_location, requested_at, matched_at, started_at, completed_at, cancelled_at,
			estimated_fare, actual_fare, distance, duration, rating, vehicle_type, payment_method,
			metadata, version, last_updated
		FROM trips
		WHERE driver_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, driverID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query trips by driver: %w", err)
	}
	defer rows.Close()

	return r.scanTrips(rows)
}

// GetActiveTrips retrieves all currently active trips
func (r *PostgreSQLTripReadModel) GetActiveTrips(ctx context.Context) ([]*types.TripAggregate, error) {
	query := `
		SELECT id, rider_id, driver_id, vehicle_id, state, pickup_location, destination_location,
			current_location, requested_at, matched_at, started_at, completed_at, cancelled_at,
			estimated_fare, actual_fare, distance, duration, rating, vehicle_type, payment_method,
			metadata, version, last_updated
		FROM trips
		WHERE state NOT IN ('completed', 'cancelled')
		ORDER BY requested_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active trips: %w", err)
	}
	defer rows.Close()

	return r.scanTrips(rows)
}

// Helper methods

func (r *PostgreSQLTripReadModel) scanTrip(row *sql.Row) (*types.TripAggregate, error) {
	var trip types.TripAggregate
	var driverID, vehicleID sql.NullString
	var pickupLocationJSON, destinationLocationJSON, currentLocationJSON, metadataJSON []byte
	var matchedAt, startedAt, completedAt, cancelledAt sql.NullTime
	var estimatedFare, actualFare, distance, rating sql.NullFloat64
	var duration sql.NullFloat64

	err := row.Scan(
		&trip.ID,
		&trip.RiderID,
		&driverID,
		&vehicleID,
		&trip.State,
		&pickupLocationJSON,
		&destinationLocationJSON,
		&currentLocationJSON,
		&trip.RequestedAt,
		&matchedAt,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&estimatedFare,
		&actualFare,
		&distance,
		&duration,
		&rating,
		&trip.VehicleType,
		&trip.PaymentMethod,
		&metadataJSON,
		&trip.Version,
		&trip.LastUpdated,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if driverID.Valid {
		trip.DriverID = driverID.String
	}
	if vehicleID.Valid {
		trip.VehicleID = vehicleID.String
	}
	if matchedAt.Valid {
		trip.MatchedAt = &matchedAt.Time
	}
	if startedAt.Valid {
		trip.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		trip.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		trip.CancelledAt = &cancelledAt.Time
	}
	if estimatedFare.Valid {
		trip.EstimatedFare = &estimatedFare.Float64
	}
	if actualFare.Valid {
		trip.ActualFare = &actualFare.Float64
	}
	if distance.Valid {
		trip.Distance = &distance.Float64
	}
	if duration.Valid {
		dur := time.Duration(duration.Float64) * time.Second
		trip.Duration = &dur
	}
	if rating.Valid {
		trip.Rating = &rating.Float64
	}

	// Unmarshal JSON fields
	if len(pickupLocationJSON) > 0 {
		json.Unmarshal(pickupLocationJSON, &trip.PickupLocation)
	}
	if len(destinationLocationJSON) > 0 {
		json.Unmarshal(destinationLocationJSON, &trip.DestinationLocation)
	}
	if len(currentLocationJSON) > 0 {
		json.Unmarshal(currentLocationJSON, &trip.CurrentLocation)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &trip.Metadata)
	}

	return &trip, nil
}

func (r *PostgreSQLTripReadModel) scanTrips(rows *sql.Rows) ([]*types.TripAggregate, error) {
	var trips []*types.TripAggregate

	for rows.Next() {
		var trip types.TripAggregate
		var driverID, vehicleID sql.NullString
		var pickupLocationJSON, destinationLocationJSON, currentLocationJSON, metadataJSON []byte
		var matchedAt, startedAt, completedAt, cancelledAt sql.NullTime
		var estimatedFare, actualFare, distance, rating sql.NullFloat64
		var duration sql.NullFloat64

		err := rows.Scan(
			&trip.ID,
			&trip.RiderID,
			&driverID,
			&vehicleID,
			&trip.State,
			&pickupLocationJSON,
			&destinationLocationJSON,
			&currentLocationJSON,
			&trip.RequestedAt,
			&matchedAt,
			&startedAt,
			&completedAt,
			&cancelledAt,
			&estimatedFare,
			&actualFare,
			&distance,
			&duration,
			&rating,
			&trip.VehicleType,
			&trip.PaymentMethod,
			&metadataJSON,
			&trip.Version,
			&trip.LastUpdated,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}

		// Handle nullable fields (same as scanTrip)
		if driverID.Valid {
			trip.DriverID = driverID.String
		}
		if vehicleID.Valid {
			trip.VehicleID = vehicleID.String
		}
		if matchedAt.Valid {
			trip.MatchedAt = &matchedAt.Time
		}
		if startedAt.Valid {
			trip.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			trip.CompletedAt = &completedAt.Time
		}
		if cancelledAt.Valid {
			trip.CancelledAt = &cancelledAt.Time
		}
		if estimatedFare.Valid {
			trip.EstimatedFare = &estimatedFare.Float64
		}
		if actualFare.Valid {
			trip.ActualFare = &actualFare.Float64
		}
		if distance.Valid {
			trip.Distance = &distance.Float64
		}
		if duration.Valid {
			dur := time.Duration(duration.Float64) * time.Second
			trip.Duration = &dur
		}
		if rating.Valid {
			trip.Rating = &rating.Float64
		}

		// Unmarshal JSON fields
		if len(pickupLocationJSON) > 0 {
			json.Unmarshal(pickupLocationJSON, &trip.PickupLocation)
		}
		if len(destinationLocationJSON) > 0 {
			json.Unmarshal(destinationLocationJSON, &trip.DestinationLocation)
		}
		if len(currentLocationJSON) > 0 {
			json.Unmarshal(currentLocationJSON, &trip.CurrentLocation)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &trip.Metadata)
		}

		trips = append(trips, &trip)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trips: %w", err)
	}

	return trips, nil
}

// Helper functions for nullable types
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func timeToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func float64ToNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func nullableJSON(data []byte) interface{} {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return data
}
