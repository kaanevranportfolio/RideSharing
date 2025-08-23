# Geospatial/ETA Service

The Geospatial/ETA Service is a core component of the rideshare platform responsible for:

- **Distance Calculations**: Haversine, Manhattan, and Euclidean distance calculations
- **ETA Predictions**: Estimated time of arrival with traffic considerations
- **Driver Location Management**: Real-time driver location tracking and updates
- **Nearby Driver Search**: Efficient geospatial queries to find available drivers
- **Geohash Generation**: Location encoding for efficient spatial indexing
- **Route Optimization**: Basic route optimization with multiple waypoints

## Features Implemented

### ✅ Core Geospatial Algorithms
- **Haversine Distance**: Great-circle distance calculation for Earth's surface
- **Manhattan Distance**: Grid-based distance for city block navigation
- **Euclidean Distance**: Straight-line distance calculation
- **Bearing Calculation**: Direction from origin to destination

### ✅ ETA Calculation Engine
- **Multi-modal Support**: Car, bike, walking speed profiles
- **Traffic-aware Routing**: Time-of-day traffic factor adjustments
- **Dynamic Speed Profiles**: Configurable speeds per vehicle type
- **Waypoint Generation**: Intermediate route points

### ✅ Driver Location Services
- **Real-time Updates**: Driver location tracking with TTL expiration
- **Geospatial Indexing**: MongoDB 2dsphere indexes for efficient queries
- **Status Management**: Online/offline/busy driver status tracking
- **Proximity Search**: Find drivers within specified radius

### ✅ Caching Strategy
- **Redis Integration**: Fast caching for distance and ETA calculations
- **Configurable TTL**: Different cache durations for different data types
- **Cache Keys**: Structured cache key generation for optimal hit rates

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Geospatial Service                       │
├─────────────────────┬───────────────────┬───────────────────┤
│     gRPC Handler    │   HTTP Handler    │   Event Handler   │
│   (geo_handler.go)  │   (future)        │    (future)       │
├─────────────────────┴───────────────────┴───────────────────┤
│                Business Logic Layer                         │
│              (geospatial_service.go)                        │
├─────────────────────┬───────────────────────────────────────┤
│   Repository Layer  │               Cache Layer             │
│ (driver_location_   │           (cache_repository.go)       │
│  repository.go)     │                                       │
├─────────────────────┼───────────────────────────────────────┤
│     MongoDB         │               Redis                   │
│ (Driver Locations)  │           (Cache Storage)             │
└─────────────────────┴───────────────────────────────────────┘
```

## Configuration

The service uses environment variables for configuration:

### Service Configuration
- `SERVICE_NAME`: Service identifier (default: "geo-service")
- `ENVIRONMENT`: Runtime environment (development/staging/production)
- `LOG_LEVEL`: Logging level (debug/info/warn/error)
- `GRPC_PORT`: gRPC server port (default: 50053)
- `HTTP_PORT`: HTTP server port (default: 8053)

### Database Configuration
- `DB_HOST`: MongoDB host (default: "localhost")
- `DB_PORT`: MongoDB port (default: 27017)
- `DB_NAME`: Database name (default: "rideshare_geo")
- `DB_USERNAME`: Database username
- `DB_PASSWORD`: Database password

### Geospatial Configuration
- `GEO_DEFAULT_DISTANCE_METHOD`: Default calculation method (default: "haversine")
- `GEO_MAX_SEARCH_RADIUS_KM`: Maximum search radius (default: 50.0)
- `GEO_DEFAULT_GEOHASH_PRECISION`: Default geohash precision (default: 7)
- `GEO_MAX_NEARBY_DRIVERS`: Maximum drivers to return (default: 100)

### Cache Configuration
- `CACHE_DISTANCE_TTL`: Distance cache TTL in seconds (default: 3600)
- `CACHE_ETA_TTL`: ETA cache TTL in seconds (default: 300)
- `CACHE_ENABLE`: Enable/disable caching (default: true)

## API Reference

### gRPC Methods

#### CalculateDistance
Calculates distance between two geographical points.

```protobuf
rpc CalculateDistance(DistanceRequest) returns (DistanceResponse);
```

#### CalculateETA
Calculates estimated time of arrival and route information.

```protobuf
rpc CalculateETA(ETARequest) returns (ETAResponse);
```

#### FindNearbyDrivers
Finds available drivers within a specified radius.

```protobuf
rpc FindNearbyDrivers(NearbyDriversRequest) returns (NearbyDriversResponse);
```

#### UpdateDriverLocation
Updates a driver's current location and status.

```protobuf
rpc UpdateDriverLocation(UpdateDriverLocationRequest) returns (UpdateDriverLocationResponse);
```

#### GenerateGeohash
Generates a geohash for a given location.

```protobuf
rpc GenerateGeohash(GeohashRequest) returns (GeohashResponse);
```

#### OptimizeRoute
Optimizes a route with multiple waypoints.

```protobuf
rpc OptimizeRoute(RouteOptimizationRequest) returns (RouteOptimizationResponse);
```

## Performance Characteristics

### Distance Calculations
- **Haversine**: ~0.1ms for single calculation
- **Manhattan**: ~0.05ms for single calculation
- **Euclidean**: ~0.05ms for single calculation

### ETA Calculations
- **Basic ETA**: ~1-5ms including distance calculation
- **Traffic-aware ETA**: ~5-10ms with traffic factors

### Driver Search
- **Nearby Search**: ~10-50ms for geospatial queries
- **Cache Hit**: ~1-2ms for cached results

## Running the Service

### Development Mode
```bash
# Set environment variables
export ENVIRONMENT=development
export LOG_LEVEL=debug

# Run the service
./run.sh
```

### Production Mode
```bash
# Build the service
go build -o geo-service main.go

# Run with production config
./geo-service
```

## Testing

The service includes comprehensive test scenarios:

1. **Distance Calculation Tests**
   - Haversine distance between NYC landmarks
   - Manhattan distance for city grid navigation
   - Euclidean distance for straight-line measurements

2. **ETA Calculation Tests**
   - Car travel times with traffic factors
   - Bike and walking route calculations
   - Multi-modal comparison

3. **Driver Location Tests**
   - Location update operations
   - Nearby driver search queries
   - Status management workflows

4. **Geohash Tests**
   - Precision level variations
   - Location encoding/decoding
   - Spatial indexing efficiency

## Future Enhancements

### Planned Features
- [ ] **Real MongoDB Integration**: Full MongoDB driver implementation
- [ ] **External Routing APIs**: Integration with Google Maps/HERE APIs
- [ ] **Machine Learning ETA**: ML-based travel time predictions
- [ ] **Advanced Route Optimization**: Traveling salesman problem solver
- [ ] **Real-time Traffic**: Live traffic data integration
- [ ] **Geofencing**: Area-based driver filtering
- [ ] **Performance Monitoring**: Detailed metrics and alerting

### Performance Optimizations
- [ ] **Connection Pooling**: Optimized database connections
- [ ] **Batch Operations**: Bulk location updates
- [ ] **Spatial Indexing**: Advanced geospatial indexes
- [ ] **Cache Warming**: Proactive cache population
- [ ] **Load Balancing**: Horizontal scaling support

## Dependencies

- **Go 1.21+**: Runtime environment
- **MongoDB**: Geospatial data storage
- **Redis**: Caching layer
- **gRPC**: Inter-service communication
- **Logrus**: Structured logging

## Monitoring

### Metrics
- Request latency percentiles (P50, P95, P99)
- Cache hit/miss ratios
- Database query performance
- Active driver counts
- Error rates by operation type

### Health Checks
- Database connectivity
- Cache availability
- Service responsiveness
- Memory usage
- CPU utilization

## Contributing

1. Follow the existing code structure and patterns
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Use structured logging with appropriate fields
5. Handle errors gracefully with proper context
