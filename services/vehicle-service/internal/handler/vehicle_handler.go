package handler

import (
	"context"

	"github.com/rideshare-platform/services/vehicle-service/internal/service"
	"github.com/rideshare-platform/shared/grpc"
	"github.com/rideshare-platform/shared/logger"
	"github.com/rideshare-platform/shared/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VehicleHandler handles gRPC requests for vehicle operations
type VehicleHandler struct {
	vehicleService *service.VehicleService
	logger         *logger.Logger
	errorHandler   *grpc.ErrorHandler
}

// NewVehicleHandler creates a new vehicle handler
func NewVehicleHandler(vehicleService *service.VehicleService, logger *logger.Logger) *VehicleHandler {
	return &VehicleHandler{
		vehicleService: vehicleService,
		logger:         logger,
		errorHandler:   grpc.NewErrorHandler(logger),
	}
}

// RegisterWithServer registers the handler with a gRPC server
func (h *VehicleHandler) RegisterWithServer(server interface{}) {
	// In a real implementation, this would register with the generated gRPC server
	// For now, we'll just log that the handler is registered
	h.logger.Logger.Info("Vehicle handler registered with gRPC server")
}

// CreateVehicle handles vehicle creation requests
func (h *VehicleHandler) CreateVehicle(ctx context.Context, req *CreateVehicleRequest) (*VehicleResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id":     req.DriverId,
		"license_plate": req.LicensePlate,
		"make":          req.Make,
		"model":         req.Model,
	}).Info("Creating vehicle")

	// Convert gRPC request to service request
	serviceReq := &service.CreateVehicleRequest{
		DriverID:              req.DriverId,
		Make:                  req.Make,
		Model:                 req.Model,
		Year:                  int(req.Year),
		Color:                 req.Color,
		LicensePlate:          req.LicensePlate,
		VehicleType:           req.VehicleType,
		Capacity:              int(req.Capacity),
		InsurancePolicyNumber: req.InsurancePolicyNumber,
	}

	// Convert timestamps
	if req.InsuranceExpiry != nil {
		expiry := req.InsuranceExpiry.AsTime()
		serviceReq.InsuranceExpiry = &expiry
	}
	if req.RegistrationExpiry != nil {
		expiry := req.RegistrationExpiry.AsTime()
		serviceReq.RegistrationExpiry = &expiry
	}

	// Call service
	vehicle, err := h.vehicleService.CreateVehicle(ctx, serviceReq)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to create vehicle")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	// Convert to gRPC response
	response := h.vehicleToResponse(vehicle)

	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicle.ID,
		"driver_id":  vehicle.DriverID,
	}).Info("Vehicle created successfully")

	return response, nil
}

// GetVehicle handles vehicle retrieval requests
func (h *VehicleHandler) GetVehicle(ctx context.Context, req *GetVehicleRequest) (*VehicleResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
	}).Debug("Getting vehicle")

	vehicle, err := h.vehicleService.GetVehicle(ctx, req.Id)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get vehicle")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	return h.vehicleToResponse(vehicle), nil
}

// GetVehiclesByDriver handles driver vehicles retrieval requests
func (h *VehicleHandler) GetVehiclesByDriver(ctx context.Context, req *GetVehiclesByDriverRequest) (*VehicleListResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id": req.DriverId,
	}).Debug("Getting vehicles by driver")

	vehicles, err := h.vehicleService.GetVehiclesByDriver(ctx, req.DriverId)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get vehicles by driver")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	// Convert to gRPC response
	response := &VehicleListResponse{
		Vehicles: make([]*VehicleResponse, len(vehicles)),
	}

	for i, vehicle := range vehicles {
		response.Vehicles[i] = h.vehicleToResponse(vehicle)
	}

	return response, nil
}

// GetAvailableVehicles handles available vehicles retrieval requests
func (h *VehicleHandler) GetAvailableVehicles(ctx context.Context, req *GetAvailableVehiclesRequest) (*VehicleListResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"driver_id": req.DriverId,
	}).Debug("Getting available vehicles")

	vehicles, err := h.vehicleService.GetAvailableVehicles(ctx, req.DriverId)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get available vehicles")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	// Convert to gRPC response
	response := &VehicleListResponse{
		Vehicles: make([]*VehicleResponse, len(vehicles)),
	}

	for i, vehicle := range vehicles {
		response.Vehicles[i] = h.vehicleToResponse(vehicle)
	}

	return response, nil
}

// UpdateVehicle handles vehicle update requests
func (h *VehicleHandler) UpdateVehicle(ctx context.Context, req *UpdateVehicleRequest) (*VehicleResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
	}).Info("Updating vehicle")

	// Convert gRPC request to service request
	serviceReq := &service.UpdateVehicleRequest{
		ID:                    req.Id,
		Make:                  req.Make,
		Model:                 req.Model,
		Year:                  int(req.Year),
		Color:                 req.Color,
		LicensePlate:          req.LicensePlate,
		VehicleType:           req.VehicleType,
		Capacity:              int(req.Capacity),
		InsurancePolicyNumber: req.InsurancePolicyNumber,
	}

	// Convert timestamps
	if req.InsuranceExpiry != nil {
		expiry := req.InsuranceExpiry.AsTime()
		serviceReq.InsuranceExpiry = &expiry
	}
	if req.RegistrationExpiry != nil {
		expiry := req.RegistrationExpiry.AsTime()
		serviceReq.RegistrationExpiry = &expiry
	}

	// Call service
	vehicle, err := h.vehicleService.UpdateVehicle(ctx, serviceReq)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to update vehicle")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": vehicle.ID,
	}).Info("Vehicle updated successfully")

	return h.vehicleToResponse(vehicle), nil
}

// UpdateVehicleStatus handles vehicle status update requests
func (h *VehicleHandler) UpdateVehicleStatus(ctx context.Context, req *UpdateVehicleStatusRequest) (*UpdateVehicleStatusResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
		"status":     req.Status,
	}).Info("Updating vehicle status")

	// Convert status
	status := models.VehicleStatus(req.Status)

	err := h.vehicleService.UpdateVehicleStatus(ctx, req.Id, status)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to update vehicle status")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
		"status":     req.Status,
	}).Info("Vehicle status updated successfully")

	return &UpdateVehicleStatusResponse{
		Success: true,
		Message: "Vehicle status updated successfully",
	}, nil
}

// DeleteVehicle handles vehicle deletion requests
func (h *VehicleHandler) DeleteVehicle(ctx context.Context, req *DeleteVehicleRequest) (*DeleteVehicleResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
	}).Info("Deleting vehicle")

	err := h.vehicleService.DeleteVehicle(ctx, req.Id)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to delete vehicle")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"vehicle_id": req.Id,
	}).Info("Vehicle deleted successfully")

	return &DeleteVehicleResponse{
		Success: true,
		Message: "Vehicle deleted successfully",
	}, nil
}

// ListVehicles handles vehicle listing requests
func (h *VehicleHandler) ListVehicles(ctx context.Context, req *ListVehiclesRequest) (*ListVehiclesResponse, error) {
	h.logger.WithContext(ctx).WithFields(logger.Fields{
		"limit":        req.Limit,
		"offset":       req.Offset,
		"status":       req.Status,
		"vehicle_type": req.VehicleType,
	}).Debug("Listing vehicles")

	// Convert gRPC request to service request
	serviceReq := &service.ListVehiclesRequest{
		Limit:       int(req.Limit),
		Offset:      int(req.Offset),
		Status:      req.Status,
		VehicleType: req.VehicleType,
	}

	// Call service
	result, err := h.vehicleService.ListVehicles(ctx, serviceReq)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to list vehicles")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	// Convert to gRPC response
	response := &ListVehiclesResponse{
		Vehicles: make([]*VehicleResponse, len(result.Vehicles)),
		Total:    result.Total,
		Limit:    int32(result.Limit),
		Offset:   int32(result.Offset),
	}

	for i, vehicle := range result.Vehicles {
		response.Vehicles[i] = h.vehicleToResponse(vehicle)
	}

	return response, nil
}

// GetVehicleStats handles vehicle statistics requests
func (h *VehicleHandler) GetVehicleStats(ctx context.Context, req *GetVehicleStatsRequest) (*GetVehicleStatsResponse, error) {
	h.logger.WithContext(ctx).Debug("Getting vehicle statistics")

	stats, err := h.vehicleService.GetVehicleStats(ctx)
	if err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get vehicle stats")
		return nil, h.errorHandler.HandleError(ctx, err)
	}

	// Convert vehicles by type map
	vehiclesByType := make(map[string]int64)
	for k, v := range stats.VehiclesByType {
		if count, ok := v.(int64); ok {
			vehiclesByType[k] = count
		}
	}

	return &GetVehicleStatsResponse{
		TotalVehicles:    stats.TotalVehicles,
		ActiveVehicles:   stats.ActiveVehicles,
		InactiveVehicles: stats.InactiveVehicles,
		VehiclesByType:   vehiclesByType,
	}, nil
}

// Helper method to convert vehicle model to gRPC response
func (h *VehicleHandler) vehicleToResponse(vehicle *models.Vehicle) *VehicleResponse {
	response := &VehicleResponse{
		Id:                    vehicle.ID,
		DriverId:              vehicle.DriverID,
		Make:                  vehicle.Make,
		Model:                 vehicle.Model,
		Year:                  int32(vehicle.Year),
		Color:                 vehicle.Color,
		LicensePlate:          vehicle.LicensePlate,
		VehicleType:           string(vehicle.VehicleType),
		Status:                string(vehicle.Status),
		Capacity:              int32(vehicle.Capacity),
		InsurancePolicyNumber: vehicle.InsurancePolicyNumber,
		CreatedAt:             timestamppb.New(vehicle.CreatedAt),
		UpdatedAt:             timestamppb.New(vehicle.UpdatedAt),
	}

	// Handle nullable timestamps
	if vehicle.InsuranceExpiry != nil {
		response.InsuranceExpiry = timestamppb.New(*vehicle.InsuranceExpiry)
	}
	if vehicle.RegistrationExpiry != nil {
		response.RegistrationExpiry = timestamppb.New(*vehicle.RegistrationExpiry)
	}

	return response
}

// gRPC message types (these would normally be generated from .proto files)

type CreateVehicleRequest struct {
	DriverId              string                 `json:"driver_id"`
	Make                  string                 `json:"make"`
	Model                 string                 `json:"model"`
	Year                  int32                  `json:"year"`
	Color                 string                 `json:"color"`
	LicensePlate          string                 `json:"license_plate"`
	VehicleType           string                 `json:"vehicle_type"`
	Capacity              int32                  `json:"capacity"`
	InsurancePolicyNumber string                 `json:"insurance_policy_number"`
	InsuranceExpiry       *timestamppb.Timestamp `json:"insurance_expiry"`
	RegistrationExpiry    *timestamppb.Timestamp `json:"registration_expiry"`
}

type UpdateVehicleRequest struct {
	Id                    string                 `json:"id"`
	Make                  string                 `json:"make"`
	Model                 string                 `json:"model"`
	Year                  int32                  `json:"year"`
	Color                 string                 `json:"color"`
	LicensePlate          string                 `json:"license_plate"`
	VehicleType           string                 `json:"vehicle_type"`
	Capacity              int32                  `json:"capacity"`
	InsurancePolicyNumber string                 `json:"insurance_policy_number"`
	InsuranceExpiry       *timestamppb.Timestamp `json:"insurance_expiry"`
	RegistrationExpiry    *timestamppb.Timestamp `json:"registration_expiry"`
}

type GetVehicleRequest struct {
	Id string `json:"id"`
}

type GetVehiclesByDriverRequest struct {
	DriverId string `json:"driver_id"`
}

type GetAvailableVehiclesRequest struct {
	DriverId string `json:"driver_id"`
}

type UpdateVehicleStatusRequest struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type DeleteVehicleRequest struct {
	Id string `json:"id"`
}

type ListVehiclesRequest struct {
	Limit       int32  `json:"limit"`
	Offset      int32  `json:"offset"`
	Status      string `json:"status"`
	VehicleType string `json:"vehicle_type"`
}

type GetVehicleStatsRequest struct{}

type VehicleResponse struct {
	Id                    string                 `json:"id"`
	DriverId              string                 `json:"driver_id"`
	Make                  string                 `json:"make"`
	Model                 string                 `json:"model"`
	Year                  int32                  `json:"year"`
	Color                 string                 `json:"color"`
	LicensePlate          string                 `json:"license_plate"`
	VehicleType           string                 `json:"vehicle_type"`
	Status                string                 `json:"status"`
	Capacity              int32                  `json:"capacity"`
	InsurancePolicyNumber string                 `json:"insurance_policy_number"`
	InsuranceExpiry       *timestamppb.Timestamp `json:"insurance_expiry"`
	RegistrationExpiry    *timestamppb.Timestamp `json:"registration_expiry"`
	CreatedAt             *timestamppb.Timestamp `json:"created_at"`
	UpdatedAt             *timestamppb.Timestamp `json:"updated_at"`
}

type VehicleListResponse struct {
	Vehicles []*VehicleResponse `json:"vehicles"`
}

type UpdateVehicleStatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DeleteVehicleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListVehiclesResponse struct {
	Vehicles []*VehicleResponse `json:"vehicles"`
	Total    int64              `json:"total"`
	Limit    int32              `json:"limit"`
	Offset   int32              `json:"offset"`
}

type GetVehicleStatsResponse struct {
	TotalVehicles    int64            `json:"total_vehicles"`
	ActiveVehicles   int64            `json:"active_vehicles"`
	InactiveVehicles int64            `json:"inactive_vehicles"`
	VehiclesByType   map[string]int64 `json:"vehicles_by_type"`
}
