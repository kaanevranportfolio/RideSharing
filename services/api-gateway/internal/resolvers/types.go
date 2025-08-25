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

func (r *PriceEstimateResolver) SurgeFare() float64 {
	return r.estimate.SurgeFare
}

func (r *PriceEstimateResolver) TotalFare() float64 {
	return r.estimate.TotalFare
}

func (r *PriceEstimateResolver) Currency() string {
	return r.estimate.Currency
}

func (r *PriceEstimateResolver) SurgeMultiplier() float64 {
	return r.estimate.SurgeMultiplier
}

func (r *PriceEstimateResolver) EstimatedDuration() int32 {
	return r.estimate.EstimatedDuration
}

func (r *PriceEstimateResolver) EstimatedDistance() float64 {
	return r.estimate.EstimatedDistance
}

// MatchResultResolver resolves MatchResult fields
type MatchResultResolver struct {
	result *matchingpb.MatchResult
}

func (r *MatchResultResolver) Success() bool {
	return r.result.Success
}

func (r *MatchResultResolver) Driver() *DriverResolver {
	if r.result.Driver == nil {
		return nil
	}
	return &DriverResolver{driver: r.result.Driver}
}

func (r *MatchResultResolver) EstimatedArrival() int32 {
	return r.result.EstimatedArrival
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
	return r.payment.PaymentMethodId
}

func (r *PaymentResolver) TransactionID() string {
	return r.payment.TransactionId
}

func (r *PaymentResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: time.Unix(r.payment.CreatedAt, 0)}
}

func (r *PaymentResolver) ProcessedAt() *graphql.Time {
	if r.payment.ProcessedAt == 0 {
		return nil
	}
	t := graphql.Time{Time: time.Unix(r.payment.ProcessedAt, 0)}
	return &t
}

// PaymentMethodResolver resolves PaymentMethod fields
type PaymentMethodResolver struct {
	method *paymentpb.PaymentMethod
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
	return r.method.LastFour
}

func (r *PaymentMethodResolver) ExpiryMonth() int32 {
	return r.method.ExpiryMonth
}

func (r *PaymentMethodResolver) ExpiryYear() int32 {
	return r.method.ExpiryYear
}

func (r *PaymentMethodResolver) CardholderName() string {
	return r.method.CardholderName
}

func (r *PaymentMethodResolver) IsDefault() bool {
	return r.method.IsDefault
}

func (r *PaymentMethodResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: time.Unix(r.method.CreatedAt, 0)}
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
	return r.update.Status.String()
}

func (r *TripUpdateResolver) Location() *LocationResolver {
	if r.update.Location == nil {
		return nil
	}
	return &LocationResolver{
		latitude:  r.update.Location.Latitude,
		longitude: r.update.Location.Longitude,
		address:   r.update.Location.Address,
	}
}

func (r *TripUpdateResolver) Timestamp() graphql.Time {
	return graphql.Time{Time: time.Unix(r.update.Timestamp, 0)}
}

func (r *TripUpdateResolver) Message() string {
	return r.update.Message
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

func (r *PricingUpdateResolver) Location() *LocationResolver {
	if r.update.Location == nil {
		return nil
	}
	return &LocationResolver{
		latitude:  r.update.Location.Latitude,
		longitude: r.update.Location.Longitude,
		address:   r.update.Location.Address,
	}
}

func (r *PricingUpdateResolver) SurgeMultiplier() float64 {
	return r.update.SurgeMultiplier
}

func (r *PricingUpdateResolver) BaseFare() float64 {
	return r.update.BaseFare
}

func (r *PricingUpdateResolver) Timestamp() graphql.Time {
	return graphql.Time{Time: time.Unix(r.update.Timestamp, 0)}
}
