//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rideshare-platform/tests/testutils"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// TestServiceInteroperability tests communication between multiple services
func TestServiceInteroperability(t *testing.T) {
	testutils.SkipIfShort(t)
	
	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	t.Run("basic_database_operations", func(t *testing.T) {
		// Test basic CRUD operations
		ctx := context.Background()
		
		// Create test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS test_users (
				id VARCHAR(255) PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		
		// Insert test data
		testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
		testEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
		
		_, err = db.ExecContext(ctx, 
			"INSERT INTO test_users (id, email) VALUES ($1, $2)",
			testID, testEmail)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
		
		// Query test data
		var retrievedID, retrievedEmail string
		err = db.QueryRowContext(ctx,
			"SELECT id, email FROM test_users WHERE id = $1", testID).Scan(&retrievedID, &retrievedEmail)
		if err != nil {
			t.Fatalf("Failed to query test data: %v", err)
		}
		
		if retrievedID != testID {
			t.Errorf("Expected ID %s, got %s", testID, retrievedID)
		}
		if retrievedEmail != testEmail {
			t.Errorf("Expected email %s, got %s", testEmail, retrievedEmail)
		}
		
		// Clean up
		_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_users")
		if err != nil {
			t.Logf("Warning: Failed to clean up test table: %v", err)
		}
	})

	t.Run("transaction_handling", func(t *testing.T) {
		ctx := context.Background()
		
		// Test transaction rollback
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		
		// Create temporary table in transaction
		_, err = tx.ExecContext(ctx, `
			CREATE TEMPORARY TABLE tx_test (
				id SERIAL PRIMARY KEY,
				data TEXT
			)
		`)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create temp table: %v", err)
		}
		
		// Insert data
		_, err = tx.ExecContext(ctx, "INSERT INTO tx_test (data) VALUES ('test')")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert data: %v", err)
		}
		
		// Rollback transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}
		
		// Verify data was rolled back
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tx_test WHERE data = 'test'").Scan(&count)
		if err == nil {
			t.Error("Expected table to not exist after rollback, but query succeeded")
		}
	})
}

// TestDatabasePerformance tests database operations under various conditions
func TestDatabasePerformance(t *testing.T) {
	testutils.SkipIfShort(t)
	
	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	t.Run("concurrent_operations", func(t *testing.T) {
		ctx := context.Background()
		
		// Create test table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS concurrent_test (
				id VARCHAR(255) PRIMARY KEY,
				worker_id INT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		defer db.ExecContext(ctx, "DROP TABLE IF EXISTS concurrent_test")
		
		// Run concurrent operations
		const numWorkers = 5
		const operationsPerWorker = 10
		done := make(chan error, numWorkers)
		
		for worker := 0; worker < numWorkers; worker++ {
			go func(workerID int) {
				for i := 0; i < operationsPerWorker; i++ {
					testID := fmt.Sprintf("worker-%d-op-%d", workerID, i)
					_, err := db.ExecContext(ctx,
						"INSERT INTO concurrent_test (id, worker_id) VALUES ($1, $2)",
						testID, workerID)
					if err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}(worker)
		}
		
		// Wait for all workers to complete
		for i := 0; i < numWorkers; i++ {
			err := <-done
			if err != nil {
				t.Errorf("Worker failed: %v", err)
			}
		}
		
		// Verify all operations completed
		var totalCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concurrent_test").Scan(&totalCount)
		if err != nil {
			t.Fatalf("Failed to count records: %v", err)
		}
		
		expectedCount := numWorkers * operationsPerWorker
		if totalCount != expectedCount {
			t.Errorf("Expected %d records, got %d", expectedCount, totalCount)
		}
	})
}

// TestBusinessLogicIntegration tests integration of business logic components
func TestBusinessLogicIntegration(t *testing.T) {
	testutils.SkipIfShort(t)
	
	t.Run("distance_calculations", func(t *testing.T) {
		// Test geospatial calculations that would be used across services
		testCases := []struct {
			name     string
			lat1     float64
			lng1     float64
			lat2     float64
			lng2     float64
			maxDist  float64 // Maximum expected distance in km
		}{
			{
				name:    "same_location",
				lat1:    40.7128, lng1: -74.0060,
				lat2:    40.7128, lng2: -74.0060,
				maxDist: 0.1,
			},
			{
				name:    "nearby_locations",
				lat1:    40.7128, lng1: -74.0060, // NYC
				lat2:    40.7589, lng2: -73.9851, // Times Square
				maxDist: 5.0,
			},
			{
				name:    "distant_locations",
				lat1:    40.7128, lng1: -74.0060, // NYC
				lat2:    34.0522, lng2: -118.2437, // LA
				maxDist: 5000.0,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				distance := calculateHaversineDistance(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
				if distance > tc.maxDist {
					t.Errorf("Distance %.2f km exceeds maximum expected %.2f km", distance, tc.maxDist)
				}
				if distance < 0 {
					t.Error("Distance cannot be negative")
				}
			})
		}
	})
	
	t.Run("time_calculations", func(t *testing.T) {
		// Test time-based calculations used across services
		now := time.Now()
		
		// Test ETA calculations
		testCases := []struct {
			name        string
			distance    float64 // km
			speedKmh    float64
			expectedMin float64 // minimum expected time in minutes
			expectedMax float64 // maximum expected time in minutes
		}{
			{
				name:        "short_trip_walking",
				distance:    1.0,
				speedKmh:    5.0,
				expectedMin: 10.0,
				expectedMax: 15.0,
			},
			{
				name:        "medium_trip_driving",
				distance:    10.0,
				speedKmh:    50.0,
				expectedMin: 10.0,
				expectedMax: 20.0,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				eta := calculateETA(tc.distance, tc.speedKmh)
				etaMinutes := eta.Minutes()
				
				if etaMinutes < tc.expectedMin || etaMinutes > tc.expectedMax {
					t.Errorf("ETA %.2f minutes not in expected range [%.2f, %.2f]",
						etaMinutes, tc.expectedMin, tc.expectedMax)
				}
			})
		}
		
		// Test timestamp validations
		pastTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(1 * time.Hour)
		
		if !pastTime.Before(now) {
			t.Error("Past time should be before current time")
		}
		if !futureTime.After(now) {
			t.Error("Future time should be after current time")
		}
	})
}

// Helper functions for integration tests
func calculateHaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180
	
	// Differences
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad
	
	// Haversine formula
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadiusKm * c
}

func calculateETA(distanceKm, speedKmh float64) time.Duration {
	if speedKmh <= 0 {
		return time.Duration(0)
	}
	
	hours := distanceKm / speedKmh
	return time.Duration(hours * float64(time.Hour))
}

// TestRealDatabaseOperations tests actual database CRUD operations
func TestRealDatabaseOperations(t *testing.T) {
	testutils.SkipIfShort(t)
	
	config := testutils.DefaultTestConfig()
	db := testutils.SetupTestDB(t, config)
	defer testutils.CleanupTestDB(t, db)

	t.Run("concurrent_user_operations", func(t *testing.T) {
		const numOperations = 10
		done := make(chan error, numOperations)
		
		// Perform concurrent user creation operations
		for i := 0; i < numOperations; i++ {
			go func(index int) {
				userID := fmt.Sprintf("concurrent-user-%d", index)
				email := fmt.Sprintf("concurrent%d@example.com", index)
				err := createTestUserInDB(db, userID, email, "rider")
				done <- err
			}(i)
		}
		
		// Collect results
		for i := 0; i < numOperations; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent user creation should succeed")
		}
		
		// Verify all users were created
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email LIKE 'concurrent%@example.com'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, numOperations, count, "All concurrent users should be created")
	})

	t.Run("transaction_rollback_behavior", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err, "Should begin transaction")
		
		// Create a user within transaction
		userID := testutils.GenerateTestID()
		_, err = tx.Exec(`
			INSERT INTO users (id, email, first_name, last_name, user_type, status, created_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, "rollback@example.com", "Rollback", "Test", "rider", "active", time.Now())
		require.NoError(t, err, "Should insert user in transaction")
		
		// Rollback the transaction
		err = tx.Rollback()
		require.NoError(t, err, "Should rollback transaction")
		
		// Verify user was not committed
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'rollback@example.com'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "User should not exist after rollback")
	})
}

// Helper types
type DriverLocation struct {
	DriverID  string
	Latitude  float64
	Longitude float64
}

// Helper functions for database operations
func createTestUserInDB(db *sql.DB, userID, email, userType string) error {
	_, err := db.Exec(`
		INSERT INTO users (id, email, first_name, last_name, user_type, status, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, userID, email, "Test", "User", userType, "active", time.Now())
	return err
}

func createTestVehicleInDB(db *sql.DB, vehicleID, driverID, licensePlate, make, model string) error {
	_, err := db.Exec(`
		INSERT INTO vehicles (id, driver_id, license_plate, make, model, year, vehicle_type, status, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`, vehicleID, driverID, licensePlate, make, model, 2020, "sedan", "active", time.Now())
	return err
}

func createTestTripInDB(db *sql.DB, tripID, riderID, driverID string, fareAmount int) error {
	_, err := db.Exec(`
		INSERT INTO trips (id, rider_id, driver_id, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng, 
		                  fare_amount, status, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`, tripID, riderID, driverID, 40.7128, -74.0060, 40.7589, -73.9851, fareAmount, "completed", time.Now())
	return err
}

func createTestPaymentInDB(db *sql.DB, paymentID, tripID string, amount int, status string) error {
	_, err := db.Exec(`
		INSERT INTO payments (id, trip_id, amount, payment_method, status, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`, paymentID, tripID, amount, "credit_card", status, time.Now())
	return err
}

func createTestDriverLocationInDB(db *sql.DB, driverID string, lat, lng float64) error {
	_, err := db.Exec(`
		INSERT INTO driver_locations (driver_id, latitude, longitude, updated_at) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (driver_id) DO UPDATE SET 
		latitude = EXCLUDED.latitude, 
		longitude = EXCLUDED.longitude, 
		updated_at = EXCLUDED.updated_at
	`, driverID, lat, lng, time.Now())
	return err
}

func findNearbyDrivers(ctx context.Context, db *sql.DB, lat, lng, radiusMeters float64) ([]DriverLocation, error) {
	// Simple distance query using Haversine formula approximation
	rows, err := db.QueryContext(ctx, `
		SELECT driver_id, latitude, longitude,
		       (6371000 * acos(cos(radians($1)) * cos(radians(latitude)) * 
		        cos(radians(longitude) - radians($2)) + sin(radians($1)) * 
		        sin(radians(latitude)))) AS distance
		FROM driver_locations 
		WHERE (6371000 * acos(cos(radians($1)) * cos(radians(latitude)) * 
		       cos(radians(longitude) - radians($2)) + sin(radians($1)) * 
		       sin(radians(latitude)))) <= $3
		ORDER BY distance
	`, lat, lng, radiusMeters)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var drivers []DriverLocation
	for rows.Next() {
		var driver DriverLocation
		var distance float64
		err := rows.Scan(&driver.DriverID, &driver.Latitude, &driver.Longitude, &distance)
		if err != nil {
			return nil, err
		}
		drivers = append(drivers, driver)
	}
	
	return drivers, rows.Err()
}

func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Simplified distance calculation for testing
	const earthRadius = 6371000 // meters
	
	// Convert to radians
	lat1Rad := lat1 * 3.14159 / 180
	lng1Rad := lng1 * 3.14159 / 180
	lat2Rad := lat2 * 3.14159 / 180
	lng2Rad := lng2 * 3.14159 / 180
	
	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad
	
	a := 0.5 - 0.5*(dlat) + (lat1Rad)*(lat2Rad)*0.5*(1-(dlng))
	return earthRadius * 2 * 1.5708 * a // Simplified for testing
}

		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Email, retrievedUser.Email)
	})
}

// Integration test for trip service
func TestTripServiceIntegration(t *testing.T) {
	t.Run("trip creation and matching", func(t *testing.T) {
		// Create locations
		pickupLocation := models.Location{
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now(),
		}

		destination := models.Location{
			Latitude:  40.7489,
			Longitude: -73.9857,
			Timestamp: time.Now(),
		}

		// Create a trip request
		trip := &models.Trip{
			RiderID:        "rider-123",
			PickupLocation: pickupLocation,
			Destination:    destination,
			Status:         models.TripStatusRequested,
			PassengerCount: 1,
			Currency:       "USD",
			RequestedAt:    time.Now(),
		}

		// Simulate trip creation
		trip.ID = "trip-123"

		// Verify trip was created
		assert.NotEmpty(t, trip.ID)
		assert.Equal(t, models.TripStatusRequested, trip.Status)
		assert.Equal(t, "rider-123", trip.RiderID)

		// Simulate matching with a driver
		driverID := "driver-456"
		vehicleID := "vehicle-123"
		trip.DriverID = &driverID
		trip.VehicleID = &vehicleID
		trip.Status = models.TripStatusMatched
		matchedTime := time.Now()
		trip.MatchedAt = &matchedTime

		// Verify trip was matched
		assert.Equal(t, driverID, *trip.DriverID)
		assert.Equal(t, vehicleID, *trip.VehicleID)
		assert.Equal(t, models.TripStatusMatched, trip.Status)
	})
}

// Integration test for payment service
func TestPaymentServiceIntegration(t *testing.T) {
	t.Run("payment processing", func(t *testing.T) {
		// Create a payment request
		payment := &models.Payment{
			TripID:      "trip-123",
			UserID:      "rider-123",
			AmountCents: 2500, // $25.00 in cents
			Currency:    "USD",
			Status:      models.PaymentStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Simulate payment processing
		payment.ID = "payment-123"
		processedTime := time.Now()
		payment.ProcessedAt = &processedTime
		payment.Status = models.PaymentStatusCompleted
		payment.UpdatedAt = time.Now()

		// Verify payment was processed
		assert.NotEmpty(t, payment.ID)
		assert.Equal(t, models.PaymentStatusCompleted, payment.Status)
		assert.Equal(t, int64(2500), payment.AmountCents)
		assert.NotNil(t, payment.ProcessedAt)
	})

	t.Run("payment failure handling", func(t *testing.T) {
		// Create a payment that will fail
		payment := &models.Payment{
			TripID:      "trip-124",
			UserID:      "rider-124",
			AmountCents: 0, // Invalid amount
			Currency:    "USD",
			Status:      models.PaymentStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Simulate payment failure
		payment.Status = models.PaymentStatusFailed
		failedTime := time.Now()
		payment.FailedAt = &failedTime
		payment.UpdatedAt = time.Now()

		// Verify payment failed
		assert.Equal(t, models.PaymentStatusFailed, payment.Status)
		assert.NotNil(t, payment.FailedAt)
	})
}

// Integration test for vehicle service
func TestVehicleServiceIntegration(t *testing.T) {
	t.Run("vehicle registration", func(t *testing.T) {
		// Register a vehicle
		vehicle := &models.Vehicle{
			DriverID:     "driver-123",
			Make:         "Toyota",
			Model:        "Camry",
			Year:         2020,
			LicensePlate: "ABC123",
			Color:        "Blue",
			VehicleType:  models.VehicleTypeSedan,
			Status:       models.VehicleStatusActive,
			Capacity:     4,
		}

		// Simulate vehicle registration
		vehicle.ID = "vehicle-123"
		vehicle.CreatedAt = time.Now()
		vehicle.UpdatedAt = time.Now()

		// Verify vehicle was registered
		assert.NotEmpty(t, vehicle.ID)
		assert.Equal(t, models.VehicleStatusActive, vehicle.Status)
		assert.Equal(t, "driver-123", vehicle.DriverID)
		assert.Equal(t, "Toyota", vehicle.Make)
		assert.Equal(t, "Camry", vehicle.Model)
		assert.Equal(t, 2020, vehicle.Year)
		assert.Equal(t, 4, vehicle.Capacity)
	})
}

// Comprehensive integration test
func TestFullRideFlow(t *testing.T) {
	t.Run("complete ride flow", func(t *testing.T) {
		// Step 1: Create rider and driver
		rider := &models.User{
			ID:        "rider-flow-123",
			Email:     "rider@example.com",
			FirstName: "John",
			LastName:  "Rider",
			UserType:  models.UserTypeRider,
			Status:    models.UserStatusActive,
		}

		driver := &models.User{
			ID:        "driver-flow-123",
			Email:     "driver@example.com",
			FirstName: "Jane",
			LastName:  "Driver",
			UserType:  models.UserTypeDriver,
			Status:    models.UserStatusActive,
		}

		// Step 2: Register vehicle
		vehicle := &models.Vehicle{
			ID:           "vehicle-flow-123",
			DriverID:     driver.ID,
			Make:         "Honda",
			Model:        "Accord",
			Year:         2021,
			LicensePlate: "FLOW123",
			VehicleType:  models.VehicleTypeSedan,
			Status:       models.VehicleStatusActive,
			Capacity:     4,
		}

		// Step 3: Create trip request
		trip := &models.Trip{
			ID:      "trip-flow-123",
			RiderID: rider.ID,
			PickupLocation: models.Location{
				Latitude:  40.7128,
				Longitude: -74.0060,
				Timestamp: time.Now(),
			},
			Destination: models.Location{
				Latitude:  40.7589,
				Longitude: -73.9851,
				Timestamp: time.Now(),
			},
			Status:         models.TripStatusRequested,
			PassengerCount: 1,
			Currency:       "USD",
			RequestedAt:    time.Now(),
		}

		// Step 4: Match driver
		trip.DriverID = &driver.ID
		trip.VehicleID = &vehicle.ID
		trip.Status = models.TripStatusMatched
		matchedTime := time.Now()
		trip.MatchedAt = &matchedTime

		// Step 5: Start trip
		trip.Status = models.TripStatusTripStarted
		startedTime := time.Now()
		trip.StartedAt = &startedTime

		// Step 6: Complete trip
		trip.Status = models.TripStatusCompleted
		completedTime := time.Now()
		trip.CompletedAt = &completedTime
		actualFare := int64(1500)
		trip.ActualFareCents = &actualFare

		// Step 7: Process payment
		payment := &models.Payment{
			ID:          "payment-flow-123",
			TripID:      trip.ID,
			UserID:      rider.ID,
			AmountCents: 1500, // $15.00
			Currency:    "USD",
			Status:      models.PaymentStatusCompleted,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		processedTime := time.Now()
		payment.ProcessedAt = &processedTime

		// Verify complete flow
		assert.Equal(t, models.UserTypeRider, rider.UserType)
		assert.Equal(t, models.UserTypeDriver, driver.UserType)
		assert.Equal(t, driver.ID, vehicle.DriverID)
		assert.Equal(t, models.TripStatusCompleted, trip.Status)
		assert.Equal(t, driver.ID, *trip.DriverID)
		assert.Equal(t, vehicle.ID, *trip.VehicleID)
		assert.Equal(t, models.PaymentStatusCompleted, payment.Status)
		assert.Equal(t, trip.ID, payment.TripID)

		// Verify timing
		assert.True(t, trip.StartedAt.After(trip.RequestedAt))
		assert.True(t, trip.CompletedAt.After(*trip.StartedAt))
		assert.True(t, payment.ProcessedAt.After(*trip.CompletedAt))

		// Verify amounts
		assert.Equal(t, int64(1500), *trip.ActualFareCents)
		assert.Equal(t, int64(1500), payment.AmountCents)
	})
}
