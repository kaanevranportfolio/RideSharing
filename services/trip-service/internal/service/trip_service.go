package service

import (
	"context"

	"github.com/rideshare-platform/services/trip-service/internal/repository"
)

type TripService struct {
	repo *repository.TripRepository
}

type CreateTripRequest struct {
	RiderID        string                 `json:"rider_id"`
	PickupLocation map[string]interface{} `json:"pickup_location"`
	Destination    map[string]interface{} `json:"destination"`
	RideType       string                 `json:"ride_type"`
}

type CreateTripResponse struct {
	TripID         string                 `json:"trip_id"`
	Status         string                 `json:"status"`
	RiderID        string                 `json:"rider_id"`
	PickupLocation map[string]interface{} `json:"pickup_location"`
	Destination    map[string]interface{} `json:"destination"`
}

func NewTripService(repo *repository.TripRepository) *TripService {
	return &TripService{
		repo: repo,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, req *CreateTripRequest) (*CreateTripResponse, error) {
	trip := &repository.Trip{
		RiderID:        req.RiderID,
		PickupLocation: req.PickupLocation,
		Destination:    req.Destination,
		Status:         "requested",
	}

	createdTrip, err := s.repo.CreateTrip(ctx, trip)
	if err != nil {
		return nil, err
	}

	return &CreateTripResponse{
		TripID:         createdTrip.ID,
		Status:         createdTrip.Status,
		RiderID:        createdTrip.RiderID,
		PickupLocation: createdTrip.PickupLocation,
		Destination:    createdTrip.Destination,
	}, nil
}

func (s *TripService) GetTrip(ctx context.Context, tripID string) (*repository.Trip, error) {
	return s.repo.GetTrip(ctx, tripID)
}
