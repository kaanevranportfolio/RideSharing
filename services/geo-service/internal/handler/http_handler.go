package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/shared/logger"
)

// HTTPHandler manages HTTP routes and handlers for the geo service
type HTTPHandler struct {
	logger *logger.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(log *logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		logger: log,
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

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", h.ping)
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
