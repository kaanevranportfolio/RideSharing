package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rideshare-platform/shared/logger"
)

// MongoTripRepository implements TripRepository using MongoDB
type MongoTripRepository struct {
	db     *mongo.Database
	trips  *mongo.Collection
	logger logger.Logger
}

// MongoEventRepository implements EventRepository using MongoDB
type MongoEventRepository struct {
	db     *mongo.Database
	events *mongo.Collection
	logger logger.Logger
}

func NewMongoTripRepository(db *mongo.Database, logger logger.Logger) *MongoTripRepository {
	return &MongoTripRepository{
		db:     db,
		trips:  db.Collection("trips"),
		logger: logger,
	}
}

func NewMongoEventRepository(db *mongo.Database, logger logger.Logger) *MongoEventRepository {
	return &MongoEventRepository{
		db:     db,
		events: db.Collection("trip_events"),
		logger: logger,
	}
}

// TripRepository Implementation

func (r *MongoTripRepository) CreateTrip(ctx context.Context, trip *Trip) error {
	r.logger.WithFields(logger.Fields{
		"trip_id":  trip.ID,
		"rider_id": trip.RiderID,
	}).Info("Creating trip in database")

	_, err := r.trips.InsertOne(ctx, trip)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create trip")
		return fmt.Errorf("failed to create trip: %w", err)
	}

	return nil
}

func (r *MongoTripRepository) GetTrip(ctx context.Context, tripID string) (*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"trip_id": tripID,
	}).Debug("Getting trip from database")

	var trip Trip
	err := r.trips.FindOne(ctx, bson.M{"_id": tripID}).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("trip not found: %s", tripID)
		}
		r.logger.WithError(err).Error("Failed to get trip")
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	return &trip, nil
}

func (r *MongoTripRepository) UpdateTrip(ctx context.Context, trip *Trip) error {
	r.logger.WithFields(logger.Fields{
		"trip_id": trip.ID,
		"status":  trip.Status,
	}).Info("Updating trip in database")

	trip.UpdatedAt = time.Now()

	filter := bson.M{"_id": trip.ID}
	update := bson.M{"$set": trip}

	result, err := r.trips.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update trip")
		return fmt.Errorf("failed to update trip: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trip not found: %s", trip.ID)
	}

	return nil
}

func (r *MongoTripRepository) GetTripsByRider(ctx context.Context, riderID string, limit int, offset int) ([]*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"rider_id": riderID,
		"limit":    limit,
		"offset":   offset,
	}).Debug("Getting trips by rider")

	filter := bson.M{"rider_id": riderID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.trips.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find trips by rider")
		return nil, fmt.Errorf("failed to find trips: %w", err)
	}
	defer cursor.Close(ctx)

	var trips []*Trip
	for cursor.Next(ctx) {
		var trip Trip
		if err := cursor.Decode(&trip); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip")
			continue
		}
		trips = append(trips, &trip)
	}

	return trips, nil
}

func (r *MongoTripRepository) GetTripsByDriver(ctx context.Context, driverID string, limit int, offset int) ([]*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"driver_id": driverID,
		"limit":     limit,
		"offset":    offset,
	}).Debug("Getting trips by driver")

	filter := bson.M{"driver_id": driverID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.trips.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find trips by driver")
		return nil, fmt.Errorf("failed to find trips: %w", err)
	}
	defer cursor.Close(ctx)

	var trips []*Trip
	for cursor.Next(ctx) {
		var trip Trip
		if err := cursor.Decode(&trip); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip")
			continue
		}
		trips = append(trips, &trip)
	}

	return trips, nil
}

func (r *MongoTripRepository) GetTripsByStatus(ctx context.Context, status string, limit int, offset int) ([]*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"status": status,
		"limit":  limit,
		"offset": offset,
	}).Debug("Getting trips by status")

	filter := bson.M{"status": status}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.trips.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find trips by status")
		return nil, fmt.Errorf("failed to find trips: %w", err)
	}
	defer cursor.Close(ctx)

	var trips []*Trip
	for cursor.Next(ctx) {
		var trip Trip
		if err := cursor.Decode(&trip); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip")
			continue
		}
		trips = append(trips, &trip)
	}

	return trips, nil
}

func (r *MongoTripRepository) GetActiveTripByRider(ctx context.Context, riderID string) (*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"rider_id": riderID,
	}).Debug("Getting active trip by rider")

	activeStatuses := []string{"requested", "matching", "matched", "driver_en_route", "driver_arrived", "started", "in_progress"}
	filter := bson.M{
		"rider_id": riderID,
		"status":   bson.M{"$in": activeStatuses},
	}

	var trip Trip
	err := r.trips.FindOne(ctx, filter).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active trip
		}
		r.logger.WithError(err).Error("Failed to get active trip by rider")
		return nil, fmt.Errorf("failed to get active trip: %w", err)
	}

	return &trip, nil
}

func (r *MongoTripRepository) GetActiveTripByDriver(ctx context.Context, driverID string) (*Trip, error) {
	r.logger.WithFields(logger.Fields{
		"driver_id": driverID,
	}).Debug("Getting active trip by driver")

	activeStatuses := []string{"matched", "driver_en_route", "driver_arrived", "started", "in_progress"}
	filter := bson.M{
		"driver_id": driverID,
		"status":    bson.M{"$in": activeStatuses},
	}

	var trip Trip
	err := r.trips.FindOne(ctx, filter).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active trip
		}
		r.logger.WithError(err).Error("Failed to get active trip by driver")
		return nil, fmt.Errorf("failed to get active trip: %w", err)
	}

	return &trip, nil
}

// EventRepository Implementation

func (r *MongoEventRepository) SaveEvent(ctx context.Context, event *TripEvent) error {
	r.logger.WithFields(logger.Fields{
		"event_id":   event.ID,
		"trip_id":    event.TripID,
		"event_type": event.EventType,
	}).Debug("Saving trip event")

	_, err := r.events.InsertOne(ctx, event)
	if err != nil {
		r.logger.WithError(err).Error("Failed to save trip event")
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func (r *MongoEventRepository) GetTripEvents(ctx context.Context, tripID string) ([]*TripEvent, error) {
	r.logger.WithFields(logger.Fields{
		"trip_id": tripID,
	}).Debug("Getting trip events")

	filter := bson.M{"trip_id": tripID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}) // Chronological order

	cursor, err := r.events.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find trip events")
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*TripEvent
	for cursor.Next(ctx) {
		var event TripEvent
		if err := cursor.Decode(&event); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip event")
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

func (r *MongoEventRepository) GetEventsByType(ctx context.Context, eventType string, limit int, offset int) ([]*TripEvent, error) {
	r.logger.WithFields(logger.Fields{
		"event_type": eventType,
		"limit":      limit,
		"offset":     offset,
	}).Debug("Getting events by type")

	filter := bson.M{"event_type": eventType}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.events.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find events by type")
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*TripEvent
	for cursor.Next(ctx) {
		var event TripEvent
		if err := cursor.Decode(&event); err != nil {
			r.logger.WithError(err).Error("Failed to decode event")
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

func (r *MongoEventRepository) GetEventsByUser(ctx context.Context, userID string, limit int, offset int) ([]*TripEvent, error) {
	r.logger.WithFields(logger.Fields{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	}).Debug("Getting events by user")

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.events.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find events by user")
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*TripEvent
	for cursor.Next(ctx) {
		var event TripEvent
		if err := cursor.Decode(&event); err != nil {
			r.logger.WithError(err).Error("Failed to decode event")
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

func (r *MongoEventRepository) GetEventsAfter(ctx context.Context, timestamp time.Time, limit int) ([]*TripEvent, error) {
	r.logger.WithFields(logger.Fields{
		"after_timestamp": timestamp,
		"limit":           limit,
	}).Debug("Getting events after timestamp")

	filter := bson.M{"timestamp": bson.M{"$gt": timestamp}}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := r.events.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to find events after timestamp")
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*TripEvent
	for cursor.Next(ctx) {
		var event TripEvent
		if err := cursor.Decode(&event); err != nil {
			r.logger.WithError(err).Error("Failed to decode event")
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}
