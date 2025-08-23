# Testing Strategy

This document outlines the comprehensive testing strategy for the rideshare platform, covering all levels of testing from unit tests to end-to-end scenarios.

## Testing Philosophy

Our testing approach follows the **Test Pyramid** principle with emphasis on:

- **Fast Feedback**: Quick unit tests for immediate feedback
- **Confidence**: Integration tests for service interactions
- **Reality**: End-to-end tests for user scenarios
- **Reliability**: Contract tests for API compatibility
- **Performance**: Load tests for scalability validation

## Testing Levels

### 1. Unit Testing (70% of tests)

#### Scope
- Individual functions and methods
- Business logic validation
- Domain model behavior
- Utility functions
- Algorithm correctness

#### Tools and Frameworks
- **Go**: `testing` package, `testify/assert`, `testify/mock`
- **Coverage Target**: 85%+ for business logic
- **Mocking**: Interface-based mocking for dependencies

#### Example Test Structure

```go
// services/pricing-service/internal/service/pricing_test.go
func TestPricingService_CalculateFare(t *testing.T) {
    tests := []struct {
        name           string
        distance       float64
        duration       int
        vehicleType    string
        surgeMultiplier float64
        expected       *Money
        expectError    bool
    }{
        {
            name:           "basic sedan fare",
            distance:       10.5,
            duration:       1800,
            vehicleType:    "sedan",
            surgeMultiplier: 1.0,
            expected:       &Money{Amount: 1250, Currency: "USD"},
            expectError:    false,
        },
        {
            name:           "surge pricing applied",
            distance:       5.0,
            duration:       900,
            vehicleType:    "sedan",
            surgeMultiplier: 2.5,
            expected:       &Money{Amount: 1875, Currency: "USD"},
            expectError:    false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockRepo := &MockPricingRepository{}
            mockRepo.On("GetPricingRules", tt.vehicleType).Return(defaultPricingRules, nil)
            
            service := NewPricingService(mockRepo)
            
            // Act
            result, err := service.CalculateFare(context.Background(), &FareRequest{
                Distance:        tt.distance,
                Duration:        tt.duration,
                VehicleType:     tt.vehicleType,
                SurgeMultiplier: tt.surgeMultiplier,
            })
            
            // Assert
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected.Amount, result.Total.Amount)
                assert.Equal(t, tt.expected.Currency, result.Total.Currency)
            }
        })
    }
}
```

#### Critical Unit Test Areas

**User Service**
- Password hashing and validation
- JWT token generation and validation
- User profile validation
- Driver status transitions

**Geo Service**
- Haversine distance calculations
- Geohash encoding/decoding
- Route optimization algorithms
- Location validation

**Matching Service**
- Driver-rider matching algorithms
- Proximity calculations
- Availability filtering
- Dispatch optimization

**Pricing Service**
- Fare calculation logic
- Surge pricing algorithms
- Promo code validation
- Time-based pricing rules

**Trip Service**
- Trip state machine transitions
- Event sourcing logic
- Trip validation rules
- Duration and distance calculations

### 2. Integration Testing (20% of tests)

#### Scope
- Service-to-service communication
- Database interactions
- External API integrations
- Message queue interactions
- Cache operations

#### Tools and Frameworks
- **Testcontainers**: For database and infrastructure testing
- **Docker Compose**: Test environment setup
- **gRPC Testing**: Service communication validation

#### Database Integration Tests

```go
// services/user-service/internal/repository/user_repository_test.go
func TestUserRepository_Integration(t *testing.T) {
    // Setup test database container
    ctx := context.Background()
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("test_db"),
        postgres.WithUsername("test_user"),
        postgres.WithPassword("test_pass"),
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)
    
    // Get connection string
    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    // Setup repository
    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)
    defer db.Close()
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    repo := NewUserRepository(db)
    
    t.Run("CreateUser", func(t *testing.T) {
        user := &User{
            Email:     "test@example.com",
            FirstName: "John",
            LastName:  "Doe",
            UserType:  "rider",
        }
        
        createdUser, err := repo.CreateUser(ctx, user)
        assert.NoError(t, err)
        assert.NotEmpty(t, createdUser.ID)
        assert.Equal(t, user.Email, createdUser.Email)
    })
    
    t.Run("GetUserByEmail", func(t *testing.T) {
        user, err := repo.GetUserByEmail(ctx, "test@example.com")
        assert.NoError(t, err)
        assert.Equal(t, "test@example.com", user.Email)
    })
}
```

#### gRPC Service Integration Tests

```go
// tests/integration/grpc_test.go
func TestGRPCServiceIntegration(t *testing.T) {
    // Start test services
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Start user service
    userServiceAddr := startUserService(t)
    defer stopUserService()
    
    // Start pricing service
    pricingServiceAddr := startPricingService(t)
    defer stopPricingService()
    
    // Create gRPC clients
    userConn, err := grpc.Dial(userServiceAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer userConn.Close()
    
    pricingConn, err := grpc.Dial(pricingServiceAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer pricingConn.Close()
    
    userClient := userpb.NewUserServiceClient(userConn)
    pricingClient := pricingpb.NewPricingServiceClient(pricingConn)
    
    t.Run("UserServiceHealthCheck", func(t *testing.T) {
        resp, err := userClient.GetUser(ctx, &userpb.GetUserRequest{
            Id: "non-existent",
        })
        assert.Error(t, err)
        assert.Nil(t, resp)
    })
    
    t.Run("PricingServiceCalculation", func(t *testing.T) {
        resp, err := pricingClient.CalculateFare(ctx, &pricingpb.CalculateFareRequest{
            PickupLocation: &commonpb.Location{
                Latitude:  37.7749,
                Longitude: -122.4194,
            },
            Destination: &commonpb.Location{
                Latitude:  37.7849,
                Longitude: -122.4094,
            },
            VehicleType: "sedan",
        })
        assert.NoError(t, err)
        assert.NotNil(t, resp)
        assert.Greater(t, resp.FareBreakdown.Total.Amount, int64(0))
    })
}
```

### 3. Contract Testing (5% of tests)

#### Scope
- API contract validation
- gRPC service contracts
- GraphQL schema validation
- Event message contracts

#### Tools and Frameworks
- **Pact**: Consumer-driven contract testing
- **Protocol Buffer validation**: Schema compatibility
- **GraphQL schema testing**: Schema validation

#### Pact Contract Tests

```go
// tests/contract/user_service_contract_test.go
func TestUserServiceContract(t *testing.T) {
    pact := &dsl.Pact{
        Consumer: "api-gateway",
        Provider: "user-service",
        Host:     "localhost",
        Port:     8001,
    }
    defer pact.Teardown()
    
    t.Run("GetUser", func(t *testing.T) {
        pact.
            AddInteraction().
            Given("User exists").
            UponReceiving("A request for user details").
            WithRequest(dsl.Request{
                Method: "POST",
                Path:   dsl.String("/user.UserService/GetUser"),
                Headers: dsl.MapMatcher{
                    "Content-Type": dsl.String("application/grpc"),
                },
                Body: dsl.Match(&userpb.GetUserRequest{
                    Id: "user-123",
                }),
            }).
            WillRespondWith(dsl.Response{
                Status: 200,
                Body: dsl.Match(&userpb.GetUserResponse{
                    User: &userpb.User{
                        Id:        "user-123",
                        Email:     "test@example.com",
                        FirstName: "John",
                        LastName:  "Doe",
                    },
                }),
            })
        
        err := pact.Verify(func() error {
            // Test implementation
            return nil
        })
        assert.NoError(t, err)
    })
}
```

### 4. End-to-End Testing (4% of tests)

#### Scope
- Complete user journeys
- Cross-service workflows
- Real-world scenarios
- UI/API integration

#### Tools and Frameworks
- **Playwright**: Web UI testing
- **GraphQL testing**: API workflow validation
- **Docker Compose**: Full environment setup

#### E2E Test Scenarios

```go
// tests/e2e/ride_journey_test.go
func TestCompleteRideJourney(t *testing.T) {
    // Setup test environment
    env := setupE2EEnvironment(t)
    defer env.Teardown()
    
    // Test data
    riderEmail := "rider@test.com"
    driverEmail := "driver@test.com"
    
    t.Run("CompleteRideFlow", func(t *testing.T) {
        // Step 1: Register rider and driver
        rider := registerUser(t, env.APIClient, riderEmail, "rider")
        driver := registerUser(t, env.APIClient, driverEmail, "driver")
        
        // Step 2: Driver goes online
        err := env.APIClient.UpdateDriverStatus(driver.ID, "online")
        assert.NoError(t, err)
        
        // Step 3: Driver updates location
        err = env.APIClient.UpdateDriverLocation(driver.ID, &Location{
            Latitude:  37.7749,
            Longitude: -122.4194,
        })
        assert.NoError(t, err)
        
        // Step 4: Rider requests ride
        rideRequest, err := env.APIClient.RequestRide(&RideRequest{
            RiderID: rider.ID,
            PickupLocation: &Location{
                Latitude:  37.7750,
                Longitude: -122.4195,
            },
            Destination: &Location{
                Latitude:  37.7850,
                Longitude: -122.4095,
            },
            VehicleType: "sedan",
        })
        assert.NoError(t, err)
        assert.Equal(t, "pending", rideRequest.Status)
        
        // Step 5: Wait for matching
        time.Sleep(2 * time.Second)
        
        // Step 6: Driver accepts ride
        trip, err := env.APIClient.AcceptRide(rideRequest.ID, driver.ID)
        assert.NoError(t, err)
        assert.Equal(t, "driver_assigned", trip.Status)
        
        // Step 7: Simulate trip progression
        err = env.APIClient.UpdateTripStatus(trip.ID, "driver_arriving")
        assert.NoError(t, err)
        
        err = env.APIClient.UpdateTripStatus(trip.ID, "driver_arrived")
        assert.NoError(t, err)
        
        err = env.APIClient.UpdateTripStatus(trip.ID, "trip_started")
        assert.NoError(t, err)
        
        err = env.APIClient.UpdateTripStatus(trip.ID, "completed")
        assert.NoError(t, err)
        
        // Step 8: Verify trip completion
        completedTrip, err := env.APIClient.GetTrip(trip.ID)
        assert.NoError(t, err)
        assert.Equal(t, "completed", completedTrip.Status)
        assert.NotNil(t, completedTrip.Fare)
        assert.Greater(t, completedTrip.Fare.Amount, int64(0))
        
        // Step 9: Verify payment processing
        payment, err := env.APIClient.GetPaymentByTripID(trip.ID)
        assert.NoError(t, err)
        assert.Equal(t, "completed", payment.Status)
    })
}
```

### 5. Performance Testing (1% of tests)

#### Scope
- Load testing
- Stress testing
- Spike testing
- Volume testing
- Endurance testing

#### Tools and Frameworks
- **K6**: Load testing
- **Artillery**: API load testing
- **JMeter**: Comprehensive performance testing

#### Load Test Scenarios

```javascript
// tests/load/api_load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp up to 200 users
    { duration: '5m', target: 200 }, // Stay at 200 users
    { duration: '2m', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.1'],    // Error rate under 10%
    errors: ['rate<0.1'],
  },
};

const BASE_URL = 'http://localhost:8080';

export default function() {
  // Test GraphQL queries
  let query = `
    query {
      nearbyDrivers(
        location: { latitude: 37.7749, longitude: -122.4194 }
        radius: 5.0
        vehicleType: SEDAN
        limit: 10
      ) {
        driverId
        distance
        rating
      }
    }
  `;
  
  let response = http.post(`${BASE_URL}/graphql`, JSON.stringify({
    query: query
  }), {
    headers: {
      'Content-Type': 'application/json',
    },
  });
  
  let success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
    'has nearby drivers': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.nearbyDrivers && body.data.nearbyDrivers.length > 0;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  sleep(1);
}

export function handleSummary(data) {
  return {
    'load_test_results.json': JSON.stringify(data),
  };
}
```

## Test Data Management

### Test Data Strategy

#### Static Test Data
- Predefined user accounts
- Standard vehicle configurations
- Fixed location coordinates
- Standard pricing rules

#### Dynamic Test Data
- Generated user profiles
- Random location data
- Variable pricing scenarios
- Simulated trip patterns

#### Test Data Factory

```go
// tests/testdata/factory.go
type TestDataFactory struct {
    faker *gofakeit.Faker
}

func NewTestDataFactory() *TestDataFactory {
    return &TestDataFactory{
        faker: gofakeit.New(0),
    }
}

func (f *TestDataFactory) CreateUser(userType string) *User {
    return &User{
        ID:        uuid.New().String(),
        Email:     f.faker.Email(),
        FirstName: f.faker.FirstName(),
        LastName:  f.faker.LastName(),
        Phone:     f.faker.Phone(),
        UserType:  userType,
        Status:    "active",
        CreatedAt: time.Now(),
    }
}

func (f *TestDataFactory) CreateLocation() *Location {
    // Generate locations within San Francisco area
    return &Location{
        Latitude:  37.7749 + (f.faker.Float64Range(-0.1, 0.1)),
        Longitude: -122.4194 + (f.faker.Float64Range(-0.1, 0.1)),
        Accuracy:  f.faker.Float64Range(5.0, 15.0),
        Timestamp: time.Now(),
    }
}

func (f *TestDataFactory) CreateTrip(riderID, driverID string) *Trip {
    return &Trip{
        ID:             uuid.New().String(),
        RiderID:        riderID,
        DriverID:       driverID,
        PickupLocation: f.CreateLocation(),
        Destination:    f.CreateLocation(),
        Status:         "requested",
        RequestedAt:    time.Now(),
    }
}
```

## Test Environment Management

### Environment Isolation

#### Local Development
- Docker Compose with isolated databases
- In-memory Redis for fast tests
- Mock external services

#### CI/CD Pipeline
- Containerized test environments
- Parallel test execution
- Test result aggregation

#### Staging Environment
- Production-like infrastructure
- Real external service integrations
- Performance baseline validation

### Test Database Management

```go
// tests/testutil/database.go
func SetupTestDatabase(t *testing.T) (*sql.DB, func()) {
    container, err := postgres.RunContainer(context.Background(),
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("test_db"),
        postgres.WithUsername("test_user"),
        postgres.WithPassword("test_pass"),
    )
    require.NoError(t, err)
    
    connStr, err := container.ConnectionString(context.Background(), "sslmode=disable")
    require.NoError(t, err)
    
    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    cleanup := func() {
        db.Close()
        container.Terminate(context.Background())
    }
    
    return db, cleanup
}
```

## Continuous Testing

### Test Automation Pipeline

```yaml
# .github/workflows/test.yml
name: Test Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run Unit Tests
      run: |
        make test-unit
        make test-coverage
    
    - name: Upload Coverage
      uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run Integration Tests
      run: make test-integration

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Start Services
      run: docker-compose -f docker-compose.test.yml up -d
    
    - name: Wait for Services
      run: make wait-for-services
    
    - name: Run E2E Tests
      run: make test-e2e
    
    - name: Cleanup
      run: docker-compose -f docker-compose.test.yml down

  performance-tests:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v3
    - name: Setup K6
      run: |
        sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
        echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
        sudo apt-get update
        sudo apt-get install k6
    
    - name: Start Services
      run: docker-compose -f docker-compose.prod.yml up -d
    
    - name: Run Load Tests
      run: k6 run tests/load/api_load_test.js
```

## Test Metrics and Reporting

### Key Metrics
- **Code Coverage**: Target 85%+ for business logic
- **Test Execution Time**: Unit tests < 10s, Integration tests < 2m
- **Test Reliability**: Flaky test rate < 1%
- **Performance Benchmarks**: API response time < 200ms (p95)

### Test Reporting

```go
// tests/testutil/reporter.go
type TestReporter struct {
    results []TestResult
}

func (r *TestReporter) RecordResult(result TestResult) {
    r.results = append(r.results, result)
}

func (r *TestReporter) GenerateReport() *TestReport {
    return &TestReport{
        TotalTests:   len(r.results),
        PassedTests:  r.countPassed(),
        FailedTests:  r.countFailed(),
        Coverage:     r.calculateCoverage(),
        Duration:     r.calculateTotalDuration(),
        Timestamp:    time.Now(),
    }
}
```

## Best Practices

### Test Organization
- Group tests by feature/domain
- Use descriptive test names
- Follow AAA pattern (Arrange, Act, Assert)
- Keep tests independent and isolated

### Test Maintenance
- Regular test review and cleanup
- Update tests with code changes
- Monitor and fix flaky tests
- Maintain test documentation

### Performance Considerations
- Parallel test execution
- Efficient test data setup/teardown
- Resource cleanup
- Test environment optimization

This comprehensive testing strategy ensures high-quality, reliable software delivery with confidence in system behavior across all scenarios and environments.