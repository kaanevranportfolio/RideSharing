package resolvers

import (
	"time"

	"github.com/graph-gophers/graphql-go"
	matchingpb "github.com/rideshare-platform/shared/proto/matching"
	paymentpb "github.com/rideshare-platform/shared/proto/payment"
	pricingpb "github.com/rideshare-platform/shared/proto/pricing"
	trippb "github.com/rideshare-platform/shared/proto/trip"
	userpb "github.com/rideshare-platform/shared/proto/user"
)

// Input types for GraphQL mutations
type CreateUserInput struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	PhoneNumber string
	Role        string
}

type CreateTripInput struct {
	RiderID         string
	PickupLocation  LocationInput
	DropoffLocation LocationInput
	VehicleType     string
}

type PriceEstimateInput struct {
	PickupLocation  LocationInput
	DropoffLocation LocationInput
	VehicleType     string
	RiderID         string
}

type NearbyDriversInput struct {
	Location    LocationInput
	Radius      float64
	VehicleType string
	MaxDrivers  int32
}

type ProcessPaymentInput struct {
	TripID          string
	Amount          float64
	Currency        string
	PaymentMethodID string
	Description     string
}

type AddPaymentMethodInput struct {
	UserID         string
	Type           string
	CardNumber     string
	ExpiryMonth    int32
	ExpiryYear     int32
	CardholderName string
	BillingAddress string
	IsDefault      bool
}

type LocationInput struct {
	Latitude  float64
	Longitude float64
	Address   string
}

type DateRangeInput struct {
	StartDate time.Time
	EndDate   time.Time
}

// GraphQL resolvers for complex types

// UserResolver resolves User fields
type UserResolver struct {
	user *userpb.User
}

func (r *UserResolver) ID() graphql.ID {
	return graphql.ID(r.user.Id)
}

func (r *UserResolver) Email() string {
	return r.user.Email
}

func (r *UserResolver) FirstName() string {
	return r.user.FirstName
}

func (r *UserResolver) LastName() string {
	return r.user.LastName
}

func (r *UserResolver) PhoneNumber() string {
	return r.user.Phone // Field is named 'Phone' not 'PhoneNumber'
}

func (r *UserResolver) Role() string {
	return r.user.Role.String()
}

func (r *UserResolver) CreatedAt() graphql.Time {
	if r.user.CreatedAt != nil {
		return graphql.Time{Time: r.user.CreatedAt.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

func (r *UserResolver) UpdatedAt() graphql.Time {
	if r.user.UpdatedAt != nil {
		return graphql.Time{Time: r.user.UpdatedAt.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

// TripResolver resolves Trip fields
type TripResolver struct {
	trip *trippb.Trip
}

func (r *TripResolver) ID() graphql.ID {
	return graphql.ID(r.trip.Id)
}

func (r *TripResolver) RiderID() string {
	return r.trip.RiderId
}

func (r *TripResolver) DriverID() *string {
	if r.trip.DriverId == "" {
		return nil
	}
	return &r.trip.DriverId
}

func (r *TripResolver) Status() string {
	return r.trip.Status.String()
}

func (r *TripResolver) PickupLocation() *LocationResolver {
	if r.trip.PickupLocation == nil {
		return nil
	}
	return &LocationResolver{
		latitude:  r.trip.PickupLocation.Latitude,
		longitude: r.trip.PickupLocation.Longitude,
		address:   r.trip.PickupLocation.Address,
	}
}

func (r *TripResolver) DestinationLocation() *LocationResolver {
	if r.trip.Destination == nil { // Field is named 'Destination'
		return nil
	}
	return &LocationResolver{
		latitude:  r.trip.Destination.Latitude,
		longitude: r.trip.Destination.Longitude,
		address:   r.trip.Destination.Address,
	}
}

func (r *TripResolver) EstimatedFare() float64 {
	return r.trip.EstimatedFare
}

func (r *TripResolver) ActualFare() *float64 {
	if r.trip.ActualFare == 0 {
		return nil
	}
	return &r.trip.ActualFare
}

func (r *TripResolver) CreatedAt() graphql.Time {
	if r.trip.RequestedAt != nil { // Trip uses RequestedAt
		return graphql.Time{Time: r.trip.RequestedAt.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

func (r *TripResolver) UpdatedAt() graphql.Time {
	if r.trip.CompletedAt != nil { // Trip uses CompletedAt as update
		return graphql.Time{Time: r.trip.CompletedAt.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

// LocationResolver resolves Location fields
type LocationResolver struct {
	latitude  float64
	longitude float64
	address   string
}

func (r *LocationResolver) Latitude() float64 {
	return r.latitude
}

func (r *LocationResolver) Longitude() float64 {
	return r.longitude
}

func (r *LocationResolver) Address() string {
	return r.address
}

// DriverResolver resolves Driver fields
type DriverResolver struct {
	driver *matchingpb.Driver
}

func (r *DriverResolver) ID() graphql.ID {
	return graphql.ID(r.driver.Id)
}

func (r *DriverResolver) Location() *LocationResolver {
	if r.driver.CurrentLocation == nil {
		return nil
	}
	return &LocationResolver{
		latitude:  r.driver.CurrentLocation.Latitude,
		longitude: r.driver.CurrentLocation.Longitude,
		address:   r.driver.CurrentLocation.Address,
	}
}

func (r *DriverResolver) Status() string {
	if r.driver.IsAvailable {
		return "AVAILABLE"
	}
	return "UNAVAILABLE"
}

func (r *DriverResolver) VehicleType() string {
	return r.driver.VehicleType
}

func (r *DriverResolver) Rating() float64 {
	return r.driver.Rating
}

func (r *DriverResolver) Distance() float64 {
	return r.driver.DistanceKm
}

func (r *DriverResolver) EstimatedArrival() int32 {
	return r.driver.EtaMinutes
}

// PriceEstimateResolver resolves PriceEstimate fields
type PriceEstimateResolver struct {
	estimate *pricingpb.PriceEstimate
}

func (r *PriceEstimateResolver) BaseFare() float64 {
	return r.estimate.BaseFare
}

func (r *PriceEstimateResolver) DistanceFare() float64 {
	return r.estimate.DistanceFare
}

func (r *PriceEstimateResolver) TimeFare() float64 {
	return r.estimate.TimeFare
}

func (r *PriceEstimateResolver) SurgeAmount() float64 {
	return r.estimate.GetSurgeAmount()
}

func (r *PriceEstimateResolver) TotalAmount() float64 {
	return r.estimate.GetTotalAmount()
}

func (r *PriceEstimateResolver) Currency() string {
	return r.estimate.GetCurrency()
}

func (r *PriceEstimateResolver) SurgeMultiplier() float64 {
	return r.estimate.GetSurgeMultiplier()
}

// MatchResultResolver resolves MatchResult fields
type MatchResultResolver struct {
	result *matchingpb.MatchResult
}

func (r *MatchResultResolver) Success() bool {
	return r.result.Success
}

func (r *MatchResultResolver) Driver() *DriverResolver {
	if r.result.GetBestMatch() == nil {
		return nil
	}
	return &DriverResolver{driver: r.result.GetBestMatch()}
}

func (r *MatchResultResolver) EstimatedArrival() int32 {
	// Returning mock data since this field doesn't exist in the protobuf
	return 300 // 5 minutes in seconds
}

func (r *MatchResultResolver) Message() string {
	return r.result.Message
}

// PaymentResolver resolves Payment fields
type PaymentResolver struct {
	payment *paymentpb.Payment
}

func (r *PaymentResolver) ID() graphql.ID {
	return graphql.ID(r.payment.Id)
}

func (r *PaymentResolver) TripID() string {
	return r.payment.TripId
}

func (r *PaymentResolver) Amount() float64 {
	return r.payment.Amount
}

func (r *PaymentResolver) Currency() string {
	return r.payment.Currency
}

func (r *PaymentResolver) Status() string {
	return r.payment.Status.String()
}

func (r *PaymentResolver) PaymentMethodID() string {
	return r.payment.GetPaymentMethod().String()
}

func (r *PaymentResolver) TransactionID() string {
	return r.payment.GetId()
}

func (r *PaymentResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: r.payment.GetCreatedAt().AsTime()}
}

func (r *PaymentResolver) ProcessedAt() *graphql.Time {
	if r.payment.GetProcessedAt() == nil {
		return nil
	}
	t := graphql.Time{Time: r.payment.GetProcessedAt().AsTime()}
	return &t
}

// PaymentMethodResolver resolves PaymentMethod fields
type PaymentMethodResolver struct {
	method *paymentpb.PaymentMethodDetails
}

func (r *PaymentMethodResolver) ID() graphql.ID {
	return graphql.ID(r.method.Id)
}

func (r *PaymentMethodResolver) UserID() string {
	return r.method.UserId
}

func (r *PaymentMethodResolver) Type() string {
	return r.method.Type.String()
}

func (r *PaymentMethodResolver) LastFour() string {
	return r.method.LastFourDigits
}

func (r *PaymentMethodResolver) ExpiryMonth() int32 {
	// Extract month from expiry date timestamp
	if r.method.ExpiryDate != nil {
		return int32(r.method.ExpiryDate.AsTime().Month())
	}
	return 0
}

func (r *PaymentMethodResolver) ExpiryYear() int32 {
	// Extract year from expiry date timestamp
	if r.method.ExpiryDate != nil {
		return int32(r.method.ExpiryDate.AsTime().Year())
	}
	return 0
}

func (r *PaymentMethodResolver) CardholderName() string {
	// Get from details map
	if r.method.Details != nil {
		return r.method.Details["cardholder_name"]
	}
	return ""
}

func (r *PaymentMethodResolver) IsDefault() bool {
	return r.method.IsDefault
}

func (r *PaymentMethodResolver) CreatedAt() graphql.Time {
	if r.method.CreatedAt != nil {
		return graphql.Time{Time: r.method.CreatedAt.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

// Analytics resolvers
type TripAnalyticsResolver struct {
	totalTrips     int32
	completedTrips int32
	cancelledTrips int32
	averageRating  float64
	totalRevenue   float64
}

func (r *TripAnalyticsResolver) TotalTrips() int32 {
	return r.totalTrips
}

func (r *TripAnalyticsResolver) CompletedTrips() int32 {
	return r.completedTrips
}

func (r *TripAnalyticsResolver) CancelledTrips() int32 {
	return r.cancelledTrips
}

func (r *TripAnalyticsResolver) AverageRating() float64 {
	return r.averageRating
}

func (r *TripAnalyticsResolver) TotalRevenue() float64 {
	return r.totalRevenue
}

// Subscription resolvers
type TripUpdateResolver struct {
	update *trippb.TripUpdateEvent
}

func (r *TripUpdateResolver) TripID() string {
	return r.update.TripId
}

func (r *TripUpdateResolver) Status() string {
	return r.update.NewStatus.String()
}

func (r *TripUpdateResolver) Location() *LocationResolver {
	if r.update.CurrentLocation == nil {
		return nil
	}
	return &LocationResolver{
		latitude:  r.update.CurrentLocation.Latitude,
		longitude: r.update.CurrentLocation.Longitude,
		address:   r.update.CurrentLocation.Address,
	}
}

func (r *TripUpdateResolver) Timestamp() graphql.Time {
	if r.update.Timestamp != nil {
		return graphql.Time{Time: r.update.Timestamp.AsTime()}
	}
	return graphql.Time{Time: time.Time{}}
}

func (r *TripUpdateResolver) Message() string {
	// Get message from metadata if available
	if r.update.Metadata != nil && r.update.Metadata["message"] != "" {
		return r.update.Metadata["message"]
	}
	return "Trip status updated"
}

type DriverLocationResolver struct {
	driverID  string
	location  *LocationResolver
	timestamp graphql.Time
}

func (r *DriverLocationResolver) DriverID() string {
	return r.driverID
}

func (r *DriverLocationResolver) Location() *LocationResolver {
	return r.location
}

func (r *DriverLocationResolver) Timestamp() graphql.Time {
	return r.timestamp
}

type PricingUpdateResolver struct {
	update *pricingpb.PricingUpdateEvent
}

func (r *PricingUpdateResolver) ZoneId() string {
	return r.update.GetZoneId()
}

func (r *PricingUpdateResolver) NewMultiplier() float64 {
	return r.update.GetNewMultiplier()
}

func (r *PricingUpdateResolver) OldMultiplier() float64 {
	return r.update.GetOldMultiplier()
}

func (r *PricingUpdateResolver) Timestamp() string {
	return r.update.GetTimestamp().AsTime().Format(time.RFC3339)
}
