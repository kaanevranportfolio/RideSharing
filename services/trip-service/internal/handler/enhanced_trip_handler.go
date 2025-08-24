package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/rideshare-platform/services/trip-service/internal/repository"
	"github.com/rideshare-platform/services/trip-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

type TripHandler struct {
	tripService service.TripService
	logger      logger.Logger
}

func NewTripHandler(tripService service.TripService, logger logger.Logger) *TripHandler {
	return &TripHandler{
		tripService: tripService,
		logger:      logger,
	}
}

// TripRequest represents the request payload for creating a trip
type TripRequest struct {
	RiderID     string          `json:"rider_id" validate:"required"`
	PickupLoc   models.Location `json:"pickup_location" validate:"required"`
	DropoffLoc  models.Location `json:"dropoff_location" validate:"required"`
	VehicleType string          `json:"vehicle_type" validate:"required"`
	Preferences TripPreferences `json:"preferences,omitempty"`
}

type TripPreferences struct {
	PetFriendly   bool   `json:"pet_friendly,omitempty"`
	AccessibleVeh bool   `json:"accessible_vehicle,omitempty"`
	QuietRide     bool   `json:"quiet_ride,omitempty"`
	Temperature   int    `json:"temperature,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

// TripResponse represents the response for trip operations
type TripResponse struct {
	Trip    *repository.Trip `json:"trip"`
	Status  string           `json:"status"`
	Message string           `json:"message,omitempty"`
}

// TripListResponse represents the response for listing trips
type TripListResponse struct {
	Trips      []*repository.Trip `json:"trips"`
	TotalCount int                `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// EventsResponse represents the response for trip events
type EventsResponse struct {
	Events []*repository.TripEvent `json:"events"`
}

// LocationUpdateRequest represents the request for updating location
type LocationUpdateRequest struct {
	UserID   string          `json:"user_id" validate:"required"`
	Location models.Location `json:"location" validate:"required"`
}

func (h *TripHandler) RegisterRoutes(router *mux.Router) {
	tripRouter := router.PathPrefix("/trips").Subrouter()

	// Trip CRUD operations
	tripRouter.HandleFunc("", h.CreateTrip).Methods("POST")
	tripRouter.HandleFunc("/{id}", h.GetTrip).Methods("GET")
	tripRouter.HandleFunc("/{id}/cancel", h.CancelTrip).Methods("POST")
	tripRouter.HandleFunc("/{id}/accept", h.AcceptTrip).Methods("POST")
	tripRouter.HandleFunc("/{id}/start", h.StartTrip).Methods("POST")
	tripRouter.HandleFunc("/{id}/complete", h.CompleteTrip).Methods("POST")
	tripRouter.HandleFunc("/{id}/location", h.UpdateLocation).Methods("POST")

	// Trip queries
	tripRouter.HandleFunc("/rider/{riderId}", h.GetTripsByRider).Methods("GET")
	tripRouter.HandleFunc("/driver/{driverId}", h.GetTripsByDriver).Methods("GET")
	tripRouter.HandleFunc("/status/{status}", h.GetTripsByStatus).Methods("GET")
	tripRouter.HandleFunc("/rider/{riderId}/active", h.GetActiveTripByRider).Methods("GET")
	tripRouter.HandleFunc("/driver/{driverId}/active", h.GetActiveTripByDriver).Methods("GET")

	// Event operations
	tripRouter.HandleFunc("/{id}/events", h.GetTripEvents).Methods("GET")
	tripRouter.HandleFunc("/events/type/{eventType}", h.GetEventsByType).Methods("GET")
	tripRouter.HandleFunc("/events/user/{userId}", h.GetEventsByUser).Methods("GET")
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Creating new trip")

	var req TripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Convert to trip preferences
	preferences := repository.TripPreferences{
		PetFriendly:       req.Preferences.PetFriendly,
		AccessibleVehicle: req.Preferences.AccessibleVeh,
		QuietRide:         req.Preferences.QuietRide,
		Temperature:       req.Preferences.Temperature,
		PaymentMethod:     req.Preferences.PaymentMethod,
	}

	trip, err := h.tripService.CreateTrip(r.Context(), req.RiderID, req.PickupLoc, req.DropoffLoc, req.VehicleType, preferences)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create trip")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create trip", err)
		return
	}

	h.respondWithSuccess(w, http.StatusCreated, TripResponse{
		Trip:   trip,
		Status: "created",
	})
}

func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	h.logger.WithFields(logger.Fields{"trip_id": tripID}).Debug("Getting trip")

	trip, err := h.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trip")
		h.respondWithError(w, http.StatusNotFound, "Trip not found", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:   trip,
		Status: "found",
	})
}

func (h *TripHandler) CancelTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	// Get user ID from request
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusBadRequest, "User ID required", nil)
		return
	}

	h.logger.WithFields(logger.Fields{
		"trip_id": tripID,
		"user_id": userID,
	}).Info("Cancelling trip")

	trip, err := h.tripService.CancelTrip(r.Context(), tripID, userID, "User requested cancellation")
	if err != nil {
		h.logger.WithError(err).Error("Failed to cancel trip")
		h.respondWithError(w, http.StatusBadRequest, "Failed to cancel trip", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:    trip,
		Status:  "cancelled",
		Message: "Trip cancelled successfully",
	})
}

func (h *TripHandler) AcceptTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	// Get driver ID from request
	driverID := r.Header.Get("X-Driver-ID")
	if driverID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Driver ID required", nil)
		return
	}

	h.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Driver accepting trip")

	trip, err := h.tripService.AcceptTrip(r.Context(), tripID, driverID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to accept trip")
		h.respondWithError(w, http.StatusBadRequest, "Failed to accept trip", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:    trip,
		Status:  "accepted",
		Message: "Trip accepted successfully",
	})
}

func (h *TripHandler) StartTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	// Get driver ID from request
	driverID := r.Header.Get("X-Driver-ID")
	if driverID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Driver ID required", nil)
		return
	}

	h.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Starting trip")

	trip, err := h.tripService.StartTrip(r.Context(), tripID, driverID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start trip")
		h.respondWithError(w, http.StatusBadRequest, "Failed to start trip", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:    trip,
		Status:  "started",
		Message: "Trip started successfully",
	})
}

func (h *TripHandler) CompleteTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	// Get driver ID from request
	driverID := r.Header.Get("X-Driver-ID")
	if driverID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Driver ID required", nil)
		return
	}

	h.logger.WithFields(logger.Fields{
		"trip_id":   tripID,
		"driver_id": driverID,
	}).Info("Completing trip")

	trip, err := h.tripService.CompleteTrip(r.Context(), tripID, driverID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to complete trip")
		h.respondWithError(w, http.StatusBadRequest, "Failed to complete trip", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:    trip,
		Status:  "completed",
		Message: "Trip completed successfully",
	})
}

func (h *TripHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	var req LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	h.logger.WithFields(logger.Fields{
		"trip_id": tripID,
		"user_id": req.UserID,
	}).Debug("Updating location")

	trip, err := h.tripService.UpdateLocation(r.Context(), tripID, req.UserID, req.Location)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update location")
		h.respondWithError(w, http.StatusBadRequest, "Failed to update location", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:   trip,
		Status: "location_updated",
	})
}

func (h *TripHandler) GetTripsByRider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	riderID := vars["riderId"]

	limit, offset := h.getPaginationParams(r)

	h.logger.WithFields(logger.Fields{
		"rider_id": riderID,
		"limit":    limit,
		"offset":   offset,
	}).Debug("Getting trips by rider")

	trips, err := h.tripService.GetTripsByRider(r.Context(), riderID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trips by rider")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get trips", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripListResponse{
		Trips:      trips,
		TotalCount: len(trips),
		Limit:      limit,
		Offset:     offset,
	})
}

func (h *TripHandler) GetTripsByDriver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	driverID := vars["driverId"]

	limit, offset := h.getPaginationParams(r)

	h.logger.WithFields(logger.Fields{
		"driver_id": driverID,
		"limit":     limit,
		"offset":    offset,
	}).Debug("Getting trips by driver")

	trips, err := h.tripService.GetTripsByDriver(r.Context(), driverID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trips by driver")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get trips", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripListResponse{
		Trips:      trips,
		TotalCount: len(trips),
		Limit:      limit,
		Offset:     offset,
	})
}

func (h *TripHandler) GetTripsByStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	status := vars["status"]

	limit, offset := h.getPaginationParams(r)

	h.logger.WithFields(logger.Fields{
		"status": status,
		"limit":  limit,
		"offset": offset,
	}).Debug("Getting trips by status")

	trips, err := h.tripService.GetTripsByStatus(r.Context(), status, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trips by status")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get trips", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripListResponse{
		Trips:      trips,
		TotalCount: len(trips),
		Limit:      limit,
		Offset:     offset,
	})
}

func (h *TripHandler) GetActiveTripByRider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	riderID := vars["riderId"]

	h.logger.WithFields(logger.Fields{
		"rider_id": riderID,
	}).Debug("Getting active trip by rider")

	trip, err := h.tripService.GetActiveTripByRider(r.Context(), riderID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active trip by rider")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get active trip", err)
		return
	}

	if trip == nil {
		h.respondWithSuccess(w, http.StatusOK, TripResponse{
			Status:  "no_active_trip",
			Message: "No active trip found",
		})
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:   trip,
		Status: "found",
	})
}

func (h *TripHandler) GetActiveTripByDriver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	driverID := vars["driverId"]

	h.logger.WithFields(logger.Fields{
		"driver_id": driverID,
	}).Debug("Getting active trip by driver")

	trip, err := h.tripService.GetActiveTripByDriver(r.Context(), driverID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active trip by driver")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get active trip", err)
		return
	}

	if trip == nil {
		h.respondWithSuccess(w, http.StatusOK, TripResponse{
			Status:  "no_active_trip",
			Message: "No active trip found",
		})
		return
	}

	h.respondWithSuccess(w, http.StatusOK, TripResponse{
		Trip:   trip,
		Status: "found",
	})
}

func (h *TripHandler) GetTripEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripID := vars["id"]

	h.logger.WithFields(logger.Fields{
		"trip_id": tripID,
	}).Debug("Getting trip events")

	events, err := h.tripService.GetTripEvents(r.Context(), tripID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trip events")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get events", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, EventsResponse{
		Events: events,
	})
}

func (h *TripHandler) GetEventsByType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventType := vars["eventType"]

	limit, offset := h.getPaginationParams(r)

	h.logger.WithFields(logger.Fields{
		"event_type": eventType,
		"limit":      limit,
		"offset":     offset,
	}).Debug("Getting events by type")

	events, err := h.tripService.GetEventsByType(r.Context(), eventType, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get events by type")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get events", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, EventsResponse{
		Events: events,
	})
}

func (h *TripHandler) GetEventsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	limit, offset := h.getPaginationParams(r)

	h.logger.WithFields(logger.Fields{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	}).Debug("Getting events by user")

	events, err := h.tripService.GetEventsByUser(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get events by user")
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get events", err)
		return
	}

	h.respondWithSuccess(w, http.StatusOK, EventsResponse{
		Events: events,
	})
}

// Helper methods

func (h *TripHandler) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 20 // default
	offset = 0 // default

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

func (h *TripHandler) respondWithError(w http.ResponseWriter, code int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := map[string]interface{}{
		"error":     message,
		"status":    "error",
		"code":      code,
		"timestamp": time.Now().Unix(),
	}

	if err != nil {
		h.logger.WithError(err).Error(message)
		response["details"] = err.Error()
	}

	json.NewEncoder(w).Encode(response)
}

func (h *TripHandler) respondWithSuccess(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := map[string]interface{}{
		"data":      data,
		"status":    "success",
		"code":      code,
		"timestamp": time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}
