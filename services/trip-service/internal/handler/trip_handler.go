package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/trip-service/internal/service"
)

// TripHandler handles HTTP requests for the trip service
type TripHandler struct {
	service *service.TripService
}

// NewTripHandler creates a new trip handler
func NewTripHandler(service *service.TripService) *TripHandler {
	return &TripHandler{
		service: service,
	}
}

// RegisterRoutes registers all routes for the trip service
func (h *TripHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", h.healthCheck)

		// Trip management
		api.POST("/trips", h.createTrip)
		api.GET("/trips/:trip_id", h.getTrip)
		api.PUT("/trips/:trip_id/status", h.updateTripStatus)
		api.PUT("/trips/:trip_id/assign", h.assignDriver)
		api.DELETE("/trips/:trip_id", h.cancelTrip)

		// User trips
		api.GET("/users/:user_id/trips", h.getUserTrips)
		api.GET("/users/:user_id/trips/active", h.getActiveTrips)

		// Metrics
		api.GET("/metrics", h.getMetrics)
	}
}

// healthCheck returns the health status of the service
func (h *TripHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "trip-service",
		"version": "1.0.0",
	})
}

// createTrip handles trip creation requests
func (h *TripHandler) createTrip(c *gin.Context) {
	var request service.CreateTripRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	trip, err := h.service.CreateTrip(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create trip",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, trip)
}

// getTrip retrieves a trip by ID
func (h *TripHandler) getTrip(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	trip, err := h.service.GetTrip(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Trip not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, trip)
}

// updateTripStatus updates the status of a trip
func (h *TripHandler) updateTripStatus(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	var request struct {
		Status service.TripStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := h.service.UpdateTripStatus(c.Request.Context(), tripID, request.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update trip status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Trip status updated successfully",
		"trip_id": tripID,
		"status":  request.Status,
	})
}

// assignDriver assigns a driver to a trip
func (h *TripHandler) assignDriver(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	var request struct {
		DriverID  string `json:"driver_id" binding:"required"`
		VehicleID string `json:"vehicle_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := h.service.AssignDriver(c.Request.Context(), tripID, request.DriverID, request.VehicleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to assign driver",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Driver assigned successfully",
		"trip_id":    tripID,
		"driver_id":  request.DriverID,
		"vehicle_id": request.VehicleID,
	})
}

// cancelTrip cancels a trip
func (h *TripHandler) cancelTrip(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	var request struct {
		CancelledBy string `json:"cancelled_by" binding:"required"`
		Reason      string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := h.service.CancelTrip(c.Request.Context(), tripID, request.CancelledBy, request.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cancel trip",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Trip cancelled successfully",
		"trip_id":      tripID,
		"cancelled_by": request.CancelledBy,
	})
}

// getUserTrips returns trip history for a user
func (h *TripHandler) getUserTrips(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing user_id parameter",
		})
		return
	}

	// Parse pagination parameters
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	trips, err := h.service.GetTripHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trip history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trips":  trips,
		"limit":  limit,
		"offset": offset,
		"count":  len(trips),
	})
}

// getActiveTrips returns active trips for a user
func (h *TripHandler) getActiveTrips(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing user_id parameter",
		})
		return
	}

	trips, err := h.service.GetActiveTrips(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get active trips",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trips": trips,
		"count": len(trips),
	})
}

// getMetrics returns trip service metrics
func (h *TripHandler) getMetrics(c *gin.Context) {
	metrics, err := h.service.GetTripMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}
