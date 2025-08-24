package grpc

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rideshare-platform/shared/models"
	geopb "github.com/rideshare-platform/shared/proto/geo"
)

// CalculateDistance implements the gRPC CalculateDistance method
func (s *Server) CalculateDistance(ctx context.Context, req *geopb.DistanceRequest) (*geopb.DistanceResponse, error) {
	if req.Origin == nil || req.Destination == nil {
		return nil, status.Error(codes.InvalidArgument, "origin and destination are required")
	}

	// Convert gRPC location to internal model
	origin := models.Location{
		Latitude:  req.Origin.Latitude,
		Longitude: req.Origin.Longitude,
		Timestamp: time.Now(),
	}
	destination := models.Location{
		Latitude:  req.Destination.Latitude,
		Longitude: req.Destination.Longitude,
		Timestamp: time.Now(),
	}

	// Calculate distance using the internal service
	distanceCalc, err := s.geoService.CalculateDistance(ctx, origin, destination, req.CalculationMethod)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate distance")
		return nil, status.Error(codes.Internal, "failed to calculate distance")
	}

	return &geopb.DistanceResponse{
		DistanceMeters:    distanceCalc.DistanceMeters,
		DistanceKm:        distanceCalc.DistanceKm,
		BearingDegrees:    distanceCalc.BearingDegrees,
		CalculationMethod: distanceCalc.CalculationMethod,
	}, nil
}

// CalculateETA implements the gRPC CalculateETA method
func (s *Server) CalculateETA(ctx context.Context, req *geopb.ETARequest) (*geopb.ETAResponse, error) {
	if req.Origin == nil || req.Destination == nil {
		return nil, status.Error(codes.InvalidArgument, "origin and destination are required")
	}

	// Convert gRPC location to internal model
	origin := models.Location{
		Latitude:  req.Origin.Latitude,
		Longitude: req.Origin.Longitude,
		Timestamp: time.Now(),
	}
	destination := models.Location{
		Latitude:  req.Destination.Latitude,
		Longitude: req.Destination.Longitude,
		Timestamp: time.Now(),
	}

	// Get departure time
	departureTime := time.Now()
	if req.DepartureTime != nil {
		departureTime = req.DepartureTime.AsTime()
	}

	// Calculate ETA using the internal service
	etaCalc, err := s.geoService.CalculateETA(ctx, origin, destination, req.VehicleType, departureTime, req.IncludeTraffic)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate ETA")
		return nil, status.Error(codes.Internal, "failed to calculate ETA")
	}

	return &geopb.ETAResponse{
		DurationSeconds:  int32(etaCalc.DurationSeconds),
		DistanceMeters:   etaCalc.DistanceMeters,
		RouteSummary:     etaCalc.RouteSummary,
		EstimatedArrival: timestamppb.New(etaCalc.EstimatedArrival),
	}, nil
}

// FindNearbyDrivers implements the gRPC FindNearbyDrivers method
func (s *Server) FindNearbyDrivers(ctx context.Context, req *geopb.NearbyDriversRequest) (*geopb.NearbyDriversResponse, error) {
	if req.Center == nil {
		return nil, status.Error(codes.InvalidArgument, "center location is required")
	}

	center := models.Location{
		Latitude:  req.Center.Latitude,
		Longitude: req.Center.Longitude,
		Timestamp: time.Now(),
	}

	// Use the internal service to find nearby drivers
	nearbyDrivers, err := s.geoService.FindNearbyDrivers(ctx, center, req.RadiusKm, int(req.Limit), req.VehicleTypes, req.OnlyAvailable)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find nearby drivers")
		return nil, status.Error(codes.Internal, "failed to find nearby drivers")
	}

	// Convert internal driver locations to gRPC format
	var grpcDrivers []*geopb.DriverLocation
	for _, driver := range nearbyDrivers {
		grpcDriver := &geopb.DriverLocation{
			DriverId:  driver.DriverID,
			VehicleId: driver.VehicleID,
			Location: &geopb.Location{
				Latitude:  driver.Location.Latitude,
				Longitude: driver.Location.Longitude,
				Timestamp: timestamppb.New(driver.Location.Timestamp),
				Address:   "", // Address field not available in current model
			},
			DistanceFromCenter: driver.DistanceFromCenter,
			Status:             driver.Status,
			VehicleType:        driver.VehicleType,
			Rating:             driver.Rating,
		}
		grpcDrivers = append(grpcDrivers, grpcDriver)
	}

	return &geopb.NearbyDriversResponse{
		Drivers:        grpcDrivers,
		TotalCount:     int32(len(nearbyDrivers)),
		SearchRadiusKm: req.RadiusKm,
	}, nil
}

// UpdateDriverLocation implements the gRPC UpdateDriverLocation method
func (s *Server) UpdateDriverLocation(ctx context.Context, req *geopb.UpdateDriverLocationRequest) (*geopb.UpdateDriverLocationResponse, error) {
	if req.DriverId == "" || req.Location == nil {
		return nil, status.Error(codes.InvalidArgument, "driver_id and location are required")
	}

	location := models.Location{
		Latitude:  req.Location.Latitude,
		Longitude: req.Location.Longitude,
		Timestamp: time.Now(),
	}

	if req.Location.Timestamp != nil {
		location.Timestamp = req.Location.Timestamp.AsTime()
	}

	// Update driver location using the internal service
	err := s.geoService.UpdateDriverLocation(ctx, req.DriverId, location, req.Status, req.VehicleId)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update driver location")
		return &geopb.UpdateDriverLocationResponse{
			Success: false,
			Message: "Failed to update driver location",
		}, nil
	}

	return &geopb.UpdateDriverLocationResponse{
		Success:   true,
		Message:   "Driver location updated successfully",
		UpdatedAt: timestamppb.New(time.Now()),
	}, nil
}

// GenerateGeohash implements the gRPC GenerateGeohash method
func (s *Server) GenerateGeohash(ctx context.Context, req *geopb.GeohashRequest) (*geopb.GeohashResponse, error) {
	if req.Location == nil {
		return nil, status.Error(codes.InvalidArgument, "location is required")
	}

	location := models.Location{
		Latitude:  req.Location.Latitude,
		Longitude: req.Location.Longitude,
		Timestamp: time.Now(),
	}

	precision := int(req.Precision)
	if precision < 1 || precision > 12 {
		precision = 7 // default precision
	}

	// Generate geohash using the internal service
	geohash, err := s.geoService.GenerateGeohash(ctx, location, precision)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate geohash")
		return nil, status.Error(codes.Internal, "failed to generate geohash")
	}

	return &geopb.GeohashResponse{
		Geohash: geohash,
		Center: &geopb.Location{
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
		},
		WidthMeters:  float64(int(1) << uint(25-5*precision/2)), // Approximate geohash cell width
		HeightMeters: float64(int(1) << uint(25-5*precision/2)), // Approximate geohash cell height
	}, nil
}

// OptimizeRoute implements the gRPC OptimizeRoute method
// Note: This is a simplified implementation since the service doesn't have OptimizeRoute
func (s *Server) OptimizeRoute(ctx context.Context, req *geopb.RouteOptimizationRequest) (*geopb.RouteOptimizationResponse, error) {
	if req.Start == nil || req.End == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end locations are required")
	}

	start := models.Location{
		Latitude:  req.Start.Latitude,
		Longitude: req.Start.Longitude,
		Timestamp: time.Now(),
	}
	end := models.Location{
		Latitude:  req.End.Latitude,
		Longitude: req.End.Longitude,
		Timestamp: time.Now(),
	}

	// For now, return a simple optimized route (start -> waypoints -> end)
	// In a real implementation, this would use advanced routing algorithms
	var optimizedRoute []*geopb.Location

	// Add start location
	optimizedRoute = append(optimizedRoute, &geopb.Location{
		Latitude:  start.Latitude,
		Longitude: start.Longitude,
	})

	// Add waypoints (in original order for now)
	for _, wp := range req.Waypoints {
		optimizedRoute = append(optimizedRoute, wp)
	}

	// Add end location
	optimizedRoute = append(optimizedRoute, &geopb.Location{
		Latitude:  end.Latitude,
		Longitude: end.Longitude,
	})

	// Calculate total distance and duration
	totalDistance := 0.0
	totalDuration := 0

	for i := 0; i < len(optimizedRoute)-1; i++ {
		curr := models.Location{
			Latitude:  optimizedRoute[i].Latitude,
			Longitude: optimizedRoute[i].Longitude,
		}
		next := models.Location{
			Latitude:  optimizedRoute[i+1].Latitude,
			Longitude: optimizedRoute[i+1].Longitude,
		}

		distCalc, err := s.geoService.CalculateDistance(ctx, curr, next, "haversine")
		if err == nil {
			totalDistance += distCalc.DistanceMeters
		}
	}

	// Estimate duration based on distance (assume 50 km/h average speed)
	totalDuration = int(totalDistance / 1000 / 50 * 3600) // seconds

	return &geopb.RouteOptimizationResponse{
		OptimizedRoute:           optimizedRoute,
		TotalDistanceKm:          totalDistance / 1000, // Convert to km
		EstimatedDurationSeconds: int32(totalDuration),
		OptimizationAlgorithm:    "nearest_neighbor",
	}, nil
}

// SubscribeToDriverLocations implements real-time driver location streaming
func (s *Server) SubscribeToDriverLocations(req *geopb.SubscribeToDriverLocationRequest, stream geopb.GeospatialService_SubscribeToDriverLocationsServer) error {
	s.logger.WithFields(map[string]interface{}{
		"area_id":    req.AreaId,
		"radius_km":  req.RadiusKm,
		"driver_ids": req.DriverIds,
	}).Info("New driver location subscription")

	// Create a context that can be cancelled
	ctx := stream.Context()

	// Mock implementation for now - in reality, this would:
	// 1. Subscribe to driver location updates from Redis/Kafka
	// 2. Filter by area and driver IDs
	// 3. Stream updates to the client

	// Simulate streaming driver locations
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Driver location subscription cancelled")
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Send mock driver location update
			event := &geopb.DriverLocationEvent{
				DriverId: "driver_123",
				Location: &geopb.Location{
					Latitude:  37.7749 + (rand.Float64()-0.5)*0.01,
					Longitude: -122.4194 + (rand.Float64()-0.5)*0.01,
					Accuracy:  5.0,
					Timestamp: timestamppb.New(time.Now()),
				},
				Status:    "available",
				VehicleId: "vehicle_456",
				SpeedKmh:  25.5,
				Heading:   180.0,
				Timestamp: timestamppb.New(time.Now()),
				Metadata: map[string]string{
					"area":      req.AreaId,
					"zone":      "downtown",
					"direction": "north",
				},
			}

			if err := stream.Send(event); err != nil {
				s.logger.WithError(err).Error("Failed to send driver location event")
				return err
			}

			s.logger.WithFields(map[string]interface{}{
				"driver_id": event.DriverId,
				"lat":       event.Location.Latitude,
				"lng":       event.Location.Longitude,
			}).Debug("Sent driver location update")
		}
	}
}

// StartLocationTracking implements location tracking session initiation
func (s *Server) StartLocationTracking(ctx context.Context, req *geopb.StartLocationTrackingRequest) (*geopb.StartLocationTrackingResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"driver_id":               req.DriverId,
		"update_interval_seconds": req.UpdateIntervalSeconds,
	}).Info("Starting location tracking session")

	// Validate request
	if req.DriverId == "" {
		return &geopb.StartLocationTrackingResponse{
			Success: false,
			Message: "Driver ID is required",
		}, nil
	}

	// Generate session ID
	sessionID := fmt.Sprintf("track_%s_%d", req.DriverId, time.Now().Unix())

	// In a real implementation, this would:
	// 1. Register the driver for location tracking
	// 2. Set up location update intervals
	// 3. Create tracking session in Redis/database

	s.logger.WithFields(map[string]interface{}{
		"driver_id":  req.DriverId,
		"session_id": sessionID,
	}).Info("Location tracking session started")

	return &geopb.StartLocationTrackingResponse{
		Success:   true,
		SessionId: sessionID,
		Message:   "Location tracking session started successfully",
	}, nil
}
