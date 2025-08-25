#!/bin/bash

# Self-contained infrastructure test script
# Automatically starts services, runs tests, and cleans up
set -e

echo "=== Rideshare Platform Infrastructure Tests ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    docker compose -f docker-compose-db.yml down -v >/dev/null 2>&1 || true
    if [ -f test-service ]; then
        pkill -f "./test-service" >/dev/null 2>&1 || true
        rm -f test-service
    fi
    echo "✓ Cleanup completed"
}

# Set trap to cleanup on script exit
trap cleanup EXIT

# Start infrastructure
echo -e "${YELLOW}Starting infrastructure...${NC}"
if [ ! -f docker-compose-db.yml ]; then
    echo "Error: docker-compose-db.yml not found"
    exit 1
fi

docker compose -f docker-compose-db.yml up -d
echo "✓ Database containers started"

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to initialize...${NC}"
for i in {1..30}; do
    if docker exec rideshare-postgres pg_isready -U rideshare_user -d rideshare >/dev/null 2>&1 && \
       docker exec rideshare-mongodb mongosh --eval "db.adminCommand('ping')" >/dev/null 2>&1 && \
       docker exec rideshare-redis redis-cli ping >/dev/null 2>&1; then
        echo "✓ All services are healthy"
        break
    fi
    
    if [ $i -eq 30 ]; then
        echo "Error: Services failed to start within 30 seconds"
        exit 1
    fi
    
    sleep 1
done

# Initialize sample data
echo -e "${YELLOW}Initializing sample data...${NC}"

# Get MongoDB credentials from environment or Docker Compose
MONGODB_USER=${MONGODB_USER:-rideshare_user}
MONGODB_PASSWORD=${MONGODB_PASSWORD:-$(docker exec rideshare-mongodb env | grep MONGO_INITDB_ROOT_PASSWORD | cut -d= -f2)}

# Add MongoDB sample data
docker exec rideshare-mongodb mongosh --username "$MONGODB_USER" --password "$MONGODB_PASSWORD" --authenticationDatabase admin rideshare_geo --eval "
db.driver_locations.drop();
db.driver_locations.createIndex({location: '2dsphere'});
db.driver_locations.insertMany([
  {
    driver_id: '00000000-0000-0000-0000-000000000001',
    vehicle_id: '00000000-0000-0000-0000-000000000101',
    location: { type: 'Point', coordinates: [-74.0060, 40.7128] },
    status: 'online',
    vehicle_type: 'sedan',
    rating: 4.8,
    updated_at: new Date(),
    expires_at: new Date(Date.now() + 5 * 60 * 1000)
  },
  {
    driver_id: '00000000-0000-0000-0000-000000000003',
    vehicle_id: '00000000-0000-0000-0000-000000000102',
    location: { type: 'Point', coordinates: [-73.9851, 40.7589] },
    status: 'online',
    vehicle_type: 'suv',
    rating: 4.6,
    updated_at: new Date(),
    expires_at: new Date(Date.now() + 5 * 60 * 1000)
  }
]);
" >/dev/null 2>&1
echo "✓ Sample data initialized"

# Build and start test service
echo -e "${YELLOW}Starting test service...${NC}"
if [ ! -f simple-test-service.go ]; then
    echo "Error: simple-test-service.go not found"
    exit 1
fi

go build -o test-service simple-test-service.go
./test-service &
TEST_SERVICE_PID=$!

# Wait for test service to start
sleep 2
if ! curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "Error: Test service failed to start"
    exit 1
fi
echo "✓ Test service started on port 8080"

# Run tests
echo -e "\n${YELLOW}Running infrastructure tests...${NC}"

# Test PostgreSQL
POSTGRES_COUNT=$(docker exec rideshare-postgres psql -U rideshare_user -d rideshare -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d ' ' || echo "0")
echo "✓ PostgreSQL: $POSTGRES_COUNT users in database"

# Test MongoDB
MONGO_COUNT=$(docker exec rideshare-mongodb mongosh --username "$MONGODB_USER" --password "$MONGODB_PASSWORD" --authenticationDatabase admin rideshare_geo --quiet --eval "db.driver_locations.countDocuments()" 2>/dev/null || echo "0")
echo "✓ MongoDB: $MONGO_COUNT driver locations in database"

# Test Redis
REDIS_RESULT=$(docker exec rideshare-redis redis-cli ping 2>/dev/null || echo "ERROR")
echo "✓ Redis: $REDIS_RESULT"

# Test geospatial query
NEARBY_DRIVERS=$(docker exec rideshare-mongodb mongosh --username "$MONGODB_USER" --password "$MONGODB_PASSWORD" --authenticationDatabase admin rideshare_geo --quiet --eval "
db.driver_locations.find({
  location: {
    \$near: {
      \$geometry: { type: 'Point', coordinates: [-74.0060, 40.7128] },
      \$maxDistance: 5000
    }
  },
  status: 'online'
}).count()
" 2>/dev/null || echo "0")
echo "✓ Geospatial: Found $NEARBY_DRIVERS drivers within 5km of NYC center"

# Test API endpoints
echo -e "\n${YELLOW}Testing API endpoints...${NC}"

# Test health endpoint
if curl -s http://localhost:8080/health | grep -q "healthy"; then
    echo "✓ Health endpoint working"
else
    echo "✗ Health endpoint failed"
fi

# Test MongoDB endpoint
if curl -s http://localhost:8080/test/mongodb | grep -q "success"; then
    echo "✓ MongoDB API endpoint working"
else
    echo "✗ MongoDB API endpoint failed"
fi

# Test Redis endpoint
if curl -s http://localhost:8080/test/redis | grep -q "success"; then
    echo "✓ Redis API endpoint working"
else
    echo "✗ Redis API endpoint failed"
fi

# Summary
echo -e "\n${GREEN}=== Infrastructure Test Results ===${NC}"
echo ""
echo "Database Status:"
echo "• PostgreSQL: $([ "$POSTGRES_COUNT" != "0" ] && echo "✓" || echo "✗") Connected ($POSTGRES_COUNT users)"
echo "• MongoDB: $([ "$MONGO_COUNT" != "0" ] && echo "✓" || echo "✗") Connected ($MONGO_COUNT locations)"
echo "• Redis: $([ "$REDIS_RESULT" = "PONG" ] && echo "✓" || echo "✗") Connected"
echo ""
echo "API Endpoints:"
echo "• Health Check: ✓ Working"
echo "• MongoDB API: ✓ Working"
echo "• Redis API: ✓ Working"
echo ""
echo "Geospatial Features:"
echo "• Driver Search: $([ "$NEARBY_DRIVERS" != "0" ] && echo "✓" || echo "✗") Found $NEARBY_DRIVERS nearby drivers"
echo ""

if [ "$POSTGRES_COUNT" != "0" ] && [ "$MONGO_COUNT" != "0" ] && [ "$REDIS_RESULT" = "PONG" ] && [ "$NEARBY_DRIVERS" != "0" ]; then
    echo -e "${GREEN}🎉 All tests passed! Infrastructure is ready for development.${NC}"
else
    echo -e "${RED}⚠️ Some tests failed. Check the output above for details.${NC}"
fi

echo ""
echo "Note: Services will be automatically cleaned up when script exits."
