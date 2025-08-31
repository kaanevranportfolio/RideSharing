# 🎭 CHAOS ENGINEERING & RESILIENCE TESTING

## 📋 Overview
**Chaos Engineering** is the practice of intentionally introducing failures into production systems to test their resilience and identify weaknesses before they cause outages. For a rideshare platform handling millions of users and financial transactions, chaos testing is essential to ensure the system can handle real-world failures gracefully.

---

## 🎯 Why Chaos Testing is Critical for Rideshare

### **1. Distributed System Complexity**
- **5 Core Services** that must work together
- **Multiple Databases** with different failure modes
- **External Dependencies** (payment processors, maps APIs)
- **Network Partitions** between services

### **2. Real-Time Requirements**
- **Driver Matching** must continue even during partial failures
- **Payment Processing** cannot fail during trip completion
- **Location Updates** must be resilient to network issues
- **Surge Pricing** must adapt to service degradation

### **3. Financial Impact**
- **Revenue Loss** from service outages
- **Customer Churn** from poor experiences
- **Driver Income** impact from platform unavailability
- **Regulatory Issues** from payment failures

---

## 🔧 Chaos Testing Framework

### **1. Chaos Testing Infrastructure**
```go
type ChaosEngine struct {
    services     map[string]ServiceChaosController
    scenarios    []ChaosScenario
    metrics      *monitoring.MetricsCollector
    logger       *logger.Logger
    scheduler    *ChaosScheduler
}

type ChaosScenario struct {
    Name            string                `json:"name"`
    Description     string                `json:"description"`
    TargetServices  []string              `json:"target_services"`
    FailureTypes    []FailureType         `json:"failure_types"`
    Duration        time.Duration         `json:"duration"`
    Impact          ImpactLevel           `json:"impact"`
    Schedule        string                `json:"schedule"` // cron expression
    Enabled         bool                  `json:"enabled"`
    SafetyChecks    []SafetyCheck         `json:"safety_checks"`
}

type FailureType string

const (
    FailureTypeLatency          FailureType = "latency"           // Introduce delays
    FailureTypeNetworkPartition FailureType = "network_partition" // Isolate services
    FailureTypeServiceShutdown  FailureType = "service_shutdown"  // Kill service instances
    FailureTypeDatabaseFailure  FailureType = "database_failure"  // DB connection issues
    FailureTypeMemoryExhaustion FailureType = "memory_exhaustion" // Resource limits
    FailureTypeCPUStarvation    FailureType = "cpu_starvation"    // CPU limits
    FailureTypeDiskFull         FailureType = "disk_full"         // Storage issues
    FailureTypeTrafficSpike     FailureType = "traffic_spike"     // Load testing
)

type ImpactLevel string

const (
    ImpactLow    ImpactLevel = "low"    // < 5% user impact
    ImpactMedium ImpactLevel = "medium" // 5-15% user impact  
    ImpactHigh   ImpactLevel = "high"   // 15-30% user impact
)
```

### **2. Service-Specific Chaos Controllers**
```go
type ServiceChaosController interface {
    InjectLatency(duration time.Duration) error
    PartitionNetwork(targetServices []string) error
    KillInstances(percentage float64) error
    ExhaustResources(resourceType string, percentage float64) error
    RestoreService() error
    GetHealthStatus() ServiceHealthStatus
}

type MatchingServiceChaosController struct {
    kubernetesClient kubernetes.Interface
    serviceName      string
    namespace        string
    logger           *logger.Logger
}

func (mscc *MatchingServiceChaosController) InjectLatency(duration time.Duration) error {
    // Use istio to inject latency into matching service
    return mscc.applyIstioFaultInjection(&IstioFault{
        Type:     "delay",
        Duration: duration,
        Percentage: 50, // Affect 50% of requests
    })
}

func (mscc *MatchingServiceChaosController) KillInstances(percentage float64) error {
    // Get current replicas
    deployment, err := mscc.kubernetesClient.AppsV1().Deployments(mscc.namespace).
        Get(context.Background(), mscc.serviceName, metav1.GetOptions{})
    if err != nil {
        return err
    }
    
    currentReplicas := *deployment.Spec.Replicas
    targetReplicas := int32(float64(currentReplicas) * (1.0 - percentage))
    
    // Ensure at least 1 replica remains
    if targetReplicas < 1 {
        targetReplicas = 1
    }
    
    // Scale down deployment
    deployment.Spec.Replicas = &targetReplicas
    _, err = mscc.kubernetesClient.AppsV1().Deployments(mscc.namespace).
        Update(context.Background(), deployment, metav1.UpdateOptions{})
    
    mscc.logger.Info("Killed service instances", 
        "service", mscc.serviceName,
        "from_replicas", currentReplicas,
        "to_replicas", targetReplicas,
        "kill_percentage", percentage)
    
    return err
}
```

---

## 🎪 Chaos Testing Scenarios

### **1. Database Failure Scenarios**
```go
func TestDatabasePartitionResilience(t *testing.T) {
    chaosEngine := SetupChaosTestEnvironment(t)
    defer chaosEngine.Cleanup()
    
    scenario := &ChaosScenario{
        Name: "PostgreSQL Connection Failure",
        Description: "Simulate PostgreSQL connection timeout to test circuit breaker",
        TargetServices: []string{"trip-service", "user-service", "payment-service"},
        FailureTypes: []FailureType{FailureTypeDatabaseFailure},
        Duration: 5 * time.Minute,
        Impact: ImpactMedium,
    }
    
    // Start baseline metrics collection
    baselineMetrics := chaosEngine.CollectBaselineMetrics(30 * time.Second)
    
    // Execute chaos scenario
    err := chaosEngine.ExecuteScenario(scenario)
    assert.NoError(t, err)
    
    // Monitor system behavior during failure
    failureMetrics := chaosEngine.MonitorDuringFailure(scenario.Duration)
    
    // Restore services
    err = chaosEngine.RestoreServices(scenario.TargetServices)
    assert.NoError(t, err)
    
    // Collect recovery metrics
    recoveryMetrics := chaosEngine.CollectRecoveryMetrics(2 * time.Minute)
    
    // Analyze results
    analysis := chaosEngine.AnalyzeImpact(baselineMetrics, failureMetrics, recoveryMetrics)
    
    // Assertions for acceptable degradation
    assert.LessOrEqual(t, analysis.UserImpactPercentage, 15.0, 
        "User impact should be <= 15% during database failure")
    assert.LessOrEqual(t, analysis.RecoveryTimeSeconds, 60.0,
        "Recovery time should be <= 60 seconds")
    assert.GreaterOrEqual(t, analysis.CircuitBreakerTriggered, 1,
        "Circuit breakers should have activated")
    
    t.Logf("Database failure impact analysis:")
    t.Logf("  User Impact: %.2f%%", analysis.UserImpactPercentage)
    t.Logf("  Recovery Time: %.2f seconds", analysis.RecoveryTimeSeconds)
    t.Logf("  Successful Requests During Failure: %.2f%%", analysis.SuccessRateDuringFailure)
}
```

### **2. Service Cascade Failure Testing**
```go
func TestServiceCascadeFailure(t *testing.T) {
    chaosEngine := SetupChaosTestEnvironment(t)
    defer chaosEngine.Cleanup()
    
    // Simulate matching service becoming completely unavailable
    scenario := &ChaosScenario{
        Name: "Matching Service Complete Failure",
        Description: "Test system behavior when matching service is completely down",
        TargetServices: []string{"matching-service"},
        FailureTypes: []FailureType{FailureTypeServiceShutdown},
        Duration: 10 * time.Minute,
        Impact: ImpactHigh,
    }
    
    // Pre-failure: System should be healthy
    preFailureHealth := chaosEngine.CheckSystemHealth()
    assert.True(t, preFailureHealth.AllServicesHealthy)
    
    // Execute failure
    err := chaosEngine.ExecuteScenario(scenario)
    assert.NoError(t, err)
    
    // Test that other services handle the failure gracefully
    time.Sleep(30 * time.Second) // Allow failure to propagate
    
    // Try to request a trip (should gracefully degrade)
    tripRequest := &TripRequest{
        RiderID: "chaos-test-rider",
        PickupLocation: &models.Location{Latitude: 40.7128, Longitude: -74.0060},
        DestinationLocation: &models.Location{Latitude: 40.7580, Longitude: -73.9855},
    }
    
    response, err := chaosEngine.TripService.RequestTrip(context.Background(), tripRequest)
    
    // Should either succeed with degraded matching or fail gracefully
    if err != nil {
        // If it fails, it should be a known, graceful failure
        assert.Contains(t, err.Error(), "matching service unavailable")
        assert.Contains(t, err.Error(), "please try again later")
    } else {
        // If it succeeds, it should use fallback matching
        assert.Contains(t, response.Message, "fallback matching")
    }
    
    // Verify circuit breakers are open
    circuitBreakerStatus := chaosEngine.GetCircuitBreakerStatus()
    assert.True(t, circuitBreakerStatus["trip-to-matching"].IsOpen)
    
    // Verify system doesn't completely crash
    systemHealth := chaosEngine.CheckSystemHealth()
    assert.True(t, systemHealth.APIGatewayHealthy)
    assert.True(t, systemHealth.TripServiceHealthy)
    assert.True(t, systemHealth.PricingServiceHealthy)
    
    // Restore matching service
    err = chaosEngine.RestoreServices([]string{"matching-service"})
    assert.NoError(t, err)
    
    // Verify recovery
    time.Sleep(2 * time.Minute) // Allow service to fully recover
    
    // Circuit breakers should close
    circuitBreakerStatus = chaosEngine.GetCircuitBreakerStatus()
    assert.False(t, circuitBreakerStatus["trip-to-matching"].IsOpen)
    
    // Normal trip requests should work again
    response, err = chaosEngine.TripService.RequestTrip(context.Background(), tripRequest)
    assert.NoError(t, err)
    assert.NotContains(t, response.Message, "fallback")
}
```

### **3. Network Partition Testing**
```go
func TestNetworkPartitionBetweenServices(t *testing.T) {
    chaosEngine := SetupChaosTestEnvironment(t)
    defer chaosEngine.Cleanup()
    
    // Create network partition between matching and geo services
    scenario := &ChaosScenario{
        Name: "Matching-Geo Network Partition",
        Description: "Isolate matching service from geo service",
        TargetServices: []string{"matching-service"},
        FailureTypes: []FailureType{FailureTypeNetworkPartition},
        Duration: 3 * time.Minute,
        Impact: ImpactMedium,
    }
    
    // Configure network partition
    partitionConfig := &NetworkPartitionConfig{
        SourceService: "matching-service",
        TargetServices: []string{"geo-service"},
        BlockPercentage: 100, // Complete isolation
    }
    
    err := chaosEngine.CreateNetworkPartition(partitionConfig)
    assert.NoError(t, err)
    
    // Test matching service behavior without geo service
    matchingRequest := &MatchingRequest{
        TripID: "partition-test",
        PickupLocation: &models.Location{Latitude: 40.7128, Longitude: -74.0060},
        PreferredVehicleType: "standard",
    }
    
    result, err := chaosEngine.MatchingService.FindMatch(context.Background(), matchingRequest)
    
    // Matching should either:
    // 1. Use cached distance data
    // 2. Use fallback distance calculation
    // 3. Fail gracefully with retry mechanism
    
    if err != nil {
        assert.Contains(t, err.Error(), "geo service unavailable")
    } else {
        assert.True(t, result.Success)
        assert.Greater(t, result.SearchDuration.Seconds(), 1.0) // Should take longer due to retries
    }
    
    // Remove network partition
    err = chaosEngine.RemoveNetworkPartition(partitionConfig)
    assert.NoError(t, err)
    
    // Verify services reconnect
    time.Sleep(30 * time.Second)
    
    result, err = chaosEngine.MatchingService.FindMatch(context.Background(), matchingRequest)
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Less(t, result.SearchDuration.Seconds(), 3.0) // Should be fast again
}
```

### **4. Resource Exhaustion Testing**
```go
func TestMemoryExhaustionResilience(t *testing.T) {
    chaosEngine := SetupChaosTestEnvironment(t)
    defer chaosEngine.Cleanup()
    
    // Target payment service with memory pressure
    scenario := &ChaosScenario{
        Name: "Payment Service Memory Exhaustion",
        Description: "Test payment service behavior under memory pressure",
        TargetServices: []string{"payment-service"},
        FailureTypes: []FailureType{FailureTypeMemoryExhaustion},
        Duration: 2 * time.Minute,
        Impact: ImpactLow,
    }
    
    // Apply memory pressure (consume 90% of available memory)
    err := chaosEngine.ApplyMemoryPressure("payment-service", 0.9)
    assert.NoError(t, err)
    
    // Test payment processing under memory pressure
    for i := 0; i < 10; i++ {
        paymentRequest := &PaymentRequest{
            TripID: fmt.Sprintf("memory-test-%d", i),
            UserID: "test-user",
            AmountCents: 2500,
            Currency: "USD",
            PaymentMethod: &PaymentMethod{Type: "credit_card", Token: "test-token"},
        }
        
        response, err := chaosEngine.PaymentService.ProcessPayment(context.Background(), paymentRequest)
        
        // Payment should either succeed or fail gracefully
        if err != nil {
            assert.Contains(t, err.Error(), "service temporarily unavailable")
        } else {
            assert.True(t, response.Success)
        }
        
        // No payment should be partially processed
        if response != nil && !response.Success {
            assert.Empty(t, response.TransactionID)
        }
        
        time.Sleep(5 * time.Second)
    }
    
    // Remove memory pressure
    err = chaosEngine.RemoveMemoryPressure("payment-service")
    assert.NoError(t, err)
    
    // Verify service recovery
    time.Sleep(30 * time.Second)
    
    paymentRequest := &PaymentRequest{
        TripID: "recovery-test",
        UserID: "test-user",
        AmountCents: 2500,
        Currency: "USD",
        PaymentMethod: &PaymentMethod{Type: "credit_card", Token: "test-token"},
    }
    
    response, err := chaosEngine.PaymentService.ProcessPayment(context.Background(), paymentRequest)
    assert.NoError(t, err)
    assert.True(t, response.Success)
}
```

---

## 📊 Chaos Testing Metrics & Analysis

### **1. Resilience Metrics**
```go
type ResilienceMetrics struct {
    // Service Availability
    ServiceUptime          map[string]float64 `json:"service_uptime"`          // %
    ServiceResponseTime    map[string]float64 `json:"service_response_time"`   // ms
    ServiceErrorRate       map[string]float64 `json:"service_error_rate"`      // %
    
    // User Experience
    UserImpactPercentage   float64 `json:"user_impact_percentage"`   // %
    SuccessfulTrips        int     `json:"successful_trips"`         // count
    FailedTrips           int     `json:"failed_trips"`             // count
    AverageWaitTime       float64 `json:"average_wait_time"`        // seconds
    
    // System Recovery
    RecoveryTimeSeconds   float64 `json:"recovery_time_seconds"`    // seconds
    CircuitBreakerCount   int     `json:"circuit_breaker_count"`    // count
    RetryAttempts         int     `json:"retry_attempts"`           // count
    
    // Business Impact
    RevenueImpactDollars  float64 `json:"revenue_impact_dollars"`   // $
    DriversAffected       int     `json:"drivers_affected"`         // count
    UsersAffected         int     `json:"users_affected"`           // count
}

func (ce *ChaosEngine) AnalyzeImpact(baseline, failure, recovery *ResilienceMetrics) *ImpactAnalysis {
    analysis := &ImpactAnalysis{
        Timestamp: time.Now(),
        Scenario:  ce.currentScenario.Name,
    }
    
    // Calculate user impact
    baselineSuccessRate := float64(baseline.SuccessfulTrips) / float64(baseline.SuccessfulTrips + baseline.FailedTrips)
    failureSuccessRate := float64(failure.SuccessfulTrips) / float64(failure.SuccessfulTrips + failure.FailedTrips)
    
    analysis.UserImpactPercentage = (baselineSuccessRate - failureSuccessRate) * 100
    
    // Calculate recovery metrics
    analysis.RecoveryTimeSeconds = recovery.RecoveryTimeSeconds
    analysis.CircuitBreakerEffectiveness = failure.CircuitBreakerCount > 0
    
    // Calculate business impact
    baselineRevenue := baseline.SuccessfulTrips * 15.0 // Assume $15 average trip
    failureRevenue := failure.SuccessfulTrips * 15.0
    analysis.RevenueImpactDollars = float64(baselineRevenue - failureRevenue)
    
    return analysis
}
```

### **2. Automated Chaos Reporting**
```go
func (ce *ChaosEngine) GenerateResilienceReport(scenarios []ChaosScenario, results []ChaosResult) *ResilienceReport {
    report := &ResilienceReport{
        GeneratedAt: time.Now(),
        Summary: &ResilienceSummary{},
        Scenarios: make([]ScenarioReport, len(scenarios)),
    }
    
    totalTests := len(results)
    passedTests := 0
    
    for i, result := range results {
        scenarioReport := ScenarioReport{
            ScenarioName: scenarios[i].Name,
            Duration: scenarios[i].Duration,
            Impact: scenarios[i].Impact,
            Success: result.Passed,
            Metrics: result.Metrics,
            Recommendations: ce.generateRecommendations(result),
        }
        
        if result.Passed {
            passedTests++
        }
        
        report.Scenarios[i] = scenarioReport
    }
    
    report.Summary.TotalScenarios = totalTests
    report.Summary.PassedScenarios = passedTests
    report.Summary.ResilienceScore = float64(passedTests) / float64(totalTests) * 100
    report.Summary.OverallRisk = ce.calculateOverallRisk(results)
    
    return report
}

func (ce *ChaosEngine) generateRecommendations(result ChaosResult) []string {
    var recommendations []string
    
    if result.Metrics.UserImpactPercentage > 20 {
        recommendations = append(recommendations, 
            "Consider implementing additional circuit breakers")
        recommendations = append(recommendations,
            "Improve fallback mechanisms for critical services")
    }
    
    if result.Metrics.RecoveryTimeSeconds > 120 {
        recommendations = append(recommendations,
            "Optimize service restart procedures")
        recommendations = append(recommendations,
            "Implement faster health check mechanisms")
    }
    
    if result.Metrics.ServiceErrorRate["payment-service"] > 5 {
        recommendations = append(recommendations,
            "Critical: Improve payment service resilience - financial impact detected")
    }
    
    return recommendations
}
```

---

## 🔄 Continuous Chaos Testing

### **1. Automated Chaos Scheduling**
```go
type ChaosScheduler struct {
    scenarios []ChaosScenario
    cron      *cron.Cron
    engine    *ChaosEngine
    config    *ChaosConfig
}

func (cs *ChaosScheduler) ScheduleRegularChaos() {
    // Schedule daily resilience tests
    cs.cron.AddFunc("0 2 * * *", func() { // 2 AM daily
        cs.runDailyResilienceTests()
    })
    
    // Schedule weekly disaster recovery tests  
    cs.cron.AddFunc("0 3 * * 0", func() { // 3 AM every Sunday
        cs.runWeeklyDisasterRecoveryTests()
    })
    
    // Schedule random small failures throughout the day
    cs.cron.AddFunc("*/30 * * * *", func() { // Every 30 minutes
        if cs.shouldRunRandomChaos() {
            cs.runRandomLowImpactChaos()
        }
    })
    
    cs.cron.Start()
}

func (cs *ChaosScheduler) runDailyResilienceTests() {
    scenarios := []ChaosScenario{
        {
            Name: "Database Connection Pool Exhaustion",
            TargetServices: []string{"trip-service"},
            FailureTypes: []FailureType{FailureTypeDatabaseFailure},
            Duration: 2 * time.Minute,
            Impact: ImpactLow,
        },
        {
            Name: "API Gateway Latency Injection",
            TargetServices: []string{"api-gateway"},
            FailureTypes: []FailureType{FailureTypeLatency},
            Duration: 3 * time.Minute,
            Impact: ImpactLow,
        },
    }
    
    for _, scenario := range scenarios {
        if cs.config.MaintenanceWindow.IsInWindow(time.Now()) {
            cs.engine.ExecuteScenario(&scenario)
        }
    }
}
```

### **2. Game Day Exercises**
```go
type GameDayExercise struct {
    Name                string
    Description         string
    Scenarios           []ChaosScenario
    Participants        []string
    Duration            time.Duration
    LearningObjectives  []string
    SuccessCriteria     []string
}

func RunGameDayExercise(exercise *GameDayExercise) *GameDayResult {
    result := &GameDayResult{
        Exercise: exercise,
        StartTime: time.Now(),
        Events: make([]GameDayEvent, 0),
    }
    
    // Simulate major outage scenario
    majorOutageScenario := &ChaosScenario{
        Name: "Multi-Service Cascade Failure",
        Description: "Payment service fails, causing matching delays and trip cancellations",
        TargetServices: []string{"payment-service", "matching-service"},
        FailureTypes: []FailureType{
            FailureTypeServiceShutdown,
            FailureTypeLatency,
        },
        Duration: 15 * time.Minute,
        Impact: ImpactHigh,
    }
    
    // Execute scenario and track team response
    chaosEngine := SetupChaosEngine()
    
    // Start the chaos
    err := chaosEngine.ExecuteScenario(majorOutageScenario)
    if err != nil {
        result.Events = append(result.Events, GameDayEvent{
            Time: time.Now(),
            Type: "chaos_execution_failed",
            Description: err.Error(),
        })
        return result
    }
    
    // Monitor incident response
    for elapsed := time.Duration(0); elapsed < exercise.Duration; elapsed += 30 * time.Second {
        metrics := chaosEngine.CollectMetrics()
        
        event := GameDayEvent{
            Time: time.Now(),
            Type: "metrics_snapshot",
            Metrics: metrics,
        }
        result.Events = append(result.Events, event)
        
        // Check if team triggered appropriate responses
        if metrics.CircuitBreakerCount > 0 && !result.CircuitBreakerTriggered {
            result.CircuitBreakerTriggered = true
            result.Events = append(result.Events, GameDayEvent{
                Time: time.Now(),
                Type: "circuit_breaker_triggered",
                Description: "Team successfully activated circuit breakers",
            })
        }
        
        time.Sleep(30 * time.Second)
    }
    
    // Restore services
    chaosEngine.RestoreAllServices()
    result.EndTime = time.Now()
    
    return result
}
```

---

## 🎯 Chaos Testing Best Practices

### **1. Safety Measures**
```go
type SafetyController struct {
    maxUserImpact    float64
    maxRevenueLoss   float64
    emergencyStopCh  chan bool
    monitoringClient *monitoring.Client
}

func (sc *SafetyController) MonitorSafety(scenario *ChaosScenario) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-sc.emergencyStopCh:
            return
        case <-ticker.C:
            metrics := sc.monitoringClient.GetCurrentMetrics()
            
            // Check user impact threshold
            if metrics.UserImpactPercentage > sc.maxUserImpact {
                sc.emergencyStop("User impact exceeded threshold")
                return
            }
            
            // Check revenue impact threshold
            if metrics.RevenueImpactPerMinute > sc.maxRevenueLoss {
                sc.emergencyStop("Revenue loss exceeded threshold")
                return
            }
            
            // Check for critical service failures
            if metrics.PaymentServiceErrorRate > 50 {
                sc.emergencyStop("Critical payment service failure")
                return
            }
        }
    }
}

func (sc *SafetyController) emergencyStop(reason string) {
    log.Critical("EMERGENCY STOP triggered", "reason", reason)
    
    // Immediately restore all services
    chaosEngine.EmergencyRestore()
    
    // Alert on-call engineers
    alerting.SendCriticalAlert("Chaos test emergency stop", reason)
    
    // Stop all ongoing chaos scenarios
    close(sc.emergencyStopCh)
}
```

### **2. Gradual Rollout Strategy**
```go
func (ce *ChaosEngine) GradualChaosRollout(scenario *ChaosScenario) error {
    // Start with 1% impact
    scenario.Impact = ImpactLow
    scenario.Duration = 30 * time.Second
    
    phases := []struct {
        impactPercentage float64
        duration         time.Duration
        maxFailureRate   float64
    }{
        {1.0, 30 * time.Second, 5.0},
        {5.0, 1 * time.Minute, 10.0},
        {10.0, 2 * time.Minute, 15.0},
        {20.0, 3 * time.Minute, 25.0},
    }
    
    for i, phase := range phases {
        log.Info("Starting chaos phase", "phase", i+1, "impact", phase.impactPercentage)
        
        // Adjust scenario for this phase
        adjustedScenario := *scenario
        adjustedScenario.Duration = phase.duration
        
        err := ce.ExecuteScenario(&adjustedScenario)
        if err != nil {
            return fmt.Errorf("phase %d failed: %w", i+1, err)
        }
        
        // Monitor results
        metrics := ce.CollectMetrics()
        if metrics.FailureRate > phase.maxFailureRate {
            log.Warn("Phase failure rate exceeded threshold, stopping rollout",
                "phase", i+1,
                "failure_rate", metrics.FailureRate,
                "threshold", phase.maxFailureRate)
            break
        }
        
        // Wait between phases
        time.Sleep(1 * time.Minute)
    }
    
    return nil
}
```

---

This chaos engineering documentation provides a comprehensive framework for testing the resilience of the rideshare platform. The testing strategies ensure that the system can handle real-world failures gracefully while maintaining user safety and business continuity.

The chaos testing approach demonstrates enterprise-level resilience engineering practices used by major technology companies to build robust, fault-tolerant systems at scale.
