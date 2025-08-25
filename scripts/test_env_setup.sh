#!/bin/bash
# Start all required services for integration tests

done
echo "API Gateway did not become healthy in time."
docker-compose logs api-gateway
echo "Starting test dependencies with docker-compose-test.yml..."
docker compose -f docker-compose-test.yml up -d

# Wait for test PostgreSQL to be healthy
POSTGRES_URL="localhost:5433"
MAX_ATTEMPTS=30
SLEEP_SEC=1

for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    STATUS=$(pg_isready -h localhost -p 5433 -U postgres | grep "accepting connections")
    if [ ! -z "$STATUS" ]; then
        echo "Test PostgreSQL is healthy."
        break
    fi
    echo "Waiting for test PostgreSQL... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done

# Wait for test MongoDB to be healthy
for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    STATUS=$(mongosh --host localhost --port 27018 --eval "db.adminCommand('ping').ok" --quiet)
    if [ "$STATUS" == "1" ]; then
        echo "Test MongoDB is healthy."
        break
    fi
    echo "Waiting for test MongoDB... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done

# Wait for test Redis to be healthy
for ((i=1;i<=MAX_ATTEMPTS;i++)); do
    STATUS=$(redis-cli -h localhost -p 6380 ping)
    if [ "$STATUS" == "PONG" ]; then
        echo "Test Redis is healthy."
        break
    fi
    echo "Waiting for test Redis... ($i/$MAX_ATTEMPTS)"
    sleep $SLEEP_SEC
done
