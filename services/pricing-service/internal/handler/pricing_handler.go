package handler

import (
	"net/http"
	"time"

	"pricing-service/internal/service"

	"github.com/gin-gonic/gin"
)

// PricingHandler handles HTTP requests for pricing operations
type PricingHandler struct {
	pricingService *service.AdvancedPricingService
}

// NewPricingHandler creates a new pricing handler
func NewPricingHandler(pricingService *service.AdvancedPricingService) *PricingHandler {
	return &PricingHandler{
		pricingService: pricingService,
	}
}

// CalculatePrice handles price calculation requests
func (h *PricingHandler) CalculatePrice(c *gin.Context) {
	var request service.PricingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Set request time if not provided
	if request.RequestTime == 0 {
		request.RequestTime = time.Now().Unix()
	}

	// Validate required fields
	if request.Distance <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_distance",
			"message": "Distance must be greater than 0",
		})
		return
	}

	if request.EstimatedTime <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_time",
			"message": "Estimated time must be greater than 0",
		})
		return
	}

	response, err := h.pricingService.CalculatePrice(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "calculation_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSurgeMultiplier handles surge multiplier requests
func (h *PricingHandler) GetSurgeMultiplier(c *gin.Context) {
	area := c.Param("area")
	if area == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_area",
			"message": "Area parameter is required",
		})
		return
	}

	multiplier, err := h.pricingService.GetSurgeMultiplier(c.Request.Context(), area)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "surge_lookup_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"area":             area,
		"surge_multiplier": multiplier,
		"surge_active":     multiplier > 1.0,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// UpdateSurgeMultiplier handles surge multiplier update requests
func (h *PricingHandler) UpdateSurgeMultiplier(c *gin.Context) {
	var request struct {
		Area             string  `json:"area" binding:"required"`
		Multiplier       float64 `json:"multiplier" binding:"required"`
		ActiveRequests   int     `json:"active_requests"`
		AvailableDrivers int     `json:"available_drivers"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Validate multiplier range
	if request.Multiplier < 1.0 || request.Multiplier > 5.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_multiplier",
			"message": "Surge multiplier must be between 1.0 and 5.0",
		})
		return
	}

	err := h.pricingService.UpdateSurgeMultiplier(
		c.Request.Context(),
		request.Area,
		request.Multiplier,
		request.ActiveRequests,
		request.AvailableDrivers,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "surge_update_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Surge multiplier updated successfully",
		"area":             request.Area,
		"surge_multiplier": request.Multiplier,
		"updated_at":       time.Now().Format(time.RFC3339),
	})
}

// ApplyDiscount handles discount application requests
func (h *PricingHandler) ApplyDiscount(c *gin.Context) {
	var request struct {
		TripID       string  `json:"trip_id" binding:"required"`
		DiscountCode string  `json:"discount_code"`
		DiscountType string  `json:"discount_type"` // percentage, fixed, promo
		Amount       float64 `json:"amount"`
		Description  string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// For now, return a mock response
	// In a real implementation, this would apply the discount to the trip
	c.JSON(http.StatusOK, gin.H{
		"message":       "Discount applied successfully",
		"trip_id":       request.TripID,
		"discount_code": request.DiscountCode,
		"discount_type": request.DiscountType,
		"amount":        request.Amount,
		"applied_at":    time.Now().Format(time.RFC3339),
	})
}

// GetPricingHistory handles pricing history requests
func (h *PricingHandler) GetPricingHistory(c *gin.Context) {
	tripID := c.Param("trip_id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_trip_id",
			"message": "Trip ID parameter is required",
		})
		return
	}

	// For now, return mock pricing history
	// In a real implementation, this would query the database
	c.JSON(http.StatusOK, gin.H{
		"trip_id": tripID,
		"history": []gin.H{
			{
				"timestamp":        time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				"action":           "price_calculated",
				"base_fare":        3.50,
				"distance_fare":    8.40,
				"time_fare":        2.25,
				"surge_multiplier": 1.0,
				"total_fare":       14.15,
			},
			{
				"timestamp":        time.Now().Add(-25 * time.Minute).Format(time.RFC3339),
				"action":           "surge_applied",
				"surge_multiplier": 1.5,
				"total_fare":       21.23,
			},
			{
				"timestamp":       time.Now().Add(-20 * time.Minute).Format(time.RFC3339),
				"action":          "discount_applied",
				"discount_type":   "first_ride",
				"discount_amount": 4.25,
				"total_fare":      16.98,
			},
		},
	})
}

// GetPricingAnalytics handles pricing analytics requests
func (h *PricingHandler) GetPricingAnalytics(c *gin.Context) {
	analytics, err := h.pricingService.GetPricingAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "analytics_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// ValidatePrice handles price validation requests
func (h *PricingHandler) ValidatePrice(c *gin.Context) {
	var request struct {
		TripID       string  `json:"trip_id" binding:"required"`
		ExpectedFare float64 `json:"expected_fare" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	isValid, cachedPrice, err := h.pricingService.ValidatePrice(
		c.Request.Context(),
		request.TripID,
		request.ExpectedFare,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "validation_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trip_id":       request.TripID,
		"is_valid":      isValid,
		"expected_fare": request.ExpectedFare,
		"cached_price":  cachedPrice,
		"validated_at":  time.Now().Format(time.RFC3339),
	})
}
