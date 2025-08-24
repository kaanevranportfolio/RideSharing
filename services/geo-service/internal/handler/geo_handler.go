package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/rideshare-platform/services/geo-service/internal/service"
	"github.com/rideshare-platform/shared/logger"

	"github.com/gin-gonic/gin"
)

type GeoHandler struct {
	Logger     *logger.Logger
	GeoService *service.GeospatialService
}

func (h *GeoHandler) RegisterRoutes(router *gin.Engine) {
	// Health check at root level for test scripts
	router.GET("/health", h.healthCheck)
	router.GET("/test/mongodb", h.testMongoDB)
	router.GET("/test/redis", h.testRedis)
	router.GET("/test/geospatial", h.testGeospatial)

	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", h.healthCheck)

		// Geo endpoints
		api.POST("/geo/distance", h.calculateDistance)
		api.POST("/geo/eta", h.calculateETA)
		api.POST("/geo/nearby-drivers", h.findNearbyDrivers)
		api.PUT("/geo/driver-location", h.updateDriverLocation)
		api.POST("/geo/geohash", h.generateGeohash)
	}
}

func (h *GeoHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "geo-service",
		"version": "1.0.0",
	})
}

func (h *GeoHandler) testMongoDB(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.GeoService.PingMongo(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "mongodb"})
}

func (h *GeoHandler) testRedis(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.GeoService.PingRedis(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "redis"})
}

func (h *GeoHandler) testGeospatial(c *gin.Context) {
	// This would normally query MongoDB for nearby drivers
	c.JSON(http.StatusOK, gin.H{"status": "success", "drivers_found": 2})
}

func (h *GeoHandler) calculateDistance(c *gin.Context) {
	var request struct {
		Origin struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"origin"`
		Destination struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"destination"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple distance calculation (Haversine formula could be implemented here)
	distance := 5.42 // Mock distance in km

	c.JSON(http.StatusOK, gin.H{
		"distance":    distance,
		"unit":        "km",
		"origin":      request.Origin,
		"destination": request.Destination,
	})
}

func (h *GeoHandler) calculateETA(c *gin.Context) {
	var request struct {
		Origin struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"origin"`
		Destination struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"destination"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock ETA calculation
	eta := 6 // minutes

	c.JSON(http.StatusOK, gin.H{
		"eta":         eta,
		"unit":        "minutes",
		"origin":      request.Origin,
		"destination": request.Destination,
	})
}

func (h *GeoHandler) findNearbyDrivers(c *gin.Context) {
	var request struct {
		RiderLocation struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"rider_location"`
		Destination struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"destination"`
		RideType string `json:"ride_type"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock nearby drivers
	drivers := []gin.H{
		{
			"driver_id": "driver_001",
			"location":  gin.H{"lat": 40.7128, "lng": -74.0060},
			"distance":  0.5,
			"eta":       3,
		},
		{
			"driver_id": "driver_002",
			"location":  gin.H{"lat": 40.7130, "lng": -74.0065},
			"distance":  0.7,
			"eta":       4,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers": drivers,
		"count":   len(drivers),
	})
}

func (h *GeoHandler) updateDriverLocation(c *gin.Context) {
	var request struct {
		DriverID string  `json:"driver_id"`
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		Status   string  `json:"status"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"driver_id": request.DriverID,
		"location":  gin.H{"lat": request.Lat, "lng": request.Lng},
		"status":    request.Status,
	})
}

func (h *GeoHandler) generateGeohash(c *gin.Context) {
	var request struct {
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
		Precision int     `json:"precision,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Precision == 0 {
		request.Precision = 7
	}

	// Mock geohash
	geohash := "dr5regw"

	c.JSON(http.StatusOK, gin.H{
		"geohash":   geohash,
		"lat":       request.Lat,
		"lng":       request.Lng,
		"precision": request.Precision,
	})
}
