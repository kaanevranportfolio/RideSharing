# Protocol Buffer Schemas

This document defines the comprehensive Protocol Buffer schemas for all gRPC services in the rideshare platform.

## Common Types

### shared/proto/common/types.proto

```protobuf
syntax = "proto3";

package common;

option go_package = "github.com/rideshare-platform/shared/proto/common";

import "google/protobuf/timestamp.proto";

// Location represents a geographical coordinate
message Location {
  double latitude = 1;
  double longitude = 2;
  double accuracy = 3; // accuracy in meters
  google.protobuf.Timestamp timestamp = 4;
}

// Address represents a physical address
message Address {
  string street = 1;
  string city = 2;
  string state = 3;
  string country = 4;
  string postal_code = 5;
  Location coordinates = 6;
}

// Money represents monetary values
message Money {
  int64 amount = 1; // amount in cents
  string currency = 2; // ISO 4217 currency code
}

// Pagination for list requests
message Pagination {
  int32 page = 1;
  int32 limit = 2;
  string sort_by = 3;
  string sort_order = 4; // "asc" or "desc"
}

// Standard response metadata
message ResponseMetadata {
  string request_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  int32 total_count = 3;
  bool has_more = 4;
}
```

### shared/proto/common/errors.proto

```protobuf
syntax = "proto3";

package common;

option go_package = "github.com/rideshare-platform/shared/proto/common";

// Error codes for the platform
enum ErrorCode {
  UNKNOWN = 0;
  INVALID_ARGUMENT = 1;
  NOT_FOUND = 2;
  ALREADY_EXISTS = 3;
  PERMISSION_DENIED = 4;
  UNAUTHENTICATED = 5;
  RESOURCE_EXHAUSTED = 6;
  FAILED_PRECONDITION = 7;
  INTERNAL = 8;
  UNAVAILABLE = 9;
  DEADLINE_EXCEEDED = 10;
}

// Standard error response
message Error {
  ErrorCode code = 1;
  string message = 2;
  map<string, string> details = 3;
}
```

### shared/proto/common/events.proto

```protobuf
syntax = "proto3";

package common;

option go_package = "github.com/rideshare-platform/shared/proto/common";

import "google/protobuf/timestamp.proto";
import "google/protobuf/any.proto";

// Base event structure
message Event {
  string id = 1;
  string type = 2;
  string aggregate_id = 3;
  string aggregate_type = 4;
  int64 version = 5;
  google.protobuf.Any data = 6;
  map<string, string> metadata = 7;
  google.protobuf.Timestamp timestamp = 8;
}

// Event stream for subscriptions
message EventStream {
  repeated Event events = 1;
  string cursor = 2;
  bool has_more = 3;
}
```

## User Service

### shared/proto/user/user.proto

```protobuf
syntax = "proto3";

package user;

option go_package = "github.com/rideshare-platform/shared/proto/user";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// User types
enum UserType {
  RIDER = 0;
  DRIVER = 1;
  ADMIN = 2;
}

// User status
enum UserStatus {
  INACTIVE = 0;
  ACTIVE = 1;
  SUSPENDED = 2;
  BANNED = 3;
}

// Driver status
enum DriverStatus {
  OFFLINE = 0;
  ONLINE = 1;
  BUSY = 2;
  BREAK = 3;
}

// User profile
message User {
  string id = 1;
  string email = 2;
  string phone = 3;
  string first_name = 4;
  string last_name = 5;
  UserType type = 6;
  UserStatus status = 7;
  string profile_image_url = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

// Driver profile
message Driver {
  string user_id = 1;
  string license_number = 2;
  google.protobuf.Timestamp license_expiry = 3;
  DriverStatus status = 4;
  double rating = 5;
  int32 total_trips = 6;
  common.Location current_location = 7;
  google.protobuf.Timestamp last_location_update = 8;
}

// Authentication requests/responses
message LoginRequest {
  string email = 1;
  string password = 2;
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  User user = 3;
  common.Error error = 4;
}

message RegisterRequest {
  string email = 1;
  string password = 2;
  string first_name = 3;
  string last_name = 4;
  string phone = 5;
  UserType type = 6;
}

message RegisterResponse {
  User user = 1;
  common.Error error = 2;
}

// User management requests/responses
message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
  Driver driver = 2; // only if user is a driver
  common.Error error = 3;
}

message UpdateUserRequest {
  string id = 1;
  string first_name = 2;
  string last_name = 3;
  string phone = 4;
  string profile_image_url = 5;
}

message UpdateUserResponse {
  User user = 1;
  common.Error error = 2;
}

message UpdateDriverLocationRequest {
  string driver_id = 1;
  common.Location location = 2;
}

message UpdateDriverLocationResponse {
  common.Error error = 1;
}

message UpdateDriverStatusRequest {
  string driver_id = 1;
  DriverStatus status = 2;
}

message UpdateDriverStatusResponse {
  common.Error error = 1;
}

// User service definition
service UserService {
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc UpdateDriverLocation(UpdateDriverLocationRequest) returns (UpdateDriverLocationResponse);
  rpc UpdateDriverStatus(UpdateDriverStatusRequest) returns (UpdateDriverStatusResponse);
}
```

## Vehicle Service

### shared/proto/vehicle/vehicle.proto

```protobuf
syntax = "proto3";

package vehicle;

option go_package = "github.com/rideshare-platform/shared/proto/vehicle";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// Vehicle types
enum VehicleType {
  SEDAN = 0;
  SUV = 1;
  HATCHBACK = 2;
  LUXURY = 3;
  VAN = 4;
}

// Vehicle status
enum VehicleStatus {
  INACTIVE = 0;
  ACTIVE = 1;
  MAINTENANCE = 2;
  RETIRED = 3;
}

// Vehicle information
message Vehicle {
  string id = 1;
  string driver_id = 2;
  string make = 3;
  string model = 4;
  int32 year = 5;
  string color = 6;
  string license_plate = 7;
  VehicleType type = 8;
  VehicleStatus status = 9;
  int32 capacity = 10;
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp updated_at = 12;
}

// Vehicle requests/responses
message RegisterVehicleRequest {
  string driver_id = 1;
  string make = 2;
  string model = 3;
  int32 year = 4;
  string color = 5;
  string license_plate = 6;
  VehicleType type = 7;
  int32 capacity = 8;
}

message RegisterVehicleResponse {
  Vehicle vehicle = 1;
  common.Error error = 2;
}

message GetVehicleRequest {
  string id = 1;
}

message GetVehicleResponse {
  Vehicle vehicle = 1;
  common.Error error = 2;
}

message GetVehiclesByDriverRequest {
  string driver_id = 1;
}

message GetVehiclesByDriverResponse {
  repeated Vehicle vehicles = 1;
  common.Error error = 2;
}

message UpdateVehicleStatusRequest {
  string id = 1;
  VehicleStatus status = 2;
}

message UpdateVehicleStatusResponse {
  common.Error error = 1;
}

// Vehicle service definition
service VehicleService {
  rpc RegisterVehicle(RegisterVehicleRequest) returns (RegisterVehicleResponse);
  rpc GetVehicle(GetVehicleRequest) returns (GetVehicleResponse);
  rpc GetVehiclesByDriver(GetVehiclesByDriverRequest) returns (GetVehiclesByDriverResponse);
  rpc UpdateVehicleStatus(UpdateVehicleStatusRequest) returns (UpdateVehicleStatusResponse);
}
```

## Geospatial Service

### shared/proto/geo/geo.proto

```protobuf
syntax = "proto3";

package geo;

option go_package = "github.com/rideshare-platform/shared/geo";

import "common/types.proto";
import "common/errors.proto";

// Distance calculation request
message CalculateDistanceRequest {
  common.Location from = 1;
  common.Location to = 2;
}

message CalculateDistanceResponse {
  double distance_km = 1;
  int32 duration_seconds = 2;
  common.Error error = 3;
}

// ETA calculation request
message CalculateETARequest {
  common.Location from = 1;
  common.Location to = 2;
  string vehicle_type = 3;
}

message CalculateETAResponse {
  int32 eta_seconds = 1;
  double distance_km = 2;
  common.Error error = 3;
}

// Nearby drivers request
message FindNearbyDriversRequest {
  common.Location location = 1;
  double radius_km = 2;
  int32 limit = 3;
  string vehicle_type = 4;
}

message NearbyDriver {
  string driver_id = 1;
  string vehicle_id = 2;
  common.Location location = 3;
  double distance_km = 4;
  double rating = 5;
}

message FindNearbyDriversResponse {
  repeated NearbyDriver drivers = 1;
  common.Error error = 2;
}

// Route optimization request
message OptimizeRouteRequest {
  common.Location start = 1;
  common.Location end = 2;
  repeated common.Location waypoints = 3;
}

message RoutePoint {
  common.Location location = 1;
  int32 eta_seconds = 2;
  double distance_from_start_km = 3;
}

message OptimizeRouteResponse {
  repeated RoutePoint route = 1;
  double total_distance_km = 2;
  int32 total_duration_seconds = 3;
  common.Error error = 4;
}

// Geospatial service definition
service GeoService {
  rpc CalculateDistance(CalculateDistanceRequest) returns (CalculateDistanceResponse);
  rpc CalculateETA(CalculateETARequest) returns (CalculateETAResponse);
  rpc FindNearbyDrivers(FindNearbyDriversRequest) returns (FindNearbyDriversResponse);
  rpc OptimizeRoute(OptimizeRouteRequest) returns (OptimizeRouteResponse);
}
```

## Matching Service

### shared/proto/matching/matching.proto

```protobuf
syntax = "proto3";

package matching;

option go_package = "github.com/rideshare-platform/shared/proto/matching";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// Match request status
enum MatchStatus {
  PENDING = 0;
  MATCHED = 1;
  ACCEPTED = 2;
  REJECTED = 3;
  CANCELLED = 4;
  EXPIRED = 5;
}

// Ride request
message RideRequest {
  string id = 1;
  string rider_id = 2;
  common.Location pickup_location = 3;
  common.Location destination = 4;
  string vehicle_type = 5;
  int32 passenger_count = 6;
  MatchStatus status = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp expires_at = 9;
}

// Driver match
message DriverMatch {
  string driver_id = 1;
  string vehicle_id = 2;
  common.Location current_location = 3;
  double distance_to_pickup_km = 4;
  int32 eta_to_pickup_seconds = 5;
  double rating = 6;
  int32 total_trips = 7;
}

// Match requests/responses
message RequestRideRequest {
  string rider_id = 1;
  common.Location pickup_location = 2;
  common.Location destination = 3;
  string vehicle_type = 4;
  int32 passenger_count = 5;
}

message RequestRideResponse {
  RideRequest ride_request = 1;
  common.Error error = 2;
}

message FindMatchRequest {
  string ride_request_id = 1;
}

message FindMatchResponse {
  repeated DriverMatch matches = 1;
  common.Error error = 2;
}

message AcceptRideRequest {
  string ride_request_id = 1;
  string driver_id = 2;
}

message AcceptRideResponse {
  string trip_id = 1;
  common.Error error = 2;
}

message RejectRideRequest {
  string ride_request_id = 1;
  string driver_id = 2;
  string reason = 3;
}

message RejectRideResponse {
  common.Error error = 1;
}

message CancelRideRequestRequest {
  string ride_request_id = 1;
  string reason = 2;
}

message CancelRideRequestResponse {
  common.Error error = 1;
}

// Streaming for real-time dispatch
message DispatchRequest {
  string driver_id = 1;
}

message DispatchNotification {
  RideRequest ride_request = 1;
  int32 expires_in_seconds = 2;
}

// Matching service definition
service MatchingService {
  rpc RequestRide(RequestRideRequest) returns (RequestRideResponse);
  rpc FindMatch(FindMatchRequest) returns (FindMatchResponse);
  rpc AcceptRide(AcceptRideRequest) returns (AcceptRideResponse);
  rpc RejectRide(RejectRideRequest) returns (RejectRideResponse);
  rpc CancelRideRequest(CancelRideRequestRequest) returns (CancelRideRequestResponse);
  
  // Streaming for real-time dispatch
  rpc DriverDispatchStream(DispatchRequest) returns (stream DispatchNotification);
}
```

## Pricing Service

### shared/proto/pricing/pricing.proto

```protobuf
syntax = "proto3";

package pricing;

option go_package = "github.com/rideshare-platform/shared/proto/pricing";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// Pricing factors
message PricingFactors {
  double base_fare = 1;
  double per_km_rate = 2;
  double per_minute_rate = 3;
  double surge_multiplier = 4;
  double booking_fee = 5;
  double service_fee = 6;
}

// Fare breakdown
message FareBreakdown {
  common.Money base_fare = 1;
  common.Money distance_fare = 2;
  common.Money time_fare = 3;
  common.Money surge_amount = 4;
  common.Money booking_fee = 5;
  common.Money service_fee = 6;
  common.Money discount = 7;
  common.Money total = 8;
}

// Pricing requests/responses
message CalculateFareRequest {
  common.Location pickup_location = 1;
  common.Location destination = 2;
  string vehicle_type = 3;
  google.protobuf.Timestamp pickup_time = 4;
  string promo_code = 5;
}

message CalculateFareResponse {
  FareBreakdown fare_breakdown = 1;
  PricingFactors pricing_factors = 2;
  common.Error error = 3;
}

message GetSurgeMultiplierRequest {
  common.Location location = 1;
  string vehicle_type = 2;
  google.protobuf.Timestamp timestamp = 3;
}

message GetSurgeMultiplierResponse {
  double surge_multiplier = 1;
  string reason = 2;
  google.protobuf.Timestamp expires_at = 3;
  common.Error error = 4;
}

message UpdateSurgeMultiplierRequest {
  common.Location location = 1;
  string vehicle_type = 2;
  double surge_multiplier = 3;
  string reason = 4;
  int32 duration_minutes = 5;
}

message UpdateSurgeMultiplierResponse {
  common.Error error = 1;
}

message ValidatePromoCodeRequest {
  string promo_code = 1;
  string user_id = 2;
  common.Money trip_amount = 3;
}

message PromoCodeDiscount {
  string type = 1; // "percentage" or "fixed"
  double value = 2;
  common.Money max_discount = 3;
  common.Money min_trip_amount = 4;
}

message ValidatePromoCodeResponse {
  bool valid = 1;
  PromoCodeDiscount discount = 2;
  string reason = 3;
  common.Error error = 4;
}

// Pricing service definition
service PricingService {
  rpc CalculateFare(CalculateFareRequest) returns (CalculateFareResponse);
  rpc GetSurgeMultiplier(GetSurgeMultiplierRequest) returns (GetSurgeMultiplierResponse);
  rpc UpdateSurgeMultiplier(UpdateSurgeMultiplierRequest) returns (UpdateSurgeMultiplierResponse);
  rpc ValidatePromoCode(ValidatePromoCodeRequest) returns (ValidatePromoCodeResponse);
}
```

## Trip Service

### shared/proto/trip/trip.proto

```protobuf
syntax = "proto3";

package trip;

option go_package = "github.com/rideshare-platform/shared/proto/trip";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// Trip status
enum TripStatus {
  REQUESTED = 0;
  MATCHED = 1;
  DRIVER_ASSIGNED = 2;
  DRIVER_ARRIVING = 3;
  DRIVER_ARRIVED = 4;
  TRIP_STARTED = 5;
  IN_PROGRESS = 6;
  COMPLETED = 7;
  CANCELLED = 8;
  FAILED = 9;
}

// Trip information
message Trip {
  string id = 1;
  string rider_id = 2;
  string driver_id = 3;
  string vehicle_id = 4;
  common.Location pickup_location = 5;
  common.Location destination = 6;
  TripStatus status = 7;
  common.Money fare = 8;
  double distance_km = 9;
  int32 duration_seconds = 10;
  google.protobuf.Timestamp requested_at = 11;
  google.protobuf.Timestamp started_at = 12;
  google.protobuf.Timestamp completed_at = 13;
  string cancellation_reason = 14;
  repeated TripEvent events = 15;
}

// Trip event for event sourcing
message TripEvent {
  string id = 1;
  string trip_id = 2;
  string event_type = 3;
  string data = 4; // JSON data
  google.protobuf.Timestamp timestamp = 5;
  string user_id = 6; // who triggered the event
}

// Trip requests/responses
message CreateTripRequest {
  string rider_id = 1;
  string driver_id = 2;
  string vehicle_id = 3;
  common.Location pickup_location = 4;
  common.Location destination = 5;
  common.Money estimated_fare = 6;
}

message CreateTripResponse {
  Trip trip = 1;
  common.Error error = 2;
}

message GetTripRequest {
  string id = 1;
}

message GetTripResponse {
  Trip trip = 1;
  common.Error error = 2;
}

message UpdateTripStatusRequest {
  string id = 1;
  TripStatus status = 2;
  string user_id = 3;
  map<string, string> metadata = 4;
}

message UpdateTripStatusResponse {
  Trip trip = 1;
  common.Error error = 2;
}

message GetTripHistoryRequest {
  string user_id = 1;
  common.Pagination pagination = 2;
}

message GetTripHistoryResponse {
  repeated Trip trips = 1;
  common.ResponseMetadata metadata = 2;
  common.Error error = 3;
}

message CancelTripRequest {
  string id = 1;
  string user_id = 2;
  string reason = 3;
}

message CancelTripResponse {
  Trip trip = 1;
  common.Error error = 2;
}

// Real-time trip updates
message TripUpdateRequest {
  string trip_id = 1;
}

message TripUpdate {
  Trip trip = 1;
  TripEvent event = 2;
}

// Trip service definition
service TripService {
  rpc CreateTrip(CreateTripRequest) returns (CreateTripResponse);
  rpc GetTrip(GetTripRequest) returns (GetTripResponse);
  rpc UpdateTripStatus(UpdateTripStatusRequest) returns (UpdateTripStatusResponse);
  rpc GetTripHistory(GetTripHistoryRequest) returns (GetTripHistoryResponse);
  rpc CancelTrip(CancelTripRequest) returns (CancelTripResponse);
  
  // Streaming for real-time updates
  rpc TripUpdateStream(TripUpdateRequest) returns (stream TripUpdate);
}
```

## Payment Service

### shared/proto/payment/payment.proto

```protobuf
syntax = "proto3";

package payment;

option go_package = "github.com/rideshare-platform/shared/proto/payment";

import "common/types.proto";
import "common/errors.proto";
import "google/protobuf/timestamp.proto";

// Payment method types
enum PaymentMethodType {
  CREDIT_CARD = 0;
  DEBIT_CARD = 1;
  DIGITAL_WALLET = 2;
  CASH = 3;
  BANK_TRANSFER = 4;
}

// Payment status
enum PaymentStatus {
  PENDING = 0;
  PROCESSING = 1;
  COMPLETED = 2;
  FAILED = 3;
  CANCELLED = 4;
  REFUNDED = 5;
}

// Payment method
message PaymentMethod {
  string id = 1;
  string user_id = 2;
  PaymentMethodType type = 3;
  string last_four = 4;
  string brand = 5;
  bool is_default = 6;
  google.protobuf.Timestamp expires_at = 7;
  google.protobuf.Timestamp created_at = 8;
}

// Payment transaction
message Payment {
  string id = 1;
  string trip_id = 2;
  string user_id = 3;
  string payment_method_id = 4;
  common.Money amount = 5;
  PaymentStatus status = 6;
  string gateway_transaction_id = 7;
  string failure_reason = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

// Payment requests/responses
message ProcessPaymentRequest {
  string trip_id = 1;
  string user_id = 2;
  string payment_method_id = 3;
  common.Money amount = 4;
  map<string, string> metadata = 5;
}

message ProcessPaymentResponse {
  Payment payment = 1;
  common.Error error = 2;
}

message GetPaymentRequest {
  string id = 1;
}

message GetPaymentResponse {
  Payment payment = 1;
  common.Error error = 2;
}

message RefundPaymentRequest {
  string payment_id = 1;
  common.Money amount = 2;
  string reason = 3;
}

message RefundPaymentResponse {
  Payment refund = 1;
  common.Error error = 2;
}

message AddPaymentMethodRequest {
  string user_id = 1;
  PaymentMethodType type = 2;
  string token = 3; // payment gateway token
  bool set_as_default = 4;
}

message AddPaymentMethodResponse {
  PaymentMethod payment_method = 1;
  common.Error error = 2;
}

message GetPaymentMethodsRequest {
  string user_id = 1;
}

message GetPaymentMethodsResponse {
  repeated PaymentMethod payment_methods = 1;
  common.Error error = 2;
}

// Payment service definition
service PaymentService {
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
  rpc GetPayment(GetPaymentRequest) returns (GetPaymentResponse);
  rpc RefundPayment(RefundPaymentRequest) returns (RefundPaymentResponse);
  rpc AddPaymentMethod(AddPaymentMethodRequest) returns (AddPaymentMethodResponse);
  rpc GetPaymentMethods(GetPaymentMethodsRequest) returns (GetPaymentMethodsResponse);
}
```

## Code Generation

To generate Go code from these Protocol Buffer definitions:

```bash
# Install required tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code for all services
make proto-gen
```

This comprehensive Protocol Buffer schema provides type-safe, efficient communication between all microservices in the rideshare platform with proper error handling, streaming capabilities, and extensible design patterns.