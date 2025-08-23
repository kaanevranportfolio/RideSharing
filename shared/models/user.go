package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// generateID generates a simple ID for models
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// UserType represents the type of user in the system
type UserType string

const (
	UserTypeRider  UserType = "rider"
	UserTypeDriver UserType = "driver"
	UserTypeAdmin  UserType = "admin"
)

// UserStatus represents the current status of a user
type UserStatus string

const (
	UserStatusInactive  UserStatus = "inactive"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
)

// DriverStatus represents the current status of a driver
type DriverStatus string

const (
	DriverStatusOffline DriverStatus = "offline"
	DriverStatusOnline  DriverStatus = "online"
	DriverStatusBusy    DriverStatus = "busy"
	DriverStatusBreak   DriverStatus = "break"
)

// User represents a user in the rideshare platform
type User struct {
	ID              string     `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	Phone           string     `json:"phone" db:"phone"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	FirstName       string     `json:"first_name" db:"first_name"`
	LastName        string     `json:"last_name" db:"last_name"`
	UserType        UserType   `json:"user_type" db:"user_type"`
	Status          UserStatus `json:"status" db:"status"`
	ProfileImageURL string     `json:"profile_image_url" db:"profile_image_url"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	PhoneVerified   bool       `json:"phone_verified" db:"phone_verified"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// Driver represents a driver profile
type Driver struct {
	UserID                  string       `json:"user_id" db:"user_id"`
	LicenseNumber           string       `json:"license_number" db:"license_number"`
	LicenseExpiry           time.Time    `json:"license_expiry" db:"license_expiry"`
	Status                  DriverStatus `json:"status" db:"status"`
	Rating                  float64      `json:"rating" db:"rating"`
	TotalTrips              int          `json:"total_trips" db:"total_trips"`
	TotalEarningsCents      int64        `json:"total_earnings_cents" db:"total_earnings_cents"`
	CurrentLatitude         *float64     `json:"current_latitude" db:"current_latitude"`
	CurrentLongitude        *float64     `json:"current_longitude" db:"current_longitude"`
	CurrentLocationAccuracy *float64     `json:"current_location_accuracy" db:"current_location_accuracy"`
	LastLocationUpdate      *time.Time   `json:"last_location_update" db:"last_location_update"`
	BackgroundCheckStatus   string       `json:"background_check_status" db:"background_check_status"`
	BackgroundCheckDate     *time.Time   `json:"background_check_date" db:"background_check_date"`
	CreatedAt               time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at" db:"updated_at"`
}

// NewUser creates a new user with default values
func NewUser(email, phone, firstName, lastName string, userType UserType) *User {
	return &User{
		ID:            generateID(),
		Email:         email,
		Phone:         phone,
		FirstName:     firstName,
		LastName:      lastName,
		UserType:      userType,
		Status:        UserStatusActive,
		EmailVerified: false,
		PhoneVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// NewDriver creates a new driver profile
func NewDriver(userID, licenseNumber string, licenseExpiry time.Time) *Driver {
	return &Driver{
		UserID:                userID,
		LicenseNumber:         licenseNumber,
		LicenseExpiry:         licenseExpiry,
		Status:                DriverStatusOffline,
		Rating:                5.0,
		TotalTrips:            0,
		TotalEarningsCents:    0,
		BackgroundCheckStatus: "pending",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive returns true if the user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsDriver returns true if the user is a driver
func (u *User) IsDriver() bool {
	return u.UserType == UserTypeDriver
}

// IsRider returns true if the user is a rider
func (u *User) IsRider() bool {
	return u.UserType == UserTypeRider
}

// IsOnline returns true if the driver is online
func (d *Driver) IsOnline() bool {
	return d.Status == DriverStatusOnline
}

// IsAvailable returns true if the driver is available for trips
func (d *Driver) IsAvailable() bool {
	return d.Status == DriverStatusOnline
}

// HasCurrentLocation returns true if the driver has a current location
func (d *Driver) HasCurrentLocation() bool {
	return d.CurrentLatitude != nil && d.CurrentLongitude != nil
}

// GetCurrentLocation returns the driver's current location
func (d *Driver) GetCurrentLocation() *Location {
	if !d.HasCurrentLocation() {
		return nil
	}

	location := &Location{
		Latitude:  *d.CurrentLatitude,
		Longitude: *d.CurrentLongitude,
	}

	if d.CurrentLocationAccuracy != nil {
		location.Accuracy = *d.CurrentLocationAccuracy
	}

	if d.LastLocationUpdate != nil {
		location.Timestamp = *d.LastLocationUpdate
	}

	return location
}

// UpdateLocation updates the driver's current location
func (d *Driver) UpdateLocation(lat, lng, accuracy float64) {
	d.CurrentLatitude = &lat
	d.CurrentLongitude = &lng
	d.CurrentLocationAccuracy = &accuracy
	now := time.Now()
	d.LastLocationUpdate = &now
	d.UpdatedAt = now
}

// UpdateStatus updates the driver's status
func (d *Driver) UpdateStatus(status DriverStatus) {
	d.Status = status
	d.UpdatedAt = time.Now()
}

// UpdateRating updates the driver's rating
func (d *Driver) UpdateRating(newRating float64, totalRatings int) {
	// Calculate weighted average
	currentTotal := d.Rating * float64(d.TotalTrips)
	d.Rating = (currentTotal + newRating) / float64(d.TotalTrips+1)
	d.UpdatedAt = time.Now()
}

// AddEarnings adds earnings to the driver's total
func (d *Driver) AddEarnings(amountCents int64) {
	d.TotalEarningsCents += amountCents
	d.UpdatedAt = time.Now()
}

// IncrementTripCount increments the driver's trip count
func (d *Driver) IncrementTripCount() {
	d.TotalTrips++
	d.UpdatedAt = time.Now()
}
