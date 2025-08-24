package handler

import (
"net/http"

"github.com/gin-gonic/gin"
"github.com/rideshare-platform/services/trip-service/internal/service"
)

type TripHandler struct {
service *service.TripService
}

func NewTripHandler(service *service.TripService) *TripHandler {
return &TripHandler{
service: service,
}
}

func (h *TripHandler) RegisterRoutes(router *gin.Engine) {
router.GET("/health", h.healthCheck)

api := router.Group("/api/v1")
{
api.GET("/health", h.healthCheck)
api.POST("/trips", h.createTrip)
api.GET("/trips/:trip_id", h.getTrip)
}
}

func (h *TripHandler) healthCheck(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"status":  "healthy",
"service": "trip-service",
"version": "1.0.0",
})
}

func (h *TripHandler) createTrip(c *gin.Context) {
var req service.CreateTripRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"error":   "Invalid request body",
"details": err.Error(),
})
return
}

if req.RiderID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"error": "rider_id is required",
})
return
}

if req.PickupLocation == nil {
c.JSON(http.StatusBadRequest, gin.H{
"error": "pickup_location is required",
})
return
}

if req.Destination == nil {
c.JSON(http.StatusBadRequest, gin.H{
"error": "destination is required",
})
return
}

trip, err := h.service.CreateTrip(c.Request.Context(), &req)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{
"error":   "Failed to create trip",
"details": err.Error(),
})
return
}

c.JSON(http.StatusCreated, trip)
}

func (h *TripHandler) getTrip(c *gin.Context) {
tripID := c.Param("trip_id")
if tripID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"error": "Missing trip_id parameter",
})
return
}

trip, err := h.service.GetTrip(c.Request.Context(), tripID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{
"error":   "Failed to retrieve trip",
"details": err.Error(),
})
return
}

if trip == nil {
c.JSON(http.StatusNotFound, gin.H{
"error":   "Trip not found",
"trip_id": tripID,
})
return
}

c.JSON(http.StatusOK, trip)
}
