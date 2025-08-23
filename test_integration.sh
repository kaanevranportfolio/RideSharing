#!/bin/bash

# Integration test script for rideshare platform services

echo "🚀 Starting Rideshare Platform Integration Tests"
echo "================================================"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if service is responding
check_service() {
    local service_name=$1
    local port=$2
    local endpoint=$3
    
    echo -n "Testing $service_name on port $port... "
    
    # Start service in background
    cd services/$service_name && ./bin/$service_name &
    local pid=$!
    
    # Wait for service to start
    sleep 3
    
    # Test endpoint
    if curl -s -f "http://localhost:$port$endpoint" > /dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        kill $pid 2>/dev/null
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        kill $pid 2>/dev/null
        return 1
    fi
}

# Test each service
echo -e "\n${YELLOW}Testing individual services:${NC}"

# Build all services first
echo "Building services..."
make build

echo -e "\n${YELLOW}Service Health Checks:${NC}"

# Test geo service
cd /home/kaan/Projects/rideshare-platform
check_service "geo-service" "8083" "/health"

# Test vehicle service  
check_service "vehicle-service" "8082" "/health"

# Test matching service
check_service "matching-service" "8084" "/health"

# Test trip service
check_service "trip-service" "8085" "/health"

echo -e "\n${YELLOW}Testing API endpoints:${NC}"

# Start geo service for endpoint testing
cd services/geo-service && ./bin/geo-service &
GEO_PID=$!
sleep 3

echo -n "Testing geo service distance calculation... "
if curl -s -X POST "http://localhost:8083/api/v1/geo/distance" \
   -H "Content-Type: application/json" \
   -d '{"origin":{"lat":40.7128,"lng":-74.0060},"destination":{"lat":40.7589,"lng":-73.9851}}' | grep -q "distance"; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
fi

kill $GEO_PID 2>/dev/null

# Start matching service for endpoint testing
cd ../matching-service && ./bin/matching-service &
MATCHING_PID=$!
sleep 3

echo -n "Testing matching service find drivers... "
if curl -s -X POST "http://localhost:8084/api/v1/matching/find-drivers" \
   -H "Content-Type: application/json" \
   -d '{"rider_location":{"lat":40.7128,"lng":-74.0060},"destination":{"lat":40.7589,"lng":-73.9851},"ride_type":"standard"}' | grep -q "drivers"; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
fi

kill $MATCHING_PID 2>/dev/null

echo -e "\n${GREEN}🎉 Integration tests completed!${NC}"
echo "Note: User service has persistent file corruption issues and is temporarily disabled."
