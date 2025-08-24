package service

import (
	"context"
	"time"

	"github.com/rideshare-platform/shared/logger"
)

// BasicTripService interface for our gRPC handler
type BasicTripService interface {
	GetTrip(ctx context.Context, tripID string) (*BasicTrip, error)
}

// BasicTrip represents a simple trip for our implementation
type BasicTrip struct {
	ID        string    `json:"id"`
	RiderID   string    `json:"rider_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Status    string    `json:"status"`
	RideType  string    `json:"ride_type"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SimpleTripService implements the BasicTripService interface
type SimpleTripService struct {
	logger *logger.Logger
}

// NewBasicTripService creates a new basic trip service instance
func NewBasicTripService(logger *logger.Logger) BasicTripService {
	return &SimpleTripService{
		logger: logger,
	}
}

// GetTrip retrieves a trip by ID (mock implementation)
func (s *SimpleTripService) GetTrip(ctx context.Context, tripID string) (*BasicTrip, error) {
	s.logger.WithFields(logger.Fields{
		"trip_id": tripID,
	}).Debug("Getting trip")

	// Mock trip for testing
	trip := &BasicTrip{
		ID:        tripID,
		RiderID:   "rider_123",
		DriverID:  "driver_456",
		Status:    "requested",
		RideType:  "standard",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		UpdatedAt: time.Now(),
	}

	return trip, nil
}
