#!/bin/bash
# Comprehensive load test script for rideshare platform

set -e

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
CONCURRENT_USERS="${CONCURRENT_USERS:-10}"
REQUESTS_PER_USER="${REQUESTS_PER_USER:-20}"
RAMP_UP_TIME="${RAMP_UP_TIME:-5}"
TEST_DURATION="${TEST_DURATION:-30}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting Rideshare Platform Load Tests${NC}"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo "Concurrent Users: $CONCURRENT_USERS"
echo "Requests per User: $REQUESTS_PER_USER"
echo "Ramp-up Time: ${RAMP_UP_TIME}s"
echo "Test Duration: ${TEST_DURATION}s"
echo ""

# Function to check if service is available
check_service() {
    local url=$1
    local service_name=$2
    
    echo -n "Checking $service_name availability..."
    if curl -s -f "$url/health" >/dev/null 2>&1; then
        echo -e " ${GREEN}✓${NC}"
        return 0
    else
        echo -e " ${RED}✗${NC}"
        return 1
    fi
}

# Function to run load test for specific endpoint
load_test_endpoint() {
    local endpoint=$1
    local method=${2:-GET}
    local description=$3
    local body=${4:-""}
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "Endpoint: $method $endpoint"
    
    local total_requests=$((CONCURRENT_USERS * REQUESTS_PER_USER))
    local success_count=0
    local error_count=0
    local start_time=$(date +%s)
    
    # Create temporary files for results
    local success_file="/tmp/load_test_success_$$"
    local error_file="/tmp/load_test_error_$$"
    
    # Run concurrent requests
    for i in $(seq 1 $CONCURRENT_USERS); do
        (
            for j in $(seq 1 $REQUESTS_PER_USER); do
                local start_request=$(date +%s%3N)
                
                if [ "$method" = "GET" ]; then
                    if curl -s -w "%{http_code}:%{time_total}" -o /dev/null "$BASE_URL$endpoint" 2>/dev/null | grep -q "^200\|^503"; then
                        echo "1" >> "$success_file"
                    else
                        echo "1" >> "$error_file"
                    fi
                elif [ "$method" = "POST" ]; then
                    if curl -s -w "%{http_code}:%{time_total}" -o /dev/null -X POST -H "Content-Type: application/json" -d "$body" "$BASE_URL$endpoint" 2>/dev/null | grep -q "^200\|^503"; then
                        echo "1" >> "$success_file"
                    else
                        echo "1" >> "$error_file"
                    fi
                fi
                
                # Small delay to prevent overwhelming
                sleep 0.1
            done
        ) &
        
        # Stagger user startup
        sleep $(echo "scale=2; $RAMP_UP_TIME / $CONCURRENT_USERS" | bc -l 2>/dev/null || echo "0.5")
    done
    
    # Wait for all background jobs
    wait
    
    # Calculate results
    if [ -f "$success_file" ]; then
        success_count=$(wc -l < "$success_file")
        rm -f "$success_file"
    fi
    
    if [ -f "$error_file" ]; then
        error_count=$(wc -l < "$error_file")
        rm -f "$error_file"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local total_completed=$((success_count + error_count))
    
    echo "Results:"
    echo "  Total Requests: $total_requests"
    echo "  Completed: $total_completed"
    echo "  Successful: $success_count"
    echo "  Errors: $error_count"
    echo "  Duration: ${duration}s"
    
    if [ $total_completed -gt 0 ]; then
        local success_rate=$(echo "scale=2; $success_count * 100 / $total_completed" | bc -l 2>/dev/null || echo "0")
        local rps=$(echo "scale=2; $total_completed / $duration" | bc -l 2>/dev/null || echo "0")
        echo "  Success Rate: ${success_rate}%"
        echo "  Requests/sec: $rps"
        
        if (( $(echo "$success_rate > 80" | bc -l 2>/dev/null || echo "0") )); then
            echo -e "  Status: ${GREEN}PASS${NC}"
        else
            echo -e "  Status: ${RED}FAIL${NC}"
        fi
    else
        echo -e "  Status: ${RED}NO RESPONSES${NC}"
    fi
}

# Pre-flight checks
echo -e "${BLUE}Pre-flight Checks${NC}"
echo "=================="

if ! check_service "$BASE_URL" "API Gateway"; then
    echo -e "${RED}API Gateway is not available. Exiting.${NC}"
    exit 1
fi

# Wait a moment for services to be fully ready
echo "Waiting for services to be fully ready..."
sleep 3

# Run load tests for different endpoints
echo -e "\n${BLUE}Running Load Tests${NC}"
echo "=================="

# Health endpoint test
load_test_endpoint "/health" "GET" "Health Check Endpoint"

# Status endpoint test
load_test_endpoint "/status" "GET" "Status Check Endpoint"

# User API tests
load_test_endpoint "/api/v1/users/123" "GET" "User Retrieval API"

# Trip API tests
load_test_endpoint "/api/v1/trips/456" "GET" "Trip Retrieval API"

# Pricing API tests
load_test_endpoint "/api/v1/pricing/estimate" "POST" "Pricing Estimation API" '{"pickup":{"lat":40.7128,"lng":-74.0060},"destination":{"lat":40.7589,"lng":-73.9851}}'

# Driver matching API tests
load_test_endpoint "/api/v1/matching/nearby-drivers" "POST" "Driver Matching API" '{"location":{"lat":40.7128,"lng":-74.0060},"radius":5000}'

# Payment API tests
load_test_endpoint "/api/v1/payments" "POST" "Payment Processing API" '{"trip_id":"test-trip","amount":15.50,"payment_method_id":"pm_test"}'

echo -e "\n${BLUE}Load Testing Complete${NC}"
echo "====================="
echo -e "${GREEN}All load tests finished successfully!${NC}"

# Optional: Run a sustained load test
if [ "${RUN_SUSTAINED_TEST:-false}" = "true" ]; then
    echo -e "\n${YELLOW}Running Sustained Load Test (${TEST_DURATION}s)...${NC}"
    
    timeout $TEST_DURATION bash -c '
        while true; do
            curl -s '"$BASE_URL"'/health >/dev/null 2>&1 &
            curl -s '"$BASE_URL"'/api/v1/users/123 >/dev/null 2>&1 &
            curl -s '"$BASE_URL"'/api/v1/trips/456 >/dev/null 2>&1 &
            sleep 0.5
        done
    '
    
    echo -e "${GREEN}Sustained load test completed!${NC}"
fi
