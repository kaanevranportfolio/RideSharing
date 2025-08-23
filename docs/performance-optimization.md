# Performance Optimization Strategy

This document outlines comprehensive performance optimization strategies for the rideshare platform, covering all aspects from database optimization to real-time processing.

## Performance Goals and SLAs

### Service Level Objectives (SLOs)

#### API Response Times
- **GraphQL API**: 95th percentile < 200ms, 99th percentile < 500ms
- **gRPC Services**: 95th percentile < 100ms, 99th percentile < 300ms
- **Database Queries**: 95th percentile < 50ms, 99th percentile < 200ms
- **Cache Operations**: 95th percentile < 5ms, 99th percentile < 20ms

#### Throughput Requirements
- **Concurrent Users**: 50,000+ active users
- **Requests per Second**: 10,000+ RPS peak load
- **Real-time Updates**: < 1 second latency for location updates
- **Trip Matching**: < 3 seconds for driver-rider matching

#### Availability and Reliability
- **Uptime**: 99.9% availability (8.76 hours downtime/year)
- **Error Rate**: < 0.1% for critical operations
- **Recovery Time**: < 5 minutes for service recovery

## Database Performance Optimization

### PostgreSQL Optimization

#### Connection Pooling

```go
// shared/database/postgres.go
type PostgresConfig struct {
    Host            string
    Port            int
    Database        string
    Username        string
    Password        string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    ConnMaxIdleTime time.Duration
}

func NewPostgresConnection(config *PostgresConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
        config.Host, config.Port, config.Username, config.Password, config.Database)
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Connection pool optimization
    db.SetMaxOpenConns(config.MaxOpenConns)     // 100 for high-load services
    db.SetMaxIdleConns(config.MaxIdleConns)     // 25 idle connections
    db.SetConnMaxLifetime(config.ConnMaxLifetime) // 1 hour
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime) // 15 minutes
    
    return db, nil
}

// Connection pool monitoring
func (db *Database) GetPoolStats() sql.DBStats {
    return db.conn.Stats()
}

func (db *Database) MonitorConnectionPool() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := db.GetPoolStats()
        
        // Log pool metrics
        log.WithFields(log.Fields{
            "open_connections": stats.OpenConnections,
            "in_use":          stats.InUse,
            "idle":            stats.Idle,
            "wait_count":      stats.WaitCount,
            "wait_duration":   stats.WaitDuration,
        }).Info("Database connection pool stats")
        
        // Alert if pool is under stress
        if float64(stats.InUse)/float64(stats.OpenConnections) > 0.8 {
            log.Warn("Database connection pool utilization high")
        }
    }
}
```

#### Query Optimization

```sql
-- Optimized indexes for common queries
-- User service indexes
CREATE INDEX CONCURRENTLY idx_users_email_active ON users(email) WHERE status = 'active';
CREATE INDEX CONCURRENTLY idx_drivers_online_location ON drivers(status, current_latitude, current_longitude) 
    WHERE status IN ('online', 'busy');

-- Trip service indexes
CREATE INDEX CONCURRENTLY idx_trips_rider_status_date ON trips(rider_id, status, requested_at DESC);
CREATE INDEX CONCURRENTLY idx_trips_driver_status_date ON trips(driver_id, status, requested_at DESC);
CREATE INDEX CONCURRENTLY idx_trips_location_pickup ON trips USING GIST(pickup_location);
CREATE INDEX CONCURRENTLY idx_trips_location_destination ON trips USING GIST(destination);

-- Partial indexes for active data
CREATE INDEX CONCURRENTLY idx_trips_active ON trips(id, status, requested_at) 
    WHERE status IN ('requested', 'matched', 'driver_assigned', 'in_progress');

-- Composite indexes for complex queries
CREATE INDEX CONCURRENTLY idx_trips_composite ON trips(rider_id, status, requested_at DESC, fare_cents);

-- Query optimization examples
-- Before: Slow query
SELECT * FROM trips WHERE rider_id = $1 ORDER BY requested_at DESC LIMIT 10;

-- After: Optimized with covering index
CREATE INDEX CONCURRENTLY idx_trips_rider_covering ON trips(rider_id, requested_at DESC) 
    INCLUDE (id, status, pickup_location, destination, fare_cents);

-- Materialized views for analytics
CREATE MATERIALIZED VIEW trip_analytics AS
SELECT 
    DATE_TRUNC('hour', requested_at) as hour,
    COUNT(*) as total_trips,
    AVG(fare_cents) as avg_fare,
    AVG(actual_duration_seconds) as avg_duration,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_trips
FROM trips 
WHERE requested_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', requested_at);

-- Refresh materialized view periodically
CREATE OR REPLACE FUNCTION refresh_trip_analytics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY trip_analytics;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh every 15 minutes
SELECT cron.schedule('refresh-analytics', '*/15 * * * *', 'SELECT refresh_trip_analytics();');
```

#### Database Partitioning

```sql
-- Partition trips table by date for better performance
CREATE TABLE trips_partitioned (
    id UUID DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL,
    driver_id UUID,
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- other columns...
    PRIMARY KEY (id, requested_at)
) PARTITION BY RANGE (requested_at);

-- Create monthly partitions
CREATE TABLE trips_2024_01 PARTITION OF trips_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE trips_2024_02 PARTITION OF trips_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Automatic partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name text, start_date date)
RETURNS void AS $$
DECLARE
    partition_name text;
    end_date date;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + interval '1 month';
    
    EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;
```

### MongoDB Optimization

#### Geospatial Query Optimization

```javascript
// Optimized geospatial indexes
db.driver_locations.createIndex(
    { "location": "2dsphere", "isOnline": 1, "isAvailable": 1 },
    { 
        name: "location_availability_idx",
        background: true 
    }
);

// Compound index for efficient filtering
db.driver_locations.createIndex(
    { 
        "geohash": 1, 
        "vehicleType": 1, 
        "isOnline": 1, 
        "isAvailable": 1,
        "timestamp": -1 
    },
    { 
        name: "geohash_filter_idx",
        background: true 
    }
);

// Optimized aggregation pipeline for nearby drivers
const nearbyDriversPipeline = [
    {
        $geoNear: {
            near: { type: "Point", coordinates: [longitude, latitude] },
            distanceField: "distance",
            maxDistance: radiusInMeters,
            spherical: true,
            query: {
                isOnline: true,
                isAvailable: true,
                vehicleType: vehicleType,
                timestamp: { $gte: new Date(Date.now() - 5 * 60 * 1000) } // 5 minutes
            }
        }
    },
    {
        $sort: { distance: 1, rating: -1 }
    },
    {
        $limit: limit
    },
    {
        $project: {
            driverId: 1,
            location: 1,
            distance: 1,
            vehicleType: 1,
            rating: 1,
            timestamp: 1
        }
    }
];

// Use read preference for geospatial queries
db.driver_locations.aggregate(nearbyDriversPipeline, {
    readPreference: "secondaryPreferred",
    maxTimeMS: 5000
});
```

#### MongoDB Connection Optimization

```go
// shared/database/mongodb.go
type MongoConfig struct {
    URI                string
    Database           string
    MaxPoolSize        uint64
    MinPoolSize        uint64
    MaxConnIdleTime    time.Duration
    MaxConnecting      uint64
    ConnectTimeout     time.Duration
    ServerSelectionTimeout time.Duration
}

func NewMongoClient(config *MongoConfig) (*mongo.Client, error) {
    opts := options.Client().
        ApplyURI(config.URI).
        SetMaxPoolSize(config.MaxPoolSize).        // 100 connections
        SetMinPoolSize(config.MinPoolSize).        // 10 connections
        SetMaxConnIdleTime(config.MaxConnIdleTime). // 30 minutes
        SetMaxConnecting(config.MaxConnecting).     // 10 concurrent connections
        SetConnectTimeout(config.ConnectTimeout).   // 10 seconds
        SetServerSelectionTimeout(config.ServerSelectionTimeout). // 5 seconds
        SetReadPreference(readpref.SecondaryPreferred()). // Read from secondaries
        SetWriteConcern(writeconcern.New(writeconcern.WMajority())). // Write majority
        SetReadConcern(readconcern.Majority()) // Read majority
    
    client, err := mongo.Connect(context.Background(), opts)
    if err != nil {
        return nil, err
    }
    
    return client, nil
}

// Optimized geospatial queries
func (r *GeoRepository) FindNearbyDrivers(ctx context.Context, location *Location, radius float64, vehicleType string, limit int) ([]*NearbyDriver, error) {
    collection := r.client.Database(r.database).Collection("driver_locations")
    
    // Use context with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    pipeline := mongo.Pipeline{
        {{Key: "$geoNear", Value: bson.D{
            {Key: "near", Value: bson.D{
                {Key: "type", Value: "Point"},
                {Key: "coordinates", Value: bson.A{location.Longitude, location.Latitude}},
            }},
            {Key: "distanceField", Value: "distance"},
            {Key: "maxDistance", Value: radius * 1000}, // Convert km to meters
            {Key: "spherical", Value: true},
            {Key: "query", Value: bson.D{
                {Key: "isOnline", Value: true},
                {Key: "isAvailable", Value: true},
                {Key: "vehicleType", Value: vehicleType},
                {Key: "timestamp", Value: bson.D{
                    {Key: "$gte", Value: time.Now().Add(-5 * time.Minute)},
                }},
            }},
        }}},
        {{Key: "$sort", Value: bson.D{
            {Key: "distance", Value: 1},
            {Key: "rating", Value: -1},
        }}},
        {{Key: "$limit", Value: limit}},
    }
    
    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var drivers []*NearbyDriver
    if err := cursor.All(ctx, &drivers); err != nil {
        return nil, err
    }
    
    return drivers, nil
}
```

## Caching Strategy

### Multi-Level Caching Architecture

```go
// shared/cache/manager.go
type CacheManager struct {
    l1Cache *ristretto.Cache // In-memory cache
    l2Cache *redis.Client    // Redis cache
    l3Cache *Database        // Database fallback
}

type CacheConfig struct {
    L1MaxCost     int64
    L1NumCounters int64
    L2TTL         time.Duration
    L3TTL         time.Duration
}

func NewCacheManager(config *CacheConfig, redisClient *redis.Client, db *Database) (*CacheManager, error) {
    l1Cache, err := ristretto.NewCache(&ristretto.Config{
        NumCounters: config.L1NumCounters, // 1M counters
        MaxCost:     config.L1MaxCost,     // 100MB
        BufferItems: 64,
    })
    if err != nil {
        return nil, err
    }
    
    return &CacheManager{
        l1Cache: l1Cache,
        l2Cache: redisClient,
        l3Cache: db,
    }, nil
}

func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, error) {
    // L1 Cache (In-memory)
    if value, found := cm.l1Cache.Get(key); found {
        cacheHits.WithLabelValues("l1").Inc()
        return value, nil
    }
    
    // L2 Cache (Redis)
    if value, err := cm.l2Cache.Get(ctx, key).Result(); err == nil {
        cacheHits.WithLabelValues("l2").Inc()
        
        // Populate L1 cache
        cm.l1Cache.Set(key, value, 1)
        return value, nil
    }
    
    cacheMisses.Inc()
    return nil, ErrCacheMiss
}

func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    // Set in L1 cache
    cm.l1Cache.Set(key, value, 1)
    
    // Set in L2 cache
    return cm.l2Cache.Set(ctx, key, value, ttl).Err()
}

// Cache-aside pattern implementation
func (cm *CacheManager) GetOrSet(ctx context.Context, key string, ttl time.Duration, fetchFunc func() (interface{}, error)) (interface{}, error) {
    // Try to get from cache
    if value, err := cm.Get(ctx, key); err == nil {
        return value, nil
    }
    
    // Fetch from source
    value, err := fetchFunc()
    if err != nil {
        return nil, err
    }
    
    // Set in cache
    cm.Set(ctx, key, value, ttl)
    
    return value, nil
}
```

### Service-Specific Caching

```go
// services/pricing-service/internal/cache/pricing_cache.go
type PricingCache struct {
    cache *CacheManager
}

func (pc *PricingCache) GetFareEstimate(ctx context.Context, req *FareRequest) (*FareEstimate, error) {
    key := fmt.Sprintf("fare:%s:%s:%s:%f:%f", 
        req.VehicleType, 
        req.PickupLocation.Hash(), 
        req.Destination.Hash(),
        req.SurgeMultiplier,
        req.Distance)
    
    return pc.cache.GetOrSet(ctx, key, 5*time.Minute, func() (interface{}, error) {
        return pc.calculateFareEstimate(ctx, req)
    }).(*FareEstimate), nil
}

func (pc *PricingCache) GetSurgeMultiplier(ctx context.Context, location *Location, vehicleType string) (float64, error) {
    geohash := location.Geohash(7) // 7-character precision
    key := fmt.Sprintf("surge:%s:%s", geohash, vehicleType)
    
    result, err := pc.cache.GetOrSet(ctx, key, 1*time.Minute, func() (interface{}, error) {
        return pc.calculateSurgeMultiplier(ctx, location, vehicleType)
    })
    
    if err != nil {
        return 1.0, err // Default to no surge
    }
    
    return result.(float64), nil
}

// Cache warming for frequently accessed data
func (pc *PricingCache) WarmCache(ctx context.Context) {
    // Warm surge pricing for major geohashes
    majorGeohashes := []string{"9q8yy", "9q8yz", "9q8yp", "9q8yr"} // San Francisco area
    vehicleTypes := []string{"sedan", "suv", "luxury"}
    
    for _, geohash := range majorGeohashes {
        for _, vehicleType := range vehicleTypes {
            location := geohash.Decode(geohash)
            pc.GetSurgeMultiplier(ctx, location, vehicleType)
        }
    }
}
```

## Real-Time Performance Optimization

### WebSocket Connection Management

```go
// services/api-gateway/internal/websocket/manager.go
type ConnectionManager struct {
    connections map[string]*websocket.Conn
    broadcast   chan []byte
    register    chan *Client
    unregister  chan *Client
    mutex       sync.RWMutex
}

type Client struct {
    ID         string
    UserID     string
    UserType   string
    Connection *websocket.Conn
    Send       chan []byte
    LastPing   time.Time
}

func (cm *ConnectionManager) Run() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case client := <-cm.register:
            cm.mutex.Lock()
            cm.connections[client.ID] = client.Connection
            cm.mutex.Unlock()
            
            activeConnections.Inc()
            
        case client := <-cm.unregister:
            cm.mutex.Lock()
            if _, ok := cm.connections[client.ID]; ok {
                delete(cm.connections, client.ID)
                close(client.Send)
            }
            cm.mutex.Unlock()
            
            activeConnections.Dec()
            
        case message := <-cm.broadcast:
            cm.mutex.RLock()
            for clientID, conn := range cm.connections {
                select {
                case conn.Send <- message:
                default:
                    // Client buffer full, disconnect
                    delete(cm.connections, clientID)
                    close(conn.Send)
                }
            }
            cm.mutex.RUnlock()
            
        case <-ticker.C:
            // Cleanup stale connections
            cm.cleanupStaleConnections()
        }
    }
}

func (cm *ConnectionManager) cleanupStaleConnections() {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    now := time.Now()
    for clientID, client := range cm.connections {
        if now.Sub(client.LastPing) > 60*time.Second {
            delete(cm.connections, clientID)
            close(client.Send)
            staleConnectionsCleanup.Inc()
        }
    }
}

// Optimized message broadcasting
func (cm *ConnectionManager) BroadcastToUsers(userIDs []string, message []byte) {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    userIDSet := make(map[string]bool)
    for _, userID := range userIDs {
        userIDSet[userID] = true
    }
    
    for _, client := range cm.connections {
        if userIDSet[client.UserID] {
            select {
            case client.Send <- message:
                messagesSent.Inc()
            default:
                messagesDropped.Inc()
            }
        }
    }
}
```

### Event Streaming Optimization

```go
// shared/events/stream.go
type EventStream struct {
    kafka    *kafka.Writer
    redis    *redis.Client
    buffer   chan *Event
    batchSize int
    flushInterval time.Duration
}

func NewEventStream(kafkaWriter *kafka.Writer, redisClient *redis.Client) *EventStream {
    es := &EventStream{
        kafka:         kafkaWriter,
        redis:         redisClient,
        buffer:        make(chan *Event, 10000), // Large buffer
        batchSize:     100,
        flushInterval: 100 * time.Millisecond,
    }
    
    go es.batchProcessor()
    return es
}

func (es *EventStream) PublishEvent(event *Event) error {
    select {
    case es.buffer <- event:
        return nil
    default:
        // Buffer full, publish directly
        return es.publishSingle(event)
    }
}

func (es *EventStream) batchProcessor() {
    ticker := time.NewTicker(es.flushInterval)
    defer ticker.Stop()
    
    batch := make([]*Event, 0, es.batchSize)
    
    for {
        select {
        case event := <-es.buffer:
            batch = append(batch, event)
            
            if len(batch) >= es.batchSize {
                es.publishBatch(batch)
                batch = batch[:0] // Reset slice
            }
            
        case <-ticker.C:
            if len(batch) > 0 {
                es.publishBatch(batch)
                batch = batch[:0]
            }
        }
    }
}

func (es *EventStream) publishBatch(events []*Event) error {
    messages := make([]kafka.Message, len(events))
    
    for i, event := range events {
        eventData, _ := json.Marshal(event)
        messages[i] = kafka.Message{
            Key:   []byte(event.AggregateID),
            Value: eventData,
            Time:  event.Timestamp,
        }
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    return es.kafka.WriteMessages(ctx, messages...)
}
```

## Service Performance Optimization

### gRPC Performance Tuning

```go
// shared/grpc/server.go
func NewOptimizedGRPCServer() *grpc.Server {
    // Connection keepalive parameters
    kaep := keepalive.EnforcementPolicy{
        MinTime:             5 * time.Second,
        PermitWithoutStream: true,
    }
    
    kasp := keepalive.ServerParameters{
        MaxConnectionIdle:     15 * time.Second,
        MaxConnectionAge:      30 * time.Second,
        MaxConnectionAgeGrace: 5 * time.Second,
        Time:                  5 * time.Second,
        Timeout:               1 * time.Second,
    }
    
    // Server options for performance
    opts := []grpc.ServerOption{
        grpc.KeepaliveEnforcementPolicy(kaep),
        grpc.KeepaliveParams(kasp),
        grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
        grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
        grpc.MaxConcurrentStreams(1000),
        grpc.ConnectionTimeout(5 * time.Second),
        grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
            grpc_recovery.UnaryServerInterceptor(),
            grpc_prometheus.UnaryServerInterceptor,
            grpc_ratelimit.UnaryServerInterceptor(rateLimiter),
        )),
        grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
            grpc_recovery.StreamServerInterceptor(),
            grpc_prometheus.StreamServerInterceptor,
        )),
    }
    
    return grpc.NewServer(opts...)
}

// Client connection pooling
type GRPCClientPool struct {
    connections map[string]*grpc.ClientConn
    mutex       sync.RWMutex
    maxConns    int
}

func (pool *GRPCClientPool) GetConnection(address string) (*grpc.ClientConn, error) {
    pool.mutex.RLock()
    if conn, exists := pool.connections[address]; exists {
        pool.mutex.RUnlock()
        return conn, nil
    }
    pool.mutex.RUnlock()
    
    pool.mutex.Lock()
    defer pool.mutex.Unlock()
    
    // Double-check pattern
    if conn, exists := pool.connections[address]; exists {
        return conn, nil
    }
    
    // Create new connection with optimization
    conn, err := grpc.Dial(address,
        grpc.WithInsecure(),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             time.Second,
            PermitWithoutStream: true,
        }),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(4*1024*1024),
            grpc.MaxCallSendMsgSize(4*1024*1024),
        ),
    )
    
    if err != nil {
        return nil, err
    }
    
    pool.connections[address] = conn
    return conn, nil
}
```

### GraphQL Performance Optimization

```go
// services/api-gateway/internal/graphql/optimization.go
type DataLoader struct {
    userLoader    *dataloader.Loader
    vehicleLoader *dataloader.Loader
    tripLoader    *dataloader.Loader
}

func NewDataLoader(userService UserServiceClient, vehicleService VehicleServiceClient, tripService TripServiceClient) *DataLoader {
    return &DataLoader{
        userLoader: dataloader.NewBatchedLoader(
            func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
                return batchLoadUsers(ctx, userService, keys)
            },
            dataloader.WithCache(&dataloader.NoCache{}), // Use external cache
            dataloader.WithBatchCapacity(100),
            dataloader.WithWait(1*time.Millisecond),
        ),
        vehicleLoader: dataloader.NewBatchedLoader(
            func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
                return batchLoadVehicles(ctx, vehicleService, keys)
            },
            dataloader.WithBatchCapacity(100),
            dataloader.WithWait(1*time.Millisecond),
        ),
        tripLoader: dataloader.NewBatchedLoader(
            func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
                return batchLoadTrips(ctx, tripService, keys)
            },
            dataloader.WithBatchCapacity(100),
            dataloader.WithWait(1*time.Millisecond),
        ),
    }
}

func batchLoadUsers(ctx context.Context, service UserServiceClient, keys dataloader.Keys) []*dataloader.Result {
    userIDs := make([]string, len(keys))
    for i, key := range keys {
        userIDs[i] = key.String()
    }
    
    // Batch request to user service
    resp, err := service.GetUsersBatch(ctx, &GetUsersBatchRequest{
        UserIds: userIDs,
    })
    
    if err != nil {
        // Return error for all keys
        results := make([]*dataloader.Result, len(keys))
        for i := range results {
            results[i] = &dataloader.Result{Error: err}
        }
        return results
    }
    
    // Map results back to keys
    userMap := make(map[string]*User)
    for _, user := range resp.Users {
        userMap[user.Id] = user
    }
    
    results := make([]*dataloader.Result, len(keys))
    for i, key := range keys {
        if user, exists := userMap[key.String()]; exists {
            results[i] = &dataloader.Result{Data: user}
        } else {
            results[i] = &dataloader.Result{Error: errors.New("user not found")}
        }
    }
    
    return results
}

// Query complexity analysis
func QueryComplexityMiddleware(maxComplexity int) gin.HandlerFunc {
    return func(c *gin.Context) {
        query := c.PostForm("query")
        if query == "" {
            c.Next()
            return
        }
        
        complexity := calculateQueryComplexity(query)
        if complexity > maxComplexity {
            c.JSON(400, gin.H{
                "error": "Query too complex",
                "complexity": complexity,
                "max_allowed": maxComplexity,
            })
            c.Abort()
            return
        }
        
        c.Set("query_complexity", complexity)
        c.Next()
    }
}
```

## Monitoring and Performance Metrics

### Custom Metrics Collection

```go
// shared/metrics/performance.go
var (
    // HTTP metrics
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    // Database metrics
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
        },
        []string{"operation", "table"},
    )
    
    // Cache metrics
    cacheHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Total number of cache hits",
        },
        []string{"cache_level"},
    )
    
    cacheMisses = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "cache_misses_total",
            Help: "Total number of cache misses",
        },
    )
    
    // Business metrics
    activeTrips = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_trips_total",
            Help: "Total number of active trips",
        },
    )
    
    onlineDrivers = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "online_drivers_total",
            Help: "Total number of online drivers",
        },
        []string{"vehicle_type"},
    )
    
    matchingLatency = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "matching_latency_seconds",
            Help:    "Driver-rider matching latency in seconds",
            Buckets: []float64{0.1, 0.5, 1,