package models

import (
	"fmt"
	"time"
)

// Money represents a monetary value
type Money struct {
	Amount   int64  `json:"amount" db:"amount"`     // amount in cents
	Currency string `json:"currency" db:"currency"` // ISO 4217 currency code
}

// PricingFactors represents the factors used in fare calculation
type PricingFactors struct {
	BaseFare        float64 `json:"base_fare" db:"base_fare"`
	PerKmRate       float64 `json:"per_km_rate" db:"per_km_rate"`
	PerMinuteRate   float64 `json:"per_minute_rate" db:"per_minute_rate"`
	SurgeMultiplier float64 `json:"surge_multiplier" db:"surge_multiplier"`
	BookingFee      float64 `json:"booking_fee" db:"booking_fee"`
	ServiceFee      float64 `json:"service_fee" db:"service_fee"`
}

// FareBreakdown represents a detailed breakdown of fare calculation
type FareBreakdown struct {
	BaseFare     Money `json:"base_fare" db:"base_fare"`
	DistanceFare Money `json:"distance_fare" db:"distance_fare"`
	TimeFare     Money `json:"time_fare" db:"time_fare"`
	SurgeAmount  Money `json:"surge_amount" db:"surge_amount"`
	BookingFee   Money `json:"booking_fee" db:"booking_fee"`
	ServiceFee   Money `json:"service_fee" db:"service_fee"`
	Discount     Money `json:"discount" db:"discount"`
	Total        Money `json:"total" db:"total"`
}

// PricingRule represents a pricing rule for a specific vehicle type and location
type PricingRule struct {
	ID                 string             `json:"id" db:"id"`
	Name               string             `json:"name" db:"name"`
	VehicleType        VehicleType        `json:"vehicle_type" db:"vehicle_type"`
	City               *string            `json:"city" db:"city"`
	BaseFareCents      int64              `json:"base_fare_cents" db:"base_fare_cents"`
	PerKmRateCents     int64              `json:"per_km_rate_cents" db:"per_km_rate_cents"`
	PerMinuteRateCents int64              `json:"per_minute_rate_cents" db:"per_minute_rate_cents"`
	BookingFeeCents    int64              `json:"booking_fee_cents" db:"booking_fee_cents"`
	ServiceFeeCents    int64              `json:"service_fee_cents" db:"service_fee_cents"`
	TimeMultipliers    map[string]float64 `json:"time_multipliers" db:"time_multipliers"`
	DayMultipliers     map[string]float64 `json:"day_multipliers" db:"day_multipliers"`
	MinFareCents       *int64             `json:"min_fare_cents" db:"min_fare_cents"`
	MaxFareCents       *int64             `json:"max_fare_cents" db:"max_fare_cents"`
	ValidFrom          time.Time          `json:"valid_from" db:"valid_from"`
	ValidUntil         *time.Time         `json:"valid_until" db:"valid_until"`
	IsActive           bool               `json:"is_active" db:"is_active"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
}

// SurgePricing represents surge pricing information
type SurgePricing struct {
	ID              string      `json:"id" db:"id"`
	LocationGeohash string      `json:"location_geohash" db:"location_geohash"`
	VehicleType     VehicleType `json:"vehicle_type" db:"vehicle_type"`
	Multiplier      float64     `json:"multiplier" db:"multiplier"`
	Reason          *string     `json:"reason" db:"reason"`
	DemandLevel     string      `json:"demand_level" db:"demand_level"`
	SupplyLevel     string      `json:"supply_level" db:"supply_level"`
	StartsAt        time.Time   `json:"starts_at" db:"starts_at"`
	ExpiresAt       time.Time   `json:"expires_at" db:"expires_at"`
	Active          bool        `json:"is_active" db:"is_active"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// PromoCode represents a promotional discount code
type PromoCode struct {
	ID                     string        `json:"id" db:"id"`
	Code                   string        `json:"code" db:"code"`
	Description            *string       `json:"description" db:"description"`
	DiscountType           string        `json:"discount_type" db:"discount_type"` // "percentage" or "fixed_amount"
	DiscountValue          float64       `json:"discount_value" db:"discount_value"`
	MaxDiscountCents       *int64        `json:"max_discount_cents" db:"max_discount_cents"`
	MinTripAmountCents     *int64        `json:"min_trip_amount_cents" db:"min_trip_amount_cents"`
	MaxUses                *int          `json:"max_uses" db:"max_uses"`
	MaxUsesPerUser         int           `json:"max_uses_per_user" db:"max_uses_per_user"`
	CurrentUses            int           `json:"current_uses" db:"current_uses"`
	ValidFrom              time.Time     `json:"valid_from" db:"valid_from"`
	ValidUntil             time.Time     `json:"valid_until" db:"valid_until"`
	Active                 bool          `json:"is_active" db:"is_active"`
	ApplicableVehicleTypes []VehicleType `json:"applicable_vehicle_types" db:"applicable_vehicle_types"`
	ApplicableCities       []string      `json:"applicable_cities" db:"applicable_cities"`
	FirstRideOnly          bool          `json:"first_ride_only" db:"first_ride_only"`
	CreatedAt              time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at" db:"updated_at"`
}

// NewMoney creates a new Money instance
func NewMoney(amountCents int64, currency string) Money {
	return Money{
		Amount:   amountCents,
		Currency: currency,
	}
}

// NewPricingRule creates a new pricing rule
func NewPricingRule(name string, vehicleType VehicleType, city *string) *PricingRule {
	return &PricingRule{
		ID:              generateID(),
		Name:            name,
		VehicleType:     vehicleType,
		City:            city,
		TimeMultipliers: make(map[string]float64),
		DayMultipliers:  make(map[string]float64),
		ValidFrom:       time.Now(),
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// NewSurgePricing creates a new surge pricing instance
func NewSurgePricing(locationGeohash string, vehicleType VehicleType, multiplier float64, durationMinutes int) *SurgePricing {
	now := time.Now()
	return &SurgePricing{
		ID:              generateID(),
		LocationGeohash: locationGeohash,
		VehicleType:     vehicleType,
		Multiplier:      multiplier,
		StartsAt:        now,
		ExpiresAt:       now.Add(time.Duration(durationMinutes) * time.Minute),
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NewPromoCode creates a new promo code
func NewPromoCode(code, discountType string, discountValue float64) *PromoCode {
	return &PromoCode{
		ID:             generateID(),
		Code:           code,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
		MaxUsesPerUser: 1,
		CurrentUses:    0,
		ValidFrom:      time.Now(),
		ValidUntil:     time.Now().AddDate(0, 1, 0), // Valid for 1 month by default
		Active:         true,
		FirstRideOnly:  false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// ToFloat64 converts Money to float64 (in major currency units)
func (m Money) ToFloat64() float64 {
	return float64(m.Amount) / 100.0
}

// Add adds another Money amount (must be same currency)
func (m Money) Add(other Money) Money {
	if m.Currency != other.Currency {
		return m // Return original if currencies don't match
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: m.Currency,
	}
}

// Subtract subtracts another Money amount (must be same currency)
func (m Money) Subtract(other Money) Money {
	if m.Currency != other.Currency {
		return m // Return original if currencies don't match
	}
	return Money{
		Amount:   m.Amount - other.Amount,
		Currency: m.Currency,
	}
}

// Multiply multiplies the money amount by a factor
func (m Money) Multiply(factor float64) Money {
	return Money{
		Amount:   int64(float64(m.Amount) * factor),
		Currency: m.Currency,
	}
}

// IsZero returns true if the amount is zero
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive returns true if the amount is positive
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// IsNegative returns true if the amount is negative
func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// Formatted returns a formatted string representation
func (m Money) Formatted() string {
	switch m.Currency {
	case "USD":
		return fmt.Sprintf("$%.2f", m.ToFloat64())
	case "EUR":
		return fmt.Sprintf("€%.2f", m.ToFloat64())
	case "GBP":
		return fmt.Sprintf("£%.2f", m.ToFloat64())
	default:
		return fmt.Sprintf("%.2f %s", m.ToFloat64(), m.Currency)
	}
}

// IsCurrentlyActive returns true if the pricing rule is currently active
func (pr *PricingRule) IsCurrentlyActive() bool {
	if !pr.IsActive {
		return false
	}

	now := time.Now()
	if now.Before(pr.ValidFrom) {
		return false
	}

	if pr.ValidUntil != nil && now.After(*pr.ValidUntil) {
		return false
	}

	return true
}

// GetTimeMultiplier returns the time-based multiplier for the current hour
func (pr *PricingRule) GetTimeMultiplier(hour int) float64 {
	if multiplier, exists := pr.TimeMultipliers[fmt.Sprintf("%d", hour)]; exists {
		return multiplier
	}
	return 1.0 // Default multiplier
}

// GetDayMultiplier returns the day-based multiplier for the current day
func (pr *PricingRule) GetDayMultiplier(weekday time.Weekday) float64 {
	dayName := weekday.String()
	if multiplier, exists := pr.DayMultipliers[dayName]; exists {
		return multiplier
	}
	return 1.0 // Default multiplier
}

// SetTimeMultiplier sets a time-based multiplier
func (pr *PricingRule) SetTimeMultiplier(hour int, multiplier float64) {
	if pr.TimeMultipliers == nil {
		pr.TimeMultipliers = make(map[string]float64)
	}
	pr.TimeMultipliers[fmt.Sprintf("%d", hour)] = multiplier
	pr.UpdatedAt = time.Now()
}

// SetDayMultiplier sets a day-based multiplier
func (pr *PricingRule) SetDayMultiplier(weekday time.Weekday, multiplier float64) {
	if pr.DayMultipliers == nil {
		pr.DayMultipliers = make(map[string]float64)
	}
	pr.DayMultipliers[weekday.String()] = multiplier
	pr.UpdatedAt = time.Now()
}

// IsCurrentlyActive returns true if the surge pricing is currently active
func (sp *SurgePricing) IsCurrentlyActive() bool {
	if !sp.Active {
		return false
	}

	now := time.Now()
	return now.After(sp.StartsAt) && now.Before(sp.ExpiresAt)
}

// Extend extends the surge pricing duration
func (sp *SurgePricing) Extend(additionalMinutes int) {
	sp.ExpiresAt = sp.ExpiresAt.Add(time.Duration(additionalMinutes) * time.Minute)
	sp.UpdatedAt = time.Now()
}

// UpdateMultiplier updates the surge multiplier
func (sp *SurgePricing) UpdateMultiplier(multiplier float64) {
	sp.Multiplier = multiplier
	sp.UpdatedAt = time.Now()
}

// IsValid returns true if the promo code is currently valid
func (pc *PromoCode) IsValid() bool {
	if !pc.Active {
		return false
	}

	now := time.Now()
	if now.Before(pc.ValidFrom) || now.After(pc.ValidUntil) {
		return false
	}

	if pc.MaxUses != nil && pc.CurrentUses >= *pc.MaxUses {
		return false
	}

	return true
}

// CanBeUsedBy returns true if the promo code can be used by the specified user
func (pc *PromoCode) CanBeUsedBy(userID string, userUsageCount int, isFirstRide bool) bool {
	if !pc.IsValid() {
		return false
	}

	if userUsageCount >= pc.MaxUsesPerUser {
		return false
	}

	if pc.FirstRideOnly && !isFirstRide {
		return false
	}

	return true
}

// CalculateDiscount calculates the discount amount for a given trip amount
func (pc *PromoCode) CalculateDiscount(tripAmountCents int64) int64 {
	if !pc.IsValid() {
		return 0
	}

	// Check minimum trip amount
	if pc.MinTripAmountCents != nil && tripAmountCents < *pc.MinTripAmountCents {
		return 0
	}

	var discountCents int64

	switch pc.DiscountType {
	case "percentage":
		discountCents = int64(float64(tripAmountCents) * pc.DiscountValue / 100.0)
	case "fixed_amount":
		discountCents = int64(pc.DiscountValue * 100) // Convert to cents
	default:
		return 0
	}

	// Apply maximum discount limit
	if pc.MaxDiscountCents != nil && discountCents > *pc.MaxDiscountCents {
		discountCents = *pc.MaxDiscountCents
	}

	// Discount cannot exceed trip amount
	if discountCents > tripAmountCents {
		discountCents = tripAmountCents
	}

	return discountCents
}

// IncrementUsage increments the usage count
func (pc *PromoCode) IncrementUsage() {
	pc.CurrentUses++
	pc.UpdatedAt = time.Now()
}

// IsApplicableForVehicleType checks if the promo code is applicable for the vehicle type
func (pc *PromoCode) IsApplicableForVehicleType(vehicleType VehicleType) bool {
	if len(pc.ApplicableVehicleTypes) == 0 {
		return true // Applicable to all vehicle types
	}

	for _, applicableType := range pc.ApplicableVehicleTypes {
		if applicableType == vehicleType {
			return true
		}
	}
	return false
}

// IsApplicableForCity checks if the promo code is applicable for the city
func (pc *PromoCode) IsApplicableForCity(city string) bool {
	if len(pc.ApplicableCities) == 0 {
		return true // Applicable to all cities
	}

	for _, applicableCity := range pc.ApplicableCities {
		if applicableCity == city {
			return true
		}
	}
	return false
}
