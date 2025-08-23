package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/vehicle-service/internal/service"
	"github.com/rideshare-platform/shared/models"
)

// VehicleHandler handles HTTP requests for vehicle operations
type VehicleHandler struct {
	vehicleService *service.VehicleService
}

// NewVehicleHandler creates a new vehicle handler
func NewVehicleHandler(vehicleService *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{
		vehicleService: vehicleService,
	}
}

// RegisterRoutes registers vehicle routes
func (h *VehicleHandler) RegisterRoutes(router *gin.Engine) {
	vehicles := router.Group("/api/v1/vehicles")
	{
		vehicles.POST("/", h.CreateVehicle)
		vehicles.GET("/:id", h.GetVehicle)
		vehicles.PUT("/:id", h.UpdateVehicle)
		vehicles.DELETE("/:id", h.DeleteVehicle)
		vehicles.GET("/driver/:driver_id", h.GetVehiclesByDriver)
	}

	// Health check
	router.GET("/health", h.HealthCheck)
}

// CreateVehicleRequest represents the request to create a vehicle
type CreateVehicleRequest struct {
	DriverID     string             `json:"driver_id" binding:"required"`
	Make         string             `json:"make" binding:"required"`
	Model        string             `json:"model" binding:"required"`
	Year         int                `json:"year" binding:"required"`
	Color        string             `json:"color" binding:"required"`
	LicensePlate string             `json:"license_plate" binding:"required"`
	VehicleType  models.VehicleType `json:"vehicle_type" binding:"required"`
	Capacity     int                `json:"capacity" binding:"required"`
}

// UpdateVehicleRequest represents the request to update a vehicle
type UpdateVehicleRequest struct {
	Make         string               `json:"make"`
	Model        string               `json:"model"`
	Year         int                  `json:"year"`
	Color        string               `json:"color"`
	LicensePlate string               `json:"license_plate"`
	VehicleType  models.VehicleType   `json:"vehicle_type"`
	Status       models.VehicleStatus `json:"status"`
	Capacity     int                  `json:"capacity"`
}

// CreateVehicle creates a new vehicle
func (h *VehicleHandler) CreateVehicle(c *gin.Context) {
	var req CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Create vehicle model
	vehicle := models.NewVehicle(req.DriverID, req.Make, req.Model, req.Year, req.Color, req.LicensePlate, req.VehicleType, req.Capacity)

	// Create vehicle
	createdVehicle, err := h.vehicleService.CreateVehicle(c.Request.Context(), vehicle)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create vehicle",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, createdVehicle)
}

// GetVehicle retrieves a vehicle by ID
func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vehicle ID is required",
		})
		return
	}

	vehicle, err := h.vehicleService.GetVehicle(c.Request.Context(), vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Vehicle not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, vehicle)
}

// UpdateVehicle updates an existing vehicle
func (h *VehicleHandler) UpdateVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vehicle ID is required",
		})
		return
	}

	var req UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Get existing vehicle
	vehicle, err := h.vehicleService.GetVehicle(c.Request.Context(), vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Vehicle not found",
			"details": err.Error(),
		})
		return
	}

	// Update fields
	if req.Make != "" {
		vehicle.Make = req.Make
	}
	if req.Model != "" {
		vehicle.Model = req.Model
	}
	if req.Year != 0 {
		vehicle.Year = req.Year
	}
	if req.Color != "" {
		vehicle.Color = req.Color
	}
	if req.LicensePlate != "" {
		vehicle.LicensePlate = req.LicensePlate
	}
	if req.VehicleType != "" {
		vehicle.VehicleType = req.VehicleType
	}
	if req.Status != "" {
		vehicle.Status = req.Status
	}
	if req.Capacity != 0 {
		vehicle.Capacity = req.Capacity
	}

	updatedVehicle, err := h.vehicleService.UpdateVehicle(c.Request.Context(), vehicle)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update vehicle",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedVehicle)
}

// DeleteVehicle deletes a vehicle by ID
func (h *VehicleHandler) DeleteVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vehicle ID is required",
		})
		return
	}

	err := h.vehicleService.DeleteVehicle(c.Request.Context(), vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to delete vehicle",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vehicle deleted successfully",
	})
}

// GetVehiclesByDriver retrieves vehicles by driver ID
func (h *VehicleHandler) GetVehiclesByDriver(c *gin.Context) {
	driverID := c.Param("driver_id")
	if driverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Driver ID is required",
		})
		return
	}

	vehicles, err := h.vehicleService.GetVehiclesByDriver(c.Request.Context(), driverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get vehicles",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vehicles": vehicles,
		"count":    len(vehicles),
	})
}

// HealthCheck returns the health status of the service
func (h *VehicleHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vehicle-service",
	})
}
