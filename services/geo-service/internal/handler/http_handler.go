package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/geo-service/internal/service"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// HTTPHandler manages HTTP routes and handlers for the geo service
type HTTPHandler struct {
	logger     *logger.Logger
	geoService *service.GeospatialService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(log *logger.Logger, geoService *service.GeospatialService) *HTTPHandler {
	return &HTTPHandler{
		logger:     log,
		geoService: geoService,
	}
}

// SetupRoutes configures the HTTP routes
func (h *HTTPHandler) SetupRoutes() *gin.Engine {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", h.healthCheck)

	// Health check endpoints for MongoDB and Redis
	router.GET("/health/mongodb", h.mongoHealth)
	router.GET("/health/redis", h.redisHealth)

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", h.ping)

		// Geo endpoints
		geo := v1.Group("/geo")
		{
			geo.POST("/distance", h.calculateDistance)
			geo.POST("/eta", h.calculateETA)
		}
	}

	return router
}

// healthCheck returns the service health status
func (h *HTTPHandler) healthCheck(c *gin.Context) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "geo-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}

// ping returns a simple pong response
func (h *HTTPHandler) ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"service": "geo-service",
	})
}

// mongoHealth checks the health of the MongoDB dependency
func (h *HTTPHandler) mongoHealth(c *gin.Context) {
	status := "healthy"
	if err := h.geoService.PingMongo(c.Request.Context()); err != nil {
		status = "unhealthy"
	}
	c.JSON(200, gin.H{"status": status, "service": "geo-service", "dependency": "mongodb"})
}

// redisHealth checks the health of the Redis dependency
func (h *HTTPHandler) redisHealth(c *gin.Context) {
	status := "healthy"
	if err := h.geoService.PingRedis(c.Request.Context()); err != nil {
		status = "unhealthy"
	}
	c.JSON(200, gin.H{"status": status, "service": "geo-service", "dependency": "redis"})
}

// LocationRequest represents a location coordinate
type LocationRequest struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

// DistanceRequest represents a distance calculation request
type DistanceRequest struct {
	Origin      LocationRequest `json:"origin" binding:"required"`
	Destination LocationRequest `json:"destination" binding:"required"`
}

// calculateDistance handles distance calculation requests
func (h *HTTPHandler) calculateDistance(c *gin.Context) {
	var req DistanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	origin := &models.Location{
		Latitude:  req.Origin.Lat,
		Longitude: req.Origin.Lng,
		Timestamp: time.Now(),
	}

	destination := &models.Location{
		Latitude:  req.Destination.Lat,
		Longitude: req.Destination.Lng,
		Timestamp: time.Now(),
	}

	distance, err := h.geoService.CalculateDistance(c.Request.Context(), *origin, *destination, "haversine")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate distance",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"distance":    distance,
		"origin":      req.Origin,
		"destination": req.Destination,
	})
}

// calculateETA handles ETA calculation requests
func (h *HTTPHandler) calculateETA(c *gin.Context) {
	var req DistanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	origin := &models.Location{
		Latitude:  req.Origin.Lat,
		Longitude: req.Origin.Lng,
		Timestamp: time.Now(),
	}

	destination := &models.Location{
		Latitude:  req.Destination.Lat,
		Longitude: req.Destination.Lng,
		Timestamp: time.Now(),
	}

	eta, err := h.geoService.CalculateETA(c.Request.Context(), *origin, *destination, "car", time.Now(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate ETA",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"eta":         eta,
		"origin":      req.Origin,
		"destination": req.Destination,
	})
}
