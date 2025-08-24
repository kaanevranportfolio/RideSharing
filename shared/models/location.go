package models

import (
	"crypto/sha256"
	"fmt"
	"math"
	"time"
)

// Location represents a geographical coordinate
type Location struct {
	Latitude  float64   `json:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" db:"longitude"`
	Accuracy  float64   `json:"accuracy" db:"accuracy"` // accuracy in meters
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// Address represents a physical address
type Address struct {
	Street      string    `json:"street" db:"street"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	Country     string    `json:"country" db:"country"`
	PostalCode  string    `json:"postal_code" db:"postal_code"`
	Coordinates *Location `json:"coordinates,omitempty"`
}

// NewLocation creates a new location with current timestamp
func NewLocation(lat, lng, accuracy float64) *Location {
	return &Location{
		Latitude:  lat,
		Longitude: lng,
		Accuracy:  accuracy,
		Timestamp: time.Now(),
	}
}

// NewAddress creates a new address
func NewAddress(street, city, state, country, postalCode string) *Address {
	return &Address{
		Street:     street,
		City:       city,
		State:      state,
		Country:    country,
		PostalCode: postalCode,
	}
}

// IsValid checks if the location coordinates are valid
func (l *Location) IsValid() bool {
	return l.Latitude >= -90 && l.Latitude <= 90 &&
		l.Longitude >= -180 && l.Longitude <= 180
}

// DistanceTo calculates the distance to another location using Haversine formula
func (l *Location) DistanceTo(other *Location) float64 {
	if !l.IsValid() || !other.IsValid() {
		return 0
	}

	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := l.Latitude * math.Pi / 180
	lon1Rad := l.Longitude * math.Pi / 180
	lat2Rad := other.Latitude * math.Pi / 180
	lon2Rad := other.Longitude * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadiusKm * c

	return distance
}

// Bearing calculates the initial bearing from this location to another
func (l *Location) Bearing(other *Location) float64 {
	if !l.IsValid() || !other.IsValid() {
		return 0
	}

	lat1Rad := l.Latitude * math.Pi / 180
	lat2Rad := other.Latitude * math.Pi / 180
	deltaLonRad := (other.Longitude - l.Longitude) * math.Pi / 180

	y := math.Sin(deltaLonRad) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLonRad)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(bearing+360, 360)
}

// Geohash generates a geohash for the location with specified precision
func (l *Location) Geohash(precision int) string {
	if !l.IsValid() || precision <= 0 {
		return ""
	}

	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

	latRange := []float64{-90.0, 90.0}
	lonRange := []float64{-180.0, 180.0}

	var geohash string
	var bit int
	var ch int
	even := true

	for len(geohash) < precision {
		if even {
			// longitude
			mid := (lonRange[0] + lonRange[1]) / 2
			if l.Longitude >= mid {
				ch |= 1 << (4 - bit)
				lonRange[0] = mid
			} else {
				lonRange[1] = mid
			}
		} else {
			// latitude
			mid := (latRange[0] + latRange[1]) / 2
			if l.Latitude >= mid {
				ch |= 1 << (4 - bit)
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

// Hash generates a hash for the location (useful for caching)
func (l *Location) Hash() string {
	data := fmt.Sprintf("%.6f,%.6f", l.Latitude, l.Longitude)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16] // Return first 16 characters
}

// IsWithinRadius checks if the location is within a specified radius of another location
func (l *Location) IsWithinRadius(other *Location, radiusKm float64) bool {
	return l.DistanceTo(other) <= radiusKm
}

// MidpointTo calculates the midpoint between this location and another
func (l *Location) MidpointTo(other *Location) *Location {
	if !l.IsValid() || !other.IsValid() {
		return nil
	}

	lat1Rad := l.Latitude * math.Pi / 180
	lon1Rad := l.Longitude * math.Pi / 180
	lat2Rad := other.Latitude * math.Pi / 180
	deltaLonRad := (other.Longitude - l.Longitude) * math.Pi / 180

	bx := math.Cos(lat2Rad) * math.Cos(deltaLonRad)
	by := math.Cos(lat2Rad) * math.Sin(deltaLonRad)

	lat3Rad := math.Atan2(
		math.Sin(lat1Rad)+math.Sin(lat2Rad),
		math.Sqrt((math.Cos(lat1Rad)+bx)*(math.Cos(lat1Rad)+bx)+by*by),
	)
	lon3Rad := lon1Rad + math.Atan2(by, math.Cos(lat1Rad)+bx)

	return &Location{
		Latitude:  lat3Rad * 180 / math.Pi,
		Longitude: lon3Rad * 180 / math.Pi,
		Timestamp: time.Now(),
	}
}

// String returns a string representation of the location
func (l *Location) String() string {
	return fmt.Sprintf("(%.6f, %.6f)", l.Latitude, l.Longitude)
}

// String returns a string representation of the address
func (a *Address) String() string {
	parts := []string{}
	if a.Street != "" {
		parts = append(parts, a.Street)
	}
	if a.City != "" {
		parts = append(parts, a.City)
	}
	if a.State != "" {
		parts = append(parts, a.State)
	}
	if a.PostalCode != "" {
		parts = append(parts, a.PostalCode)
	}
	if a.Country != "" {
		parts = append(parts, a.Country)
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		result += part
	}
	return result
}

// IsComplete checks if the address has all required fields
func (a *Address) IsComplete() bool {
	return a.Street != "" && a.City != "" && a.State != "" && a.Country != ""
}

// DecodeGeohash decodes a geohash string back to a location
func DecodeGeohash(geohash string) *Location {
	if geohash == "" {
		return nil
	}

	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

	latRange := []float64{-90.0, 90.0}
	lonRange := []float64{-180.0, 180.0}

	even := true

	for _, char := range geohash {
		idx := -1
		for i, c := range base32 {
			if c == char {
				idx = i
				break
			}
		}

		if idx == -1 {
			return nil // Invalid character
		}

		for i := 4; i >= 0; i-- {
			bit := (idx >> i) & 1

			if even {
				// longitude
				mid := (lonRange[0] + lonRange[1]) / 2
				if bit == 1 {
					lonRange[0] = mid
				} else {
					lonRange[1] = mid
				}
			} else {
				// latitude
				mid := (latRange[0] + latRange[1]) / 2
				if bit == 1 {
					latRange[0] = mid
				} else {
					latRange[1] = mid
				}
			}

			even = !even
		}
	}

	return &Location{
		Latitude:  (latRange[0] + latRange[1]) / 2,
		Longitude: (lonRange[0] + lonRange[1]) / 2,
		Timestamp: time.Now(),
	}
}

// DriverLocation represents a driver's real-time location with additional metadata
type DriverLocation struct {
	DriverID           string    `json:"driver_id" db:"driver_id"`
	VehicleID          string    `json:"vehicle_id" db:"vehicle_id"`
	Location           *Location `json:"location" db:"location"`
	DistanceFromCenter float64   `json:"distance_from_center" db:"distance_from_center"`
	Status             string    `json:"status" db:"status"`
	VehicleType        string    `json:"vehicle_type" db:"vehicle_type"`
	Rating             float32   `json:"rating" db:"rating"`
	LastUpdated        time.Time `json:"last_updated" db:"last_updated"`
}

// NewDriverLocation creates a new driver location entry
func NewDriverLocation(driverID, vehicleID string, location *Location, status, vehicleType string, rating float32) *DriverLocation {
	return &DriverLocation{
		DriverID:    driverID,
		VehicleID:   vehicleID,
		Location:    location,
		Status:      status,
		VehicleType: vehicleType,
		Rating:      rating,
		LastUpdated: time.Now(),
	}
}
