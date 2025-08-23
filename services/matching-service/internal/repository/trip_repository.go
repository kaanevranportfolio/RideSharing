package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TripRepository handles trip data operations
type TripRepository struct {
	collection *mongo.Collection
	logger     *logger.Logger
}

// NewTripRepository creates a new trip repository
func NewTripRepository(db *mongo.Database, logger *logger.Logger) *TripRepository {
	return &TripRepository{
		collection: db.Collection("trips"),
		logger:     logger,
	}
}

// CreateTrip creates a new trip
func (r *TripRepository) CreateTrip(ctx context.Context, trip *models.Trip) error {
	trip.RequestedAt = time.Now()
	trip.Status = models.TripStatusRequested

	_, err := r.collection.InsertOne(ctx, trip)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create trip")
		return fmt.Errorf("failed to create trip: %w", err)
	}

	r.logger.WithField("trip_id", trip.ID).Info("Trip created successfully")
	return nil
}

// GetTrip retrieves a trip by ID
func (r *TripRepository) GetTrip(ctx context.Context, tripID string) (*models.Trip, error) {
	var trip models.Trip
	err := r.collection.FindOne(ctx, bson.M{"id": tripID}).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("trip not found: %s", tripID)
		}
		r.logger.WithError(err).WithField("trip_id", tripID).Error("Failed to get trip")
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	return &trip, nil
}

// UpdateTripStatus updates the status of a trip
func (r *TripRepository) UpdateTripStatus(ctx context.Context, tripID string, status models.TripStatus) error {
	update := bson.M{"$set": bson.M{"status": status}}

	// Add timestamp based on status
	switch status {
	case models.TripStatusMatched:
		now := time.Now()
		update["$set"].(bson.M)["matched_at"] = now
	case models.TripStatusDriverAssigned:
		now := time.Now()
		update["$set"].(bson.M)["driver_assigned_at"] = now
	case models.TripStatusDriverArrived:
		now := time.Now()
		update["$set"].(bson.M)["driver_arrived_at"] = now
	case models.TripStatusTripStarted:
		now := time.Now()
		update["$set"].(bson.M)["started_at"] = now
	case models.TripStatusCompleted:
		now := time.Now()
		update["$set"].(bson.M)["completed_at"] = now
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"id": tripID}, update)
	if err != nil {
		r.logger.WithError(err).WithFields(logger.Fields{
			"trip_id": tripID,
			"status":  status,
		}).Error("Failed to update trip status")
		return fmt.Errorf("failed to update trip status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trip not found: %s", tripID)
	}

	r.logger.WithFields(logger.Fields{
		"trip_id": tripID,
		"status":  status,
	}).Info("Trip status updated successfully")

	return nil
}

// AssignDriver assigns a driver to a trip
func (r *TripRepository) AssignDriver(ctx context.Context, tripID, driverID, vehicleID string) error {
	update := bson.M{
		"$set": bson.M{
			"driver_id":          driverID,
			"vehicle_id":         vehicleID,
			"status":             models.TripStatusDriverAssigned,
			"driver_assigned_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"id": tripID}, update)
	if err != nil {
		r.logger.WithError(err).WithFields(logger.Fields{
			"trip_id":    tripID,
			"driver_id":  driverID,
			"vehicle_id": vehicleID,
		}).Error("Failed to assign driver to trip")
		return fmt.Errorf("failed to assign driver to trip: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trip not found: %s", tripID)
	}

	r.logger.WithFields(logger.Fields{
		"trip_id":    tripID,
		"driver_id":  driverID,
		"vehicle_id": vehicleID,
	}).Info("Driver assigned to trip successfully")

	return nil
}

// GetPendingTrips retrieves all trips that are pending matching
func (r *TripRepository) GetPendingTrips(ctx context.Context, limit int) ([]*models.Trip, error) {
	filter := bson.M{"status": models.TripStatusRequested}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.M{"requested_at": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get pending trips")
		return nil, fmt.Errorf("failed to get pending trips: %w", err)
	}
	defer cursor.Close(ctx)

	var trips []*models.Trip
	for cursor.Next(ctx) {
		var trip models.Trip
		if err := cursor.Decode(&trip); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip")
			continue
		}
		trips = append(trips, &trip)
	}

	if err := cursor.Err(); err != nil {
		r.logger.WithError(err).Error("Cursor error while getting pending trips")
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return trips, nil
}

// GetActiveTripsForDriver retrieves all active trips for a driver
func (r *TripRepository) GetActiveTripsForDriver(ctx context.Context, driverID string) ([]*models.Trip, error) {
	filter := bson.M{
		"driver_id": driverID,
		"status": bson.M{"$in": []models.TripStatus{
			models.TripStatusDriverAssigned,
			models.TripStatusDriverArriving,
			models.TripStatusDriverArrived,
			models.TripStatusTripStarted,
			models.TripStatusInProgress,
		}},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		r.logger.WithError(err).WithField("driver_id", driverID).Error("Failed to get active trips for driver")
		return nil, fmt.Errorf("failed to get active trips for driver: %w", err)
	}
	defer cursor.Close(ctx)

	var trips []*models.Trip
	for cursor.Next(ctx) {
		var trip models.Trip
		if err := cursor.Decode(&trip); err != nil {
			r.logger.WithError(err).Error("Failed to decode trip")
			continue
		}
		trips = append(trips, &trip)
	}

	return trips, nil
}
