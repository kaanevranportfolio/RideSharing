package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/rideshare-platform/services/api-gateway/internal/grpc"
	geopb "github.com/rideshare-platform/shared/proto/geo"
	matchingpb "github.com/rideshare-platform/shared/proto/matching"
	paymentpb "github.com/rideshare-platform/shared/proto/payment"
	pricingpb "github.com/rideshare-platform/shared/proto/pricing"
	trippb "github.com/rideshare-platform/shared/proto/trip"
	userpb "github.com/rideshare-platform/shared/proto/user"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	grpcClient *grpc.ClientManager
}

// NewResolver creates a new GraphQL resolver
func NewResolver(grpcClient *grpc.ClientManager) *Resolver {
	return &Resolver{
		grpcClient: grpcClient,
	}
}

// User resolvers
func (r *Resolver) User(ctx context.Context, args struct{ ID graphql.ID }) (*UserResolver, error) {
	id := string(args.ID)

	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "user")
	defer cancel()

	resp, err := r.grpcClient.UserClient.GetUser(grpcCtx, &userpb.GetUserRequest{
		UserId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &UserResolver{user: resp.User}, nil
}

func (r *Resolver) CreateUser(ctx context.Context, args struct {
	Input CreateUserInput
}) (*UserResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "user")
	defer cancel()

	resp, err := r.grpcClient.UserClient.CreateUser(grpcCtx, &userpb.CreateUserRequest{
		Email:       args.Input.Email,
		Password:    args.Input.Password,
		FirstName:   args.Input.FirstName,
		LastName:    args.Input.LastName,
		PhoneNumber: args.Input.PhoneNumber,
		Role:        userpb.UserRole(userpb.UserRole_value[args.Input.Role]),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &UserResolver{user: resp.User}, nil
}

// Trip resolvers
func (r *Resolver) Trip(ctx context.Context, args struct{ ID graphql.ID }) (*TripResolver, error) {
	id := string(args.ID)

	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "trip")
	defer cancel()

	resp, err := r.grpcClient.TripClient.GetTrip(grpcCtx, &trippb.GetTripRequest{
		TripId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	return &TripResolver{trip: resp.Trip}, nil
}

func (r *Resolver) CreateTrip(ctx context.Context, args struct {
	Input CreateTripInput
}) (*TripResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "trip")
	defer cancel()

	resp, err := r.grpcClient.TripClient.CreateTrip(grpcCtx, &trippb.CreateTripRequest{
		RiderId:         args.Input.RiderID,
		PickupLocation:  convertToGRPCLocation(args.Input.PickupLocation),
		DropoffLocation: convertToGRPCLocation(args.Input.DropoffLocation),
		VehicleType:     args.Input.VehicleType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	return &TripResolver{trip: resp.Trip}, nil
}

// Pricing resolvers
func (r *Resolver) GetPriceEstimate(ctx context.Context, args struct {
	Input PriceEstimateInput
}) (*PriceEstimateResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "pricing")
	defer cancel()

	resp, err := r.grpcClient.PricingClient.GetPriceEstimate(grpcCtx, &pricingpb.GetPriceEstimateRequest{
		PickupLocation:  convertToGRPCLocation(args.Input.PickupLocation),
		DropoffLocation: convertToGRPCLocation(args.Input.DropoffLocation),
		VehicleType:     args.Input.VehicleType,
		RiderId:         args.Input.RiderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get price estimate: %w", err)
	}

	return &PriceEstimateResolver{estimate: resp.Estimate}, nil
}

// Driver matching resolvers
func (r *Resolver) FindNearbyDrivers(ctx context.Context, args struct {
	Input NearbyDriversInput
}) ([]*DriverResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "matching")
	defer cancel()

	resp, err := r.grpcClient.MatchingClient.FindNearbyDrivers(grpcCtx, &matchingpb.FindNearbyDriversRequest{
		Location:    convertToGRPCLocation(args.Input.Location),
		Radius:      args.Input.Radius,
		VehicleType: args.Input.VehicleType,
		MaxDrivers:  args.Input.MaxDrivers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	drivers := make([]*DriverResolver, len(resp.Drivers))
	for i, driver := range resp.Drivers {
		drivers[i] = &DriverResolver{driver: driver}
	}

	return drivers, nil
}

func (r *Resolver) MatchDriver(ctx context.Context, args struct {
	TripID string
}) (*MatchResultResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "matching")
	defer cancel()

	resp, err := r.grpcClient.MatchingClient.MatchDriver(grpcCtx, &matchingpb.MatchDriverRequest{
		TripId: args.TripID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to match driver: %w", err)
	}

	return &MatchResultResolver{result: resp.MatchResult}, nil
}

// Payment resolvers
func (r *Resolver) ProcessPayment(ctx context.Context, args struct {
	Input ProcessPaymentInput
}) (*PaymentResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "payment")
	defer cancel()

	resp, err := r.grpcClient.PaymentClient.ProcessPayment(grpcCtx, &paymentpb.ProcessPaymentRequest{
		TripId:          args.Input.TripID,
		Amount:          args.Input.Amount,
		Currency:        args.Input.Currency,
		PaymentMethodId: args.Input.PaymentMethodID,
		Description:     args.Input.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	return &PaymentResolver{payment: resp.Payment}, nil
}

func (r *Resolver) AddPaymentMethod(ctx context.Context, args struct {
	Input AddPaymentMethodInput
}) (*PaymentMethodResolver, error) {
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "payment")
	defer cancel()

	resp, err := r.grpcClient.PaymentClient.AddPaymentMethod(grpcCtx, &paymentpb.AddPaymentMethodRequest{
		UserId:         args.Input.UserID,
		Type:           paymentpb.PaymentMethodType(paymentpb.PaymentMethodType_value[args.Input.Type]),
		CardNumber:     args.Input.CardNumber,
		ExpiryMonth:    args.Input.ExpiryMonth,
		ExpiryYear:     args.Input.ExpiryYear,
		CardholderName: args.Input.CardholderName,
		BillingAddress: args.Input.BillingAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add payment method: %w", err)
	}

	return &PaymentMethodResolver{method: resp.PaymentMethod}, nil
}

// Analytics resolvers
func (r *Resolver) TripAnalytics(ctx context.Context, args struct {
	DateRange DateRangeInput
}) (*TripAnalyticsResolver, error) {
	// This could aggregate data from multiple services
	grpcCtx, cancel := r.grpcClient.WithTimeout(ctx, "trip")
	defer cancel()

	// Get trip statistics (simplified example)
	resp, err := r.grpcClient.TripClient.GetTrip(grpcCtx, &trippb.GetTripRequest{
		TripId: "analytics", // This would be a different endpoint in practice
	})
	if err != nil {
		// Return mock analytics for now
		return &TripAnalyticsResolver{
			totalTrips:     1000,
			completedTrips: 950,
			cancelledTrips: 50,
			averageRating:  4.5,
			totalRevenue:   50000.0,
		}, nil
	}

	return &TripAnalyticsResolver{
		totalTrips:     1000,
		completedTrips: 950,
		cancelledTrips: 50,
		averageRating:  4.5,
		totalRevenue:   50000.0,
	}, nil
}

// Utility functions
func convertToGRPCLocation(loc LocationInput) *geopb.Location {
	return &geopb.Location{
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Address:   loc.Address,
	}
}

func convertFromGRPCLocation(loc *geopb.Location) *LocationResolver {
	return &LocationResolver{
		latitude:  loc.Latitude,
		longitude: loc.Longitude,
		address:   loc.Address,
	}
}

func convertTimestamp(ts int64) graphql.Time {
	return graphql.Time{Time: time.Unix(ts, 0)}
}

// Subscription resolvers (simplified examples)
func (r *Resolver) TripUpdates(ctx context.Context, args struct {
	TripID string
}) (<-chan *TripUpdateResolver, error) {
	ch := make(chan *TripUpdateResolver)

	go func() {
		defer close(ch)

		// In a real implementation, this would stream from the gRPC service
		stream, err := r.grpcClient.TripClient.SubscribeToTripUpdates(ctx, &trippb.SubscribeToTripUpdatesRequest{
			TripId: args.TripID,
		})
		if err != nil {
			return
		}

		for {
			update, err := stream.Recv()
			if err != nil {
				return
			}

			select {
			case ch <- &TripUpdateResolver{update: update}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (r *Resolver) DriverLocationUpdates(ctx context.Context, args struct {
	TripID string
}) (<-chan *DriverLocationResolver, error) {
	ch := make(chan *DriverLocationResolver)

	go func() {
		defer close(ch)

		// Stream driver location updates
		stream, err := r.grpcClient.MatchingClient.StreamDriverUpdates(ctx, &matchingpb.StreamDriverUpdatesRequest{
			TripId: args.TripID,
		})
		if err != nil {
			return
		}

		for {
			update, err := stream.Recv()
			if err != nil {
				return
			}

			select {
			case ch <- &DriverLocationResolver{
				driverID:  update.DriverId,
				location:  convertFromGRPCLocation(update.Location),
				timestamp: convertTimestamp(update.Timestamp),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (r *Resolver) PricingUpdates(ctx context.Context, args struct {
	Location LocationInput
}) (<-chan *PricingUpdateResolver, error) {
	ch := make(chan *PricingUpdateResolver)

	go func() {
		defer close(ch)

		// Stream pricing updates
		stream, err := r.grpcClient.PricingClient.SubscribeToPricingUpdates(ctx, &pricingpb.SubscribeToPricingUpdatesRequest{
			Location: convertToGRPCLocation(args.Location),
		})
		if err != nil {
			return
		}

		for {
			update, err := stream.Recv()
			if err != nil {
				return
			}

			select {
			case ch <- &PricingUpdateResolver{update: update}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}
