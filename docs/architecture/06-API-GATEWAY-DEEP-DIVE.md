# 🚪 API GATEWAY - DEEP DIVE

## 📋 Overview
The **API Gateway** serves as the single entry point for all client applications (mobile apps, web clients, admin panels). It provides a unified GraphQL interface that aggregates data from all backend microservices, handles authentication, and manages real-time subscriptions.

---

## 🎯 Core Responsibilities

### **1. Request Orchestration**
```go
type APIGateway struct {
    grpcClients    *GRPCClientManager
    graphqlServer  *GraphQLServer
    authMiddleware *AuthenticationMiddleware
    rateLimiter    *RateLimiter
    websocketHub   *WebSocketHub
}

// The gateway orchestrates requests across multiple services
func (gw *APIGateway) ProcessRideRequest(ctx context.Context, req *RideRequest) (*RideResponse, error) {
    // 1. Authenticate user
    user, err := gw.authMiddleware.ValidateToken(ctx, req.AuthToken)
    if err != nil {
        return nil, fmt.Errorf("authentication failed: %v", err)
    }
    
    // 2. Get price estimate (parallel call)
    priceChan := make(chan *PriceEstimate, 1)
    go func() {
        price, err := gw.grpcClients.PricingService.CalculatePrice(ctx, req)
        if err != nil {
            priceChan <- &PriceEstimate{Error: err}
        } else {
            priceChan <- price
        }
    }()
    
    // 3. Find nearby drivers (parallel call)
    driversChan := make(chan *NearbyDrivers, 1)
    go func() {
        drivers, err := gw.grpcClients.GeoService.FindNearbyDrivers(ctx, req.PickupLocation)
        if err != nil {
            driversChan <- &NearbyDrivers{Error: err}
        } else {
            driversChan <- drivers
        }
    }()
    
    // 4. Wait for both responses
    price := <-priceChan
    drivers := <-driversChan
    
    // 5. Create trip if drivers available
    if len(drivers.Available) > 0 {
        trip, err := gw.grpcClients.TripService.CreateTrip(ctx, &CreateTripRequest{
            UserID:      user.ID,
            Pickup:      req.PickupLocation,
            Destination: req.DestinationLocation,
            PriceEstimate: price.Amount,
        })
        if err != nil {
            return nil, err
        }
        
        return &RideResponse{
            TripID:          trip.ID,
            EstimatedPrice:  price.Amount,
            AvailableDrivers: len(drivers.Available),
            ETA:            drivers.AverageETA,
        }, nil
    }
    
    return nil, fmt.Errorf("no drivers available")
}
```

### **2. GraphQL Schema Design**
```graphql
# Core GraphQL schema structure
type Query {
    # User queries
    me: User
    user(id: ID!): User
    
    # Trip queries
    trip(id: ID!): Trip
    myTrips(limit: Int, offset: Int): [Trip!]!
    
    # Driver queries
    nearbyDrivers(location: LocationInput!, radius: Float!): [Driver!]!
    
    # Price queries
    priceEstimate(pickup: LocationInput!, destination: LocationInput!): PriceEstimate
}

type Mutation {
    # Authentication
    login(email: String!, password: String!): AuthResponse
    register(input: RegisterInput!): AuthResponse
    
    # Trip management
    requestRide(input: RideRequestInput!): Trip
    cancelTrip(tripId: ID!): Boolean
    
    # Driver actions
    acceptTrip(tripId: ID!): Boolean
    startTrip(tripId: ID!): Boolean
    completeTrip(tripId: ID!, finalLocation: LocationInput!): Boolean
    
    # Payments
    addPaymentMethod(input: PaymentMethodInput!): PaymentMethod
    processPayment(tripId: ID!, paymentMethodId: ID!): PaymentResult
}

type Subscription {
    # Real-time trip updates
    tripUpdates(tripId: ID!): TripUpdate
    
    # Driver location tracking
    driverLocation(driverId: ID!): LocationUpdate
    
    # Live price updates
    priceUpdates(pickup: LocationInput!, destination: LocationInput!): PriceUpdate
}

# Complex types
type Trip {
    id: ID!
    rider: User!
    driver: Driver
    pickup: Location!
    destination: Location!
    status: TripStatus!
    price: Price!
    createdAt: DateTime!
    updatedAt: DateTime!
    
    # Real-time computed fields
    currentLocation: Location @live
    estimatedArrival: DateTime @live
    route: [Location!]! @live
}
```

### **3. Authentication & Authorization**
```go
type AuthenticationMiddleware struct {
    jwtSecret     []byte
    userService   UserServiceClient
    sessionStore  SessionStore
    rateLimiter   *RateLimiter
}

func (auth *AuthenticationMiddleware) ValidateToken(ctx context.Context, token string) (*User, error) {
    // 1. Parse JWT token
    claims, err := auth.parseJWTToken(token)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %v", err)
    }
    
    // 2. Check token expiration
    if claims.ExpiresAt < time.Now().Unix() {
        return nil, fmt.Errorf("token expired")
    }
    
    // 3. Validate with session store (for immediate revocation capability)
    session, err := auth.sessionStore.GetSession(ctx, claims.SessionID)
    if err != nil || !session.IsActive {
        return nil, fmt.Errorf("session invalid or revoked")
    }
    
    // 4. Fetch fresh user data from User Service
    user, err := auth.userService.GetUser(ctx, &GetUserRequest{
        UserID: claims.UserID,
    })
    if err != nil {
        return nil, fmt.Errorf("user not found: %v", err)
    }
    
    // 5. Check rate limiting
    if !auth.rateLimiter.Allow(claims.UserID) {
        return nil, fmt.Errorf("rate limit exceeded")
    }
    
    return user, nil
}

// Role-based access control
func (auth *AuthenticationMiddleware) CheckPermission(user *User, resource string, action string) bool {
    switch user.Role {
    case "admin":
        return true // Admins have all permissions
    case "driver":
        return auth.checkDriverPermissions(resource, action)
    case "rider":
        return auth.checkRiderPermissions(resource, action)
    default:
        return false
    }
}

func (auth *AuthenticationMiddleware) checkDriverPermissions(resource, action string) bool {
    driverPermissions := map[string][]string{
        "trips":    {"read", "update"}, // Can read and update trips
        "vehicles": {"read", "update"}, // Can manage their vehicles
        "earnings": {"read"},           // Can view earnings
        "profile":  {"read", "update"}, // Can update profile
    }
    
    allowedActions := driverPermissions[resource]
    for _, allowedAction := range allowedActions {
        if allowedAction == action {
            return true
        }
    }
    return false
}
```

### **4. Real-time Communication (WebSocket)**
```go
type WebSocketHub struct {
    clients     map[string]*WebSocketClient // userID -> client
    broadcast   chan []byte
    register    chan *WebSocketClient
    unregister  chan *WebSocketClient
    tripUpdates chan *TripUpdate
}

type WebSocketClient struct {
    userID     string
    conn       *websocket.Conn
    send       chan []byte
    hub        *WebSocketHub
    subscriptions map[string]bool // subscribed topics
}

func (hub *WebSocketHub) Run() {
    for {
        select {
        case client := <-hub.register:
            hub.clients[client.userID] = client
            log.Printf("Client %s connected", client.userID)
            
        case client := <-hub.unregister:
            if _, ok := hub.clients[client.userID]; ok {
                delete(hub.clients, client.userID)
                close(client.send)
                log.Printf("Client %s disconnected", client.userID)
            }
            
        case tripUpdate := <-hub.tripUpdates:
            // Send trip updates to relevant users (rider and driver)
            if riderClient, ok := hub.clients[tripUpdate.RiderID]; ok {
                riderClient.send <- marshalTripUpdate(tripUpdate)
            }
            if driverClient, ok := hub.clients[tripUpdate.DriverID]; ok {
                driverClient.send <- marshalTripUpdate(tripUpdate)
            }
            
        case message := <-hub.broadcast:
            // Broadcast to all connected clients
            for userID, client := range hub.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(hub.clients, userID)
                }
            }
        }
    }
}

// Handle incoming messages from clients
func (client *WebSocketClient) handleMessage(message []byte) {
    var msg WebSocketMessage
    if err := json.Unmarshal(message, &msg); err != nil {
        log.Printf("Error unmarshaling message: %v", err)
        return
    }
    
    switch msg.Type {
    case "subscribe_trip_updates":
        client.subscriptions["trip_updates_"+msg.TripID] = true
        
    case "subscribe_driver_location":
        client.subscriptions["driver_location_"+msg.DriverID] = true
        
    case "location_update":
        // Driver sending location update
        go client.hub.processLocationUpdate(client.userID, msg.LocationData)
        
    case "ping":
        // Heartbeat
        client.send <- []byte(`{"type":"pong"}`)
    }
}
```

### **5. Service Discovery & Health Monitoring**
```go
type GRPCClientManager struct {
    clients map[string]interface{}
    config  *Config
    healthCheckers map[string]*HealthChecker
}

func (gcm *GRPCClientManager) Initialize() error {
    services := []ServiceConfig{
        {Name: "user-service", Address: "user-service:50051"},
        {Name: "vehicle-service", Address: "vehicle-service:50052"},
        {Name: "geo-service", Address: "geo-service:50053"},
        {Name: "matching-service", Address: "matching-service:50054"},
        {Name: "pricing-service", Address: "pricing-service:50055"},
        {Name: "trip-service", Address: "trip-service:50056"},
        {Name: "payment-service", Address: "payment-service:50057"},
    }
    
    for _, svc := range services {
        conn, err := grpc.Dial(svc.Address, 
            grpc.WithInsecure(),
            grpc.WithTimeout(10*time.Second),
            grpc.WithKeepaliveParams(keepalive.ClientParameters{
                Time:                10 * time.Second,
                Timeout:             3 * time.Second,
                PermitWithoutStream: true,
            }),
        )
        if err != nil {
            log.Printf("Failed to connect to %s: %v", svc.Name, err)
            continue
        }
        
        // Create service client based on service name
        switch svc.Name {
        case "user-service":
            gcm.clients["user"] = pb.NewUserServiceClient(conn)
        case "vehicle-service":
            gcm.clients["vehicle"] = pb.NewVehicleServiceClient(conn)
        case "geo-service":
            gcm.clients["geo"] = pb.NewGeoServiceClient(conn)
        case "matching-service":
            gcm.clients["matching"] = pb.NewMatchingServiceClient(conn)
        case "pricing-service":
            gcm.clients["pricing"] = pb.NewPricingServiceClient(conn)
        case "trip-service":
            gcm.clients["trip"] = pb.NewTripServiceClient(conn)
        case "payment-service":
            gcm.clients["payment"] = pb.NewPaymentServiceClient(conn)
        }
        
        // Setup health checker
        gcm.healthCheckers[svc.Name] = &HealthChecker{
            serviceName: svc.Name,
            client:      grpc_health_v1.NewHealthClient(conn),
        }
    }
    
    // Start health checking
    go gcm.startHealthChecking()
    
    return nil
}

func (gcm *GRPCClientManager) HealthCheck(ctx context.Context) map[string]bool {
    health := make(map[string]bool)
    
    for serviceName, checker := range gcm.healthCheckers {
        resp, err := checker.client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
            Service: serviceName,
        })
        
        health[serviceName] = err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
    }
    
    return health
}
```

### **6. Rate Limiting & Security**
```go
type RateLimiter struct {
    rules    map[string]*RateLimitRule
    store    RateLimitStore // Redis-based
    metrics  *PrometheusMetrics
}

type RateLimitRule struct {
    Resource      string        // e.g., "trip_requests"
    MaxRequests   int           // e.g., 100
    TimeWindow    time.Duration // e.g., 1 hour
    BurstAllowed  int           // e.g., 10 requests in burst
}

func (rl *RateLimiter) Allow(userID string, resource string) bool {
    rule, exists := rl.rules[resource]
    if !exists {
        return true // No rate limit defined
    }
    
    key := fmt.Sprintf("rate_limit:%s:%s", userID, resource)
    
    // Get current count from Redis
    count, err := rl.store.GetCount(key, rule.TimeWindow)
    if err != nil {
        log.Printf("Rate limit check failed: %v", err)
        return true // Fail open
    }
    
    if count >= rule.MaxRequests {
        rl.metrics.RateLimitExceeded.WithLabelValues(resource).Inc()
        return false
    }
    
    // Increment counter
    rl.store.IncrementCount(key, rule.TimeWindow)
    return true
}

// Security middleware
func (gw *APIGateway) SecurityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
        
        // 2. Security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // 3. Request size limit
        if r.ContentLength > 10*1024*1024 { // 10MB limit
            http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
            return
        }
        
        // 4. IP-based rate limiting
        clientIP := getClientIP(r)
        if !gw.rateLimiter.Allow(clientIP, "api_requests") {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

---

## 🔧 Technical Implementation Details

### **GraphQL Resolvers**
```go
type Resolver struct {
    grpcClients *GRPCClientManager
    authService *AuthenticationService
    cache       *RedisCache
}

func (r *Resolver) Trip() TripResolver { return &tripResolver{r} }
func (r *Resolver) User() UserResolver { return &userResolver{r} }
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

type tripResolver struct{ *Resolver }

func (r *tripResolver) Driver(ctx context.Context, obj *model.Trip) (*model.Driver, error) {
    if obj.DriverID == nil {
        return nil, nil
    }
    
    // Check cache first
    cacheKey := fmt.Sprintf("driver:%s", *obj.DriverID)
    if cached := r.cache.Get(cacheKey); cached != nil {
        return cached.(*model.Driver), nil
    }
    
    // Fetch from User Service
    driver, err := r.grpcClients.UserService.GetDriver(ctx, &pb.GetDriverRequest{
        DriverId: *obj.DriverID,
    })
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    r.cache.Set(cacheKey, driver, 5*time.Minute)
    
    return convertDriverProtoToGraphQL(driver), nil
}

func (r *tripResolver) Route(ctx context.Context, obj *model.Trip) ([]*model.Location, error) {
    // Get route from Geo Service
    route, err := r.grpcClients.GeoService.CalculateRoute(ctx, &pb.RouteRequest{
        Origin:      locationToProto(obj.Pickup),
        Destination: locationToProto(obj.Destination),
        TripId:      obj.ID,
    })
    if err != nil {
        return nil, err
    }
    
    return convertRouteProtoToGraphQL(route.Waypoints), nil
}
```

### **Error Handling & Resilience**
```go
type CircuitBreaker struct {
    maxFailures  int
    timeout      time.Duration
    resetTimeout time.Duration
    state        CircuitState
    failures     int
    lastFailTime time.Time
    mutex        sync.RWMutex
}

func (cb *CircuitBreaker) Call(fn func() (interface{}, error)) (interface{}, error) {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    if cb.state == StateOpen {
        if time.Since(cb.lastFailTime) > cb.resetTimeout {
            cb.state = StateHalfOpen
            cb.failures = 0
        } else {
            return nil, fmt.Errorf("circuit breaker is open")
        }
    }
    
    result, err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = StateOpen
        }
        
        return nil, err
    }
    
    // Success - reset circuit breaker
    cb.failures = 0
    cb.state = StateClosed
    return result, nil
}

// Graceful degradation example
func (gw *APIGateway) GetNearbyDriversWithFallback(ctx context.Context, location *Location) ([]*Driver, error) {
    // Try primary Geo Service
    result, err := gw.circuitBreaker.Call(func() (interface{}, error) {
        return gw.grpcClients.GeoService.FindNearbyDrivers(ctx, &pb.NearbyDriversRequest{
            Location: locationToProto(location),
            Radius:   5000, // 5km
        })
    })
    
    if err == nil {
        return convertDriversProtoToGraphQL(result.(*pb.NearbyDriversResponse).Drivers), nil
    }
    
    // Fallback 1: Try cached data
    cached := gw.cache.GetNearbyDrivers(location)
    if len(cached) > 0 {
        log.Printf("Using cached drivers due to Geo Service failure")
        return cached, nil
    }
    
    // Fallback 2: Return empty list with degraded service indicator
    return []*Driver{}, fmt.Errorf("service temporarily unavailable")
}
```

---

## 📊 Performance & Monitoring

### **Metrics Collection**
```go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "api_gateway_request_duration_seconds",
            Help: "Request duration in seconds",
        },
        []string{"method", "endpoint", "status"},
    )
    
    grpcCallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "api_gateway_grpc_call_duration_seconds",
            Help: "gRPC call duration in seconds",
        },
        []string{"service", "method", "status"},
    )
    
    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "api_gateway_active_websocket_connections",
            Help: "Number of active WebSocket connections",
        },
    )
)

func instrumentGRPCCall(service, method string, fn func() error) error {
    start := time.Now()
    err := fn()
    duration := time.Since(start)
    
    status := "success"
    if err != nil {
        status = "error"
    }
    
    grpcCallDuration.WithLabelValues(service, method, status).Observe(duration.Seconds())
    
    return err
}
```

### **Health Checks & Monitoring**
```go
func (gw *APIGateway) HealthHandler(w http.ResponseWriter, r *http.Request) {
    health := &HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Services:  make(map[string]ServiceHealth),
    }
    
    // Check all downstream services
    for serviceName, checker := range gw.grpcClients.healthCheckers {
        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
        defer cancel()
        
        resp, err := checker.client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
            Service: serviceName,
        })
        
        serviceHealth := ServiceHealth{
            Status:      "unhealthy",
            LastChecked: time.Now(),
        }
        
        if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
            serviceHealth.Status = "healthy"
        } else {
            serviceHealth.Error = err.Error()
            health.Status = "degraded"
        }
        
        health.Services[serviceName] = serviceHealth
    }
    
    statusCode := http.StatusOK
    if health.Status == "degraded" {
        statusCode = http.StatusServiceUnavailable
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(health)
}
```

The API Gateway is the most complex service as it orchestrates all interactions between clients and the backend services. It provides a unified, secure, and performant interface while handling concerns like authentication, rate limiting, real-time communication, and graceful degradation.
