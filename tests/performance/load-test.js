import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const userCreationTrend = new Trend('user_creation_duration');
const rideRequestTrend = new Trend('ride_request_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 20 }, // Ramp up
    { duration: '5m', target: 50 }, // Stay at 50 users
    { duration: '2m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 0 }, // Ramp down
  ],
  thresholds: {
    errors: ['rate<0.1'], // Error rate should be less than 10%
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    user_creation_duration: ['p(95)<1000'], // User creation should be under 1s
    ride_request_duration: ['p(95)<2000'], // Ride requests should be under 2s
  },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data generators
function generateUser() {
  const userId = `user_${Math.random().toString(36).substr(2, 9)}`;
  return {
    email: `${userId}@example.com`,
    phone: `+1555${Math.floor(Math.random() * 10000000).toString().padStart(7, '0')}`,
    first_name: `FirstName${userId.substr(-4)}`,
    last_name: `LastName${userId.substr(-4)}`,
    user_type: Math.random() < 0.7 ? 'rider' : 'driver',
    password: 'testpassword123'
  };
}

function generateLocation() {
  // San Francisco area coordinates
  const lat = 37.7749 + (Math.random() - 0.5) * 0.1;
  const lng = -122.4194 + (Math.random() - 0.5) * 0.1;
  return { latitude: lat, longitude: lng };
}

function generateVehicle() {
  const makes = ['Toyota', 'Honda', 'Ford', 'BMW', 'Tesla'];
  const models = ['Camry', 'Civic', 'F-150', 'X3', 'Model 3'];
  
  return {
    make: makes[Math.floor(Math.random() * makes.length)],
    model: models[Math.floor(Math.random() * models.length)],
    year: 2015 + Math.floor(Math.random() * 9),
    license_plate: `ABC${Math.floor(Math.random() * 1000)}`,
    color: 'Black',
    vehicle_type: 'sedan'
  };
}

// Test scenarios
export default function() {
  const scenario = Math.random();
  
  if (scenario < 0.4) {
    testUserJourney();
  } else if (scenario < 0.7) {
    testDriverJourney();
  } else {
    testRideMatching();
  }
  
  sleep(1);
}

function testUserJourney() {
  const user = generateUser();
  
  // 1. Create user
  const createUserStart = Date.now();
  const createUserResponse = http.post(`${BASE_URL}/api/v1/users`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const createUserDuration = Date.now() - createUserStart;
  userCreationTrend.add(createUserDuration);
  
  const createUserSuccess = check(createUserResponse, {
    'user creation status is 201': (r) => r.status === 201,
    'user creation response has id': (r) => JSON.parse(r.body).id !== undefined,
  });
  
  if (!createUserSuccess) {
    errorRate.add(1);
    return;
  }
  
  const userId = JSON.parse(createUserResponse.body).id;
  
  // 2. Authenticate user
  const authResponse = http.post(`${BASE_URL}/api/v1/users/auth`, JSON.stringify({
    email: user.email,
    password: user.password
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const authSuccess = check(authResponse, {
    'authentication status is 200': (r) => r.status === 200,
    'authentication response has token': (r) => JSON.parse(r.body).token !== undefined,
  });
  
  if (!authSuccess) {
    errorRate.add(1);
    return;
  }
  
  const token = JSON.parse(authResponse.body).token;
  const headers = { 
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  };
  
  // 3. Get user profile
  const profileResponse = http.get(`${BASE_URL}/api/v1/users/${userId}`, { headers });
  
  check(profileResponse, {
    'profile retrieval status is 200': (r) => r.status === 200,
    'profile contains user data': (r) => JSON.parse(r.body).email === user.email,
  });
  
  // 4. Request a ride (if rider)
  if (user.user_type === 'rider') {
    const rideRequestStart = Date.now();
    const pickup = generateLocation();
    const destination = generateLocation();
    
    const rideResponse = http.post(`${BASE_URL}/api/v1/trips`, JSON.stringify({
      pickup_location: pickup,
      destination_location: destination,
      vehicle_type: 'sedan'
    }), { headers });
    
    const rideRequestDuration = Date.now() - rideRequestStart;
    rideRequestTrend.add(rideRequestDuration);
    
    const rideSuccess = check(rideResponse, {
      'ride request status is 201 or 202': (r) => r.status === 201 || r.status === 202,
    });
    
    if (!rideSuccess) {
      errorRate.add(1);
    }
  }
}

function testDriverJourney() {
  const driver = generateUser();
  driver.user_type = 'driver';
  
  // Create driver
  const createResponse = http.post(`${BASE_URL}/api/v1/users`, JSON.stringify(driver), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (createResponse.status !== 201) {
    errorRate.add(1);
    return;
  }
  
  const driverId = JSON.parse(createResponse.body).id;
  
  // Authenticate
  const authResponse = http.post(`${BASE_URL}/api/v1/users/auth`, JSON.stringify({
    email: driver.email,
    password: driver.password
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (authResponse.status !== 200) {
    errorRate.add(1);
    return;
  }
  
  const token = JSON.parse(authResponse.body).token;
  const headers = { 
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  };
  
  // Register vehicle
  const vehicle = generateVehicle();
  vehicle.driver_id = driverId;
  
  const vehicleResponse = http.post(`${BASE_URL}/api/v1/vehicles`, JSON.stringify(vehicle), { headers });
  
  const vehicleSuccess = check(vehicleResponse, {
    'vehicle registration status is 201': (r) => r.status === 201,
  });
  
  if (!vehicleSuccess) {
    errorRate.add(1);
    return;
  }
  
  const vehicleId = JSON.parse(vehicleResponse.body).id;
  
  // Update driver location
  const location = generateLocation();
  const locationResponse = http.put(`${BASE_URL}/api/v1/vehicles/${vehicleId}/location`, 
    JSON.stringify(location), { headers });
  
  check(locationResponse, {
    'location update status is 200': (r) => r.status === 200,
  });
  
  // Go online
  const onlineResponse = http.put(`${BASE_URL}/api/v1/vehicles/${vehicleId}/status`, 
    JSON.stringify({ status: 'available' }), { headers });
  
  check(onlineResponse, {
    'go online status is 200': (r) => r.status === 200,
  });
}

function testRideMatching() {
  // Test the matching service directly
  const location = generateLocation();
  
  const matchResponse = http.get(`${BASE_URL}/api/v1/matching/nearby-drivers?` + 
    `lat=${location.latitude}&lng=${location.longitude}&radius=5`);
  
  const matchSuccess = check(matchResponse, {
    'matching service status is 200': (r) => r.status === 200,
    'matching service returns array': (r) => Array.isArray(JSON.parse(r.body)),
  });
  
  if (!matchSuccess) {
    errorRate.add(1);
  }
  
  // Test pricing calculation
  const pickup = generateLocation();
  const destination = generateLocation();
  
  const pricingResponse = http.post(`${BASE_URL}/api/v1/pricing/calculate`, JSON.stringify({
    pickup_location: pickup,
    destination_location: destination,
    vehicle_type: 'sedan'
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(pricingResponse, {
    'pricing service status is 200': (r) => r.status === 200,
    'pricing response has estimate': (r) => JSON.parse(r.body).estimated_fare !== undefined,
  });
}

// Spike test for handling traffic bursts
export function handleSummary(data) {
  return {
    'performance-results.json': JSON.stringify(data, null, 2),
    stdout: `
Performance Test Summary:
========================
Total Requests: ${data.metrics.http_reqs.count}
Failed Requests: ${data.metrics.http_req_failed.count}
Error Rate: ${(data.metrics.http_req_failed.count / data.metrics.http_reqs.count * 100).toFixed(2)}%
Average Response Time: ${data.metrics.http_req_duration.avg.toFixed(2)}ms
95th Percentile: ${data.metrics.http_req_duration['p(95)'].toFixed(2)}ms
    `,
  };
}
