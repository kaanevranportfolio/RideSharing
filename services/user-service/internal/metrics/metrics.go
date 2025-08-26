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
			Name: "user_service_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_service_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Business metrics
	usersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_service_users_created_total",
			Help: "Total number of users created",
		},
	)

	usersActiveTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "user_service_users_active_total",
			Help: "Total number of active users",
		},
	)

	databaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "user_service_db_connections_active",
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

// RecordUserCreated increments the users created counter
func RecordUserCreated() {
	usersCreatedTotal.Inc()
}

// SetActiveUsers sets the current number of active users
func SetActiveUsers(count float64) {
	usersActiveTotal.Set(count)
}

// SetDatabaseConnections sets the current number of database connections
func SetDatabaseConnections(count float64) {
	databaseConnectionsActive.Set(count)
}
