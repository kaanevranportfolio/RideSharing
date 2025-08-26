
#!/bin/bash
# Start all required services for integration tests

# Set PROJECT_ROOT to the parent directory of this script
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Starting test dependencies with docker-compose-test.yml..."
docker compose -f docker-compose-test.yml up -d


# Start application services needed for integration tests
echo "Starting application services for integration tests..."
SERVICES=("api-gateway" "user-service" "vehicle-service" "geo-service" "matching-service" "trip-service" "payment-service" "pricing-service")
for svc in "${SERVICES[@]}"; do
    echo "Building $svc..."
    (cd "$PROJECT_ROOT/services/$svc" && go build -o $svc main.go)
    echo "Starting $svc..."
    (cd "$PROJECT_ROOT/services/$svc" && nohup ./$svc > "$PROJECT_ROOT/services/$svc/$svc.log" 2>&1 &)
done

# Wait for API Gateway to be healthy
echo "Waiting for API Gateway to be healthy..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ API Gateway is healthy"
        break
    fi
    echo "Waiting for API Gateway... ($i/30)"
    sleep 1
done

if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ API Gateway failed to start"
    exit 1
fi

# Wait for test PostgreSQL to be healthy using docker exec
POSTGRES_URL="localhost:5433"
MAX_ATTEMPTS=30
SLEEP_SEC=1

echo "Waiting for test PostgreSQL to be ready..."
for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    if docker exec rideshare-platform-postgres-test-1 pg_isready -U postgres -q 2>/dev/null; then
        echo "Test PostgreSQL is healthy."
        break
    fi
    echo "Waiting for test PostgreSQL... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done

# Wait for test MongoDB to be healthy using docker exec
echo "Waiting for test MongoDB to be ready..."
for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    if docker exec rideshare-platform-mongodb-test-1 mongosh --eval "db.adminCommand('ping').ok" --quiet 2>/dev/null | grep -q "1"; then
        echo "Test MongoDB is healthy."
        break
    fi
    echo "Waiting for test MongoDB... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done

# Wait for test Redis to be healthy using docker exec
echo "Waiting for test Redis to be ready..."
for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    if docker exec rideshare-platform-redis-test-1 redis-cli ping 2>/dev/null | grep -q "PONG"; then
        echo "Test Redis is healthy."
        break
    fi
    echo "Waiting for test Redis... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done

echo "✅ Test infrastructure is ready (databases only)"
echo "ℹ️  Application services require proper build setup"

# Start test API mock for E2E and integration tests requiring API Gateway
if [[ "$ENABLE_API_MOCK" == "true" ]]; then
    echo "🎭 Starting test API mock..."
    cd "${PROJECT_ROOT}/scripts"
    nohup go run test-api-mock.go > "${PROJECT_ROOT}/logs/test-api-mock.log" 2>&1 &
    echo $! > "${PROJECT_ROOT}/logs/test-api-mock.pid"
    
    # Wait for API mock to be ready
    for i in {1..10}; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            echo "✅ Test API mock is healthy"
            break
        fi
        echo "Waiting for test API mock... ($i/10)"
        sleep 1
    done
    
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "❌ Test API mock failed to start"
        exit 1
    fi
fi

echo "✅ Test environment ready"
