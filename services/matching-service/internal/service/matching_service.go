package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/rideshare-platform/services/matching-service/internal/config"
)

// MatchingService handles trip matching logic
type MatchingService struct {
	config *config.Config
	// repositories would be added here
}

// NewMatchingService creates a new matching service
func NewMatchingService(cfg *config.Config) *MatchingService {
	return &MatchingService{
		config: cfg,
	}
}

// MatchingRequest represents a trip matching request
type MatchingRequest struct {
	TripID         string    `json:"trip_id"`
	RiderID        string    `json:"rider_id"`
	PickupLocation Location  `json:"pickup_location"`
	Destination    Location  `json:"destination"`
	PassengerCount int       `json:"passenger_count"`
	VehicleType    string    `json:"vehicle_type"`
	RequestedAt    time.Time `json:"requested_at"`
	SpecialNeeds   []string  `json:"special_needs,omitempty"`
}

// Location represents a geographic location
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// DriverInfo represents driver information for matching
type DriverInfo struct {
	DriverID        string    `json:"driver_id"`
	VehicleID       string    `json:"vehicle_id"`
	CurrentLocation Location  `json:"current_location"`
	VehicleType     string    `json:"vehicle_type"`
	Rating          float64   `json:"rating"`
	Status          string    `json:"status"`
	LastUpdate      time.Time `json:"last_update"`
	Distance        float64   `json:"distance"` // calculated distance from pickup
}

// MatchingResult represents the result of a matching attempt
type MatchingResult struct {
	TripID       string       `json:"trip_id"`
	Success      bool         `json:"success"`
	DriverID     string       `json:"driver_id,omitempty"`
	VehicleID    string       `json:"vehicle_id,omitempty"`
	EstimatedETA int          `json:"estimated_eta,omitempty"` // seconds
	Reason       string       `json:"reason,omitempty"`
	Candidates   []DriverInfo `json:"candidates,omitempty"`
}

// FindMatch attempts to find a suitable driver for a trip
func (s *MatchingService) FindMatch(ctx context.Context, request *MatchingRequest) (*MatchingResult, error) {
	// For now, simulate the matching process
	// In a real implementation, this would:
	// 1. Query nearby drivers
	// 2. Filter by availability and vehicle type
	// 3. Apply matching algorithm
	// 4. Return best match

	result := &MatchingResult{
		TripID:  request.TripID,
		Success: false,
		Reason:  "No implementation yet - this is a demo service",
	}

	// Simulate some candidates for demo
	candidates := s.generateMockCandidates(request.PickupLocation)
	result.Candidates = candidates

	if len(candidates) > 0 {
		// Select the best candidate (closest for now)
		best := candidates[0]
		result.Success = true
		result.DriverID = best.DriverID
		result.VehicleID = best.VehicleID
		result.EstimatedETA = int(best.Distance * 60) // rough estimate: 1km = 1 minute
		result.Reason = "Mock match successful"
	}

	return result, nil
}

// generateMockCandidates generates mock driver candidates for testing
func (s *MatchingService) generateMockCandidates(pickup Location) []DriverInfo {
	candidates := []DriverInfo{
		{
			DriverID:        "driver_001",
			VehicleID:       "vehicle_001",
			CurrentLocation: Location{Latitude: pickup.Latitude + 0.01, Longitude: pickup.Longitude + 0.01},
			VehicleType:     "sedan",
			Rating:          4.8,
			Status:          "available",
			LastUpdate:      time.Now(),
		},
		{
			DriverID:        "driver_002",
			VehicleID:       "vehicle_002",
			CurrentLocation: Location{Latitude: pickup.Latitude - 0.005, Longitude: pickup.Longitude + 0.015},
			VehicleType:     "suv",
			Rating:          4.6,
			Status:          "available",
			LastUpdate:      time.Now(),
		},
		{
			DriverID:        "driver_003",
			VehicleID:       "vehicle_003",
			CurrentLocation: Location{Latitude: pickup.Latitude + 0.02, Longitude: pickup.Longitude - 0.01},
			VehicleType:     "sedan",
			Rating:          4.9,
			Status:          "available",
			LastUpdate:      time.Now(),
		},
	}

	// Calculate distances and sort by distance
	for i := range candidates {
		candidates[i].Distance = calculateDistance(pickup, candidates[i].CurrentLocation)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Distance < candidates[j].Distance
	})

	return candidates
}

// calculateDistance calculates the distance between two locations using Haversine formula
func calculateDistance(loc1, loc2 Location) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers

	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLat := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLon := (loc2.Longitude - loc1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// GetMatchingStatus returns the status of ongoing matching processes
func (s *MatchingService) GetMatchingStatus(ctx context.Context, tripID string) (map[string]interface{}, error) {
	// Mock implementation
	return map[string]interface{}{
		"trip_id":      tripID,
		"status":       "searching",
		"started_at":   time.Now().Add(-30 * time.Second),
		"attempts":     2,
		"max_attempts": s.config.MatchingRetryAttempts,
	}, nil
}

// CancelMatching cancels an ongoing matching process
func (s *MatchingService) CancelMatching(ctx context.Context, tripID string) error {
	// Mock implementation
	return nil
}

// GetMatchingMetrics returns metrics about the matching service
func (s *MatchingService) GetMatchingMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_requests":     1234,
		"successful_matches": 1089,
		"success_rate":       88.2,
		"avg_match_time":     "15.3s",
		"active_searches":    5,
	}, nil
}
