#!/usr/bin/env bash

# =============================================================================
# 🚀 TEST INFRASTRUCTURE SETUP SCRIPT
# =============================================================================
# Sets up all required infrastructure for comprehensive testing
# Author: Senior Software Engineer

set -euo pipefail

# Color definitions
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

# Icons
readonly CHECK="✅"
readonly CROSS="❌"
readonly WARNING="⚠️"
readonly INFO="ℹ️"
readonly ROCKET="🚀"
readonly GEAR="⚙️"

echo -e "${CYAN}${BOLD}🚀 SETTING UP TEST INFRASTRUCTURE${NC}"
echo "=================================="

# Function to check if a service is healthy
wait_for_service() {
    local service_name="$1"
    local max_attempts=30
    local attempt=0
    
    echo -e "${BLUE}${INFO} Waiting for $service_name to be ready...${NC}"
    
    while [ $attempt -lt $max_attempts ]; do
        if docker compose -f docker-compose-test.yml ps "$service_name" | grep -q "healthy"; then
            echo -e "${GREEN}${CHECK} $service_name is ready${NC}"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -e "${YELLOW}${WARNING} Attempt $attempt/$max_attempts - $service_name not ready yet...${NC}"
        sleep 2
    done
    
    echo -e "${RED}${CROSS} $service_name failed to become ready${NC}"
    return 1
}

# Start test infrastructure
echo -e "${GEAR} Starting test databases..."
docker compose -f docker-compose-test.yml up -d

# Wait for services to be healthy
wait_for_service "postgres-test"
wait_for_service "mongodb-test"
wait_for_service "redis-test"

# Export test environment variables
echo -e "${GEAR} Setting up test environment variables..."
export TEST_POSTGRES_HOST="localhost"
export TEST_POSTGRES_PORT="5433"
export TEST_POSTGRES_USER="postgres"
export TEST_POSTGRES_PASSWORD="${TEST_POSTGRES_PASSWORD:-$(openssl rand -base64 12)}"
export TEST_POSTGRES_DB="rideshare_test"

export TEST_MONGODB_HOST="localhost"
export TEST_MONGODB_PORT="27018"
export TEST_MONGODB_USER="admin"
export TEST_MONGODB_PASSWORD="${TEST_MONGODB_PASSWORD:-$(openssl rand -base64 12)}"
export TEST_MONGODB_DB="rideshare_test"

export TEST_REDIS_HOST="localhost"
export TEST_REDIS_PORT="6380"

# Generate JWT secret for testing
export TEST_JWT_SECRET="${TEST_JWT_SECRET:-$(openssl rand -base64 24)}"

# Create test environment file
cat > .env.test << EOF
# Test Database Configuration - Auto-generated passwords
TEST_POSTGRES_HOST=localhost
TEST_POSTGRES_PORT=5433
TEST_POSTGRES_USER=postgres
TEST_POSTGRES_PASSWORD=$TEST_POSTGRES_PASSWORD
TEST_POSTGRES_DB=rideshare_test

TEST_MONGODB_HOST=localhost
TEST_MONGODB_PORT=27018
TEST_MONGODB_USER=admin
TEST_MONGODB_PASSWORD=$TEST_MONGODB_PASSWORD
TEST_MONGODB_DB=rideshare_test

TEST_REDIS_HOST=localhost
TEST_REDIS_PORT=6380

TEST_JWT_SECRET=$TEST_JWT_SECRET

# Service URLs for Integration Tests
API_GATEWAY_URL=http://localhost:8080
USER_SERVICE_URL=http://localhost:8081
VEHICLE_SERVICE_URL=http://localhost:8082
GEO_SERVICE_URL=http://localhost:8083
TRIP_SERVICE_URL=http://localhost:8084
MATCHING_SERVICE_URL=http://localhost:8085
PAYMENT_SERVICE_URL=http://localhost:8086
PRICING_SERVICE_URL=http://localhost:8087
EOF

echo -e "${GREEN}${CHECK} Test infrastructure is ready!${NC}"
echo -e "${INFO} Test environment variables saved to .env.test${NC}"
echo -e "${INFO} Use: source .env.test before running tests${NC}"

# Install additional test dependencies
echo -e "${GEAR} Installing test dependencies..."
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest || echo "gosec installation failed"
go install github.com/sonatard/nancy@latest || echo "nancy installation failed"

# Test database connections
echo -e "${GEAR} Testing database connections..."

# Test PostgreSQL
if PGPASSWORD="$TEST_POSTGRES_PASSWORD" psql -h "$TEST_POSTGRES_HOST" -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -d "$TEST_POSTGRES_DB" -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}${CHECK} PostgreSQL connection successful${NC}"
else
    echo -e "${RED}${CROSS} PostgreSQL connection failed${NC}"
fi

# Test MongoDB
if mongosh "mongodb://admin:$TEST_MONGODB_PASSWORD@localhost:27018/rideshare_test?authSource=admin" --eval "db.runCommand({ping: 1})" > /dev/null 2>&1; then
    echo -e "${GREEN}${CHECK} MongoDB connection successful${NC}"
else
    echo -e "${RED}${CROSS} MongoDB connection failed${NC}"
fi

# Test Redis
if redis-cli -h "$TEST_REDIS_HOST" -p "$TEST_REDIS_PORT" ping > /dev/null 2>&1; then
    echo -e "${GREEN}${CHECK} Redis connection successful${NC}"
else
    echo -e "${RED}${CROSS} Redis connection failed${NC}"
fi

echo ""
echo -e "${CYAN}${BOLD}🎉 TEST INFRASTRUCTURE SETUP COMPLETE!${NC}"
echo -e "${INFO} You can now run: ${BOLD}make test-all${NC} or ${BOLD}./scripts/test-orchestrator.sh all${NC}"
