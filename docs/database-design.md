# Database Design and Schema

This document outlines the comprehensive database design for the rideshare platform, including PostgreSQL schemas, MongoDB collections, and Redis data structures.

## Database Architecture Overview

The platform uses a polyglot persistence approach:

- **PostgreSQL**: Transactional data, event sourcing, user management
- **MongoDB**: Geospatial data, location indexing, route optimization
- **Redis**: Caching, real-time state, session management

## PostgreSQL Schema Design

### User Management Database

#### users table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    user_type VARCHAR(20) NOT NULL CHECK (user_type IN ('rider', 'driver', 'admin')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('inactive', 'active', 'suspended', 'banned')),
    profile_image_url TEXT,
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_type ON users(user_type);
CREATE INDEX idx_users_status ON users(status);
```

#### drivers table
```sql
CREATE TABLE drivers (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    license_number VARCHAR(50) UNIQUE NOT NULL,
    license_expiry DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('offline', 'online', 'busy', 'break')),
    rating DECIMAL(3,2) DEFAULT 5.00 CHECK (rating >= 0 AND rating <= 5),
    total_trips INTEGER DEFAULT 0,
    total_earnings_cents BIGINT DEFAULT 0,
    current_latitude DECIMAL(10,8),
    current_longitude DECIMAL(11,8),
    current_location_accuracy DECIMAL(8,2),
    last_location_update TIMESTAMP WITH TIME ZONE,
    background_check_status VARCHAR(20) DEFAULT 'pending',
    background_check_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_drivers_status ON drivers(status);
CREATE INDEX idx_drivers_rating ON drivers(rating);
CREATE INDEX idx_drivers_location ON drivers(current_latitude, current_longitude);
```

#### vehicles table
```sql
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(user_id) ON DELETE CASCADE,
    make VARCHAR(50) NOT NULL,
    model VARCHAR(50) NOT NULL,
    year INTEGER NOT NULL CHECK (year >= 1990 AND year <= EXTRACT(YEAR FROM NOW()) + 1),
    color VARCHAR(30) NOT NULL,
    license_plate VARCHAR(20) UNIQUE NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('sedan', 'suv', 'hatchback', 'luxury', 'van')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('inactive', 'active', 'maintenance', 'retired')),
    capacity INTEGER NOT NULL DEFAULT 4 CHECK (capacity >= 1 AND capacity <= 8),
    insurance_policy_number VARCHAR(100),
    insurance_expiry DATE,
    registration_expiry DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_vehicles_driver_id ON vehicles(driver_id);
CREATE INDEX idx_vehicles_type ON vehicles(vehicle_type);
CREATE INDEX idx_vehicles_status ON vehicles(status);
CREATE INDEX idx_vehicles_license_plate ON vehicles(license_plate);
```

### Trip Management Database

#### trips table
```sql
CREATE TABLE trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id),
    driver_id UUID REFERENCES drivers(user_id),
    vehicle_id UUID REFERENCES vehicles(id),
    
    -- Location data (stored as JSON for flexibility)
    pickup_location JSONB NOT NULL,
    destination JSONB NOT NULL,
    actual_route JSONB, -- actual route taken
    
    -- Trip details
    status VARCHAR(20) NOT NULL DEFAULT 'requested' CHECK (status IN (
        'requested', 'matched', 'driver_assigned', 'driver_arriving', 
        'driver_arrived', 'trip_started', 'in_progress', 'completed', 
        'cancelled', 'failed'
    )),
    
    -- Pricing
    estimated_fare_cents BIGINT,
    actual_fare_cents BIGINT,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Metrics
    estimated_distance_km DECIMAL(8,2),
    actual_distance_km DECIMAL(8,2),
    estimated_duration_seconds INTEGER,
    actual_duration_seconds INTEGER,
    
    -- Timestamps
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    matched_at TIMESTAMP WITH TIME ZONE,
    driver_assigned_at TIMESTAMP WITH TIME ZONE,
    driver_arrived_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Cancellation
    cancelled_by VARCHAR(20), -- 'rider', 'driver', 'system'
    cancellation_reason TEXT,
    
    -- Additional metadata
    passenger_count INTEGER DEFAULT 1,
    special_requests TEXT,
    promo_code VARCHAR(50),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_trips_rider_id ON trips(rider_id);
CREATE INDEX idx_trips_driver_id ON trips(driver_id);
CREATE INDEX idx_trips_status ON trips(status);
CREATE INDEX idx_trips_requested_at ON trips(requested_at);
CREATE INDEX idx_trips_completed_at ON trips(completed_at);
CREATE INDEX idx_trips_pickup_location ON trips USING GIN (pickup_location);
CREATE INDEX idx_trips_destination ON trips USING GIN (destination);
```

#### trip_events table (Event Sourcing)
```sql
CREATE TABLE trip_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    event_version INTEGER NOT NULL,
    user_id UUID REFERENCES users(id),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_trip_events_trip_id ON trip_events(trip_id);
CREATE INDEX idx_trip_events_type ON trip_events(event_type);
CREATE INDEX idx_trip_events_timestamp ON trip_events(timestamp);
CREATE UNIQUE INDEX idx_trip_events_version ON trip_events(trip_id, event_version);
```

### Payment Database

#### payment_methods table
```sql
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('credit_card', 'debit_card', 'digital_wallet', 'cash', 'bank_transfer')),
    provider VARCHAR(50), -- 'stripe', 'paypal', etc.
    provider_payment_method_id VARCHAR(255),
    last_four VARCHAR(4),
    brand VARCHAR(20), -- 'visa', 'mastercard', etc.
    is_default BOOLEAN DEFAULT FALSE,
    expires_at DATE,
    billing_address JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX idx_payment_methods_default ON payment_methods(user_id, is_default);
```

#### payments table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id),
    user_id UUID NOT NULL REFERENCES users(id),
    payment_method_id UUID REFERENCES payment_methods(id),
    
    -- Amount
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'processing', 'completed', 'failed', 'cancelled', 'refunded'
    )),
    
    -- Gateway details
    gateway_provider VARCHAR(50),
    gateway_transaction_id VARCHAR(255),
    gateway_response JSONB,
    
    -- Failure details
    failure_code VARCHAR(50),
    failure_reason TEXT,
    
    -- Refund details
    refunded_amount_cents BIGINT DEFAULT 0,
    refund_reason TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_payments_trip_id ON payments(trip_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_gateway_transaction_id ON payments(gateway_transaction_id);
```

### Pricing Database

#### pricing_rules table
```sql
CREATE TABLE pricing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL,
    city VARCHAR(100),
    
    -- Base pricing
    base_fare_cents BIGINT NOT NULL,
    per_km_rate_cents BIGINT NOT NULL,
    per_minute_rate_cents BIGINT NOT NULL,
    booking_fee_cents BIGINT DEFAULT 0,
    service_fee_cents BIGINT DEFAULT 0,
    
    -- Time-based rules
    time_multipliers JSONB, -- hour-based multipliers
    day_multipliers JSONB,  -- day-of-week multipliers
    
    -- Distance-based rules
    min_fare_cents BIGINT,
    max_fare_cents BIGINT,
    
    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_pricing_rules_vehicle_type ON pricing_rules(vehicle_type);
CREATE INDEX idx_pricing_rules_city ON pricing_rules(city);
CREATE INDEX idx_pricing_rules_active ON pricing_rules(is_active);
```

#### surge_pricing table
```sql
CREATE TABLE surge_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_geohash VARCHAR(20) NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL,
    multiplier DECIMAL(4,2) NOT NULL CHECK (multiplier >= 1.0),
    reason VARCHAR(255),
    demand_level VARCHAR(20) CHECK (demand_level IN ('low', 'medium', 'high', 'very_high')),
    supply_level VARCHAR(20) CHECK (supply_level IN ('low', 'medium', 'high', 'very_high')),
    
    -- Validity
    starts_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_surge_pricing_geohash ON surge_pricing(location_geohash);
CREATE INDEX idx_surge_pricing_vehicle_type ON surge_pricing(vehicle_type);
CREATE INDEX idx_surge_pricing_active ON surge_pricing(is_active, expires_at);
```

#### promo_codes table
```sql
CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    
    -- Discount details
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value DECIMAL(10,2) NOT NULL,
    max_discount_cents BIGINT,
    min_trip_amount_cents BIGINT,
    
    -- Usage limits
    max_uses INTEGER,
    max_uses_per_user INTEGER DEFAULT 1,
    current_uses INTEGER DEFAULT 0,
    
    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Restrictions
    applicable_vehicle_types VARCHAR(255)[], -- array of vehicle types
    applicable_cities VARCHAR(255)[],        -- array of cities
    first_ride_only BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_promo_codes_code ON promo_codes(code);
CREATE INDEX idx_promo_codes_active ON promo_codes(is_active, valid_until);
```

## MongoDB Schema Design

### Geospatial Collections

#### driver_locations collection
```javascript
{
  _id: ObjectId,
  driverId: "uuid",
  location: {
    type: "Point",
    coordinates: [longitude, latitude] // GeoJSON format
  },
  accuracy: 10.5, // meters
  heading: 45.0,  // degrees
  speed: 25.5,    // km/h
  timestamp: ISODate,
  geohash: "9q8yy", // for efficient querying
  isOnline: true,
  isAvailable: true,
  vehicleType: "sedan",
  currentTripId: "uuid" // null if available
}

// Indexes
db.driver_locations.createIndex({ "location": "2dsphere" })
db.driver_locations.createIndex({ "driverId": 1 })
db.driver_locations.createIndex({ "geohash": 1 })
db.driver_locations.createIndex({ "timestamp": 1 })
db.driver_locations.createIndex({ "isOnline": 1, "isAvailable": 1 })
```

#### trip_routes collection
```javascript
{
  _id: ObjectId,
  tripId: "uuid",
  route: {
    type: "LineString",
    coordinates: [[longitude, latitude], ...] // actual path taken
  },
  estimatedRoute: {
    type: "LineString",
    coordinates: [[longitude, latitude], ...] // planned route
  },
  waypoints: [
    {
      location: {
        type: "Point",
        coordinates: [longitude, latitude]
      },
      timestamp: ISODate,
      type: "pickup" | "destination" | "waypoint"
    }
  ],
  distance: 15.5, // km
  duration: 1800,  // seconds
  createdAt: ISODate,
  updatedAt: ISODate
}

// Indexes
db.trip_routes.createIndex({ "tripId": 1 })
db.trip_routes.createIndex({ "route": "2dsphere" })
db.trip_routes.createIndex({ "estimatedRoute": "2dsphere" })
```

#### geofences collection
```javascript
{
  _id: ObjectId,
  name: "Downtown Area",
  type: "city_zone" | "airport" | "surge_zone" | "restricted",
  geometry: {
    type: "Polygon",
    coordinates: [[[longitude, latitude], ...]]
  },
  properties: {
    surgeMultiplier: 1.5,
    restrictions: ["no_pickup", "no_dropoff"],
    specialRules: {}
  },
  isActive: true,
  createdAt: ISODate,
  updatedAt: ISODate
}

// Indexes
db.geofences.createIndex({ "geometry": "2dsphere" })
db.geofences.createIndex({ "type": 1, "isActive": 1 })
```

#### location_history collection (for analytics)
```javascript
{
  _id: ObjectId,
  entityId: "uuid", // driver or trip ID
  entityType: "driver" | "trip",
  locations: [
    {
      coordinates: [longitude, latitude],
      timestamp: ISODate,
      accuracy: 10.5,
      speed: 25.5,
      heading: 45.0
    }
  ],
  date: ISODate, // for partitioning
  createdAt: ISODate
}

// Indexes
db.location_history.createIndex({ "entityId": 1, "date": 1 })
db.location_history.createIndex({ "date": 1 }) // for TTL
```

## Redis Data Structures

### Session Management
```redis
# User sessions
SET session:user:{user_id} "{jwt_token}" EX 3600

# Driver online status
SET driver:online:{driver_id} "true" EX 300
SADD drivers:online:sedan {driver_id1} {driver_id2} ...

# Driver availability
SET driver:available:{driver_id} "true" EX 60
SADD drivers:available:geohash:{geohash} {driver_id1} {driver_id2} ...
```

### Real-time Matching
```redis
# Ride requests
HSET ride_request:{request_id} 
  rider_id {rider_id}
  pickup_lat {latitude}
  pickup_lng {longitude}
  dest_lat {latitude}
  dest_lng {longitude}
  vehicle_type {type}
  status "pending"
  expires_at {timestamp}

# Driver dispatch queue
LPUSH driver:dispatch:{driver_id} {ride_request_id}
EXPIRE driver:dispatch:{driver_id} 300

# Matching state
SET matching:request:{request_id} "searching" EX 300
SADD matching:candidates:{request_id} {driver_id1} {driver_id2} ...
```

### Pricing Cache
```redis
# Surge pricing
HSET surge:geohash:{geohash}:{vehicle_type}
  multiplier {multiplier}
  reason "{reason}"
  expires_at {timestamp}

# Fare estimates cache
SET fare_estimate:{pickup_hash}:{dest_hash}:{vehicle_type} 
  "{fare_breakdown_json}" EX 300

# Promo code validation cache
SET promo:validation:{code}:{user_id} 
  "{validation_result_json}" EX 1800
```

### Real-time Updates
```redis
# Trip status updates
PUBLISH trip:updates:{trip_id} "{trip_update_json}"
PUBLISH driver:location:{driver_id} "{location_update_json}"

# Driver dispatch notifications
PUBLISH driver:dispatch:{driver_id} "{dispatch_notification_json}"

# Surge pricing updates
PUBLISH surge:updates:{geohash} "{surge_update_json}"
```

### Rate Limiting
```redis
# API rate limiting
INCR rate_limit:user:{user_id}:{endpoint}:{window}
EXPIRE rate_limit:user:{user_id}:{endpoint}:{window} {window_seconds}

# Driver location update rate limiting
INCR rate_limit:location:{driver_id}:{minute}
EXPIRE rate_limit:location:{driver_id}:{minute} 60
```

## Database Triggers and Functions

### PostgreSQL Triggers

#### Update timestamps trigger
```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to all tables with updated_at column
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_drivers_updated_at BEFORE UPDATE ON drivers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ... apply to other tables
```

#### Trip event sourcing trigger
```sql
CREATE OR REPLACE FUNCTION create_trip_event()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert event when trip status changes
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO trip_events (trip_id, event_type, event_data, event_version)
        VALUES (
            NEW.id,
            'status_changed',
            jsonb_build_object(
                'old_status', OLD.status,
                'new_status', NEW.status,
                'changed_at', NOW()
            ),
            (SELECT COALESCE(MAX(event_version), 0) + 1 FROM trip_events WHERE trip_id = NEW.id)
        );
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trip_status_event AFTER UPDATE ON trips
    FOR EACH ROW EXECUTE FUNCTION create_trip_event();
```

## Data Retention and Archival

### Retention Policies

```sql
-- Archive old trips (older than 2 years)
CREATE TABLE trips_archive (LIKE trips INCLUDING ALL);

-- Archive old location history
-- MongoDB TTL index for location_history
db.location_history.createIndex(
  { "createdAt": 1 }, 
  { expireAfterSeconds: 7776000 } // 90 days
)

-- Redis key expiration is handled per key
```

### Backup Strategy

```sql
-- Daily backup script
pg_dump rideshare_platform > backup_$(date +%Y%m%d).sql

-- MongoDB backup
mongodump --db rideshare_geo --out /backup/mongo_$(date +%Y%m%d)

-- Redis backup
redis-cli BGSAVE
```

This comprehensive database design provides scalable, efficient data storage with proper indexing, event sourcing capabilities, and real-time data structures optimized for a high-performance rideshare platform.