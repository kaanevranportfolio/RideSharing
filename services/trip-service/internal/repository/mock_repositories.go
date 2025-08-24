package repository

import (
	"context"
	"fmt"

	"github.com/rideshare-platform/services/trip-service/internal/types"
)

// MockEventStore implements TripEventStore for testing
type MockEventStore struct {
	events map[string][]*types.TripEvent
}

// NewMockEventStore creates a new mock event store
func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events: make(map[string][]*types.TripEvent),
	}
}

// SaveEvent saves an event to memory
func (m *MockEventStore) SaveEvent(ctx context.Context, event *types.TripEvent) error {
	if _, exists := m.events[event.TripID]; !exists {
		m.events[event.TripID] = []*types.TripEvent{}
	}
	m.events[event.TripID] = append(m.events[event.TripID], event)
	return nil
}

// GetEvents returns all events for a trip
func (m *MockEventStore) GetEvents(ctx context.Context, tripID string) ([]*types.TripEvent, error) {
	events, exists := m.events[tripID]
	if !exists {
		return []*types.TripEvent{}, nil
	}
	return events, nil
}

// GetEventsAfterVersion returns events after a specific version
func (m *MockEventStore) GetEventsAfterVersion(ctx context.Context, tripID string, version int) ([]*types.TripEvent, error) {
	allEvents, err := m.GetEvents(ctx, tripID)
	if err != nil {
		return nil, err
	}

	var filteredEvents []*types.TripEvent
	for _, event := range allEvents {
		if event.Version > version {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents, nil
}

// MockReadModel implements TripReadModel for testing
type MockReadModel struct {
	trips map[string]*types.TripAggregate
}

// NewMockReadModel creates a new mock read model
func NewMockReadModel() *MockReadModel {
	return &MockReadModel{
		trips: make(map[string]*types.TripAggregate),
	}
}

// SaveTrip saves a trip to memory
func (m *MockReadModel) SaveTrip(ctx context.Context, trip *types.TripAggregate) error {
	m.trips[trip.ID] = trip
	return nil
}

// GetTrip retrieves a trip by ID
func (m *MockReadModel) GetTrip(ctx context.Context, tripID string) (*types.TripAggregate, error) {
	trip, exists := m.trips[tripID]
	if !exists {
		return nil, fmt.Errorf("trip not found: %s", tripID)
	}
	return trip, nil
}

// GetTripsByRider retrieves trips for a rider
func (m *MockReadModel) GetTripsByRider(ctx context.Context, riderID string, limit, offset int) ([]*types.TripAggregate, error) {
	var trips []*types.TripAggregate
	count := 0

	for _, trip := range m.trips {
		if trip.RiderID == riderID {
			if count >= offset && len(trips) < limit {
				trips = append(trips, trip)
			}
			count++
		}
	}
	return trips, nil
}

// GetTripsByDriver retrieves trips for a driver
func (m *MockReadModel) GetTripsByDriver(ctx context.Context, driverID string, limit, offset int) ([]*types.TripAggregate, error) {
	var trips []*types.TripAggregate
	count := 0

	for _, trip := range m.trips {
		if trip.DriverID == driverID {
			if count >= offset && len(trips) < limit {
				trips = append(trips, trip)
			}
			count++
		}
	}
	return trips, nil
}

// GetActiveTrips retrieves all active trips
func (m *MockReadModel) GetActiveTrips(ctx context.Context) ([]*types.TripAggregate, error) {
	var activeTrips []*types.TripAggregate

	for _, trip := range m.trips {
		if trip.State != types.TripStateCompleted && trip.State != types.TripStateCancelled {
			activeTrips = append(activeTrips, trip)
		}
	}
	return activeTrips, nil
}
