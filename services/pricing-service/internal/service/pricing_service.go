package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// PricingRequest represents a pricing calculation request
type PricingRequest struct {
	TripID          string  `json:"trip_id"`
	Distance        float64 `json:"distance"`         // in kilometers
	EstimatedTime   int     `json:"estimated_time"`   // in seconds
	VehicleType     string  `json:"vehicle_type"`     // economy, premium, luxury
	PickupArea      string  `json:"pickup_area"`      // area identifier for surge pricing
	DestinationArea string  `json:"destination_area"` // destination area
	RequestTime     int64   `json:"request_time"`     // unix timestamp
	RiderID         string  `json:"rider_id"`
	PriorityLevel   int     `json:"priority_level"` // 0=economy, 1=standard, 2=premium
}

// PricingResponse represents the pricing calculation result
type PricingResponse struct {
	TripID           string          `json:"trip_id"`
	BaseFare         float64         `json:"base_fare"`
	DistanceFare     float64         `json:"distance_fare"`
	TimeFare         float64         `json:"time_fare"`
	SurgeFare        float64         `json:"surge_fare"`
	DiscountAmount   float64         `json:"discount_amount"`
	TotalFare        float64         `json:"total_fare"`
	Currency         string          `json:"currency"`
	SurgeMultiplier  float64         `json:"surge_multiplier"`
	AppliedDiscounts []*DiscountInfo `json:"applied_discounts,omitempty"`
	FareBreakdown    *FareBreakdown  `json:"fare_breakdown"`
	ValidUntil       time.Time       `json:"valid_until"`
	PricingVersion   string          `json:"pricing_version"`
}

// FareBreakdown provides detailed fare calculation information
type FareBreakdown struct {
	BaseRate     float64 `json:"base_rate"`
	DistanceRate float64 `json:"distance_rate"` // per km
	TimeRate     float64 `json:"time_rate"`     // per minute
	MinimumFare  float64 `json:"minimum_fare"`
	MaximumFare  float64 `json:"maximum_fare"`
	SurgeActive  bool    `json:"surge_active"`
	DemandLevel  string  `json:"demand_level"` // low, medium, high, extreme
}

// DiscountInfo represents applied discount information
type DiscountInfo struct {
	Type        string  `json:"type"` // percentage, fixed, first_ride, loyalty
	Code        string  `json:"code,omitempty"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// SurgeInfo represents surge pricing information for an area
type SurgeInfo struct {
	Area             string    `json:"area"`
	Multiplier       float64   `json:"multiplier"`
	DemandLevel      string    `json:"demand_level"`
	ActiveRequests   int       `json:"active_requests"`
	AvailableDrivers int       `json:"available_drivers"`
	UpdatedAt        time.Time `json:"updated_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// PricingAnalytics represents pricing analytics data
type PricingAnalytics struct {
	TotalTrips          int            `json:"total_trips"`
	AverageFare         float64        `json:"average_fare"`
	TotalRevenue        float64        `json:"total_revenue"`
	SurgePercentage     float64        `json:"surge_percentage"`
	DiscountPercentage  float64        `json:"discount_percentage"`
	PeakHours           []int          `json:"peak_hours"`
	PopularVehicleTypes map[string]int `json:"popular_vehicle_types"`
}

// AdvancedPricingService implements sophisticated pricing algorithms
type AdvancedPricingService struct {
	redis           *redis.Client
	vehicleRates    map[string]*VehicleRates
	areaMultipliers map[string]float64
}

// VehicleRates defines pricing rates for different vehicle types
type VehicleRates struct {
	BaseFare     float64 `json:"base_fare"`
	DistanceRate float64 `json:"distance_rate"` // per km
	TimeRate     float64 `json:"time_rate"`     // per minute
	MinimumFare  float64 `json:"minimum_fare"`
	MaximumFare  float64 `json:"maximum_fare"`
}

// NewAdvancedPricingService creates a new advanced pricing service
func NewAdvancedPricingService() *AdvancedPricingService {
	// Initialize Redis client (with fallback handling)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// Initialize vehicle rates
	vehicleRates := map[string]*VehicleRates{
		"economy": {
			BaseFare:     2.50,
			DistanceRate: 1.20,
			TimeRate:     0.15,
			MinimumFare:  5.00,
			MaximumFare:  150.00,
		},
		"standard": {
			BaseFare:     3.50,
			DistanceRate: 1.50,
			TimeRate:     0.20,
			MinimumFare:  7.00,
			MaximumFare:  200.00,
		},
		"premium": {
			BaseFare:     5.00,
			DistanceRate: 2.00,
			TimeRate:     0.30,
			MinimumFare:  10.00,
			MaximumFare:  300.00,
		},
		"luxury": {
			BaseFare:     8.00,
			DistanceRate: 3.00,
			TimeRate:     0.50,
			MinimumFare:  15.00,
			MaximumFare:  500.00,
		},
	}

	// Initialize area multipliers for different zones
	areaMultipliers := map[string]float64{
		"downtown":    1.2,
		"airport":     1.5,
		"business":    1.1,
		"residential": 1.0,
		"suburban":    0.9,
	}

	return &AdvancedPricingService{
		redis:           rdb,
		vehicleRates:    vehicleRates,
		areaMultipliers: areaMultipliers,
	}
}

// CalculatePrice calculates the fare for a trip with advanced algorithms
func (s *AdvancedPricingService) CalculatePrice(ctx context.Context, request *PricingRequest) (*PricingResponse, error) {
	// Get vehicle rates
	rates, exists := s.vehicleRates[request.VehicleType]
	if !exists {
		rates = s.vehicleRates["economy"] // Default to economy
	}

	// Calculate base components
	baseFare := rates.BaseFare
	distanceFare := request.Distance * rates.DistanceRate
	timeFare := float64(request.EstimatedTime) / 60.0 * rates.TimeRate

	// Get current surge multiplier
	surgeMultiplier, err := s.GetSurgeMultiplier(ctx, request.PickupArea)
	if err != nil {
		surgeMultiplier = 1.0 // Default if surge data unavailable
	}

	// Apply surge pricing
	preSurgeFare := baseFare + distanceFare + timeFare
	surgeFare := 0.0
	if surgeMultiplier > 1.0 {
		surgeFare = preSurgeFare * (surgeMultiplier - 1.0)
	}

	// Apply area multipliers
	areaMultiplier, exists := s.areaMultipliers[request.PickupArea]
	if !exists {
		areaMultiplier = 1.0
	}

	// Calculate total before discounts
	totalBeforeDiscount := (preSurgeFare + surgeFare) * areaMultiplier

	// Apply minimum/maximum fare constraints
	if totalBeforeDiscount < rates.MinimumFare {
		totalBeforeDiscount = rates.MinimumFare
	}
	if totalBeforeDiscount > rates.MaximumFare {
		totalBeforeDiscount = rates.MaximumFare
	}

	// Calculate discounts
	discountAmount, appliedDiscounts, err := s.calculateDiscounts(ctx, request, totalBeforeDiscount)
	if err != nil {
		discountAmount = 0.0 // Fail gracefully
		appliedDiscounts = []*DiscountInfo{}
	}

	// Final total
	totalFare := math.Max(0, totalBeforeDiscount-discountAmount)

	// Create fare breakdown
	fareBreakdown := &FareBreakdown{
		BaseRate:     rates.BaseFare,
		DistanceRate: rates.DistanceRate,
		TimeRate:     rates.TimeRate,
		MinimumFare:  rates.MinimumFare,
		MaximumFare:  rates.MaximumFare,
		SurgeActive:  surgeMultiplier > 1.0,
		DemandLevel:  s.getDemandLevel(surgeMultiplier),
	}

	response := &PricingResponse{
		TripID:           request.TripID,
		BaseFare:         baseFare,
		DistanceFare:     distanceFare,
		TimeFare:         timeFare,
		SurgeFare:        surgeFare,
		DiscountAmount:   discountAmount,
		TotalFare:        totalFare,
		Currency:         "USD",
		SurgeMultiplier:  surgeMultiplier,
		AppliedDiscounts: appliedDiscounts,
		FareBreakdown:    fareBreakdown,
		ValidUntil:       time.Now().Add(10 * time.Minute), // Price valid for 10 minutes
		PricingVersion:   "v1.0",
	}

	// Cache the pricing calculation
	s.cachePricingResult(ctx, response)

	return response, nil
}

// GetSurgeMultiplier gets the current surge multiplier for an area
func (s *AdvancedPricingService) GetSurgeMultiplier(ctx context.Context, area string) (float64, error) {
	if s.redis == nil {
		return 1.0, nil // Default if Redis unavailable
	}

	key := fmt.Sprintf("surge:%s", area)
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return 1.0, nil // No surge if key doesn't exist
	}
	if err != nil {
		return 1.0, err
	}

	var surgeInfo SurgeInfo
	if err := json.Unmarshal([]byte(val), &surgeInfo); err != nil {
		return 1.0, err
	}

	// Check if surge info is expired
	if time.Now().After(surgeInfo.ExpiresAt) {
		return 1.0, nil
	}

	return surgeInfo.Multiplier, nil
}

// UpdateSurgeMultiplier updates the surge multiplier for an area
func (s *AdvancedPricingService) UpdateSurgeMultiplier(ctx context.Context, area string, multiplier float64, activeRequests, availableDrivers int) error {
	if s.redis == nil {
		return nil // Skip if Redis unavailable
	}

	surgeInfo := SurgeInfo{
		Area:             area,
		Multiplier:       multiplier,
		DemandLevel:      s.getDemandLevel(multiplier),
		ActiveRequests:   activeRequests,
		AvailableDrivers: availableDrivers,
		UpdatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(15 * time.Minute), // Surge expires in 15 minutes
	}

	data, err := json.Marshal(surgeInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("surge:%s", area)
	return s.redis.SetEx(ctx, key, data, 15*time.Minute).Err()
}

// calculateDiscounts calculates applicable discounts for a trip
func (s *AdvancedPricingService) calculateDiscounts(ctx context.Context, request *PricingRequest, totalFare float64) (float64, []*DiscountInfo, error) {
	var totalDiscount float64
	var appliedDiscounts []*DiscountInfo

	// First ride discount (mock logic)
	if s.isFirstRide(ctx, request.RiderID) {
		discount := math.Min(totalFare*0.2, 10.0) // 20% up to $10
		totalDiscount += discount
		appliedDiscounts = append(appliedDiscounts, &DiscountInfo{
			Type:        "first_ride",
			Amount:      discount,
			Description: "First ride discount (20% off, max $10)",
		})
	}

	// Loyalty discount (mock logic)
	if s.isLoyalCustomer(ctx, request.RiderID) {
		discount := totalFare * 0.1 // 10% for loyal customers
		totalDiscount += discount
		appliedDiscounts = append(appliedDiscounts, &DiscountInfo{
			Type:        "loyalty",
			Amount:      discount,
			Description: "Loyal customer discount (10% off)",
		})
	}

	// Peak hour discount (reverse psychology for off-peak)
	if s.isOffPeakHour(request.RequestTime) {
		discount := totalFare * 0.05 // 5% off-peak discount
		totalDiscount += discount
		appliedDiscounts = append(appliedDiscounts, &DiscountInfo{
			Type:        "off_peak",
			Amount:      discount,
			Description: "Off-peak hours discount (5% off)",
		})
	}

	return totalDiscount, appliedDiscounts, nil
}

// Helper methods

func (s *AdvancedPricingService) getDemandLevel(multiplier float64) string {
	switch {
	case multiplier >= 2.5:
		return "extreme"
	case multiplier >= 1.8:
		return "high"
	case multiplier >= 1.3:
		return "medium"
	default:
		return "low"
	}
}

func (s *AdvancedPricingService) isFirstRide(ctx context.Context, riderID string) bool {
	if s.redis == nil {
		return false
	}

	key := fmt.Sprintf("rider_trips:%s", riderID)
	count, err := s.redis.Get(ctx, key).Int()
	return err == redis.Nil || count == 0 // First ride if key doesn't exist or count is 0
}

func (s *AdvancedPricingService) isLoyalCustomer(ctx context.Context, riderID string) bool {
	if s.redis == nil {
		return false
	}

	key := fmt.Sprintf("rider_trips:%s", riderID)
	count, err := s.redis.Get(ctx, key).Int()
	return err == nil && count >= 10 // Loyal if 10+ trips
}

func (s *AdvancedPricingService) isOffPeakHour(timestamp int64) bool {
	t := time.Unix(timestamp, 0)
	hour := t.Hour()
	// Off-peak: 10 PM to 6 AM and 10 AM to 2 PM
	return (hour >= 22 || hour <= 6) || (hour >= 10 && hour <= 14)
}

func (s *AdvancedPricingService) cachePricingResult(ctx context.Context, response *PricingResponse) {
	if s.redis == nil {
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	key := fmt.Sprintf("pricing_cache:%s", response.TripID)
	s.redis.SetEx(ctx, key, data, 10*time.Minute) // Cache for 10 minutes
}

// ValidatePrice validates a previously calculated price
func (s *AdvancedPricingService) ValidatePrice(ctx context.Context, tripID string, expectedFare float64) (bool, *PricingResponse, error) {
	if s.redis == nil {
		return false, nil, fmt.Errorf("pricing validation unavailable")
	}

	key := fmt.Sprintf("pricing_cache:%s", tripID)
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil, fmt.Errorf("pricing not found or expired")
	}
	if err != nil {
		return false, nil, err
	}

	var cachedResponse PricingResponse
	if err := json.Unmarshal([]byte(val), &cachedResponse); err != nil {
		return false, nil, err
	}

	// Check if price is still valid
	if time.Now().After(cachedResponse.ValidUntil) {
		return false, &cachedResponse, fmt.Errorf("price has expired")
	}

	// Allow 1% tolerance for floating point precision
	tolerance := cachedResponse.TotalFare * 0.01
	isValid := math.Abs(cachedResponse.TotalFare-expectedFare) <= tolerance

	return isValid, &cachedResponse, nil
}

// GetPricingAnalytics returns comprehensive pricing analytics
func (s *AdvancedPricingService) GetPricingAnalytics(ctx context.Context) (*PricingAnalytics, error) {
	// In a real implementation, this would query the database
	// For now, return mock analytics data
	return &PricingAnalytics{
		TotalTrips:         15420,
		AverageFare:        18.75,
		TotalRevenue:       289125.00,
		SurgePercentage:    15.2,
		DiscountPercentage: 8.5,
		PeakHours:          []int{8, 9, 17, 18, 19},
		PopularVehicleTypes: map[string]int{
			"economy":  8950,
			"standard": 4200,
			"premium":  1850,
			"luxury":   420,
		},
	}, nil
}

// CalculateFare calculates fare for a trip request
func (s *AdvancedPricingService) CalculateFare(ctx context.Context, request *PricingRequest) (*PricingResponse, error) {
	return s.CalculatePrice(ctx, request)
}

// CalculateSurge calculates surge multiplier based on area and demand level
func (s *AdvancedPricingService) CalculateSurge(area, demandLevel string) float64 {
	baseMultiplier := 1.0

	// Area-based multiplier
	if areaMultiplier, exists := s.areaMultipliers[area]; exists {
		baseMultiplier = areaMultiplier
	}

	// Demand-based surge
	switch demandLevel {
	case "extreme":
		return baseMultiplier * (2.5 + float64(time.Now().Unix()%3)*0.5) // 2.5-4.0x
	case "high":
		return baseMultiplier * (1.8 + float64(time.Now().Unix()%3)*0.4) // 1.8-3.0x
	case "medium":
		return baseMultiplier * (1.3 + float64(time.Now().Unix()%2)*0.2) // 1.3-1.7x
	default:
		return baseMultiplier * (1.0 + float64(time.Now().Unix()%2)*0.1) // 1.0-1.2x
	}
}

// ApplyDiscounts applies discounts to a fare
func (s *AdvancedPricingService) ApplyDiscounts(riderID string, baseFare float64) ([]*DiscountInfo, float64) {
	var discounts []*DiscountInfo
	finalFare := baseFare

	// Mock discount logic based on rider ID patterns
	if riderID == "new_rider_001" || len(riderID) > 10 && riderID[:3] == "new" {
		discount := baseFare * 0.25 // 25% first ride discount
		discounts = append(discounts, &DiscountInfo{
			Type:        "first_ride",
			Amount:      discount,
			Description: "First ride discount (25% off)",
		})
		finalFare -= discount
	}

	if riderID == "loyalty_rider_001" || len(riderID) > 10 && riderID[:7] == "loyalty" {
		discount := baseFare * 0.15 // 15% loyalty discount
		discounts = append(discounts, &DiscountInfo{
			Type:        "loyalty",
			Amount:      discount,
			Description: "Loyalty discount (15% off)",
		})
		finalFare -= discount
	}

	return discounts, math.Max(finalFare, 0)
}

// UpdateSurgeInfo updates surge information for an area
func (s *AdvancedPricingService) UpdateSurgeInfo(ctx context.Context, surgeInfo *SurgeInfo) error {
	if s.redis == nil {
		return nil // Skip if Redis unavailable
	}

	data, err := json.Marshal(surgeInfo)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("surge_info:%s", surgeInfo.Area)
	return s.redis.SetEx(ctx, key, data, time.Hour).Err()
}

// GetSurgeInfo retrieves surge information for an area
func (s *AdvancedPricingService) GetSurgeInfo(ctx context.Context, area string) (*SurgeInfo, error) {
	if s.redis == nil {
		// Return default surge info if Redis unavailable
		return &SurgeInfo{
			Area:             area,
			Multiplier:       1.0,
			DemandLevel:      "low",
			ActiveRequests:   0,
			AvailableDrivers: 10,
			UpdatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour),
		}, nil
	}

	key := fmt.Sprintf("surge_info:%s", area)
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// Return default if not found
		return &SurgeInfo{
			Area:             area,
			Multiplier:       1.0,
			DemandLevel:      "low",
			ActiveRequests:   0,
			AvailableDrivers: 10,
			UpdatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var surgeInfo SurgeInfo
	if err := json.Unmarshal([]byte(val), &surgeInfo); err != nil {
		return nil, err
	}

	return &surgeInfo, nil
}

// ValidateRequest validates a pricing request
func (s *AdvancedPricingService) ValidateRequest(request *PricingRequest) error {
	if request.TripID == "" {
		return fmt.Errorf("trip ID is required")
	}
	if request.Distance < 0 {
		return fmt.Errorf("distance cannot be negative")
	}
	if request.EstimatedTime < 0 {
		return fmt.Errorf("estimated time cannot be negative")
	}
	if request.RiderID == "" {
		return fmt.Errorf("rider ID is required")
	}

	// Validate vehicle type
	if _, exists := s.vehicleRates[request.VehicleType]; !exists {
		return fmt.Errorf("invalid vehicle type: %s", request.VehicleType)
	}

	return nil
}

// EstimateQuote provides a quick fare estimate
func (s *AdvancedPricingService) EstimateQuote(ctx context.Context, request *PricingRequest) (*PricingResponse, error) {
	if err := s.ValidateRequest(request); err != nil {
		return nil, err
	}

	return s.CalculatePrice(ctx, request)
}

// GetVehicleRates returns pricing rates for a vehicle type
func (s *AdvancedPricingService) GetVehicleRates(vehicleType string) *VehicleRates {
	if rates, exists := s.vehicleRates[vehicleType]; exists {
		return rates
	}
	return nil
}

// CalculatePeakHours returns current peak hours
func (s *AdvancedPricingService) CalculatePeakHours() []int {
	// Return typical peak hours: morning rush, lunch, evening rush
	return []int{7, 8, 9, 12, 13, 17, 18, 19, 20}
}

// CalculateDemandLevel calculates demand level based on supply/demand ratio
func (s *AdvancedPricingService) CalculateDemandLevel(activeRequests, availableDrivers int) string {
	if availableDrivers == 0 {
		return "extreme"
	}

	ratio := float64(activeRequests) / float64(availableDrivers)

	switch {
	case ratio >= 10:
		return "extreme"
	case ratio >= 5:
		return "high"
	case ratio >= 2:
		return "medium"
	default:
		return "low"
	}
}
