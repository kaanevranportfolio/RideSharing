package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rideshare-platform/services/vehicle-service/internal/service"
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
		vehicles.GET("/", h.ListVehicles)
	}

	// Health check
	router.GET("/health", h.HealthCheck)
}

// CreateVehicle creates a new vehicle
func (h *VehicleHandler) CreateVehicle(c *gin.Context) {
	var req service.CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}
	createdVehicle, err := h.vehicleService.CreateVehicle(c.Request.Context(), &req)
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

	var req service.UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set the ID from the URL param
	req.ID = vehicleID

	updatedVehicle, err := h.vehicleService.UpdateVehicle(c.Request.Context(), &req)
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

// ListVehicles returns a list of vehicles
func (h *VehicleHandler) ListVehicles(c *gin.Context) {
	// Parse query params for pagination and filtering
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}
	status := c.Query("status")
	vehicleType := c.Query("vehicle_type")

	req := &service.ListVehiclesRequest{
		Limit:       limit,
		Offset:      offset,
		Status:      status,
		VehicleType: vehicleType,
	}

	resp, err := h.vehicleService.ListVehicles(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// HealthCheck returns the health status of the service
func (h *VehicleHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vehicle-service",
	})
}
