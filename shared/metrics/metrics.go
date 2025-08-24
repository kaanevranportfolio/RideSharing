package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Business metrics
	TripsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trips_created_total",
			Help: "Total number of trips created",
		},
		[]string{"status"},
	)

	TripsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trips_completed_total",
			Help: "Total number of trips completed",
		},
		[]string{"payment_method"},
	)

	UsersRegistered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "users_registered_total",
			Help: "Total number of users registered",
		},
	)

	DriversOnline = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "drivers_online",
			Help: "Number of drivers currently online",
		},
	)

	MatchingAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "matching_attempts_total",
			Help: "Total number of driver matching attempts",
		},
		[]string{"result"},
	)

	MatchingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "matching_duration_seconds",
			Help:    "Duration of driver matching process",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
	)

	PaymentProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "payment_processing_duration_seconds",
			Help:    "Duration of payment processing",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0},
		},
	)

	DatabaseConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
		[]string{"database"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordTripCreated records trip creation
func RecordTripCreated(status string) {
	TripsCreated.WithLabelValues(status).Inc()
}

// RecordTripCompleted records trip completion
func RecordTripCompleted(paymentMethod string) {
	TripsCompleted.WithLabelValues(paymentMethod).Inc()
}

// RecordUserRegistration records user registration
func RecordUserRegistration() {
	UsersRegistered.Inc()
}

// SetDriversOnline sets the number of drivers online
func SetDriversOnline(count float64) {
	DriversOnline.Set(count)
}

// RecordMatchingAttempt records driver matching attempt
func RecordMatchingAttempt(result string, duration time.Duration) {
	MatchingAttempts.WithLabelValues(result).Inc()
	MatchingDuration.Observe(duration.Seconds())
}

// RecordPaymentProcessing records payment processing duration
func RecordPaymentProcessing(duration time.Duration) {
	PaymentProcessingDuration.Observe(duration.Seconds())
}

// SetDatabaseConnections sets active database connections
func SetDatabaseConnections(database string, count float64) {
	DatabaseConnectionsActive.WithLabelValues(database).Set(count)
}

// RecordCacheHit records cache hit
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records cache miss
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}
