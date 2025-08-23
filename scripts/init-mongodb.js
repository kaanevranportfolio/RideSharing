// Initialize MongoDB database for Geospatial service

// Switch to the rideshare_geo database
db = db.getSiblingDB('rideshare_geo');

// Create driver_locations collection with geospatial index
db.createCollection('driver_locations');

// Create 2dsphere index for geospatial queries
db.driver_locations.createIndex({
    "location": "2dsphere"
}, {
    name: "location_2dsphere"
});

// Create TTL index for automatic expiration
db.driver_locations.createIndex({
    "expires_at": 1
}, {
    expireAfterSeconds: 0,
    name: "expires_at_ttl"
});

// Create compound index for efficient queries
db.driver_locations.createIndex({
    "status": 1,
    "vehicle_type": 1,
    "rating": -1
}, {
    name: "status_vehicle_rating"
});

// Insert sample driver locations for testing
const sampleLocations = [
    {
        driver_id: "00000000-0000-0000-0000-000000000001",
        vehicle_id: "00000000-0000-0000-0000-000000000101",
        location: {
            type: "Point",
            coordinates: [-74.0060, 40.7128], // [longitude, latitude] for GeoJSON
            accuracy: 10.0,
            timestamp: new Date()
        },
        status: "online",
        vehicle_type: "sedan",
        rating: 4.8,
        updated_at: new Date(),
        expires_at: new Date(Date.now() + 5 * 60 * 1000) // 5 minutes from now
    },
    {
        driver_id: "00000000-0000-0000-0000-000000000003",
        vehicle_id: "00000000-0000-0000-0000-000000000102",
        location: {
            type: "Point",
            coordinates: [-73.9851, 40.7589], // [longitude, latitude] for GeoJSON
            accuracy: 15.0,
            timestamp: new Date()
        },
        status: "online",
        vehicle_type: "suv",
        rating: 4.6,
        updated_at: new Date(),
        expires_at: new Date(Date.now() + 5 * 60 * 1000) // 5 minutes from now
    },
    {
        driver_id: "driver_test_001",
        vehicle_id: "vehicle_test_001",
        location: {
            type: "Point",
            coordinates: [-73.9934, 40.7505], // [longitude, latitude] for GeoJSON
            accuracy: 8.0,
            timestamp: new Date()
        },
        status: "online",
        vehicle_type: "sedan",
        rating: 4.9,
        updated_at: new Date(),
        expires_at: new Date(Date.now() + 5 * 60 * 1000) // 5 minutes from now
    }
];

// Insert sample data
db.driver_locations.insertMany(sampleLocations);

print("MongoDB initialization completed for rideshare_geo database");
print("Created collections: driver_locations");
print("Created indexes: location_2dsphere, expires_at_ttl, status_vehicle_rating");
print("Inserted " + sampleLocations.length + " sample driver locations");
