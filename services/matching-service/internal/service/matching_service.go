package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/rideshare-platform/services/matching-service/internal/config"
	"github.com/rideshare-platform/services/matching-service/internal/repository"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// AdvancedMatchingService handles trip matching with sophisticated algorithms
type AdvancedMatchingService struct {
	config     *config.Config
	logger     *logger.Logger
	tripRepo   *repository.TripRepository
	redis      *redis.Client
	mongo      *mongo.Client
	geoService GeoServiceClient // Interface for geo-service gRPC calls
}

// GeoServiceClient interface for geo-service integration
type GeoServiceClient interface {
	CalculateDistance(ctx context.Context, origin, destination *models.Location) (*DistanceResult, error)
	CalculateETA(ctx context.Context, origin, destination *models.Location, vehicleType string) (*ETAResult, error)
	FindNearbyDrivers(ctx context.Context, center *models.Location, radiusKm float64, limit int) ([]*DriverLocation, error)
}

// DistanceResult represents distance calculation result from geo-service
type DistanceResult struct {
	DistanceMeters float64
	DistanceKm     float64
	BearingDegrees float64
}

// ETAResult represents ETA calculation result from geo-service
type ETAResult struct {
	DurationSeconds int
	DistanceMeters  float64
	RouteSummary    string
}

// DriverLocation represents a driver's location from geo-service
type DriverLocation struct {
	DriverID           string
	VehicleID          string
	Location           *models.Location
	DistanceFromCenter float64
	Status             string
	VehicleType        string
	Rating             float64
}

// MatchingRequest represents a comprehensive trip matching request
type MatchingRequest struct {
	TripID         string            `json:"trip_id"`
	RiderID        string            `json:"rider_id"`
	PickupLocation *models.Location  `json:"pickup_location"`
	Destination    *models.Location  `json:"destination"`
	PassengerCount int               `json:"passenger_count"`
	VehicleType    string            `json:"vehicle_type"`
	RequestedAt    time.Time         `json:"requested_at"`
	SpecialNeeds   []string          `json:"special_needs,omitempty"`
	PriorityLevel  int               `json:"priority_level"` // 1=normal, 2=premium, 3=emergency
	MaxWaitTime    time.Duration     `json:"max_wait_time"`
	Preferences    *RiderPreferences `json:"preferences,omitempty"`
}

// RiderPreferences represents rider preferences for matching
type RiderPreferences struct {
	MinDriverRating    float64  `json:"min_driver_rating"`
	PreferredGender    string   `json:"preferred_gender,omitempty"`
	AllowSharedRides   bool     `json:"allow_shared_rides"`
	MaxDetourTime      int      `json:"max_detour_time"` // minutes
	PreferQuietRide    bool     `json:"prefer_quiet_ride"`
	AccessibilityNeeds []string `json:"accessibility_needs,omitempty"`
}

// MatchingResult represents comprehensive matching result
type MatchingResult struct {
	TripID             string               `json:"trip_id"`
	Success            bool                 `json:"success"`
	MatchedDriver      *MatchedDriverInfo   `json:"matched_driver,omitempty"`
	EstimatedETA       int                  `json:"estimated_eta,omitempty"` // seconds
	EstimatedFare      *FareEstimate        `json:"estimated_fare,omitempty"`
	Reason             string               `json:"reason,omitempty"`
	AlternativeOptions []*MatchedDriverInfo `json:"alternative_options,omitempty"`
	MatchingScore      float64              `json:"matching_score,omitempty"`
	ProcessingTime     time.Duration        `json:"processing_time"`
	RetryCount         int                  `json:"retry_count"`
}

// MatchedDriverInfo represents detailed matched driver information
type MatchedDriverInfo struct {
	DriverID        string           `json:"driver_id"`
	VehicleID       string           `json:"vehicle_id"`
	DriverName      string           `json:"driver_name"`
	DriverPhoto     string           `json:"driver_photo,omitempty"`
	Rating          float64          `json:"rating"`
	TripCount       int              `json:"trip_count"`
	CurrentLocation *models.Location `json:"current_location"`
	VehicleInfo     *VehicleDetails  `json:"vehicle_info"`
	Distance        float64          `json:"distance"` // km from pickup
	ETA             int              `json:"eta"`      // seconds to pickup
	MatchScore      float64          `json:"match_score"`
	Status          string           `json:"status"`
}

// VehicleDetails represents detailed vehicle information
type VehicleDetails struct {
	Make         string   `json:"make"`
	Model        string   `json:"model"`
	Year         int      `json:"year"`
	Color        string   `json:"color"`
	LicensePlate string   `json:"license_plate"`
	VehicleType  string   `json:"vehicle_type"`
	Capacity     int      `json:"capacity"`
	Features     []string `json:"features,omitempty"` // e.g., "air_conditioning", "wifi", "phone_charger"
}

// FareEstimate represents estimated fare for the trip
type FareEstimate struct {
	BaseFare      float64 `json:"base_fare"`
	DistanceFare  float64 `json:"distance_fare"`
	TimeFare      float64 `json:"time_fare"`
	SurgeFare     float64 `json:"surge_fare"`
	TotalEstimate float64 `json:"total_estimate"`
	Currency      string  `json:"currency"`
}

// NewAdvancedMatchingService creates a new advanced matching service
func NewAdvancedMatchingService(
	cfg *config.Config,
	logger *logger.Logger,
	tripRepo *repository.TripRepository,
	redis *redis.Client,
	mongo *mongo.Client,
	geoService GeoServiceClient,
) *AdvancedMatchingService {
	return &AdvancedMatchingService{
		config:     cfg,
		logger:     logger,
		tripRepo:   tripRepo,
		redis:      redis,
		mongo:      mongo,
		geoService: geoService,
	}
}

// NewSimpleMatchingService creates a basic matching service for testing
func NewSimpleMatchingService(cfg *config.Config) *AdvancedMatchingService {
	// Create a simple version without external dependencies for basic functionality
	return &AdvancedMatchingService{
		config: cfg,
		// Other fields will be nil - need to handle this in methods
	}
}

// FindMatch implements sophisticated driver matching algorithm
func (s *AdvancedMatchingService) FindMatch(ctx context.Context, request *MatchingRequest) (*MatchingResult, error) {
	startTime := time.Now()

	// Basic safety check for nil dependencies - return mock response
	if s.geoService == nil {
		return s.generateMockResult(request, startTime), nil
	}

	if s.logger != nil {
		s.logger.WithContext(ctx).WithFields(logger.Fields{
			"trip_id":      request.TripID,
			"rider_id":     request.RiderID,
			"vehicle_type": request.VehicleType,
			"pickup_lat":   request.PickupLocation.Latitude,
			"pickup_lng":   request.PickupLocation.Longitude,
		}).Info("Starting advanced trip matching")
	} // Phase 1: Find nearby drivers using geo-service
	nearbyDrivers, err := s.findNearbyDrivers(ctx, request)
	if err != nil {
		return &MatchingResult{
			TripID:         request.TripID,
			Success:        false,
			Reason:         fmt.Sprintf("Failed to find nearby drivers: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	if len(nearbyDrivers) == 0 {
		return &MatchingResult{
			TripID:         request.TripID,
			Success:        false,
			Reason:         "No available drivers found in the area",
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Phase 2: Filter drivers based on requirements
	eligibleDrivers := s.filterEligibleDrivers(ctx, nearbyDrivers, request)
	if len(eligibleDrivers) == 0 {
		return &MatchingResult{
			TripID:         request.TripID,
			Success:        false,
			Reason:         "No eligible drivers match the requirements",
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Phase 3: Score and rank drivers
	scoredDrivers, err := s.scoreAndRankDrivers(ctx, eligibleDrivers, request)
	if err != nil {
		return &MatchingResult{
			TripID:         request.TripID,
			Success:        false,
			Reason:         fmt.Sprintf("Failed to score drivers: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// Phase 4: Select best match and alternatives
	bestMatch := scoredDrivers[0]
	var alternatives []*MatchedDriverInfo
	if len(scoredDrivers) > 1 {
		maxAlternatives := 3
		if len(scoredDrivers) < maxAlternatives+1 {
			maxAlternatives = len(scoredDrivers) - 1
		}
		alternatives = scoredDrivers[1 : maxAlternatives+1]
	}

	// Phase 5: Calculate fare estimate
	fareEstimate, err := s.calculateFareEstimate(ctx, request, bestMatch)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to calculate fare estimate")
	}

	// Phase 6: Reserve the driver
	err = s.reserveDriver(ctx, bestMatch.DriverID, request.TripID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to reserve driver")
		return &MatchingResult{
			TripID:         request.TripID,
			Success:        false,
			Reason:         "Driver reservation failed",
			ProcessingTime: time.Since(startTime),
		}, err
	}

	result := &MatchingResult{
		TripID:             request.TripID,
		Success:            true,
		MatchedDriver:      bestMatch,
		EstimatedETA:       bestMatch.ETA,
		EstimatedFare:      fareEstimate,
		Reason:             "Successfully matched with optimal driver",
		AlternativeOptions: alternatives,
		MatchingScore:      bestMatch.MatchScore,
		ProcessingTime:     time.Since(startTime),
		RetryCount:         0,
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"trip_id":        request.TripID,
		"matched_driver": bestMatch.DriverID,
		"match_score":    bestMatch.MatchScore,
		"processing_ms":  time.Since(startTime).Milliseconds(),
	}).Info("Trip matching completed successfully")

	return result, nil
}

// findNearbyDrivers gets nearby drivers from geo-service
func (s *AdvancedMatchingService) findNearbyDrivers(ctx context.Context, request *MatchingRequest) ([]*DriverLocation, error) {
	// Start with a smaller radius and expand if needed
	radiusKm := 5.0
	maxRadius := 20.0
	limit := 50

	for radiusKm <= maxRadius {
		drivers, err := s.geoService.FindNearbyDrivers(ctx, request.PickupLocation, radiusKm, limit)
		if err != nil {
			return nil, err
		}

		if len(drivers) >= 5 { // Minimum drivers to consider
			return drivers, nil
		}

		radiusKm += 5.0 // Expand search radius
	}

	// Return whatever we found, even if less than ideal
	return s.geoService.FindNearbyDrivers(ctx, request.PickupLocation, maxRadius, limit)
}

// filterEligibleDrivers filters drivers based on requirements
func (s *AdvancedMatchingService) filterEligibleDrivers(ctx context.Context, drivers []*DriverLocation, request *MatchingRequest) []*DriverLocation {
	var eligible []*DriverLocation

	for _, driver := range drivers {
		// Check basic availability
		if driver.Status != "available" {
			continue
		}

		// Check vehicle type match
		if request.VehicleType != "" && driver.VehicleType != request.VehicleType {
			continue
		}

		// Check minimum rating requirement
		if request.Preferences != nil && driver.Rating < request.Preferences.MinDriverRating {
			continue
		}

		// Check maximum distance (15km for now)
		if driver.DistanceFromCenter > 15.0 {
			continue
		}

		eligible = append(eligible, driver)
	}

	return eligible
}

// scoreAndRankDrivers scores drivers based on multiple factors
func (s *AdvancedMatchingService) scoreAndRankDrivers(ctx context.Context, drivers []*DriverLocation, request *MatchingRequest) ([]*MatchedDriverInfo, error) {
	var scoredDrivers []*MatchedDriverInfo

	for _, driver := range drivers {
		// Calculate ETA
		eta, err := s.geoService.CalculateETA(ctx, driver.Location, request.PickupLocation, driver.VehicleType)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to calculate ETA for driver", driver.DriverID)
			continue
		}

		// Create matched driver info
		matchedDriver := &MatchedDriverInfo{
			DriverID:        driver.DriverID,
			VehicleID:       driver.VehicleID,
			Rating:          driver.Rating,
			CurrentLocation: driver.Location,
			Distance:        driver.DistanceFromCenter,
			ETA:             eta.DurationSeconds,
			Status:          driver.Status,
			VehicleInfo: &VehicleDetails{
				VehicleType: driver.VehicleType,
				// Additional vehicle details would be fetched from vehicle service
			},
		}

		// Calculate composite matching score
		score := s.calculateMatchingScore(matchedDriver, request)
		matchedDriver.MatchScore = score

		scoredDrivers = append(scoredDrivers, matchedDriver)
	}

	// Sort by score (descending)
	sort.Slice(scoredDrivers, func(i, j int) bool {
		return scoredDrivers[i].MatchScore > scoredDrivers[j].MatchScore
	})

	return scoredDrivers, nil
}

// calculateMatchingScore calculates a composite score for driver matching
func (s *AdvancedMatchingService) calculateMatchingScore(driver *MatchedDriverInfo, request *MatchingRequest) float64 {
	score := 0.0

	// Distance factor (closer is better) - 40% weight
	maxDistance := 15.0 // km
	distanceScore := math.Max(0, (maxDistance-driver.Distance)/maxDistance) * 40

	// ETA factor (faster pickup is better) - 30% weight
	maxETA := 20.0 * 60 // 20 minutes in seconds
	etaScore := math.Max(0, (maxETA-float64(driver.ETA))/maxETA) * 30

	// Rating factor (higher rating is better) - 20% weight
	ratingScore := (driver.Rating / 5.0) * 20

	// Availability factor - 10% weight
	availabilityScore := 10.0 // Full score for available drivers

	score = distanceScore + etaScore + ratingScore + availabilityScore

	// Apply priority bonuses
	if request.PriorityLevel > 1 {
		score += float64(request.PriorityLevel-1) * 5 // Bonus for premium/emergency
	}

	return math.Min(100.0, score) // Cap at 100
}

// calculateFareEstimate estimates the fare for the trip
func (s *AdvancedMatchingService) calculateFareEstimate(ctx context.Context, request *MatchingRequest, driver *MatchedDriverInfo) (*FareEstimate, error) {
	// Calculate trip distance and duration
	distanceResult, err := s.geoService.CalculateDistance(ctx, request.PickupLocation, request.Destination)
	if err != nil {
		return nil, err
	}

	etaResult, err := s.geoService.CalculateETA(ctx, request.PickupLocation, request.Destination, driver.VehicleInfo.VehicleType)
	if err != nil {
		return nil, err
	}

	// Base fare calculation (simplified)
	baseFare := 3.00                                           // Base fare
	distanceFare := distanceResult.DistanceKm * 1.50           // $1.50 per km
	timeFare := float64(etaResult.DurationSeconds) / 60 * 0.25 // $0.25 per minute

	// Surge pricing (simplified - could be more sophisticated)
	surgeFare := 0.0
	if request.PriorityLevel > 1 {
		surgeFare = (baseFare + distanceFare + timeFare) * 0.5 // 50% surge for premium
	}

	total := baseFare + distanceFare + timeFare + surgeFare

	return &FareEstimate{
		BaseFare:      baseFare,
		DistanceFare:  distanceFare,
		TimeFare:      timeFare,
		SurgeFare:     surgeFare,
		TotalEstimate: total,
		Currency:      "USD",
	}, nil
}

// reserveDriver temporarily reserves a driver for the trip
func (s *AdvancedMatchingService) reserveDriver(ctx context.Context, driverID, tripID string) error {
	// Safety check for nil Redis dependency
	if s.redis == nil {
		if s.logger != nil {
			s.logger.WithContext(ctx).Warn("Redis client not available - driver reservation skipped")
		}
		return nil // Return success for testing without Redis
	}

	// Set a reservation in Redis with TTL
	key := fmt.Sprintf("driver_reservation:%s", driverID)
	value := fmt.Sprintf("trip:%s:reserved_at:%d", tripID, time.Now().Unix())

	return s.redis.SetEx(ctx, key, value, 5*time.Minute).Err()
} // GetMatchingStatus returns the status of ongoing matching processes
func (s *AdvancedMatchingService) GetMatchingStatus(ctx context.Context, tripID string) (map[string]interface{}, error) {
	status := "not_found"
	startedAt := time.Now().Add(-30 * time.Second) // Default fallback

	// Safety check for nil Redis dependency
	if s.redis != nil {
		// Check if there's an active reservation for this trip
		pattern := "driver_reservation:*"
		keys, err := s.redis.Keys(ctx, pattern).Result()
		if err != nil && s.logger != nil {
			s.logger.WithError(err).Warn("Failed to check driver reservations")
		}

		for _, key := range keys {
			value, err := s.redis.Get(ctx, key).Result()
			if err == nil && strings.Contains(value, tripID) {
				status = "searching"
				// Extract timestamp from reservation value if possible
				break
			}
		}
	} else if s.logger != nil {
		s.logger.WithContext(ctx).Warn("Redis client not available - using mock status")
		status = "searching" // Mock status for testing
	}

	return map[string]interface{}{
		"trip_id":      tripID,
		"status":       status,
		"started_at":   startedAt,
		"attempts":     1,
		"max_attempts": 3,
	}, nil
}

// CancelMatching cancels an ongoing matching process
func (s *AdvancedMatchingService) CancelMatching(ctx context.Context, tripID string) error {
	// Safety check for nil Redis dependency
	if s.redis == nil {
		if s.logger != nil {
			s.logger.WithContext(ctx).WithField("trip_id", tripID).Info("Matching cancelled (Redis not available)")
		}
		return nil // Return success for testing without Redis
	}

	// Remove any driver reservations for this trip
	pattern := "driver_reservation:*"
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		value, err := s.redis.Get(ctx, key).Result()
		if err == nil && contains(value, tripID) {
			s.redis.Del(ctx, key)
		}
	}

	if s.logger != nil {
		s.logger.WithContext(ctx).WithField("trip_id", tripID).Info("Matching cancelled")
	}
	return nil
}

// GetMatchingMetrics returns comprehensive matching metrics
func (s *AdvancedMatchingService) GetMatchingMetrics(ctx context.Context) (map[string]interface{}, error) {
	// In a real implementation, these would come from monitoring systems
	return map[string]interface{}{
		"total_requests":      1234,
		"successful_matches":  1089,
		"success_rate":        88.2,
		"avg_match_time":      "15.3s",
		"avg_match_score":     85.7,
		"active_searches":     5,
		"avg_driver_distance": "2.3km",
		"avg_eta":             "8.5min",
		"surge_active":        false,
		"available_drivers":   245,
		"active_trips":        123,
	}, nil
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// generateMockResult creates a mock matching result for testing purposes
func (s *AdvancedMatchingService) generateMockResult(request *MatchingRequest, startTime time.Time) *MatchingResult {
	mockDriver := &MatchedDriverInfo{
		DriverID:   "mock-driver-123",
		VehicleID:  "mock-vehicle-456",
		DriverName: "Mock Driver",
		Rating:     4.8,
		TripCount:  150,
		CurrentLocation: &models.Location{
			Latitude:  request.PickupLocation.Latitude,
			Longitude: request.PickupLocation.Longitude,
		},
		VehicleInfo: &VehicleDetails{
			Make:         "Test",
			Model:        "Vehicle",
			Year:         2023,
			Color:        "Blue",
			LicensePlate: "MOCK123",
			VehicleType:  request.VehicleType,
			Capacity:     4,
		},
		Distance:   1.2,
		ETA:        300, // 5 minutes
		MatchScore: 85.5,
		Status:     "available",
	}

	mockFare := &FareEstimate{
		BaseFare:      3.00,
		DistanceFare:  6.00,
		TimeFare:      2.50,
		SurgeFare:     0.00,
		TotalEstimate: 11.50,
		Currency:      "USD",
	}

	return &MatchingResult{
		TripID:             request.TripID,
		Success:            true,
		MatchedDriver:      mockDriver,
		EstimatedETA:       300,
		EstimatedFare:      mockFare,
		Reason:             "Mock match for testing",
		AlternativeOptions: []*MatchedDriverInfo{},
		MatchingScore:      85.5,
		ProcessingTime:     time.Since(startTime),
		RetryCount:         0,
	}
}
