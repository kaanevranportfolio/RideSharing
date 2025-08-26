package geo_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHaversineDistanceFormula tests the Haversine formula implementation
func TestHaversineDistanceFormula(t *testing.T) {
	// Test locations: Times Square and Central Park
	lat1 := 40.7589 // Times Square
	lon1 := -73.9851
	lat2 := 40.7829 // Central Park
	lon2 := -73.9654

	distance := calculateHaversineDistance(lat1, lon1, lat2, lon2)

	// Expected distance is approximately 2.7km
	assert.Greater(t, distance, 2000.0, "Distance should be greater than 2km")
	assert.Less(t, distance, 4000.0, "Distance should be less than 4km")
}

// TestManhattanDistanceCalculation tests Manhattan distance calculation
func TestManhattanDistanceCalculation(t *testing.T) {
	lat1 := 40.7589
	lon1 := -73.9851
	lat2 := 40.7829
	lon2 := -73.9654

	distance := calculateManhattanDistance(lat1, lon1, lat2, lon2)

	// Manhattan distance should be greater than Haversine
	haversine := calculateHaversineDistance(lat1, lon1, lat2, lon2)
	assert.Greater(t, distance, haversine, "Manhattan distance should be greater than Haversine")
}

// TestGeohashGeneration tests geohash calculation
func TestGeohashGeneration(t *testing.T) {
	lat := 40.7589
	lon := -73.9851
	precision := 6

	geohash := calculateGeohash(lat, lon, precision)

	assert.Equal(t, precision, len(geohash), "Geohash should have correct precision")

	// Test consistency
	geohash2 := calculateGeohash(lat, lon, precision)
	assert.Equal(t, geohash, geohash2, "Same location should produce same geohash")
}

// TestTrafficFactorCalculation tests traffic factor algorithms
func TestTrafficFactorCalculation(t *testing.T) {
	// Rush hour
	rushHourFactor := getTrafficFactor(8) // 8 AM
	assert.Equal(t, 1.5, rushHourFactor, "Rush hour should have 1.5x factor")

	// Normal hours
	normalFactor := getTrafficFactor(14) // 2 PM
	assert.Equal(t, 1.0, normalFactor, "Normal hours should have 1.0x factor")

	// Late night
	lateNightFactor := getTrafficFactor(2) // 2 AM
	assert.Equal(t, 0.8, lateNightFactor, "Late night should have 0.8x factor")
}

// TestBoundaryConditions tests edge cases
func TestBoundaryConditions(t *testing.T) {
	// Same location
	distance := calculateHaversineDistance(40.7589, -73.9851, 40.7589, -73.9851)
	assert.InDelta(t, 0.0, distance, 0.1, "Same location should have ~0 distance")

	// Extreme coordinates
	distance = calculateHaversineDistance(-90, -180, 90, 180)
	assert.Greater(t, distance, 10000000.0, "Extreme coordinates should give large distance")
}

// Helper functions implementing the algorithms

func calculateHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadiusKm * c * 1000 // Convert to meters

	return distance
}

func calculateManhattanDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const degreesToMeters = 111000

	deltaLat := math.Abs(lat2-lat1) * degreesToMeters
	deltaLng := math.Abs(lon2-lon1) * degreesToMeters * math.Cos(lat1*math.Pi/180)

	return deltaLat + deltaLng
}

func calculateGeohash(lat, lng float64, precision int) string {
	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"
	var geohash string

	latRange := []float64{-90.0, 90.0}
	lngRange := []float64{-180.0, 180.0}

	var even bool = true
	var bit int = 0
	var ch int = 0

	for len(geohash) < precision {
		if even {
			mid := (lngRange[0] + lngRange[1]) / 2
			if lng >= mid {
				ch |= (1 << (4 - bit))
				lngRange[0] = mid
			} else {
				lngRange[1] = mid
			}
		} else {
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

func getTrafficFactor(hour int) float64 {
	// Rush hour times (7-9 AM, 5-7 PM)
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		return 1.5 // rush_hour factor
	}

	// Late night (11 PM - 5 AM)
	if hour >= 23 || hour <= 5 {
		return 0.8 // late_night factor
	}

	// Normal hours
	return 1.0 // normal factor
}
