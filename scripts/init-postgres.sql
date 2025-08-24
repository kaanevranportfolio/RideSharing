-- Initialize PostgreSQL database for User and Vehicle services

-- Create users table
CREATE TABLE IF NOT EXISTS users (
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

-- Create indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_type ON users(user_type);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- Create drivers table
CREATE TABLE IF NOT EXISTS drivers (
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

-- Create indexes for drivers table
CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);
CREATE INDEX IF NOT EXISTS idx_drivers_rating ON drivers(rating);
CREATE INDEX IF NOT EXISTS idx_drivers_location ON drivers(current_latitude, current_longitude);

-- Create vehicles table
CREATE TABLE IF NOT EXISTS vehicles (
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

-- Create indexes for vehicles table
CREATE INDEX IF NOT EXISTS idx_vehicles_driver_id ON vehicles(driver_id);
CREATE INDEX IF NOT EXISTS idx_vehicles_type ON vehicles(vehicle_type);
CREATE INDEX IF NOT EXISTS idx_vehicles_status ON vehicles(status);
CREATE INDEX IF NOT EXISTS idx_vehicles_license_plate ON vehicles(license_plate);

-- Insert sample data for testing
INSERT INTO users (id, email, phone, password_hash, first_name, last_name, user_type, status)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'john.driver@example.com', '+1234567890', '$2a$10$example', 'John', 'Driver', 'driver', 'active'),
    ('00000000-0000-0000-0000-000000000002', 'jane.rider@example.com', '+1234567891', '$2a$10$example', 'Jane', 'Rider', 'rider', 'active'),
    ('00000000-0000-0000-0000-000000000003', 'mike.driver@example.com', '+1234567892', '$2a$10$example', 'Mike', 'Driver', 'driver', 'active')
ON CONFLICT (email) DO NOTHING;

INSERT INTO drivers (user_id, license_number, license_expiry, status, rating, current_latitude, current_longitude)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'DL123456789', '2025-12-31', 'online', 4.8, 40.7128, -74.0060),
    ('00000000-0000-0000-0000-000000000003', 'DL987654321', '2025-06-30', 'online', 4.6, 40.7589, -73.9851)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO vehicles (id, driver_id, make, model, year, color, license_plate, vehicle_type, capacity)
VALUES 
    ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'Toyota', 'Camry', 2022, 'Silver', 'ABC123', 'sedan', 4),
    ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000003', 'Honda', 'CR-V', 2023, 'Blue', 'XYZ789', 'suv', 5)
ON CONFLICT (license_plate) DO NOTHING;

-- Create trips table
CREATE TABLE IF NOT EXISTS trips (
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

-- Create indexes for trips table
CREATE INDEX IF NOT EXISTS idx_trips_rider_id ON trips(rider_id);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id ON trips(driver_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
CREATE INDEX IF NOT EXISTS idx_trips_requested_at ON trips(requested_at);
CREATE INDEX IF NOT EXISTS idx_trips_completed_at ON trips(completed_at);
