package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/rideshare-platform/services/geo-service/internal/config"
	"github.com/rideshare-platform/services/geo-service/internal/repository"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
)

// GeospatialService handles all geospatial calculations and operations
type GeospatialService struct {
	config     *config.Config
	logger     *logger.Logger
	driverRepo *repository.DriverLocationRepository
	cacheRepo  *repository.CacheRepository
	mongo      *mongo.Client
	redis      *redis.Client
}

// NewGeospatialService creates a new geospatial service
func NewGeospatialService(
	cfg *config.Config,
	log *logger.Logger,
	driverRepo *repository.DriverLocationRepository,
	cacheRepo *repository.CacheRepository,
	mongo *mongo.Client,
	redis *redis.Client,
) *GeospatialService {
	return &GeospatialService{
		config:     cfg,
		logger:     log,
		driverRepo: driverRepo,
		cacheRepo:  cacheRepo,
		mongo:      mongo,
		redis:      redis,
	}
}

// DistanceCalculation represents the result of a distance calculation
type DistanceCalculation struct {
	DistanceMeters    float64 `json:"distance_meters"`
	DistanceKm        float64 `json:"distance_km"`
	BearingDegrees    float64 `json:"bearing_degrees"`
	CalculationMethod string  `json:"calculation_method"`
}

// ETACalculation represents the result of an ETA calculation
type ETACalculation struct {
	DurationSeconds  int               `json:"duration_seconds"`
	DistanceMeters   float64           `json:"distance_meters"`
	RouteSummary     string            `json:"route_summary"`
	Waypoints        []models.Location `json:"waypoints"`
	EstimatedArrival time.Time         `json:"estimated_arrival"`
}

// NearbyDriver represents a driver with location and distance information
type NearbyDriver struct {
	DriverID           string          `json:"driver_id"`
	VehicleID          string          `json:"vehicle_id"`
	Location           models.Location `json:"location"`
	DistanceFromCenter float64         `json:"distance_from_center"`
	Status             string          `json:"status"`
	VehicleType        string          `json:"vehicle_type"`
	Rating             float64         `json:"rating"`
}

// CalculateDistance calculates the distance between two geographical points
func (s *GeospatialService) CalculateDistance(ctx context.Context, origin, destination models.Location, method string) (*DistanceCalculation, error) {
	// Use default method if not specified
	if method == "" {
		method = s.config.Geospatial.DefaultDistanceMethod
	}

	// Check cache first
	cacheKey := fmt.Sprintf("distance:%s:%.6f,%.6f:%.6f,%.6f", method, origin.Latitude, origin.Longitude, destination.Latitude, destination.Longitude)
	if s.config.Cache.EnableCaching {
		if _, err := s.cacheRepo.Get(ctx, cacheKey); err == nil {
			s.logger.WithContext(ctx).Debug("Distance calculation cache hit")
			// In a real implementation, you'd unmarshal and return the cached result
			// For now, we'll continue with calculation
		}
	}

	var distance float64
	var bearing float64

	switch method {
	case "haversine":
		distance, bearing = s.calculateHaversineDistance(origin, destination)
	case "manhattan":
		distance, bearing = s.calculateManhattanDistance(origin, destination)
	case "euclidean":
		distance, bearing = s.calculateEuclideanDistance(origin, destination)
	default:
		return nil, fmt.Errorf("unsupported calculation method: %s", method)
	}

	result := &DistanceCalculation{
		DistanceMeters:    distance,
		DistanceKm:        distance / 1000,
		BearingDegrees:    bearing,
		CalculationMethod: method,
	}

	// Cache the result
	if s.config.Cache.EnableCaching {
		s.cacheRepo.Set(ctx, cacheKey, result, time.Duration(s.config.Cache.DistanceCacheTTL)*time.Second)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"method":      method,
		"distance_km": result.DistanceKm,
		"bearing":     result.BearingDegrees,
	}).Debug("Distance calculated")

	return result, nil
}

// CalculateETA calculates estimated time of arrival and route information
func (s *GeospatialService) CalculateETA(ctx context.Context, origin, destination models.Location, vehicleType string, departureTime time.Time, includeTraffic bool) (*ETACalculation, error) {
	// Calculate base distance
	distanceCalc, err := s.CalculateDistance(ctx, origin, destination, "haversine")
	if err != nil {
		return nil, fmt.Errorf("failed to calculate distance for ETA: %w", err)
	}

	// Get vehicle speed
	speed, exists := s.config.Geospatial.RouteOptimization.DefaultSpeeds[vehicleType]
	if !exists {
		speed = s.config.Geospatial.RouteOptimization.DefaultSpeeds["car"] // default to car speed
	}

	// Calculate base duration (distance / speed)
	baseDurationHours := distanceCalc.DistanceKm / speed
	baseDurationSeconds := int(baseDurationHours * 3600)

	// Apply traffic factors if requested
	if includeTraffic {
		trafficFactor := s.getTrafficFactor(departureTime)
		baseDurationSeconds = int(float64(baseDurationSeconds) * trafficFactor)
	}

	estimatedArrival := departureTime.Add(time.Duration(baseDurationSeconds) * time.Second)

	// Generate route summary
	routeSummary := fmt.Sprintf("Route from (%.6f, %.6f) to (%.6f, %.6f) via %s - %.2f km",
		origin.Latitude, origin.Longitude,
		destination.Latitude, destination.Longitude,
		vehicleType, distanceCalc.DistanceKm)

	// Generate waypoints (simplified - in reality would use routing service)
	waypoints := s.generateWaypoints(origin, destination, 3)

	result := &ETACalculation{
		DurationSeconds:  baseDurationSeconds,
		DistanceMeters:   distanceCalc.DistanceMeters,
		RouteSummary:     routeSummary,
		Waypoints:        waypoints,
		EstimatedArrival: estimatedArrival,
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_type":     vehicleType,
		"duration_minutes": baseDurationSeconds / 60,
		"distance_km":      distanceCalc.DistanceKm,
		"include_traffic":  includeTraffic,
	}).Debug("ETA calculated")

	return result, nil
}

// FindNearbyDrivers finds drivers within a specified radius of a location
func (s *GeospatialService) FindNearbyDrivers(ctx context.Context, center models.Location, radiusKm float64, limit int, vehicleTypes []string, onlyAvailable bool) ([]NearbyDriver, error) {
	// Validate radius
	if radiusKm > s.config.Geospatial.MaxSearchRadiusKm {
		radiusKm = s.config.Geospatial.MaxSearchRadiusKm
	}

	// Validate limit
	if limit > s.config.Geospatial.MaxNearbyDrivers {
		limit = s.config.Geospatial.MaxNearbyDrivers
	}

	// Get driver locations from repository
	driverLocations, err := s.driverRepo.FindNearbyDrivers(ctx, center, radiusKm, vehicleTypes, onlyAvailable)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	// Calculate distances and sort
	var nearbyDrivers []NearbyDriver
	for _, driverLoc := range driverLocations {
		distance, _ := s.calculateHaversineDistance(center, driverLoc.Location)

		nearbyDrivers = append(nearbyDrivers, NearbyDriver{
			DriverID:           driverLoc.DriverID,
			VehicleID:          driverLoc.VehicleID,
			Location:           driverLoc.Location,
			DistanceFromCenter: distance / 1000, // convert to km
			Status:             driverLoc.Status,
			VehicleType:        driverLoc.VehicleType,
			Rating:             driverLoc.Rating,
		})
	}

	// Sort by distance
	sort.Slice(nearbyDrivers, func(i, j int) bool {
		return nearbyDrivers[i].DistanceFromCenter < nearbyDrivers[j].DistanceFromCenter
	})

	// Apply limit
	if len(nearbyDrivers) > limit {
		nearbyDrivers = nearbyDrivers[:limit]
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"center_lat":     center.Latitude,
		"center_lng":     center.Longitude,
		"radius_km":      radiusKm,
		"drivers_found":  len(nearbyDrivers),
		"only_available": onlyAvailable,
		"vehicle_types":  vehicleTypes,
	}).Info("Nearby drivers search completed")

	return nearbyDrivers, nil
}

// UpdateDriverLocation updates a driver's location
func (s *GeospatialService) UpdateDriverLocation(ctx context.Context, driverID string, location models.Location, status string, vehicleID string) error {
	driverLocation := &repository.DriverLocation{
		DriverID:  driverID,
		VehicleID: vehicleID,
		Location:  location,
		Status:    status,
		UpdatedAt: time.Now(),
	}

	err := s.driverRepo.UpdateDriverLocation(ctx, driverLocation)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":  driverID,
		"vehicle_id": vehicleID,
		"latitude":   location.Latitude,
		"longitude":  location.Longitude,
		"status":     status,
	}).Info("Driver location updated")

	return nil
}

// GenerateGeohash generates a geohash for a location
func (s *GeospatialService) GenerateGeohash(ctx context.Context, location models.Location, precision int) (string, error) {
	if precision <= 0 {
		precision = s.config.Geospatial.DefaultGeohashPrecision
	}

	// Validate precision
	if precision < 1 || precision > 12 {
		return "", fmt.Errorf("invalid geohash precision: %d (must be 1-12)", precision)
	}

	geohash := s.calculateGeohash(location.Latitude, location.Longitude, precision)

	s.logger.WithContext(ctx).WithFields(logger.Fields{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"precision": precision,
		"geohash":   geohash,
	}).Debug("Geohash generated")

	return geohash, nil
}

// PingMongo pings the MongoDB instance
func (s *GeospatialService) PingMongo(ctx context.Context) error {
	if s.mongo == nil {
		return errors.New("mongo client not initialized")
	}
	return s.mongo.Ping(ctx, nil)
}

// PingRedis pings the Redis instance
func (s *GeospatialService) PingRedis(ctx context.Context) error {
	if s.redis == nil {
		return errors.New("redis client not initialized")
	}
	return s.redis.Ping(ctx).Err()
}

// Private helper methods

// calculateHaversineDistance calculates the great-circle distance between two points
func (s *GeospatialService) calculateHaversineDistance(origin, destination models.Location) (float64, float64) {
	const earthRadiusKm = 6371

	lat1Rad := origin.Latitude * math.Pi / 180
	lat2Rad := destination.Latitude * math.Pi / 180
	deltaLatRad := (destination.Latitude - origin.Latitude) * math.Pi / 180
	deltaLngRad := (destination.Longitude - origin.Longitude) * math.Pi / 180

	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLngRad/2)*math.Sin(deltaLngRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceKm := earthRadiusKm * c
	distanceMeters := distanceKm * 1000

	// Calculate bearing
	y := math.Sin(deltaLngRad) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLngRad)
	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}

	return distanceMeters, bearing
}

// calculateManhattanDistance calculates Manhattan distance (for city grids)
func (s *GeospatialService) calculateManhattanDistance(origin, destination models.Location) (float64, float64) {
	const degreesToMeters = 111000 // approximate meters per degree at equator

	deltaLat := math.Abs(destination.Latitude - origin.Latitude)
	deltaLng := math.Abs(destination.Longitude - origin.Longitude)

	latDistance := deltaLat * degreesToMeters
	lngDistance := deltaLng * degreesToMeters * math.Cos(origin.Latitude*math.Pi/180)

	distance := latDistance + lngDistance

	// Calculate approximate bearing
	bearing := math.Atan2(destination.Longitude-origin.Longitude, destination.Latitude-origin.Latitude) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}

	return distance, bearing
}

// calculateEuclideanDistance calculates straight-line distance
func (s *GeospatialService) calculateEuclideanDistance(origin, destination models.Location) (float64, float64) {
	const degreesToMeters = 111000

	deltaLat := (destination.Latitude - origin.Latitude) * degreesToMeters
	deltaLng := (destination.Longitude - origin.Longitude) * degreesToMeters * math.Cos(origin.Latitude*math.Pi/180)

	distance := math.Sqrt(deltaLat*deltaLat + deltaLng*deltaLng)

	bearing := math.Atan2(deltaLng, deltaLat) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}

	return distance, bearing
}

// getTrafficFactor returns traffic multiplier based on time of day
func (s *GeospatialService) getTrafficFactor(departureTime time.Time) float64 {
	hour := departureTime.Hour()

	// Rush hour times (7-9 AM, 5-7 PM)
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		return s.config.Geospatial.RouteOptimization.TrafficFactors["rush_hour"]
	}

	// Late night (11 PM - 5 AM)
	if hour >= 23 || hour <= 5 {
		return s.config.Geospatial.RouteOptimization.TrafficFactors["late_night"]
	}

	// Normal hours
	return s.config.Geospatial.RouteOptimization.TrafficFactors["normal"]
}

// generateWaypoints generates intermediate waypoints for a route
func (s *GeospatialService) generateWaypoints(origin, destination models.Location, count int) []models.Location {
	var waypoints []models.Location

	// Add origin
	waypoints = append(waypoints, origin)

	// Generate intermediate points
	for i := 1; i < count; i++ {
		ratio := float64(i) / float64(count)

		lat := origin.Latitude + ratio*(destination.Latitude-origin.Latitude)
		lng := origin.Longitude + ratio*(destination.Longitude-origin.Longitude)

		waypoints = append(waypoints, models.Location{
			Latitude:  lat,
			Longitude: lng,
			Timestamp: time.Now(),
		})
	}

	// Add destination
	waypoints = append(waypoints, destination)

	return waypoints
}

// calculateGeohash generates a geohash for given coordinates
func (s *GeospatialService) calculateGeohash(lat, lng float64, precision int) string {
	// Simplified geohash implementation
	// In production, use a proper geohash library

	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"
	var geohash string

	latRange := []float64{-90.0, 90.0}
	lngRange := []float64{-180.0, 180.0}

	var even bool = true
	var bit int = 0
	var ch int = 0

	for len(geohash) < precision {
		if even {
			// longitude
			mid := (lngRange[0] + lngRange[1]) / 2
			if lng >= mid {
				ch |= (1 << (4 - bit))
				lngRange[0] = mid
			} else {
				lngRange[1] = mid
			}
		} else {
			// latitude
			mid := (latRange[0] + latRange[1]) / 2
			if lat >= mid {
				ch |= (1 << (4 - bit))
				latRange[0] = mid
			} else {
				latRange[1] = mid
			}
		}

		even = !even
		bit++

		if bit == 5 {
			geohash += string(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return geohash
}
