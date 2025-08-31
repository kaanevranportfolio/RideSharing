# 🛣️ TRIP SERVICE - DEEP DIVE

## 📋 Overview
The **Trip Service** is the orchestrator of the entire ride experience, managing the complete lifecycle of a trip from initial request to final completion. It implements event sourcing patterns, coordinates with all other services, and maintains the authoritative state of every ride in the system.

---

## 🎯 Core Responsibilities

### **1. Trip Lifecycle Management**
- **State Machine**: Manages complex trip states and transitions
- **Event Sourcing**: Records every trip event for audit and replay
- **Coordination**: Orchestrates interactions between all services
- **Business Rules**: Enforces trip policies and constraints

### **2. Trip State Transitions**
```
REQUESTED → SEARCHING → MATCHED → CONFIRMED → 
DRIVER_ARRIVING → DRIVER_ARRIVED → STARTED → 
IN_PROGRESS → COMPLETED/CANCELLED
```

### **3. Service Orchestration**
- **Matching**: Coordinates with matching service for driver assignment
- **Pricing**: Gets fare calculations and handles payment processing
- **Navigation**: Manages route tracking and ETA updates
- **Notifications**: Sends real-time updates to riders and drivers

### **4. Event Management**
- **Event Storage**: Persists all trip events for complete audit trail
- **Event Replay**: Reconstruct trip state from event history
- **Real-time Streaming**: Publishes events for real-time features
- **Analytics**: Provides data for business intelligence

---

## 🏗️ Architecture Components

### **Production Service Structure**
```go
type ProductionTripService struct {
    tripRepo       TripRepositoryInterface      // Trip data persistence
    eventRepo      TripEventRepositoryInterface // Event sourcing
    matchingClient MatchingServiceClient        // Driver matching
    pricingClient  PricingServiceClient         // Fare calculations
    paymentClient  PaymentServiceClient         // Payment processing
    logger         *logger.Logger               // Logging system
}
```

### **Event Sourcing Implementation**
```go
type TripEvent struct {
    ID        string                 `json:"id"`
    TripID    string                 `json:"trip_id"`
    Type      TripEventType          `json:"type"`
    Data      map[string]interface{} `json:"data"`
    ActorID   string                 `json:"actor_id"`   // Who triggered the event
    ActorType ActorType              `json:"actor_type"` // rider, driver, system
    Timestamp time.Time              `json:"timestamp"`
    Version   int                    `json:"version"`    // Event version for schema evolution
}

type TripEventType string

const (
    EventTripRequested     TripEventType = "trip_requested"
    EventMatchingStarted   TripEventType = "matching_started"
    EventDriverMatched     TripEventType = "driver_matched"
    EventDriverConfirmed   TripEventType = "driver_confirmed"
    EventDriverArriving    TripEventType = "driver_arriving"
    EventDriverArrived     TripEventType = "driver_arrived"
    EventTripStarted       TripEventType = "trip_started"
    EventLocationUpdated   TripEventType = "location_updated"
    EventETAUpdated        TripEventType = "eta_updated"
    EventTripCompleted     TripEventType = "trip_completed"
    EventTripCancelled     TripEventType = "trip_cancelled"
    EventPaymentProcessed  TripEventType = "payment_processed"
    EventPaymentFailed     TripEventType = "payment_failed"
)
```

---

## 🔄 Trip State Machine

### **1. State Definitions**
```go
type TripStatus string

const (
    TripStatusRequested      TripStatus = "requested"       // User requested ride
    TripStatusSearching      TripStatus = "searching"       // Looking for driver
    TripStatusMatched        TripStatus = "matched"         // Driver found
    TripStatusConfirmed      TripStatus = "confirmed"       // Driver accepted
    TripStatusDriverArriving TripStatus = "driver_arriving" // Driver en route to pickup
    TripStatusDriverArrived  TripStatus = "driver_arrived"  // Driver at pickup location
    TripStatusStarted        TripStatus = "started"         // Trip started
    TripStatusInProgress     TripStatus = "in_progress"     // Trip ongoing
    TripStatusCompleted      TripStatus = "completed"       // Trip finished
    TripStatusCancelled      TripStatus = "cancelled"       // Trip cancelled
)
```

### **2. State Transition Validation**
```go
func (s *ProductionTripService) validateStateTransition(currentStatus, newStatus TripStatus) error {
    validTransitions := map[TripStatus][]TripStatus{
        TripStatusRequested: {
            TripStatusSearching,
            TripStatusCancelled,
        },
        TripStatusSearching: {
            TripStatusMatched,
            TripStatusCancelled,
        },
        TripStatusMatched: {
            TripStatusConfirmed,
            TripStatusCancelled,
        },
        TripStatusConfirmed: {
            TripStatusDriverArriving,
            TripStatusCancelled,
        },
        TripStatusDriverArriving: {
            TripStatusDriverArrived,
            TripStatusCancelled,
        },
        TripStatusDriverArrived: {
            TripStatusStarted,
            TripStatusCancelled,
        },
        TripStatusStarted: {
            TripStatusInProgress,
            TripStatusCancelled,
        },
        TripStatusInProgress: {
            TripStatusCompleted,
            TripStatusCancelled,
        },
        // Terminal states
        TripStatusCompleted: {},
        TripStatusCancelled: {},
    }
    
    allowedTransitions, exists := validTransitions[currentStatus]
    if !exists {
        return fmt.Errorf("unknown current status: %s", currentStatus)
    }
    
    for _, allowed := range allowedTransitions {
        if allowed == newStatus {
            return nil
        }
    }
    
    return fmt.Errorf("invalid state transition from %s to %s", currentStatus, newStatus)
}
```

---

## 🚀 Complete Trip Lifecycle Implementation

### **1. Trip Request Processing**
```go
func (s *ProductionTripService) RequestTrip(ctx context.Context, request *TripRequest) (*TripResponse, error) {
    startTime := time.Now()
    
    // 1. Validate request
    if err := s.validateTripRequest(request); err != nil {
        return nil, fmt.Errorf("invalid trip request: %w", err)
    }
    
    // 2. Create trip record
    trip := &models.Trip{
        ID:                  s.generateTripID(),
        RiderID:             request.RiderID,
        PickupLocation:      request.PickupLocation,
        DestinationLocation: request.DestinationLocation,
        RequestedVehicleType: request.VehicleType,
        SpecialRequirements: request.SpecialRequirements,
        Status:              TripStatusRequested,
        RequestedAt:         time.Now(),
    }
    
    // 3. Save trip to database
    if err := s.tripRepo.CreateTrip(ctx, trip); err != nil {
        return nil, fmt.Errorf("failed to create trip: %w", err)
    }
    
    // 4. Record trip requested event
    event := &TripEvent{
        ID:        s.generateEventID(),
        TripID:    trip.ID,
        Type:      EventTripRequested,
        Data:      s.tripToEventData(trip),
        ActorID:   request.RiderID,
        ActorType: ActorTypeRider,
        Timestamp: time.Now(),
        Version:   1,
    }
    
    if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
        s.logger.Error("Failed to create trip event", "error", err)
        // Continue - don't fail the request for event storage issues
    }
    
    // 5. Get initial fare estimate
    fareEstimate, err := s.pricingClient.EstimateFare(ctx, &FareEstimateRequest{
        PickupLocation:      trip.PickupLocation,
        DestinationLocation: trip.DestinationLocation,
        VehicleType:         trip.RequestedVehicleType,
        TimeOfDay:          time.Now(),
    })
    
    if err != nil {
        s.logger.Warn("Failed to get fare estimate", "trip_id", trip.ID, "error", err)
        fareEstimate = &FareEstimateResponse{EstimatedFare: 0.0}
    }
    
    // 6. Start matching process asynchronously
    go s.startMatching(context.Background(), trip)
    
    response := &TripResponse{
        TripID:        trip.ID,
        Status:        string(trip.Status),
        EstimatedFare: fareEstimate.EstimatedFare,
        CreatedAt:     trip.RequestedAt,
        ProcessingTime: time.Since(startTime),
    }
    
    return response, nil
}
```

### **2. Driver Matching Orchestration**
```go
func (s *ProductionTripService) startMatching(ctx context.Context, trip *models.Trip) {
    // Update trip status to searching
    if err := s.updateTripStatus(ctx, trip.ID, TripStatusSearching, ActorTypeSystem, "system"); err != nil {
        s.logger.Error("Failed to update trip status to searching", "trip_id", trip.ID, "error", err)
        return
    }
    
    // Call matching service
    matchingRequest := &MatchingRequest{
        TripID:                 trip.ID,
        RiderID:               trip.RiderID,
        PickupLocation:        trip.PickupLocation,
        DestinationLocation:   trip.DestinationLocation,
        PreferredVehicleType:  trip.RequestedVehicleType,
        SpecialRequirements:   trip.SpecialRequirements,
        RequestedAt:           trip.RequestedAt,
    }
    
    matchResult, err := s.matchingClient.FindMatch(ctx, matchingRequest)
    if err != nil {
        s.logger.Error("Matching failed", "trip_id", trip.ID, "error", err)
        s.handleMatchingFailure(ctx, trip)
        return
    }
    
    if !matchResult.Success {
        s.logger.Info("No drivers available", "trip_id", trip.ID, "reason", matchResult.Reason)
        s.handleMatchingFailure(ctx, trip)
        return
    }
    
    // Driver found - update trip with driver info
    if err := s.assignDriver(ctx, trip, matchResult); err != nil {
        s.logger.Error("Failed to assign driver", "trip_id", trip.ID, "error", err)
        s.handleMatchingFailure(ctx, trip)
        return
    }
    
    s.logger.Info("Driver successfully matched", 
        "trip_id", trip.ID, 
        "driver_id", matchResult.Driver.DriverID,
        "eta", matchResult.ETA)
}
```

### **3. Driver Assignment Process**
```go
func (s *ProductionTripService) assignDriver(ctx context.Context, trip *models.Trip, matchResult *MatchingResult) error {
    // 1. Update trip with driver information
    trip.DriverID = matchResult.Driver.DriverID
    trip.VehicleID = matchResult.Driver.VehicleID
    trip.EstimatedFare = matchResult.EstimatedFare
    trip.PickupETA = matchResult.ETA
    
    if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
        return fmt.Errorf("failed to update trip with driver info: %w", err)
    }
    
    // 2. Update trip status to matched
    if err := s.updateTripStatus(ctx, trip.ID, TripStatusMatched, ActorTypeSystem, "system"); err != nil {
        return fmt.Errorf("failed to update trip status to matched: %w", err)
    }
    
    // 3. Record driver matched event
    eventData := map[string]interface{}{
        "driver_id":      matchResult.Driver.DriverID,
        "vehicle_id":     matchResult.Driver.VehicleID,
        "estimated_fare": matchResult.EstimatedFare,
        "pickup_eta":     matchResult.ETA,
        "match_score":    matchResult.MatchScore,
    }
    
    event := &TripEvent{
        ID:        s.generateEventID(),
        TripID:    trip.ID,
        Type:      EventDriverMatched,
        Data:      eventData,
        ActorID:   "system",
        ActorType: ActorTypeSystem,
        Timestamp: time.Now(),
        Version:   1,
    }
    
    if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
        s.logger.Error("Failed to create driver matched event", "error", err)
    }
    
    // 4. Send notifications to rider and driver
    s.sendMatchNotifications(ctx, trip, matchResult.Driver)
    
    // 5. Start driver confirmation timeout
    go s.startDriverConfirmationTimeout(ctx, trip.ID)
    
    return nil
}
```

### **4. Real-time Location Tracking**
```go
func (s *ProductionTripService) UpdateDriverLocation(ctx context.Context, request *LocationUpdateRequest) error {
    trip, err := s.tripRepo.GetTripById(ctx, request.TripID)
    if err != nil {
        return fmt.Errorf("trip not found: %w", err)
    }
    
    // Validate that the driver can update this trip
    if trip.DriverID != request.DriverID {
        return fmt.Errorf("unauthorized driver location update")
    }
    
    // Update driver location in trip
    trip.DriverCurrentLocation = request.Location
    trip.LastLocationUpdate = time.Now()
    
    if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
        return fmt.Errorf("failed to update trip location: %w", err)
    }
    
    // Record location update event
    eventData := map[string]interface{}{
        "latitude":  request.Location.Latitude,
        "longitude": request.Location.Longitude,
        "timestamp": time.Now(),
    }
    
    event := &TripEvent{
        ID:        s.generateEventID(),
        TripID:    trip.ID,
        Type:      EventLocationUpdated,
        Data:      eventData,
        ActorID:   request.DriverID,
        ActorType: ActorTypeDriver,
        Timestamp: time.Now(),
        Version:   1,
    }
    
    s.eventRepo.CreateEvent(ctx, event)
    
    // Calculate and update ETA if trip is in progress
    if trip.Status == TripStatusDriverArriving || trip.Status == TripStatusInProgress {
        s.updateETAEstimate(ctx, trip)
    }
    
    // Publish real-time location update
    s.publishLocationUpdate(ctx, trip, request.Location)
    
    return nil
}
```

---

## 🎯 Advanced Trip Features

### **1. Trip Completion Process**
```go
func (s *ProductionTripService) CompleteTrip(ctx context.Context, request *CompleteTripRequest) (*CompleteTripResponse, error) {
    trip, err := s.tripRepo.GetTripById(ctx, request.TripID)
    if err != nil {
        return nil, fmt.Errorf("trip not found: %w", err)
    }
    
    // Validate trip can be completed
    if trip.Status != TripStatusInProgress {
        return nil, fmt.Errorf("trip cannot be completed in status: %s", trip.Status)
    }
    
    // Calculate final fare
    fareRequest := &FareCalculationRequest{
        TripID:              trip.ID,
        UserID:              trip.RiderID,
        VehicleType:         trip.VehicleType,
        DistanceKm:          request.FinalDistanceKm,
        DurationMinutes:     request.FinalDurationMinutes,
        PickupLocation:      trip.PickupLocation,
        DestinationLocation: trip.DestinationLocation,
        TimeOfDay:          trip.StartedAt,
    }
    
    fareResponse, err := s.pricingClient.CalculateFare(ctx, fareRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to calculate final fare: %w", err)
    }
    
    // Update trip with final details
    trip.Status = TripStatusCompleted
    trip.CompletedAt = time.Now()
    trip.FinalFare = fareResponse.TotalFare
    trip.DistanceKm = request.FinalDistanceKm
    trip.DurationMinutes = int(request.FinalDurationMinutes)
    trip.EndLocation = request.EndLocation
    
    if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
        return nil, fmt.Errorf("failed to update completed trip: %w", err)
    }
    
    // Record completion event
    completionData := map[string]interface{}{
        "final_fare":        fareResponse.TotalFare,
        "distance_km":       request.FinalDistanceKm,
        "duration_minutes":  request.FinalDurationMinutes,
        "end_location":      request.EndLocation,
        "fare_breakdown":    fareResponse.FareBreakdown,
    }
    
    event := &TripEvent{
        ID:        s.generateEventID(),
        TripID:    trip.ID,
        Type:      EventTripCompleted,
        Data:      completionData,
        ActorID:   request.ActorID,
        ActorType: request.ActorType,
        Timestamp: time.Now(),
        Version:   1,
    }
    
    s.eventRepo.CreateEvent(ctx, event)
    
    // Process payment
    go s.processPayment(context.Background(), trip, fareResponse)
    
    // Send completion notifications
    s.sendCompletionNotifications(ctx, trip, fareResponse)
    
    return &CompleteTripResponse{
        TripID:    trip.ID,
        FinalFare: fareResponse.TotalFare,
        Receipt:   s.generateReceipt(trip, fareResponse),
    }, nil
}
```

### **2. Trip Cancellation Handling**
```go
func (s *ProductionTripService) CancelTrip(ctx context.Context, request *CancelTripRequest) error {
    trip, err := s.tripRepo.GetTripById(ctx, request.TripID)
    if err != nil {
        return fmt.Errorf("trip not found: %w", err)
    }
    
    // Validate cancellation is allowed
    if trip.Status == TripStatusCompleted {
        return fmt.Errorf("cannot cancel completed trip")
    }
    
    // Calculate cancellation fee based on trip status
    var cancellationFee float64
    switch trip.Status {
    case TripStatusRequested, TripStatusSearching:
        cancellationFee = 0.0 // Free cancellation
    case TripStatusMatched, TripStatusConfirmed:
        cancellationFee = 2.0 // Small fee
    case TripStatusDriverArriving:
        cancellationFee = 3.0 // Medium fee
    case TripStatusDriverArrived, TripStatusStarted:
        cancellationFee = 5.0 // Higher fee
    default:
        cancellationFee = 0.0
    }
    
    // Update trip status
    trip.Status = TripStatusCancelled
    trip.CancelledAt = time.Now()
    trip.CancellationReason = request.Reason
    trip.CancelledBy = request.CancelledBy
    trip.CancellationFee = cancellationFee
    
    if err := s.tripRepo.UpdateTrip(ctx, trip); err != nil {
        return fmt.Errorf("failed to update cancelled trip: %w", err)
    }
    
    // Record cancellation event
    cancellationData := map[string]interface{}{
        "reason":            request.Reason,
        "cancelled_by":      request.CancelledBy,
        "cancellation_fee":  cancellationFee,
        "trip_status":       trip.Status,
    }
    
    event := &TripEvent{
        ID:        s.generateEventID(),
        TripID:    trip.ID,
        Type:      EventTripCancelled,
        Data:      cancellationData,
        ActorID:   request.CancelledBy,
        ActorType: request.ActorType,
        Timestamp: time.Now(),
        Version:   1,
    }
    
    s.eventRepo.CreateEvent(ctx, event)
    
    // Handle post-cancellation logic
    s.handlePostCancellation(ctx, trip)
    
    return nil
}
```

---

## 🔧 Integration Points

### **1. With All Services**
```go
// The trip service coordinates with every other service:

// Matching Service - for driver assignment
matchResult, err := s.matchingClient.FindMatch(ctx, matchingRequest)

// Pricing Service - for fare calculations
fareResponse, err := s.pricingClient.CalculateFare(ctx, fareRequest)

// Payment Service - for payment processing
paymentResult, err := s.paymentClient.ProcessPayment(ctx, paymentRequest)

// Geo Service - for distance/ETA calculations (indirectly through other services)

// User Service - for user validation and notifications (indirectly)

// Vehicle Service - for vehicle information (indirectly through matching)
```

### **2. Event Publishing for Real-time Features**
```go
func (s *ProductionTripService) publishTripEvent(ctx context.Context, event *TripEvent) {
    // Publish to Redis for real-time subscriptions
    eventJSON, _ := json.Marshal(event)
    
    // Channel for this specific trip
    tripChannel := fmt.Sprintf("trip_events:%s", event.TripID)
    s.redis.Publish(ctx, tripChannel, eventJSON)
    
    // Global trip events channel
    s.redis.Publish(ctx, "global_trip_events", eventJSON)
    
    // User-specific channels
    if event.ActorType == ActorTypeRider {
        riderChannel := fmt.Sprintf("rider_events:%s", event.ActorID)
        s.redis.Publish(ctx, riderChannel, eventJSON)
    } else if event.ActorType == ActorTypeDriver {
        driverChannel := fmt.Sprintf("driver_events:%s", event.ActorID)
        s.redis.Publish(ctx, driverChannel, eventJSON)
    }
}
```

---

## 📊 Event Sourcing Benefits

### **1. Complete Audit Trail**
```go
func (s *ProductionTripService) GetTripHistory(ctx context.Context, tripID string) (*TripHistory, error) {
    events, err := s.eventRepo.GetEventsByTripID(ctx, tripID)
    if err != nil {
        return nil, err
    }
    
    history := &TripHistory{
        TripID: tripID,
        Events: make([]*TripEventView, len(events)),
    }
    
    for i, event := range events {
        history.Events[i] = &TripEventView{
            Type:        string(event.Type),
            Timestamp:   event.Timestamp,
            Actor:       fmt.Sprintf("%s (%s)", event.ActorID, event.ActorType),
            Description: s.eventToDescription(event),
            Data:        event.Data,
        }
    }
    
    return history, nil
}
```

### **2. State Reconstruction**
```go
func (s *ProductionTripService) reconstructTripFromEvents(ctx context.Context, tripID string) (*models.Trip, error) {
    events, err := s.eventRepo.GetEventsByTripID(ctx, tripID)
    if err != nil {
        return nil, err
    }
    
    trip := &models.Trip{}
    
    // Replay events to rebuild trip state
    for _, event := range events {
        switch event.Type {
        case EventTripRequested:
            s.applyTripRequestedEvent(trip, event)
        case EventDriverMatched:
            s.applyDriverMatchedEvent(trip, event)
        case EventTripStarted:
            s.applyTripStartedEvent(trip, event)
        case EventTripCompleted:
            s.applyTripCompletedEvent(trip, event)
        // ... handle other events
        }
    }
    
    return trip, nil
}
```

---

## 🌟 Advanced Features

### **1. Trip Analytics**
```go
func (s *ProductionTripService) GenerateTripAnalytics(ctx context.Context, timeRange time.Duration) (*TripAnalytics, error) {
    analytics := &TripAnalytics{
        TimeRange: timeRange,
        StartTime: time.Now().Add(-timeRange),
        EndTime:   time.Now(),
    }
    
    // Get trip statistics
    analytics.TotalTrips = s.getTripCount(ctx, timeRange)
    analytics.CompletedTrips = s.getCompletedTripCount(ctx, timeRange)
    analytics.CancelledTrips = s.getCancelledTripCount(ctx, timeRange)
    analytics.CompletionRate = float64(analytics.CompletedTrips) / float64(analytics.TotalTrips)
    
    // Average metrics
    analytics.AvgTripDuration = s.getAvgTripDuration(ctx, timeRange)
    analytics.AvgTripDistance = s.getAvgTripDistance(ctx, timeRange)
    analytics.AvgFare = s.getAvgFare(ctx, timeRange)
    
    // Peak times analysis
    analytics.PeakHours = s.analyzePeakHours(ctx, timeRange)
    
    return analytics, nil
}
```

### **2. Predictive ETA Updates**
```go
func (s *ProductionTripService) updateETAWithTrafficPrediction(ctx context.Context, trip *models.Trip) {
    // Use machine learning to predict accurate ETA
    features := &ETAFeatures{
        CurrentLocation:   trip.DriverCurrentLocation,
        DestinationLocation: trip.PickupLocation,
        TimeOfDay:        time.Now(),
        DayOfWeek:        time.Now().Weekday(),
        Weather:          s.getWeatherConditions(trip.PickupLocation),
        TrafficPatterns:  s.getHistoricalTrafficPatterns(trip.PickupLocation),
    }
    
    predictedETA := s.etaPredictor.PredictETA(features)
    
    if math.Abs(predictedETA-float64(trip.PickupETA)) > 60 { // 1 minute difference
        trip.PickupETA = int(predictedETA)
        s.tripRepo.UpdateTrip(ctx, trip)
        
        // Notify rider of ETA change
        s.sendETAUpdateNotification(ctx, trip)
    }
}
```

---

## 🎯 Why This Service is Critical

### **1. Central Orchestrator**
- **Single Source of Truth**: Authoritative state for all trips
- **Service Coordination**: Manages interactions between all services
- **Business Logic Enforcement**: Ensures trip rules and policies

### **2. Data Integrity**
- **Event Sourcing**: Complete audit trail and state reconstruction
- **State Machine**: Prevents invalid state transitions
- **Consistency**: Ensures data consistency across services

### **3. Real-time Operations**
- **Live Updates**: Real-time trip status and location updates
- **Notifications**: Instant updates to riders and drivers
- **Analytics**: Real-time business intelligence

### **4. Business Intelligence**
- **Complete History**: Every trip action is recorded
- **Analytics Ready**: Rich data for business analysis
- **Debugging**: Full event trail for troubleshooting

---

This Trip Service represents the **operational heart** of the rideshare platform. Its event sourcing architecture provides unparalleled auditability and debugging capabilities, while its orchestration role ensures all services work together seamlessly to deliver a smooth ride experience. The production implementation handles the complexity of coordinating multiple distributed services while maintaining data consistency and providing real-time updates to all stakeholders.
