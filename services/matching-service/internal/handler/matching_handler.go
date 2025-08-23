package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/matching-service/internal/service"
)

// MatchingHandler handles HTTP requests for the matching service
type MatchingHandler struct {
	service *service.MatchingService
}

// NewMatchingHandler creates a new matching handler
func NewMatchingHandler(service *service.MatchingService) *MatchingHandler {
	return &MatchingHandler{
		service: service,
	}
}

// RegisterRoutes registers all routes for the matching service
func (h *MatchingHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", h.healthCheck)

		// Matching endpoints
		api.POST("/match", h.findMatch)
		api.GET("/match/:trip_id/status", h.getMatchingStatus)
		api.DELETE("/match/:trip_id", h.cancelMatching)

		// Driver finding endpoints
		matching := api.Group("/matching")
		{
			matching.POST("/find-drivers", h.findDrivers)
		}

		// Metrics
		api.GET("/metrics", h.getMetrics)
	}
}

// healthCheck returns the health status of the service
func (h *MatchingHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "matching-service",
		"version": "1.0.0",
	})
}

// findMatch handles trip matching requests
func (h *MatchingHandler) findMatch(c *gin.Context) {
	var request service.MatchingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if request.TripID == "" || request.RiderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required fields: trip_id, rider_id",
		})
		return
	}

	result, err := h.service.FindMatch(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find match",
			"details": err.Error(),
		})
		return
	}

	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusNotFound, result)
	}
}

// getMatchingStatus returns the status of a matching request
func (h *MatchingHandler) getMatchingStatus(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	status, err := h.service.GetMatchingStatus(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get matching status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// cancelMatching cancels an ongoing matching request
func (h *MatchingHandler) cancelMatching(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing trip_id parameter",
		})
		return
	}

	err := h.service.CancelMatching(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cancel matching",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Matching cancelled successfully",
		"trip_id": tripID,
	})
}

// getMetrics returns matching service metrics
func (h *MatchingHandler) getMetrics(c *gin.Context) {
	metrics, err := h.service.GetMatchingMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// FindDriversRequest represents a request to find available drivers
type FindDriversRequest struct {
	RiderLocation struct {
		Lat float64 `json:"lat" binding:"required"`
		Lng float64 `json:"lng" binding:"required"`
	} `json:"rider_location" binding:"required"`
	Destination struct {
		Lat float64 `json:"lat" binding:"required"`
		Lng float64 `json:"lng" binding:"required"`
	} `json:"destination" binding:"required"`
	RideType string `json:"ride_type" binding:"required"`
}

// findDrivers handles requests to find available drivers
func (h *MatchingHandler) findDrivers(c *gin.Context) {
	var request FindDriversRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Mock response for now - return sample drivers
	drivers := []map[string]interface{}{
		{
			"driver_id":    "driver-001",
			"vehicle_type": "sedan",
			"distance_km":  1.2,
			"eta_minutes":  5,
			"rating":       4.8,
		},
		{
			"driver_id":    "driver-002",
			"vehicle_type": "suv",
			"distance_km":  2.1,
			"eta_minutes":  8,
			"rating":       4.6,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers":       drivers,
		"total_found":   len(drivers),
		"ride_type":     request.RideType,
		"search_radius": 5.0,
	})
}
