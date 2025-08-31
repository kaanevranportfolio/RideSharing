package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/rideshare-platform/shared/logger"
)

// MetricsCollector collects and exposes metrics for the rideshare platform
type MetricsCollector struct {
	redis  *redis.Client
	logger *logger.Logger

	// Prometheus metrics
	tripMetrics     *TripMetrics
	driverMetrics   *DriverMetrics
	matchingMetrics *MatchingMetrics
	paymentMetrics  *PaymentMetrics
	systemMetrics   *SystemMetrics
}

// TripMetrics contains trip-related Prometheus metrics
type TripMetrics struct {
	TripsTotal         prometheus.Counter
	TripsActive        prometheus.Gauge
	TripDuration       prometheus.Histogram
	TripsByStatus      *prometheus.CounterVec
	TripsByVehicleType *prometheus.CounterVec
	TripRevenue        prometheus.Counter
	TripCancellations  prometheus.Counter
}

// DriverMetrics contains driver-related Prometheus metrics
type DriverMetrics struct {
	DriversOnline     prometheus.Gauge
	DriversAvailable  prometheus.Gauge
	DriversBusy       prometheus.Gauge
	DriverUtilization prometheus.Histogram
	DriverRatings     prometheus.Histogram
	DriverEarnings    prometheus.Counter
}

// MatchingMetrics contains matching-related Prometheus metrics
type MatchingMetrics struct {
	MatchRequests   prometheus.Counter
	MatchSuccessful prometheus.Counter
	MatchFailed     prometheus.Counter
	MatchDuration   prometheus.Histogram
	MatchDistance   prometheus.Histogram
	MatchingQueue   prometheus.Gauge
}

// PaymentMetrics contains payment-related Prometheus metrics
type PaymentMetrics struct {
	PaymentsTotal    prometheus.Counter
	PaymentsByMethod *prometheus.CounterVec
	PaymentsByStatus *prometheus.CounterVec
	PaymentAmount    prometheus.Counter
	PaymentFailures  prometheus.Counter
	RefundsTotal     prometheus.Counter
	FraudDetections  prometheus.Counter
}

// SystemMetrics contains system-level Prometheus metrics
type SystemMetrics struct {
	APIRequests          *prometheus.CounterVec
	APILatency           *prometheus.HistogramVec
	DatabaseQueries      *prometheus.CounterVec
	DatabaseLatency      *prometheus.HistogramVec
	RedisOperations      *prometheus.CounterVec
	ErrorsTotal          *prometheus.CounterVec
	WebSocketConnections prometheus.Gauge
}

// BusinessMetrics represents business KPIs
type BusinessMetrics struct {
	TotalTrips           int64     `json:"total_trips"`
	ActiveTrips          int64     `json:"active_trips"`
	CompletedTrips       int64     `json:"completed_trips"`
	CancelledTrips       int64     `json:"cancelled_trips"`
	TotalRevenue         float64   `json:"total_revenue"`
	AverageRating        float64   `json:"average_rating"`
	AverageTripDuration  float64   `json:"average_trip_duration"`
	DriverUtilization    float64   `json:"driver_utilization"`
	CustomerSatisfaction float64   `json:"customer_satisfaction"`
	Timestamp            time.Time `json:"timestamp"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status       string                   `json:"status"` // healthy, degraded, unhealthy
	Services     map[string]ServiceHealth `json:"services"`
	OverallScore float64                  `json:"overall_score"`
	LastChecked  time.Time                `json:"last_checked"`
	Alerts       []Alert                  `json:"alerts"`
}

// ServiceHealth represents individual service health
type ServiceHealth struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	ResponseTime float64   `json:"response_time_ms"`
	ErrorRate    float64   `json:"error_rate"`
	Availability float64   `json:"availability"`
	LastChecked  time.Time `json:"last_checked"`
	Dependencies []string  `json:"dependencies"`
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Severity    string                 `json:"severity"` // critical, warning, info
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Service     string                 `json:"service"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(redis *redis.Client, logger *logger.Logger) *MetricsCollector {
	collector := &MetricsCollector{
		redis:  redis,
		logger: logger,
	}

	collector.initializeMetrics()
	return collector
}

// initializeMetrics initializes all Prometheus metrics
func (mc *MetricsCollector) initializeMetrics() {
	// Trip metrics
	mc.tripMetrics = &TripMetrics{
		TripsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_trips_total",
			Help: "Total number of trips requested",
		}),
		TripsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_trips_active",
			Help: "Number of currently active trips",
		}),
		TripDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rideshare_trip_duration_seconds",
			Help:    "Trip duration in seconds",
			Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1 min to ~17 hours
		}),
		TripsByStatus: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_trips_by_status_total",
			Help: "Total trips by status",
		}, []string{"status"}),
		TripsByVehicleType: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_trips_by_vehicle_type_total",
			Help: "Total trips by vehicle type",
		}, []string{"vehicle_type"}),
		TripRevenue: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_trip_revenue_cents_total",
			Help: "Total trip revenue in cents",
		}),
		TripCancellations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_trip_cancellations_total",
			Help: "Total trip cancellations",
		}),
	}

	// Driver metrics
	mc.driverMetrics = &DriverMetrics{
		DriversOnline: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_drivers_online",
			Help: "Number of drivers currently online",
		}),
		DriversAvailable: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_drivers_available",
			Help: "Number of drivers currently available",
		}),
		DriversBusy: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_drivers_busy",
			Help: "Number of drivers currently busy",
		}),
		DriverUtilization: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rideshare_driver_utilization_ratio",
			Help:    "Driver utilization ratio (busy time / online time)",
			Buckets: prometheus.LinearBuckets(0, 0.1, 11), // 0% to 100%
		}),
		DriverRatings: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rideshare_driver_ratings",
			Help:    "Driver ratings distribution",
			Buckets: prometheus.LinearBuckets(1, 0.5, 9), // 1.0 to 5.0
		}),
		DriverEarnings: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_driver_earnings_cents_total",
			Help: "Total driver earnings in cents",
		}),
	}

	// Matching metrics
	mc.matchingMetrics = &MatchingMetrics{
		MatchRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_match_requests_total",
			Help: "Total matching requests",
		}),
		MatchSuccessful: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_match_successful_total",
			Help: "Total successful matches",
		}),
		MatchFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_match_failed_total",
			Help: "Total failed matches",
		}),
		MatchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rideshare_match_duration_seconds",
			Help:    "Time to find a match in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 8), // 1s to 4+ minutes
		}),
		MatchDistance: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rideshare_match_distance_km",
			Help:    "Distance between rider and matched driver in km",
			Buckets: prometheus.LinearBuckets(0, 1, 21), // 0 to 20+ km
		}),
		MatchingQueue: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_matching_queue_size",
			Help: "Number of riders waiting for matches",
		}),
	}

	// Payment metrics
	mc.paymentMetrics = &PaymentMetrics{
		PaymentsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_payments_total",
			Help: "Total payment attempts",
		}),
		PaymentsByMethod: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_payments_by_method_total",
			Help: "Total payments by method",
		}, []string{"method"}),
		PaymentsByStatus: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_payments_by_status_total",
			Help: "Total payments by status",
		}, []string{"status"}),
		PaymentAmount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_payment_amount_cents_total",
			Help: "Total payment amount in cents",
		}),
		PaymentFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_payment_failures_total",
			Help: "Total payment failures",
		}),
		RefundsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_refunds_total",
			Help: "Total refunds processed",
		}),
		FraudDetections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rideshare_fraud_detections_total",
			Help: "Total fraud detections",
		}),
	}

	// System metrics
	mc.systemMetrics = &SystemMetrics{
		APIRequests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_api_requests_total",
			Help: "Total API requests",
		}, []string{"service", "method", "endpoint", "status"}),
		APILatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rideshare_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"service", "method", "endpoint"}),
		DatabaseQueries: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_database_queries_total",
			Help: "Total database queries",
		}, []string{"service", "operation", "table"}),
		DatabaseLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rideshare_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"service", "operation", "table"}),
		RedisOperations: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_redis_operations_total",
			Help: "Total Redis operations",
		}, []string{"service", "operation"}),
		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rideshare_errors_total",
			Help: "Total errors by service and type",
		}, []string{"service", "error_type"}),
		WebSocketConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rideshare_websocket_connections",
			Help: "Number of active WebSocket connections",
		}),
	}
}

// RecordTripCreated records a trip creation event
func (mc *MetricsCollector) RecordTripCreated(vehicleType string) {
	mc.tripMetrics.TripsTotal.Inc()
	mc.tripMetrics.TripsByVehicleType.WithLabelValues(vehicleType).Inc()
	mc.tripMetrics.TripsByStatus.WithLabelValues("requested").Inc()
}

// RecordTripStatusChange records a trip status change
func (mc *MetricsCollector) RecordTripStatusChange(oldStatus, newStatus string) {
	mc.tripMetrics.TripsByStatus.WithLabelValues(newStatus).Inc()

	if newStatus == "completed" || newStatus == "cancelled" {
		mc.tripMetrics.TripsActive.Dec()
	}
	if oldStatus == "requested" && newStatus == "matched" {
		mc.tripMetrics.TripsActive.Inc()
	}
}

// RecordTripCompleted records a completed trip
func (mc *MetricsCollector) RecordTripCompleted(durationSeconds float64, revenueCents int64) {
	mc.tripMetrics.TripDuration.Observe(durationSeconds)
	mc.tripMetrics.TripRevenue.Add(float64(revenueCents))
	mc.tripMetrics.TripsByStatus.WithLabelValues("completed").Inc()
}

// RecordTripCancelled records a cancelled trip
func (mc *MetricsCollector) RecordTripCancelled(reason string) {
	mc.tripMetrics.TripCancellations.Inc()
	mc.tripMetrics.TripsByStatus.WithLabelValues("cancelled").Inc()
}

// RecordMatchRequest records a matching request
func (mc *MetricsCollector) RecordMatchRequest() {
	mc.matchingMetrics.MatchRequests.Inc()
}

// RecordMatchResult records the result of a matching attempt
func (mc *MetricsCollector) RecordMatchResult(success bool, durationSeconds float64, distanceKm float64) {
	mc.matchingMetrics.MatchDuration.Observe(durationSeconds)

	if success {
		mc.matchingMetrics.MatchSuccessful.Inc()
		mc.matchingMetrics.MatchDistance.Observe(distanceKm)
	} else {
		mc.matchingMetrics.MatchFailed.Inc()
	}
}

// RecordPayment records a payment attempt
func (mc *MetricsCollector) RecordPayment(method, status string, amountCents int64) {
	mc.paymentMetrics.PaymentsTotal.Inc()
	mc.paymentMetrics.PaymentsByMethod.WithLabelValues(method).Inc()
	mc.paymentMetrics.PaymentsByStatus.WithLabelValues(status).Inc()

	if status == "completed" {
		mc.paymentMetrics.PaymentAmount.Add(float64(amountCents))
	} else if status == "failed" {
		mc.paymentMetrics.PaymentFailures.Inc()
	}
}

// RecordAPIRequest records an API request
func (mc *MetricsCollector) RecordAPIRequest(service, method, endpoint, status string, duration float64) {
	mc.systemMetrics.APIRequests.WithLabelValues(service, method, endpoint, status).Inc()
	mc.systemMetrics.APILatency.WithLabelValues(service, method, endpoint).Observe(duration)
}

// RecordDatabaseQuery records a database query
func (mc *MetricsCollector) RecordDatabaseQuery(service, operation, table string, duration float64) {
	mc.systemMetrics.DatabaseQueries.WithLabelValues(service, operation, table).Inc()
	mc.systemMetrics.DatabaseLatency.WithLabelValues(service, operation, table).Observe(duration)
}

// UpdateDriverCounts updates driver status counts
func (mc *MetricsCollector) UpdateDriverCounts(online, available, busy int) {
	mc.driverMetrics.DriversOnline.Set(float64(online))
	mc.driverMetrics.DriversAvailable.Set(float64(available))
	mc.driverMetrics.DriversBusy.Set(float64(busy))
}

// RecordDriverUtilization records driver utilization
func (mc *MetricsCollector) RecordDriverUtilization(utilizationRatio float64) {
	mc.driverMetrics.DriverUtilization.Observe(utilizationRatio)
}

// GetBusinessMetrics collects and returns current business metrics
func (mc *MetricsCollector) GetBusinessMetrics(ctx context.Context) (*BusinessMetrics, error) {
	// In a real implementation, this would query the database
	// For now, return computed metrics

	metrics := &BusinessMetrics{
		Timestamp: time.Now(),
	}

	// If Redis is available, try to get cached metrics
	if mc.redis != nil {
		data, err := mc.redis.Get(ctx, "business_metrics:current").Result()
		if err == nil {
			if err := json.Unmarshal([]byte(data), metrics); err == nil {
				return metrics, nil
			}
		}
	}

	// Fallback to mock data
	metrics.TotalTrips = 15420
	metrics.ActiveTrips = 234
	metrics.CompletedTrips = 14890
	metrics.CancelledTrips = 530
	metrics.TotalRevenue = 289125.50
	metrics.AverageRating = 4.72
	metrics.AverageTripDuration = 18.5  // minutes
	metrics.DriverUtilization = 0.68    // 68%
	metrics.CustomerSatisfaction = 0.89 // 89%

	// Cache the metrics if Redis is available
	if mc.redis != nil {
		data, _ := json.Marshal(metrics)
		mc.redis.SetEx(ctx, "business_metrics:current", data, 5*time.Minute)
	}

	return metrics, nil
}

// GetSystemHealth returns the current system health status
func (mc *MetricsCollector) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {
	health := &SystemHealth{
		Services:    make(map[string]ServiceHealth),
		LastChecked: time.Now(),
		Alerts:      []Alert{},
	}

	// Check individual services
	services := []string{
		"api-gateway", "user-service", "vehicle-service",
		"trip-service", "matching-service", "pricing-service",
		"payment-service", "geo-service",
	}

	totalScore := 0.0
	healthyServices := 0

	for _, service := range services {
		serviceHealth := mc.checkServiceHealth(ctx, service)
		health.Services[service] = serviceHealth

		// Calculate score based on availability and error rate
		score := serviceHealth.Availability * (1 - serviceHealth.ErrorRate)
		totalScore += score

		if serviceHealth.Status == "healthy" {
			healthyServices++
		}
	}

	// Calculate overall score
	health.OverallScore = totalScore / float64(len(services))

	// Determine overall status
	switch {
	case health.OverallScore >= 0.95:
		health.Status = "healthy"
	case health.OverallScore >= 0.80:
		health.Status = "degraded"
	default:
		health.Status = "unhealthy"
	}

	// Generate alerts for unhealthy services
	for serviceName, service := range health.Services {
		if service.Status != "healthy" {
			alert := Alert{
				ID:       fmt.Sprintf("service_%s_%d", serviceName, time.Now().Unix()),
				Severity: "warning",
				Title:    fmt.Sprintf("Service %s is %s", serviceName, service.Status),
				Description: fmt.Sprintf("Service availability: %.1f%%, Error rate: %.1f%%",
					service.Availability*100, service.ErrorRate*100),
				Service:   serviceName,
				CreatedAt: time.Now(),
			}
			if service.Status == "unhealthy" {
				alert.Severity = "critical"
			}
			health.Alerts = append(health.Alerts, alert)
		}
	}

	return health, nil
}

// checkServiceHealth checks the health of an individual service
func (mc *MetricsCollector) checkServiceHealth(ctx context.Context, serviceName string) ServiceHealth {
	// In a real implementation, this would make health check calls
	// For now, return mock data with some variation

	baseAvailability := 0.99
	baseErrorRate := 0.01
	baseResponseTime := 50.0

	// Add some realistic variation
	availability := baseAvailability - (float64(time.Now().Unix()%10) * 0.001)
	errorRate := baseErrorRate + (float64(time.Now().Unix()%5) * 0.002)
	responseTime := baseResponseTime + (float64(time.Now().Unix()%20) * 2.0)

	status := "healthy"
	if availability < 0.95 || errorRate > 0.05 {
		status = "degraded"
	}
	if availability < 0.90 || errorRate > 0.10 {
		status = "unhealthy"
	}

	return ServiceHealth{
		Name:         serviceName,
		Status:       status,
		ResponseTime: responseTime,
		ErrorRate:    errorRate,
		Availability: availability,
		LastChecked:  time.Now(),
		Dependencies: []string{"postgresql", "redis", "mongodb"},
	}
}

// StartMetricsCollection starts periodic collection of metrics
func (mc *MetricsCollector) StartMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mc.logger.Info("Stopping metrics collection")
			return
		case <-ticker.C:
			mc.collectPlatformMetrics(ctx)
		}
	}
}

// collectPlatformMetrics collects platform-wide metrics
func (mc *MetricsCollector) collectPlatformMetrics(ctx context.Context) {
	// Update queue sizes
	if mc.redis != nil {
		queueSize, err := mc.redis.LLen(ctx, "matching_queue").Result()
		if err == nil {
			mc.matchingMetrics.MatchingQueue.Set(float64(queueSize))
		}

		// Count WebSocket connections
		connections, err := mc.redis.SCard(ctx, "websocket_connections").Result()
		if err == nil {
			mc.systemMetrics.WebSocketConnections.Set(float64(connections))
		}
	}

	mc.logger.Debug("Platform metrics collected")
}
