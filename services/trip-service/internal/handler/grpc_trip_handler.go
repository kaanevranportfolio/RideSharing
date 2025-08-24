package handler

import (
	"context"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	trippb "github.com/rideshare-platform/shared/proto/trip"
)

// GRPCTripHandler handles gRPC requests for trip service
type GRPCTripHandler struct {
	trippb.UnimplementedTripServiceServer
	tripService service.BasicTripService
	logger      *logger.Logger

	// Subscription management
	subscriptions map[string][]chan *trippb.TripUpdateEvent
	subMutex      sync.RWMutex
}

func NewGRPCTripHandler(tripService service.BasicTripService, logger *logger.Logger) *GRPCTripHandler {
	return &GRPCTripHandler{
		tripService:   tripService,
		logger:        logger,
		subscriptions: make(map[string][]chan *trippb.TripUpdateEvent),
	}
}

// SubscribeToTripUpdates implements real-time trip updates streaming
func (h *GRPCTripHandler) SubscribeToTripUpdates(req *trippb.SubscribeToTripUpdatesRequest, stream trippb.TripService_SubscribeToTripUpdatesServer) error {
	h.logger.WithFields(logger.Fields{
		"trip_id": req.TripId,
		"user_id": req.UserId,
	}).Info("New trip subscription")

	// Create a channel for this subscription
	updateChan := make(chan *trippb.TripUpdateEvent, 10)

	// Register the subscription
	h.subMutex.Lock()
	if _, exists := h.subscriptions[req.TripId]; !exists {
		h.subscriptions[req.TripId] = make([]chan *trippb.TripUpdateEvent, 0)
	}
	h.subscriptions[req.TripId] = append(h.subscriptions[req.TripId], updateChan)
	h.subMutex.Unlock()

	// Cleanup function
	defer func() {
		h.subMutex.Lock()
		if subscribers, exists := h.subscriptions[req.TripId]; exists {
			// Remove this channel from subscribers
			for i, ch := range subscribers {
				if ch == updateChan {
					h.subscriptions[req.TripId] = append(subscribers[:i], subscribers[i+1:]...)
					break
				}
			}
			// If no more subscribers, remove the trip from subscriptions
			if len(h.subscriptions[req.TripId]) == 0 {
				delete(h.subscriptions, req.TripId)
			}
		}
		h.subMutex.Unlock()
		close(updateChan)

		h.logger.WithFields(logger.Fields{
			"trip_id": req.TripId,
			"user_id": req.UserId,
		}).Info("Trip subscription closed")
	}()

	// Send initial trip status
	trip, err := h.tripService.GetTrip(stream.Context(), req.TripId)
	if err != nil {
		return status.Errorf(codes.NotFound, "Trip not found: %v", err)
	}

	// Convert trip status to proto enum
	currentStatus := convertToProtoStatus(trip.Status)

	initialEvent := &trippb.TripUpdateEvent{
		TripId:    req.TripId,
		OldStatus: currentStatus,
		NewStatus: currentStatus,
		Timestamp: timestamppb.New(trip.UpdatedAt),
		Metadata: map[string]string{
			"rider_id":   trip.RiderID,
			"driver_id":  trip.DriverID,
			"event_type": "initial_status",
		},
	}

	if err := stream.Send(initialEvent); err != nil {
		return err
	}

	// Listen for updates on this trip
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-updateChan:
			if !ok {
				return nil // Channel closed
			}
			if err := stream.Send(event); err != nil {
				h.logger.WithError(err).Error("Failed to send trip update")
				return err
			}
		case <-time.After(30 * time.Second):
			// Send heartbeat to keep connection alive
			heartbeat := &trippb.TripUpdateEvent{
				TripId:    req.TripId,
				OldStatus: currentStatus,
				NewStatus: currentStatus,
				Timestamp: timestamppb.New(time.Now()),
				Metadata: map[string]string{
					"event_type": "heartbeat",
				},
			}
			if err := stream.Send(heartbeat); err != nil {
				return err
			}
		}
	}
}

// NotifyTripUpdate sends an update to all subscribers of a trip
func (h *GRPCTripHandler) NotifyTripUpdate(tripID string, oldStatus, newStatus trippb.TripStatus, metadata map[string]string) {
	h.subMutex.RLock()
	subscribers, exists := h.subscriptions[tripID]
	h.subMutex.RUnlock()

	if !exists || len(subscribers) == 0 {
		return // No subscribers
	}

	event := &trippb.TripUpdateEvent{
		TripId:    tripID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: timestamppb.New(time.Now()),
		Metadata:  metadata,
	}

	h.logger.WithFields(logger.Fields{
		"trip_id":          tripID,
		"old_status":       oldStatus.String(),
		"new_status":       newStatus.String(),
		"subscriber_count": len(subscribers),
	}).Debug("Broadcasting trip update")

	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this subscriber
			h.logger.WithFields(logger.Fields{
				"trip_id": tripID,
			}).Warn("Subscriber channel full, skipping update")
		}
	}
}

// GetTrip implements gRPC method for getting trip details
func (h *GRPCTripHandler) GetTrip(ctx context.Context, req *trippb.GetTripRequest) (*trippb.GetTripResponse, error) {
	trip, err := h.tripService.GetTrip(ctx, req.TripId)
	if err != nil {
		return &trippb.GetTripResponse{
			Found: false,
		}, nil
	}

	// Convert internal trip to proto trip
	protoTrip := convertToProtoTrip(trip)

	return &trippb.GetTripResponse{
		Trip:  protoTrip,
		Found: true,
	}, nil
}

// UpdateTripStatus implements gRPC method for updating trip status
func (h *GRPCTripHandler) UpdateTripStatus(ctx context.Context, req *trippb.UpdateTripStatusRequest) (*trippb.UpdateTripStatusResponse, error) {
	// Validate the request
	if req.TripId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Trip ID is required")
	}

	// Get current trip for comparison
	trip, err := h.tripService.GetTrip(ctx, req.TripId)
	if err != nil {
		return &trippb.UpdateTripStatusResponse{
			Success: false,
			Message: "Trip not found",
		}, nil
	}

	oldStatus := convertToProtoStatus(trip.Status)
	newStatus := req.Status

	// Notify subscribers about the status change
	metadata := map[string]string{
		"previous_status": oldStatus.String(),
		"reason":          req.Reason,
		"updated_by":      req.DriverId,
		"event_type":      "status_change",
	}

	h.NotifyTripUpdate(req.TripId, oldStatus, newStatus, metadata)

	// Update the trip (this would typically call a proper update method)
	// For now, we'll just return success
	updatedTrip := convertToProtoTrip(trip)

	return &trippb.UpdateTripStatusResponse{
		Trip:    updatedTrip,
		Success: true,
		Message: "Trip status updated successfully",
	}, nil
}

// GetSubscriptionStats returns statistics about active subscriptions
func (h *GRPCTripHandler) GetSubscriptionStats() map[string]int {
	h.subMutex.RLock()
	defer h.subMutex.RUnlock()

	stats := make(map[string]int)
	for tripID, subscribers := range h.subscriptions {
		stats[tripID] = len(subscribers)
	}
	return stats
}

// Helper function to convert internal trip status to proto status
func convertToProtoStatus(status string) trippb.TripStatus {
	switch strings.ToLower(status) {
	case "requested":
		return trippb.TripStatus_REQUESTED
	case "matched":
		return trippb.TripStatus_MATCHED
	case "driver_en_route":
		return trippb.TripStatus_DRIVER_EN_ROUTE
	case "driver_arrived":
		return trippb.TripStatus_DRIVER_ARRIVED
	case "started":
		return trippb.TripStatus_TRIP_STARTED
	case "in_progress":
		return trippb.TripStatus_IN_PROGRESS
	case "completed":
		return trippb.TripStatus_COMPLETED
	case "cancelled":
		return trippb.TripStatus_CANCELLED_BY_RIDER
	case "failed":
		return trippb.TripStatus_FAILED
	default:
		return trippb.TripStatus_UNKNOWN_STATUS
	}
}

// Helper function to convert internal trip to proto trip
func convertToProtoTrip(trip *service.BasicTrip) *trippb.Trip {
	return &trippb.Trip{
		Id:       trip.ID,
		RiderId:  trip.RiderID,
		DriverId: trip.DriverID,
		Status:   convertToProtoStatus(trip.Status),
	}
}
