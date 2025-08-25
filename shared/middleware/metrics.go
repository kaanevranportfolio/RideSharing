package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/shared/logger"
)

// MetricsMiddleware provides basic metrics collection
type MetricsMiddleware struct {
	logger *logger.Logger
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(logger *logger.Logger) *MetricsMiddleware {
	return &MetricsMiddleware{
		logger: logger,
	}
}

// Handler returns a gin middleware function for basic metrics
func (m *MetricsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log basic metrics
		duration := time.Since(start)
		m.logger.Info("Request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", duration,
		)
	}
}

// GetMetricsHandler returns a simple metrics endpoint
func (m *MetricsMiddleware) GetMetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "basic metrics enabled",
			"timestamp": time.Now(),
		})
	}
}
