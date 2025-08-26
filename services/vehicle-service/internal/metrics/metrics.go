package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicle_service_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vehicle_service_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Business metrics
	vehiclesRegisteredTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vehicle_service_vehicles_registered_total",
			Help: "Total number of vehicles registered",
		},
	)

	vehiclesActiveTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vehicle_service_vehicles_active_total",
			Help: "Total number of active vehicles",
		},
	)

	databaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vehicle_service_db_connections_active",
			Help: "Number of active database connections",
		},
	)
)

// PrometheusMiddleware creates a Gin middleware for Prometheus metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := string(rune(c.Writer.Status()))

		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			status,
		).Inc()

		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)
	}
}

// RecordVehicleRegistered increments the vehicles registered counter
func RecordVehicleRegistered() {
	vehiclesRegisteredTotal.Inc()
}

// SetActiveVehicles sets the current number of active vehicles
func SetActiveVehicles(count float64) {
	vehiclesActiveTotal.Set(count)
}

// SetDatabaseConnections sets the current number of database connections
func SetDatabaseConnections(count float64) {
	databaseConnectionsActive.Set(count)
}
