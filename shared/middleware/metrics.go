package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rideshare-platform/shared/logger"
)

// MetricsMiddleware provides Prometheus metrics collection
type MetricsMiddleware struct {
	logger           *logger.Logger
	requestDuration  *prometheus.HistogramVec
	requestsTotal    *prometheus.CounterVec
	requestsInFlight prometheus.Gauge
	requestSize      *prometheus.HistogramVec
	responseSize     *prometheus.HistogramVec
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(serviceName string, log *logger.Logger) *MetricsMiddleware {
	requestDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "endpoint", "status_code"},
	)

	requestsTotal := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "endpoint", "status_code"},
	)

	requestsInFlight := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	requestSize := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"service", "method", "endpoint"},
	)

	responseSize := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"service", "method", "endpoint", "status_code"},
	)

	return &MetricsMiddleware{
		logger:           log,
		requestDuration:  requestDuration,
		requestsTotal:    requestsTotal,
		requestsInFlight: requestsInFlight,
		requestSize:      requestSize,
		responseSize:     responseSize,
	}
}

// PrometheusMetrics collects Prometheus metrics for HTTP requests
func (m *MetricsMiddleware) PrometheusMetrics(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics collection for metrics endpoint
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Increment in-flight requests
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		// Record request size
		if c.Request.ContentLength > 0 {
			m.requestSize.WithLabelValues(
				serviceName,
				c.Request.Method,
				c.FullPath(),
			).Observe(float64(c.Request.ContentLength))
		}

		// Create response writer wrapper to capture response size
		writer := &metricsResponseWriter{
			ResponseWriter: c.Writer,
			size:          0,
		}
		c.Writer = writer

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get status code
		statusCode := strconv.Itoa(c.Writer.Status())

		// Record metrics
		m.requestDuration.WithLabelValues(
			serviceName,
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Observe(duration.Seconds())

		m.requestsTotal.WithLabelValues(
			serviceName,
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Inc()

		m.responseSize.WithLabelValues(
			serviceName,
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Observe(float64(writer.size))

		// Log metrics
		m.logger.LogMetric(c.Request.Context(), "http_request_duration", duration.Seconds(), map[string]string{
			"service":     serviceName,
			"method":      c.Request.Method,
			"endpoint":    c.FullPath(),
			"status_code": statusCode,
		})
	}
}

// metricsResponseWriter wraps gin.ResponseWriter to capture response size
type metricsResponseWriter struct {
	gin.ResponseWriter
	size int
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// BusinessMetrics provides business-specific metrics
type BusinessMetrics struct {
	logger *logger.Logger
	
	// Trip metrics
	tripsCreated    *prometheus.CounterVec
	tripsCompleted  *prometheus.CounterVec
	tripsCancelled  *prometheus.CounterVec
	tripDuration    *prometheus.HistogramVec
	tripDistance    *prometheus.HistogramVec
	tripFare        *prometheus.HistogramVec
	
	// User metrics
	usersRegistered *prometheus.CounterVec
	usersActive     *prometheus.GaugeVec
	
	// Driver metrics
	driversOnline   prometheus.Gauge
	driversActive   prometheus.Gauge
	
	// Matching metrics
	matchingTime    *prometheus.HistogramVec
	matchingSuccess *prometheus.CounterVec
	matchingFailed  *prometheus.CounterVec
	
	// Payment metrics
	paymentsProcessed *prometheus.CounterVec
	paymentsFailed    *prometheus.CounterVec
	paymentAmount     *prometheus.HistogramVec
}

// NewBusinessMetrics creates business metrics collectors
func NewBusinessMetrics(log *logger.Logger) *BusinessMetrics {
	return &BusinessMetrics{
		logger: log,
		
		tripsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trips_created_total",
				Help: "Total number of trips created",
			},
			[]string{"service", "user_type"},
		),
		
		tripsCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trips_completed_total",
				Help: "Total number of trips completed",
			},
			[]string{"service"},
		),
		
		tripsCancelled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trips_cancelled_total",
				Help: "Total number of trips cancelled",
			},
			[]string{"service", "cancelled_by"},
		),
		
		tripDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "trip_duration_seconds",
				Help:    "Duration of completed trips in seconds",
				Buckets: []float64{60, 300, 600, 1200, 1800, 3600, 7200},
			},
			[]string{"service"},
		),
		
		tripDistance: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "trip_distance_km",
				Help:    "Distance of completed trips in kilometers",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"service"},
		),
		
		tripFare: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "trip_fare_amount",
				Help:    "Fare amount of completed trips",
				Buckets: []float64{5, 10, 20, 50, 100, 200, 500},
			},
			[]string{"service", "currency"},
		),
		
		usersRegistered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "users_registered_total",
				Help: "Total number of users registered",
			},
			[]string{"service", "user_type"},
		),
		
		usersActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "users_active",
				Help: "Number of currently active users",
			},
			[]string{"service", "user_type"},
		),
		
		driversOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "drivers_online",
				Help: "Number of drivers currently online",
			},
		),
		
		driversActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "drivers_active",
				Help: "Number of drivers currently on a trip",
			},
		),
		
		matchingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "matching_time_seconds",
				Help:    "Time taken to match rider with driver",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
			},
			[]string{"service"},
		),
		
		matchingSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "matching_success_total",
				Help: "Total number of successful matches",
			},
			[]string{"service"},
		),
		
		matchingFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "matching_failed_total",
				Help: "Total number of failed matches",
			},
			[]string{"service", "reason"},
		),
		
		paymentsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payments_processed_total",
				Help: "Total number of payments processed",
			},
			[]string{"service", "payment_method"},
		),
		
		paymentsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payments_failed_total",
				Help: "Total number of failed payments",
			},
			[]string{"service", "payment_method", "error_type"},
		),
		
		paymentAmount: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_amount",
				Help:    "Amount of processed payments",
				Buckets: []float64{5, 10, 20, 50, 100, 200, 500},
			},
			[]string{"service", "currency", "payment_method"},
		),
	}
}

// RecordTripCreated records a trip creation event
func (bm *BusinessMetrics) RecordTripCreated(service, userType string) {
	bm.tripsCreated.WithLabelValues(service, userType).Inc()
}

// RecordTripCompleted records a trip completion event
func (bm *BusinessMetrics) RecordTripCompleted(service string, duration time.Duration, distance, fare float64, currency string) {
	bm.tripsCompleted.WithLabelValues(service).Inc()
	bm.tripDuration.WithLabelValues(service).Observe(duration.Seconds())
	bm.tripDistance.WithLabelValues(service).Observe(distance)
	bm.tripFare.WithLabelValues(service, currency).Observe(fare)
}

// RecordTripCancelled records a trip cancellation event
func (bm *BusinessMetrics) RecordTripCancelled(service, cancelledBy string) {
	bm.tripsCancelled.WithLabelValues(service, cancelledBy).Inc()
}

// RecordUserRegistration records a user registration event
func (bm *BusinessMetrics) RecordUserRegistration(service, userType string) {
	bm.usersRegistered.WithLabelValues(service, userType).Inc()
}

// SetActiveUsers sets the number of active users
func (bm *BusinessMetrics) SetActiveUsers(service, userType string, count float64) {
	bm.usersActive.WithLabelValues(service, userType).Set(count)
}

// SetDriversOnline sets the number of online drivers
func (bm *BusinessMetrics) SetDriversOnline(count float64) {
	bm.driversOnline.Set(count)
}

// SetDriversActive sets the number of active drivers
func (bm *BusinessMetrics) SetDriversActive(count float64) {
	bm.driversActive.Set(count)
}

// RecordMatchingSuccess records a successful match
func (bm *BusinessMetrics) RecordMatchingSuccess(service string, duration time.Duration) {
	bm.matchingSuccess.WithLabelValues(service).Inc()
	bm.matchingTime.WithLabelValues(service).Observe(duration.Seconds())
}

// RecordMatchingFailed records a failed match
func (bm *BusinessMetrics) RecordMatchingFailed(service, reason string) {
	bm.matchingFailed.WithLabelValues(service, reason).Inc()
}

// RecordPaymentProcessed records a successful payment
func (bm *BusinessMetrics) RecordPaymentProcessed(service, paymentMethod, currency string, amount float64) {
	bm.paymentsProcessed.WithLabelValues(service, paymentMethod).Inc()
	bm.paymentAmount.WithLabelValues(service, currency, paymentMethod).Observe(amount)
}

// RecordPaymentFailed records a failed payment
func (bm *BusinessMetrics) RecordPaymentFailed(service, paymentMethod, errorType string) {
	bm.paymentsFailed.WithLabelValues(service, paymentMethod, errorType).Inc()
}