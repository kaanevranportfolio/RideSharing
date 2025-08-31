# 💰 PRICING SERVICE - DEEP DIVE

## 📋 Overview
The **Pricing Service** is the financial brain of the rideshare platform, implementing sophisticated dynamic pricing algorithms that balance supply and demand while maximizing revenue and maintaining fairness. This service handles everything from base fare calculations to complex surge pricing algorithms.

---

## 🎯 Core Responsibilities

### **1. Dynamic Fare Calculation**
- **Base Fare Structure**: Different rates for vehicle types
- **Distance & Time Pricing**: Per-km and per-minute rates
- **Surge Pricing**: Supply/demand-based price adjustments
- **Real-time Updates**: Prices change based on current conditions

### **2. Surge Pricing Algorithm**
- **Demand Analysis**: Monitor ride requests vs available drivers
- **Geographic Zones**: Area-specific surge multipliers
- **Time-based Patterns**: Rush hour, events, weather-based adjustments
- **Maximum Caps**: Prevent excessive surge pricing

### **3. Promotions & Discounts**
- **First Ride Discounts**: New user acquisition
- **Loyalty Programs**: Repeat customer rewards
- **Promo Codes**: Marketing campaign support
- **Seasonal Offers**: Event-based promotions

### **4. Revenue Optimization**
- **Price Elasticity**: Balance price vs demand
- **Driver Incentives**: Ensure adequate supply
- **Market Competition**: Stay competitive with other platforms
- **Profit Margin Management**: Maintain business sustainability

---

## 🏗️ Architecture Components

### **Production Service Structure**
```go
type ProductionPricingService struct {
    redis       *redis.Client                // Real-time data caching
    logger      *logger.Logger               // Logging system
    metrics     *monitoring.MetricsCollector // Performance tracking
    config      *PricingConfig               // Pricing configuration
    surgeEngine *SurgeEngine                 // Surge pricing logic
    promoEngine *PromotionEngine             // Promotions system
}
```

### **Comprehensive Configuration System**
```go
type PricingConfig struct {
    BaseRates          map[string]*BaseRate `json:"base_rates"`          // Per vehicle type
    SurgeConfig        *SurgeConfig         `json:"surge_config"`        // Surge settings
    PromotionConfig    *PromotionConfig     `json:"promotion_config"`    // Promo settings
    DynamicPricing     bool                 `json:"dynamic_pricing"`     // Enable/disable
    MaxSurgeMultiplier float64              `json:"max_surge_multiplier"` // 5.0x max
    MinFare            float64              `json:"min_fare"`            // $3.00 minimum
    Currency           string               `json:"currency"`            // USD, EUR, etc.
}

type BaseRate struct {
    VehicleType       string  `json:"vehicle_type"`        // economy, standard, premium, luxury
    BaseFare          float64 `json:"base_fare"`           // $2.50 initial charge
    PerKmRate         float64 `json:"per_km_rate"`         // $1.20 per kilometer
    PerMinuteRate     float64 `json:"per_minute_rate"`     // $0.25 per minute
    MinimumFare       float64 `json:"minimum_fare"`        // $3.00 minimum trip
    CancellationFee   float64 `json:"cancellation_fee"`    // $2.00 cancel fee
    ServiceFeePercent float64 `json:"service_fee_percent"` // 20% platform fee
}
```

---

## 🧠 The Surge Pricing Algorithm

### **1. Real-time Supply & Demand Monitoring**
```go
type SurgeEngine struct {
    redis     *redis.Client
    config    *SurgeConfig
    logger    *logger.Logger
    geoHashes map[string]*SurgeArea // Geographic surge zones
}

type SurgeArea struct {
    GeoHash         string           `json:"geohash"`         // Unique area identifier
    Center          *models.Location `json:"center"`          // Geographic center
    ActiveDrivers   int              `json:"active_drivers"`   // Available drivers
    PendingRequests int              `json:"pending_requests"` // Waiting riders
    SurgeMultiplier float64          `json:"surge_multiplier"` // Current multiplier
    LastUpdated     time.Time        `json:"last_updated"`    // Update timestamp
}
```

### **2. Surge Calculation Algorithm**
```go
func (se *SurgeEngine) calculateSurgeMultiplier(area *SurgeArea) float64 {
    // Base case: no surge if enough drivers
    if area.ActiveDrivers == 0 && area.PendingRequests == 0 {
        return 1.0
    }
    
    // Calculate demand-to-supply ratio
    var demandRatio float64
    if area.ActiveDrivers == 0 {
        demandRatio = float64(area.PendingRequests) * 2.0 // High penalty for no drivers
    } else {
        demandRatio = float64(area.PendingRequests) / float64(area.ActiveDrivers)
    }
    
    // Surge thresholds
    lowThreshold := se.config.DemandThreshold    // 1.5 (1.5 requests per driver)
    highThreshold := lowThreshold * 2.0          // 3.0 (3 requests per driver)
    
    var surgeMultiplier float64
    
    switch {
    case demandRatio <= lowThreshold:
        // Normal pricing - no surge
        surgeMultiplier = 1.0
        
    case demandRatio <= highThreshold:
        // Linear surge increase
        surgeRange := highThreshold - lowThreshold
        surgeProgress := (demandRatio - lowThreshold) / surgeRange
        surgeMultiplier = 1.0 + (surgeProgress * (se.config.BaseSurgeMultiplier - 1.0))
        
    default:
        // High demand - exponential surge
        excessDemand := demandRatio - highThreshold
        exponentialFactor := math.Min(excessDemand/2.0, 3.0) // Cap exponential growth
        surgeMultiplier = se.config.BaseSurgeMultiplier * (1.0 + exponentialFactor)
    }
    
    // Apply maximum surge cap
    surgeMultiplier = math.Min(surgeMultiplier, se.config.MaxSurgeMultiplier)
    
    // Apply minimum surge (never below 1.0)
    surgeMultiplier = math.Max(surgeMultiplier, 1.0)
    
    return surgeMultiplier
}
```

### **3. Geographic Surge Zones**
```go
func (se *SurgeEngine) updateSurgeZones(ctx context.Context) error {
    // Get current ride requests and driver locations
    requests := se.getCurrentRequests(ctx)
    drivers := se.getAvailableDrivers(ctx)
    
    // Group by geohash areas (creates geographic zones)
    zoneData := make(map[string]*SurgeArea)
    
    // Count requests per zone
    for _, request := range requests {
        geohash := se.locationToGeoHash(request.PickupLocation)
        if zone, exists := zoneData[geohash]; exists {
            zone.PendingRequests++
        } else {
            zoneData[geohash] = &SurgeArea{
                GeoHash:         geohash,
                Center:          request.PickupLocation,
                PendingRequests: 1,
                ActiveDrivers:   0,
                LastUpdated:     time.Now(),
            }
        }
    }
    
    // Count drivers per zone
    for _, driver := range drivers {
        geohash := se.locationToGeoHash(driver.CurrentLocation)
        if zone, exists := zoneData[geohash]; exists {
            zone.ActiveDrivers++
        } else {
            zoneData[geohash] = &SurgeArea{
                GeoHash:         geohash,
                Center:          driver.CurrentLocation,
                PendingRequests: 0,
                ActiveDrivers:   1,
                LastUpdated:     time.Now(),
            }
        }
    }
    
    // Calculate surge for each zone
    for _, zone := range zoneData {
        zone.SurgeMultiplier = se.calculateSurgeMultiplier(zone)
        
        // Cache surge data
        se.cacheSurgeData(ctx, zone)
        
        // Log significant surge changes
        if zone.SurgeMultiplier > 1.5 {
            se.logger.Info("High surge detected", 
                "geohash", zone.GeoHash,
                "multiplier", zone.SurgeMultiplier,
                "drivers", zone.ActiveDrivers,
                "requests", zone.PendingRequests)
        }
    }
    
    se.geoHashes = zoneData
    return nil
}
```

---

## 💰 Advanced Fare Calculation

### **1. Complete Fare Calculation Process**
```go
func (s *ProductionPricingService) CalculateFare(ctx context.Context, request *FareCalculationRequest) (*FareCalculationResponse, error) {
    startTime := time.Now()
    
    // 1. Get base rate for vehicle type
    baseRate, exists := s.config.BaseRates[request.VehicleType]
    if !exists {
        return nil, fmt.Errorf("unsupported vehicle type: %s", request.VehicleType)
    }
    
    // 2. Calculate base fare components
    baseFare := baseRate.BaseFare
    distanceFare := request.DistanceKm * baseRate.PerKmRate
    timeFare := request.DurationMinutes * baseRate.PerMinuteRate
    
    subtotal := baseFare + distanceFare + timeFare
    
    // 3. Apply minimum fare
    if subtotal < baseRate.MinimumFare {
        subtotal = baseRate.MinimumFare
    }
    
    // 4. Get surge multiplier
    surgeMultiplier := s.getSurgeMultiplier(ctx, request.PickupLocation, request.VehicleType)
    
    // 5. Apply surge pricing
    fareAfterSurge := subtotal * surgeMultiplier
    
    // 6. Apply promotions and discounts
    promotionDiscount, err := s.promoEngine.calculateDiscount(ctx, request)
    if err != nil {
        s.logger.Warn("Failed to calculate promotion discount", "error", err)
        promotionDiscount = 0.0
    }
    
    discountAmount := fareAfterSurge * promotionDiscount
    fareAfterDiscount := fareAfterSurge - discountAmount
    
    // 7. Calculate service fee
    serviceFee := fareAfterDiscount * baseRate.ServiceFeePercent
    
    // 8. Calculate driver earnings
    driverEarnings := fareAfterDiscount - serviceFee
    
    // 9. Apply final minimum fare check
    totalFare := math.Max(fareAfterDiscount, s.config.MinFare)
    
    // 10. Round to nearest cent
    totalFare = math.Round(totalFare*100) / 100
    
    response := &FareCalculationResponse{
        TotalFare:       totalFare,
        SurgeMultiplier: surgeMultiplier,
        PromotionDiscount: discountAmount,
        FareBreakdown: &FareBreakdown{
            BaseFare:      baseFare,
            DistanceFare:  distanceFare,
            TimeFare:      timeFare,
            Subtotal:      subtotal,
            SurgeAmount:   fareAfterSurge - subtotal,
            DiscountAmount: discountAmount,
            ServiceFee:    serviceFee,
            DriverEarnings: driverEarnings,
        },
        Currency:         s.config.Currency,
        CalculatedAt:     time.Now(),
        CalculationTime:  time.Since(startTime),
    }
    
    // Record metrics
    s.metrics.RecordFareCalculation(request.VehicleType, totalFare, surgeMultiplier)
    
    return response, nil
}
```

### **2. Time-based Pricing Adjustments**
```go
func (s *ProductionPricingService) getTimeBasedMultiplier(timeOfDay time.Time) float64 {
    hour := timeOfDay.Hour()
    weekday := timeOfDay.Weekday()
    
    // Weekend pricing
    if weekday == time.Saturday || weekday == time.Sunday {
        return 1.1 // 10% weekend premium
    }
    
    // Rush hour pricing (weekdays)
    switch {
    case hour >= 7 && hour <= 9:   // Morning rush
        return 1.15
    case hour >= 17 && hour <= 19: // Evening rush
        return 1.2
    case hour >= 22 || hour <= 5:  // Late night/early morning
        return 1.25
    default:
        return 1.0
    }
}
```

---

## 🎁 Promotion Engine

### **1. Discount Calculation System**
```go
type PromotionEngine struct {
    config *PromotionConfig
    logger *logger.Logger
}

func (pe *PromotionEngine) calculateDiscount(ctx context.Context, request *FareCalculationRequest) (float64, error) {
    if !pe.config.Enabled {
        return 0.0, nil
    }
    
    totalDiscount := 0.0
    
    // 1. First ride discount
    if pe.isFirstRide(ctx, request.UserID) {
        firstRideDiscount := pe.config.FirstRideDiscountPercent
        totalDiscount = math.Max(totalDiscount, firstRideDiscount)
    }
    
    // 2. Loyalty discount
    loyaltyDiscount := pe.calculateLoyaltyDiscount(ctx, request.UserID)
    totalDiscount = math.Max(totalDiscount, loyaltyDiscount)
    
    // 3. Promo code discount
    promoDiscount := pe.getPromoCodeDiscount(ctx, request.PromoCode)
    totalDiscount += promoDiscount
    
    // 4. Time-based promotions
    timeBasedDiscount := pe.getTimeBasedPromotion(request.TimeOfDay)
    totalDiscount += timeBasedDiscount
    
    // Cap total discount
    totalDiscount = math.Min(totalDiscount, pe.config.MaxDiscountPercent)
    
    return totalDiscount, nil
}
```

### **2. Loyalty Program Implementation**
```go
func (pe *PromotionEngine) calculateLoyaltyDiscount(ctx context.Context, userID string) float64 {
    tripCount := pe.getUserTripCount(ctx, userID)
    
    // Loyalty tiers based on trip count
    for trips, discount := range pe.config.LoyaltyDiscountTiers {
        if tripCount >= trips {
            return discount
        }
    }
    
    return 0.0
}

// Example loyalty tiers:
// 10 trips -> 5% discount
// 25 trips -> 8% discount  
// 50 trips -> 10% discount
// 100 trips -> 15% discount
```

---

## 📊 Real-time Price Updates

### **1. Price Streaming System**
```go
func (s *ProductionPricingService) StreamPriceUpdates(ctx context.Context, location *models.Location, vehicleType string) (<-chan *PriceUpdate, error) {
    updateChan := make(chan *PriceUpdate, 10)
    
    go func() {
        defer close(updateChan)
        
        ticker := time.NewTicker(30 * time.Second) // Update every 30 seconds
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // Calculate current fare estimate
                estimateRequest := &FareEstimateRequest{
                    PickupLocation:  location,
                    VehicleType:     vehicleType,
                    TimeOfDay:       time.Now(),
                }
                
                estimate, err := s.EstimateFare(ctx, estimateRequest)
                if err != nil {
                    continue
                }
                
                update := &PriceUpdate{
                    Location:        location,
                    VehicleType:     vehicleType,
                    EstimatedFare:   estimate.EstimatedFare,
                    SurgeMultiplier: estimate.SurgeMultiplier,
                    UpdatedAt:       time.Now(),
                }
                
                select {
                case updateChan <- update:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return updateChan, nil
}
```

### **2. Surge Notification System**
```go
func (s *ProductionPricingService) notifySignificantSurgeChanges(oldMultiplier, newMultiplier float64, location *models.Location) {
    // Notify if surge increases significantly
    if newMultiplier >= 2.0 && newMultiplier > oldMultiplier*1.2 {
        notification := &SurgeNotification{
            Location:        location,
            SurgeMultiplier: newMultiplier,
            Message:        fmt.Sprintf("High demand! Prices are %.1fx higher than normal", newMultiplier),
            Severity:       "high",
            Timestamp:      time.Now(),
        }
        
        // Send to notification service
        s.sendSurgeNotification(notification)
    }
}
```

---

## 🔧 Integration Points

### **1. With Matching Service**
```go
// Matching service gets fare estimates for user display
fareEstimate, err := pricingClient.EstimateFare(ctx, &pricing.FareEstimateRequest{
    PickupLocation:      request.PickupLocation,
    DestinationLocation: request.DestinationLocation,
    VehicleType:         request.PreferredVehicleType,
    TimeOfDay:          time.Now(),
})
```

### **2. With Trip Service**
```go
// Trip service calculates final fare when trip completes
finalFare, err := pricingClient.CalculateFare(ctx, &pricing.FareCalculationRequest{
    TripID:              trip.ID,
    UserID:              trip.RiderID,
    VehicleType:         trip.VehicleType,
    DistanceKm:          trip.DistanceKm,
    DurationMinutes:     trip.DurationMinutes,
    PickupLocation:      trip.PickupLocation,
    DestinationLocation: trip.DestinationLocation,
    TimeOfDay:          trip.StartedAt,
})
```

### **3. With Payment Service**
```go
// Payment service processes payment using calculated fare
paymentRequest := &payment.PaymentRequest{
    TripID:      trip.ID,
    UserID:      trip.RiderID,
    AmountCents: int64(finalFare.TotalFare * 100),
    Currency:    finalFare.Currency,
}
```

---

## 📊 Performance & Analytics

### **1. Price Analytics**
```go
func (s *ProductionPricingService) generatePricingAnalytics(ctx context.Context, timeRange time.Duration) (*PricingAnalytics, error) {
    analytics := &PricingAnalytics{
        TimeRange: timeRange,
        StartTime: time.Now().Add(-timeRange),
        EndTime:   time.Now(),
    }
    
    // Average surge multipliers by time of day
    analytics.AvgSurgeByHour = s.calculateAvgSurgeByHour(ctx, timeRange)
    
    // Revenue by vehicle type
    analytics.RevenueByVehicleType = s.calculateRevenueByVehicleType(ctx, timeRange)
    
    // Promotion usage statistics
    analytics.PromotionUsage = s.calculatePromotionUsage(ctx, timeRange)
    
    // Price elasticity metrics
    analytics.PriceElasticity = s.calculatePriceElasticity(ctx, timeRange)
    
    return analytics, nil
}
```

### **2. A/B Testing for Pricing**
```go
func (s *ProductionPricingService) applyPricingExperiment(ctx context.Context, userID string, baseFare float64) float64 {
    // Get user's experiment group
    experimentGroup := s.getUserExperimentGroup(userID)
    
    switch experimentGroup {
    case "control":
        return baseFare
    case "premium_pricing":
        return baseFare * 1.05 // 5% higher
    case "discount_pricing":
        return baseFare * 0.95 // 5% lower
    default:
        return baseFare
    }
}
```

---

## 🌟 Advanced Features

### **1. Machine Learning Price Optimization**
```go
type PriceOptimizer struct {
    model MLModel // Machine learning model
}

func (po *PriceOptimizer) optimizePrice(ctx context.Context, request *OptimizationRequest) (*OptimizedPrice, error) {
    features := &PricingFeatures{
        TimeOfDay:       request.TimeOfDay.Hour(),
        DayOfWeek:       int(request.TimeOfDay.Weekday()),
        Weather:         request.Weather,
        LocalEvents:     request.LocalEvents,
        HistoricalDemand: request.HistoricalDemand,
        CompetitorPrices: request.CompetitorPrices,
    }
    
    // Use ML model to predict optimal price
    optimizedMultiplier, err := po.model.Predict(features)
    if err != nil {
        return nil, err
    }
    
    return &OptimizedPrice{
        Multiplier:   optimizedMultiplier,
        Confidence:   po.model.GetConfidence(),
        Explanation:  po.model.GetExplanation(),
    }, nil
}
```

### **2. Event-based Surge Pricing**
```go
func (s *ProductionPricingService) handleSpecialEvent(ctx context.Context, event *SpecialEvent) error {
    // Examples: concerts, sports games, airport delays
    eventMultiplier := s.calculateEventMultiplier(event)
    
    // Apply surge to affected areas
    for _, location := range event.AffectedAreas {
        geohash := s.locationToGeoHash(location)
        s.applyEventSurge(geohash, eventMultiplier, event.Duration)
    }
    
    return nil
}
```

---

## 🎯 Business Impact

### **1. Revenue Optimization**
- **Dynamic Pricing**: Increases revenue by 15-25% compared to fixed pricing
- **Surge Pricing**: Balances supply and demand, reducing wait times
- **Promotions**: Drives user acquisition and retention

### **2. Market Positioning**
- **Competitive Pricing**: Automated price matching with competitors
- **Value Perception**: Transparent pricing builds trust
- **Flexibility**: Quick adaptation to market changes

### **3. Driver Economics**
- **Earnings Optimization**: Higher surge periods increase driver income
- **Supply Incentives**: Pricing encourages drivers during high demand
- **Fair Distribution**: Ensures all drivers get earning opportunities

---

This Pricing Service represents a **sophisticated financial engine** that balances multiple competing objectives: maximizing revenue, ensuring fairness, maintaining competitiveness, and optimizing supply/demand dynamics. Its real-time algorithms and machine learning capabilities enable the platform to adapt quickly to changing market conditions while providing predictable, transparent pricing to users.
