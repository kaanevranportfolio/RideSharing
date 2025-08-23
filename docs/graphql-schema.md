# GraphQL Schema Design

This document defines the comprehensive GraphQL schema for the rideshare platform's client-facing API gateway.

## Schema Overview

The GraphQL API provides a unified interface for all client applications (web, mobile, admin) to interact with the rideshare platform. It aggregates data from multiple microservices and provides real-time subscriptions for live updates.

## Core Types

### User Types

```graphql
enum UserType {
  RIDER
  DRIVER
  ADMIN
}

enum UserStatus {
  INACTIVE
  ACTIVE
  SUSPENDED
  BANNED
}

enum DriverStatus {
  OFFLINE
  ONLINE
  BUSY
  BREAK
}

type User {
  id: ID!
  email: String!
  phone: String!
  firstName: String!
  lastName: String!
  type: UserType!
  status: UserStatus!
  profileImageUrl: String
  createdAt: DateTime!
  updatedAt: DateTime!
  
  # Driver-specific fields (only populated if user is a driver)
  driver: Driver
  
  # Relationships
  trips: [Trip!]!
  vehicles: [Vehicle!]!
  paymentMethods: [PaymentMethod!]!
}

type Driver {
  userId: ID!
  licenseNumber: String!
  licenseExpiry: DateTime!
  status: DriverStatus!
  rating: Float!
  totalTrips: Int!
  currentLocation: Location
  lastLocationUpdate: DateTime
  
  # Relationships
  user: User!
  vehicles: [Vehicle!]!
  activeTrip: Trip
}
```

### Vehicle Types

```graphql
enum VehicleType {
  SEDAN
  SUV
  HATCHBACK
  LUXURY
  VAN
}

enum VehicleStatus {
  INACTIVE
  ACTIVE
  MAINTENANCE
  RETIRED
}

type Vehicle {
  id: ID!
  driverId: ID!
  make: String!
  model: String!
  year: Int!
  color: String!
  licensePlate: String!
  type: VehicleType!
  status: VehicleStatus!
  capacity: Int!
  createdAt: DateTime!
  updatedAt: DateTime!
  
  # Relationships
  driver: Driver!
  trips: [Trip!]!
}
```

### Location and Geography Types

```graphql
type Location {
  latitude: Float!
  longitude: Float!
  accuracy: Float
  timestamp: DateTime
  address: Address
}

type Address {
  street: String
  city: String
  state: String
  country: String
  postalCode: String
  coordinates: Location
}

type Route {
  points: [RoutePoint!]!
  totalDistance: Float!
  totalDuration: Int!
}

type RoutePoint {
  location: Location!
  etaSeconds: Int!
  distanceFromStart: Float!
}

type NearbyDriver {
  driverId: ID!
  vehicleId: ID!
  location: Location!
  distance: Float!
  rating: Float!
  driver: Driver!
  vehicle: Vehicle!
}
```

### Trip Types

```graphql
enum TripStatus {
  REQUESTED
  MATCHED
  DRIVER_ASSIGNED
  DRIVER_ARRIVING
  DRIVER_ARRIVED
  TRIP_STARTED
  IN_PROGRESS
  COMPLETED
  CANCELLED
  FAILED
}

type Trip {
  id: ID!
  riderId: ID!
  driverId: ID
  vehicleId: ID
  pickupLocation: Location!
  destination: Location!
  status: TripStatus!
  fare: Money
  distance: Float
  duration: Int
  requestedAt: DateTime!
  startedAt: DateTime
  completedAt: DateTime
  cancellationReason: String
  
  # Relationships
  rider: User!
  driver: Driver
  vehicle: Vehicle
  payment: Payment
  events: [TripEvent!]!
  
  # Computed fields
  estimatedFare: Money
  estimatedDuration: Int
  currentDriverLocation: Location
}

type TripEvent {
  id: ID!
  tripId: ID!
  eventType: String!
  data: JSON
  timestamp: DateTime!
  userId: ID
  user: User
}
```

### Pricing Types

```graphql
type Money {
  amount: Int! # amount in cents
  currency: String!
  formatted: String! # formatted string like "$12.34"
}

type PricingFactors {
  baseFare: Float!
  perKmRate: Float!
  perMinuteRate: Float!
  surgeMultiplier: Float!
  bookingFee: Float!
  serviceFee: Float!
}

type FareBreakdown {
  baseFare: Money!
  distanceFare: Money!
  timeFare: Money!
  surgeAmount: Money!
  bookingFee: Money!
  serviceFee: Money!
  discount: Money!
  total: Money!
}

type FareEstimate {
  fareBreakdown: FareBreakdown!
  pricingFactors: PricingFactors!
  validUntil: DateTime!
}

type SurgeInfo {
  multiplier: Float!
  reason: String!
  expiresAt: DateTime!
}
```

### Payment Types

```graphql
enum PaymentMethodType {
  CREDIT_CARD
  DEBIT_CARD
  DIGITAL_WALLET
  CASH
  BANK_TRANSFER
}

enum PaymentStatus {
  PENDING
  PROCESSING
  COMPLETED
  FAILED
  CANCELLED
  REFUNDED
}

type PaymentMethod {
  id: ID!
  userId: ID!
  type: PaymentMethodType!
  lastFour: String!
  brand: String!
  isDefault: Boolean!
  expiresAt: DateTime
  createdAt: DateTime!
}

type Payment {
  id: ID!
  tripId: ID!
  userId: ID!
  paymentMethodId: ID!
  amount: Money!
  status: PaymentStatus!
  gatewayTransactionId: String
  failureReason: String
  createdAt: DateTime!
  updatedAt: DateTime!
  
  # Relationships
  trip: Trip!
  user: User!
  paymentMethod: PaymentMethod!
}
```

### Matching Types

```graphql
enum MatchStatus {
  PENDING
  MATCHED
  ACCEPTED
  REJECTED
  CANCELLED
  EXPIRED
}

type RideRequest {
  id: ID!
  riderId: ID!
  pickupLocation: Location!
  destination: Location!
  vehicleType: VehicleType!
  passengerCount: Int!
  status: MatchStatus!
  createdAt: DateTime!
  expiresAt: DateTime!
  
  # Relationships
  rider: User!
  matches: [DriverMatch!]!
}

type DriverMatch {
  driverId: ID!
  vehicleId: ID!
  currentLocation: Location!
  distanceToPickup: Float!
  etaToPickup: Int!
  rating: Float!
  totalTrips: Int!
  
  # Relationships
  driver: Driver!
  vehicle: Vehicle!
}
```

### Common Types

```graphql
scalar DateTime
scalar JSON

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
  totalCount: Int!
}

interface Node {
  id: ID!
}

type Error {
  code: String!
  message: String!
  field: String
}
```

## Query Types

```graphql
type Query {
  # User queries
  me: User
  user(id: ID!): User
  users(
    first: Int
    after: String
    filter: UserFilter
    sort: UserSort
  ): UserConnection!
  
  # Driver queries
  driver(id: ID!): Driver
  nearbyDrivers(
    location: LocationInput!
    radius: Float = 5.0
    vehicleType: VehicleType
    limit: Int = 10
  ): [NearbyDriver!]!
  
  # Vehicle queries
  vehicle(id: ID!): Vehicle
  vehicles(driverId: ID!): [Vehicle!]!
  
  # Trip queries
  trip(id: ID!): Trip
  trips(
    userId: ID
    status: TripStatus
    first: Int
    after: String
    sort: TripSort
  ): TripConnection!
  myTrips(
    status: TripStatus
    first: Int
    after: String
    sort: TripSort
  ): TripConnection!
  
  # Pricing queries
  fareEstimate(
    pickup: LocationInput!
    destination: LocationInput!
    vehicleType: VehicleType!
    pickupTime: DateTime
    promoCode: String
  ): FareEstimate!
  surgeInfo(
    location: LocationInput!
    vehicleType: VehicleType!
  ): SurgeInfo!
  
  # Payment queries
  payment(id: ID!): Payment
  paymentMethods(userId: ID): [PaymentMethod!]!
  
  # Geospatial queries
  calculateDistance(
    from: LocationInput!
    to: LocationInput!
  ): DistanceResult!
  calculateETA(
    from: LocationInput!
    to: LocationInput!
    vehicleType: VehicleType
  ): ETAResult!
  optimizeRoute(
    start: LocationInput!
    end: LocationInput!
    waypoints: [LocationInput!]
  ): Route!
  
  # Matching queries
  rideRequest(id: ID!): RideRequest
  activeRideRequest(riderId: ID!): RideRequest
}
```

## Mutation Types

```graphql
type Mutation {
  # Authentication mutations
  login(email: String!, password: String!): AuthPayload!
  register(input: RegisterInput!): AuthPayload!
  refreshToken(refreshToken: String!): AuthPayload!
  logout: Boolean!
  
  # User mutations
  updateProfile(input: UpdateProfileInput!): User!
  updateDriverStatus(status: DriverStatus!): Driver!
  updateDriverLocation(location: LocationInput!): Driver!
  
  # Vehicle mutations
  registerVehicle(input: RegisterVehicleInput!): Vehicle!
  updateVehicle(id: ID!, input: UpdateVehicleInput!): Vehicle!
  updateVehicleStatus(id: ID!, status: VehicleStatus!): Vehicle!
  
  # Trip mutations
  requestRide(input: RequestRideInput!): RideRequest!
  cancelRideRequest(id: ID!, reason: String): RideRequest!
  acceptRide(rideRequestId: ID!): Trip!
  rejectRide(rideRequestId: ID!, reason: String): Boolean!
  updateTripStatus(id: ID!, status: TripStatus!): Trip!
  cancelTrip(id: ID!, reason: String): Trip!
  
  # Payment mutations
  addPaymentMethod(input: AddPaymentMethodInput!): PaymentMethod!
  removePaymentMethod(id: ID!): Boolean!
  setDefaultPaymentMethod(id: ID!): PaymentMethod!
  processPayment(input: ProcessPaymentInput!): Payment!
  refundPayment(paymentId: ID!, amount: MoneyInput, reason: String): Payment!
  
  # Pricing mutations
  updateSurgeMultiplier(input: UpdateSurgeInput!): SurgeInfo!
  validatePromoCode(code: String!, tripAmount: MoneyInput!): PromoCodeResult!
}
```

## Subscription Types

```graphql
type Subscription {
  # Trip subscriptions
  tripUpdates(tripId: ID!): TripUpdate!
  myTripUpdates: TripUpdate!
  
  # Driver subscriptions
  driverDispatch(driverId: ID!): DispatchNotification!
  driverLocationUpdates(driverId: ID!): LocationUpdate!
  
  # Pricing subscriptions
  surgeUpdates(location: LocationInput!, vehicleType: VehicleType!): SurgeUpdate!
  fareUpdates(
    pickup: LocationInput!
    destination: LocationInput!
    vehicleType: VehicleType!
  ): FareUpdate!
  
  # Matching subscriptions
  rideRequestUpdates(rideRequestId: ID!): RideRequestUpdate!
  nearbyDriverUpdates(
    location: LocationInput!
    radius: Float!
    vehicleType: VehicleType
  ): NearbyDriverUpdate!
}
```

## Input Types

```graphql
input LocationInput {
  latitude: Float!
  longitude: Float!
  accuracy: Float
}

input RegisterInput {
  email: String!
  password: String!
  firstName: String!
  lastName: String!
  phone: String!
  type: UserType!
}

input UpdateProfileInput {
  firstName: String
  lastName: String
  phone: String
  profileImageUrl: String
}

input RegisterVehicleInput {
  make: String!
  model: String!
  year: Int!
  color: String!
  licensePlate: String!
  type: VehicleType!
  capacity: Int!
}

input UpdateVehicleInput {
  color: String
  status: VehicleStatus
  capacity: Int
}

input RequestRideInput {
  pickupLocation: LocationInput!
  destination: LocationInput!
  vehicleType: VehicleType!
  passengerCount: Int = 1
}

input AddPaymentMethodInput {
  type: PaymentMethodType!
  token: String! # payment gateway token
  setAsDefault: Boolean = false
}

input ProcessPaymentInput {
  tripId: ID!
  paymentMethodId: ID!
  amount: MoneyInput!
}

input MoneyInput {
  amount: Int!
  currency: String!
}

input UpdateSurgeInput {
  location: LocationInput!
  vehicleType: VehicleType!
  multiplier: Float!
  reason: String!
  durationMinutes: Int!
}

input UserFilter {
  type: UserType
  status: UserStatus
  search: String
}

input UserSort {
  field: UserSortField!
  direction: SortDirection!
}

enum UserSortField {
  CREATED_AT
  UPDATED_AT
  FIRST_NAME
  LAST_NAME
}

enum SortDirection {
  ASC
  DESC
}

input TripSort {
  field: TripSortField!
  direction: SortDirection!
}

enum TripSortField {
  CREATED_AT
  STARTED_AT
  COMPLETED_AT
  FARE
}
```

## Connection Types (for Pagination)

```graphql
type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
}

type UserEdge {
  node: User!
  cursor: String!
}

type TripConnection {
  edges: [TripEdge!]!
  pageInfo: PageInfo!
}

type TripEdge {
  node: Trip!
  cursor: String!
}
```

## Response Types

```graphql
type AuthPayload {
  accessToken: String!
  refreshToken: String!
  user: User!
  expiresIn: Int!
}

type DistanceResult {
  distance: Float!
  duration: Int!
}

type ETAResult {
  eta: Int!
  distance: Float!
}

type PromoCodeResult {
  valid: Boolean!
  discount: PromoCodeDiscount
  reason: String
}

type PromoCodeDiscount {
  type: String! # "percentage" or "fixed"
  value: Float!
  maxDiscount: Money
  minTripAmount: Money
}

# Subscription payload types
type TripUpdate {
  trip: Trip!
  event: TripEvent!
}

type DispatchNotification {
  rideRequest: RideRequest!
  expiresInSeconds: Int!
}

type LocationUpdate {
  driverId: ID!
  location: Location!
  timestamp: DateTime!
}

type SurgeUpdate {
  location: Location!
  vehicleType: VehicleType!
  surgeInfo: SurgeInfo!
}

type FareUpdate {
  pickup: Location!
  destination: Location!
  vehicleType: VehicleType!
  fareEstimate: FareEstimate!
}

type RideRequestUpdate {
  rideRequest: RideRequest!
  event: String!
}

type NearbyDriverUpdate {
  drivers: [NearbyDriver!]!
  location: Location!
  timestamp: DateTime!
}
```

## Schema Directives

```graphql
# Authentication directive
directive @auth(requires: UserType) on FIELD_DEFINITION

# Rate limiting directive
directive @rateLimit(max: Int!, window: Int!) on FIELD_DEFINITION

# Caching directive
directive @cacheControl(maxAge: Int, scope: CacheControlScope) on FIELD_DEFINITION | OBJECT

enum CacheControlScope {
  PUBLIC
  PRIVATE
}
```

## Example Usage

### Query Examples

```graphql
# Get current user with driver info
query Me {
  me {
    id
    firstName
    lastName
    type
    driver {
      status
      rating
      currentLocation {
        latitude
        longitude
      }
    }
  }
}

# Get fare estimate
query FareEstimate($pickup: LocationInput!, $destination: LocationInput!) {
  fareEstimate(
    pickup: $pickup
    destination: $destination
    vehicleType: SEDAN
  ) {
    fareBreakdown {
      total {
        formatted
      }
      baseFare {
        formatted
      }
      surgeAmount {
        formatted
      }
    }
    pricingFactors {
      surgeMultiplier
    }
  }
}

# Get trip history
query TripHistory($first: Int!, $after: String) {
  myTrips(first: $first, after: $after, sort: { field: CREATED_AT, direction: DESC }) {
    edges {
      node {
        id
        status
        pickupLocation {
          address {
            street
            city
          }
        }
        destination {
          address {
            street
            city
          }
        }
        fare {
          formatted
        }
        createdAt
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
```

### Mutation Examples

```graphql
# Request a ride
mutation RequestRide($input: RequestRideInput!) {
  requestRide(input: $input) {
    id
    status
    pickupLocation {
      latitude
      longitude
    }
    destination {
      latitude
      longitude
    }
    expiresAt
  }
}

# Update driver status
mutation UpdateDriverStatus($status: DriverStatus!) {
  updateDriverStatus(status: $status) {
    status
    currentLocation {
      latitude
      longitude
    }
  }
}
```

### Subscription Examples

```graphql
# Subscribe to trip updates
subscription TripUpdates($tripId: ID!) {
  tripUpdates(tripId: $tripId) {
    trip {
      id
      status
      driver {
        user {
          firstName
        }
        currentLocation {
          latitude
          longitude
        }
      }
    }
    event {
      eventType
      timestamp
    }
  }
}

# Subscribe to driver dispatch notifications
subscription DriverDispatch($driverId: ID!) {
  driverDispatch(driverId: $driverId) {
    rideRequest {
      id
      pickupLocation {
        address {
          street
          city
        }
      }
      destination {
        address {
          street
          city
        }
      }
      rider {
        firstName
        rating
      }
    }
    expiresInSeconds
  }
}
```

This comprehensive GraphQL schema provides a powerful, type-safe API for all client applications while maintaining clean separation from the underlying microservices architecture. The schema supports real-time features, efficient data fetching, and scalable pagination patterns.