package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/logger"
)

// EventType represents the type of event
type EventType string

const (
	// User events
	UserRegisteredEvent  EventType = "user.registered"
	UserUpdatedEvent     EventType = "user.updated"
	UserDeactivatedEvent EventType = "user.deactivated"

	// Driver events
	DriverOnlineEvent     EventType = "driver.online"
	DriverOfflineEvent    EventType = "driver.offline"
	DriverLocationUpdated EventType = "driver.location_updated"

	// Trip events
	TripRequestedEvent EventType = "trip.requested"
	TripMatchedEvent   EventType = "trip.matched"
	TripStartedEvent   EventType = "trip.started"
	TripCompletedEvent EventType = "trip.completed"
	TripCancelledEvent EventType = "trip.cancelled"

	// Payment events
	PaymentProcessedEvent EventType = "payment.processed"
	PaymentFailedEvent    EventType = "payment.failed"
	PaymentRefundedEvent  EventType = "payment.refunded"

	// Vehicle events
	VehicleRegisteredEvent  EventType = "vehicle.registered"
	VehicleUpdatedEvent     EventType = "vehicle.updated"
	VehicleDeactivatedEvent EventType = "vehicle.deactivated"
)

// Event represents a domain event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	AggregateID string                 `json:"aggregate_id"`
	Version     int                    `json:"version"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, aggregateID string, version int, data map[string]interface{}, source string) *Event {
	return &Event{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateID: aggregateID,
		Version:     version,
		Data:        data,
		Metadata:    make(map[string]interface{}),
		Timestamp:   time.Now().UTC(),
		Source:      source,
	}
}

// AddMetadata adds metadata to the event
func (e *Event) AddMetadata(key string, value interface{}) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
}

// ToJSON converts the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON creates an event from JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// EventHandler represents an event handler function
type EventHandler func(ctx context.Context, event *Event) error

// EventBus represents an event bus for publishing and subscribing to events
type EventBus interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType, handler EventHandler) error
	Close() error
}

// InMemoryEventBus is a simple in-memory event bus implementation
type InMemoryEventBus struct {
	handlers map[EventType][]EventHandler
	logger   *logger.Logger
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(log *logger.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[EventType][]EventHandler),
		logger:   log,
	}
}

// Publish publishes an event to all subscribers
func (bus *InMemoryEventBus) Publish(ctx context.Context, event *Event) error {
	handlers, exists := bus.handlers[event.Type]
	if !exists {
		bus.logger.WithContext(ctx).WithFields(logger.Fields{
			"event_type": event.Type,
			"event_id":   event.ID,
		}).Debug("No handlers registered for event type")
		return nil
	}

	bus.logger.WithContext(ctx).WithFields(logger.Fields{
		"event_type":    event.Type,
		"event_id":      event.ID,
		"aggregate_id":  event.AggregateID,
		"handler_count": len(handlers),
	}).Info("Publishing event")

	// Execute handlers concurrently
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(ctx, event); err != nil {
				bus.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
					"event_type": event.Type,
					"event_id":   event.ID,
				}).Error("Event handler failed")
			}
		}(handler)
	}

	return nil
}

// Subscribe subscribes to events of a specific type
func (bus *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.WithFields(logger.Fields{
		"event_type":    eventType,
		"handler_count": len(bus.handlers[eventType]),
	}).Info("Event handler subscribed")

	return nil
}

// Unsubscribe removes a handler from event subscriptions
func (bus *InMemoryEventBus) Unsubscribe(eventType EventType, handler EventHandler) error {
	handlers, exists := bus.handlers[eventType]
	if !exists {
		return nil
	}

	// Remove handler (simple implementation - in production, use proper handler identification)
	for i, h := range handlers {
		if &h == &handler {
			bus.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	bus.logger.WithFields(logger.Fields{
		"event_type":    eventType,
		"handler_count": len(bus.handlers[eventType]),
	}).Info("Event handler unsubscribed")

	return nil
}

// Close closes the event bus
func (bus *InMemoryEventBus) Close() error {
	bus.handlers = make(map[EventType][]EventHandler)
	bus.logger.Logger.Info("Event bus closed")
	return nil
}

// EventStore represents an event store for persisting events
type EventStore interface {
	SaveEvent(ctx context.Context, event *Event) error
	GetEvents(ctx context.Context, aggregateID string) ([]*Event, error)
	GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]*Event, error)
	GetEventsAfter(ctx context.Context, timestamp time.Time, limit int) ([]*Event, error)
}

// InMemoryEventStore is a simple in-memory event store implementation
type InMemoryEventStore struct {
	events []Event
	logger *logger.Logger
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(log *logger.Logger) *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make([]Event, 0),
		logger: log,
	}
}

// SaveEvent saves an event to the store
func (store *InMemoryEventStore) SaveEvent(ctx context.Context, event *Event) error {
	store.events = append(store.events, *event)

	store.logger.WithContext(ctx).WithFields(logger.Fields{
		"event_type":   event.Type,
		"event_id":     event.ID,
		"aggregate_id": event.AggregateID,
		"version":      event.Version,
	}).Debug("Event saved to store")

	return nil
}

// GetEvents retrieves all events for an aggregate
func (store *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string) ([]*Event, error) {
	var events []*Event

	for _, event := range store.events {
		if event.AggregateID == aggregateID {
			events = append(events, &event)
		}
	}

	store.logger.WithContext(ctx).WithFields(logger.Fields{
		"aggregate_id": aggregateID,
		"event_count":  len(events),
	}).Debug("Retrieved events for aggregate")

	return events, nil
}

// GetEventsByType retrieves events by type
func (store *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]*Event, error) {
	var events []*Event
	count := 0

	for i := len(store.events) - 1; i >= 0 && count < limit; i-- {
		if store.events[i].Type == eventType {
			events = append(events, &store.events[i])
			count++
		}
	}

	store.logger.WithContext(ctx).WithFields(logger.Fields{
		"event_type":  eventType,
		"event_count": len(events),
		"limit":       limit,
	}).Debug("Retrieved events by type")

	return events, nil
}

// GetEventsAfter retrieves events after a specific timestamp
func (store *InMemoryEventStore) GetEventsAfter(ctx context.Context, timestamp time.Time, limit int) ([]*Event, error) {
	var events []*Event
	count := 0

	for i := len(store.events) - 1; i >= 0 && count < limit; i-- {
		if store.events[i].Timestamp.After(timestamp) {
			events = append(events, &store.events[i])
			count++
		}
	}

	store.logger.WithContext(ctx).WithFields(logger.Fields{
		"after":       timestamp,
		"event_count": len(events),
		"limit":       limit,
	}).Debug("Retrieved events after timestamp")

	return events, nil
}

// EventPublisher combines event bus and event store
type EventPublisher struct {
	bus    EventBus
	store  EventStore
	logger *logger.Logger
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(bus EventBus, store EventStore, log *logger.Logger) *EventPublisher {
	return &EventPublisher{
		bus:    bus,
		store:  store,
		logger: log,
	}
}

// PublishEvent publishes an event to both the bus and store
func (pub *EventPublisher) PublishEvent(ctx context.Context, event *Event) error {
	// Save to store first
	if err := pub.store.SaveEvent(ctx, event); err != nil {
		pub.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"event_type": event.Type,
			"event_id":   event.ID,
		}).Error("Failed to save event to store")
		return fmt.Errorf("failed to save event: %w", err)
	}

	// Then publish to bus
	if err := pub.bus.Publish(ctx, event); err != nil {
		pub.logger.WithContext(ctx).WithError(err).WithFields(logger.Fields{
			"event_type": event.Type,
			"event_id":   event.ID,
		}).Error("Failed to publish event to bus")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	pub.logger.WithContext(ctx).WithFields(logger.Fields{
		"event_type":   event.Type,
		"event_id":     event.ID,
		"aggregate_id": event.AggregateID,
	}).Info("Event published successfully")

	return nil
}

// Subscribe subscribes to events
func (pub *EventPublisher) Subscribe(eventType EventType, handler EventHandler) error {
	return pub.bus.Subscribe(eventType, handler)
}

// GetEvents retrieves events from store
func (pub *EventPublisher) GetEvents(ctx context.Context, aggregateID string) ([]*Event, error) {
	return pub.store.GetEvents(ctx, aggregateID)
}

// Close closes the publisher
func (pub *EventPublisher) Close() error {
	return pub.bus.Close()
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Simple implementation - in production, use proper UUID generation
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// EventHandlerRegistry manages event handlers
type EventHandlerRegistry struct {
	handlers map[EventType][]EventHandler
	logger   *logger.Logger
}

// NewEventHandlerRegistry creates a new event handler registry
func NewEventHandlerRegistry(log *logger.Logger) *EventHandlerRegistry {
	return &EventHandlerRegistry{
		handlers: make(map[EventType][]EventHandler),
		logger:   log,
	}
}

// Register registers an event handler
func (registry *EventHandlerRegistry) Register(eventType EventType, handler EventHandler) {
	registry.handlers[eventType] = append(registry.handlers[eventType], handler)

	registry.logger.WithFields(logger.Fields{
		"event_type":    eventType,
		"handler_count": len(registry.handlers[eventType]),
	}).Info("Event handler registered")
}

// GetHandlers returns handlers for an event type
func (registry *EventHandlerRegistry) GetHandlers(eventType EventType) []EventHandler {
	return registry.handlers[eventType]
}

// Clear clears all handlers
func (registry *EventHandlerRegistry) Clear() {
	registry.handlers = make(map[EventType][]EventHandler)
	registry.logger.Logger.Info("Event handler registry cleared")
}
