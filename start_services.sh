#!/bin/bash

# Start all rideshare platform services with proper environment configuration
set -e

# Load environment variables
source .env

# Export additional service-specific environment variables
export POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
export MONGODB_PASSWORD=${MONGODB_PASSWORD}
export JWT_SECRET_KEY=${JWT_SECRET_KEY}

# Service-specific ports
export USER_SERVICE_PORT=8081
export VEHICLE_SERVICE_PORT=8082
export GEO_SERVICE_PORT=8083
export MATCHING_SERVICE_PORT=8084
export TRIP_SERVICE_PORT=8085
export PAYMENT_SERVICE_PORT=8086
export PRICING_SERVICE_PORT=8087
export API_GATEWAY_PORT=8080

echo "🚀 Starting Rideshare Platform Services..."

# Start User Service
echo "Starting User Service on port $USER_SERVICE_PORT..."
cd services/user-service
HTTP_PORT=$USER_SERVICE_PORT nohup ./user-service > user-service.log 2>&1 &
USER_PID=$!
echo "✅ User Service started (PID: $USER_PID)"
cd ../..

# Start Vehicle Service
echo "Starting Vehicle Service on port $VEHICLE_SERVICE_PORT..."
cd services/vehicle-service
HTTP_PORT=$VEHICLE_SERVICE_PORT nohup ./vehicle-service > vehicle-service.log 2>&1 &
VEHICLE_PID=$!
echo "✅ Vehicle Service started (PID: $VEHICLE_PID)"
cd ../..

# Start Geo Service
echo "Starting Geo Service on port $GEO_SERVICE_PORT..."
cd services/geo-service
HTTP_PORT=$GEO_SERVICE_PORT nohup ./geo-service > geo-service.log 2>&1 &
GEO_PID=$!
echo "✅ Geo Service started (PID: $GEO_PID)"
cd ../..

# Start Matching Service
echo "Starting Matching Service on port $MATCHING_SERVICE_PORT..."
cd services/matching-service
HTTP_PORT=$MATCHING_SERVICE_PORT nohup ./matching-service > matching-service.log 2>&1 &
MATCHING_PID=$!
echo "✅ Matching Service started (PID: $MATCHING_PID)"
cd ../..

# Start Trip Service
echo "Starting Trip Service on port $TRIP_SERVICE_PORT..."
cd services/trip-service
HTTP_PORT=$TRIP_SERVICE_PORT nohup ./trip-service > trip-service.log 2>&1 &
TRIP_PID=$!
echo "✅ Trip Service started (PID: $TRIP_PID)"
cd ../..

# Start Payment Service
echo "Starting Payment Service on port $PAYMENT_SERVICE_PORT..."
cd services/payment-service
HTTP_PORT=$PAYMENT_SERVICE_PORT nohup ./payment-service > payment-service.log 2>&1 &
PAYMENT_PID=$!
echo "✅ Payment Service started (PID: $PAYMENT_PID)"
cd ../..

# Start Pricing Service
echo "Starting Pricing Service on port $PRICING_SERVICE_PORT..."
cd services/pricing-service
HTTP_PORT=$PRICING_SERVICE_PORT nohup ./pricing-service > pricing-service.log 2>&1 &
PRICING_PID=$!
echo "✅ Pricing Service started (PID: $PRICING_PID)"
cd ../..

# Start API Gateway
echo "Starting API Gateway on port $API_GATEWAY_PORT..."
cd services/api-gateway
HTTP_PORT=$API_GATEWAY_PORT nohup ./api-gateway > api-gateway.log 2>&1 &
GATEWAY_PID=$!
echo "✅ API Gateway started (PID: $GATEWAY_PID)"
cd ../..

# Give services time to start
echo "⏳ Waiting for services to initialize..."
sleep 5

echo ""
echo "🎉 All services started successfully!"
echo ""
echo "📊 Service Status:"
echo "• User Service:    http://localhost:$USER_SERVICE_PORT"
echo "• Vehicle Service: http://localhost:$VEHICLE_SERVICE_PORT"
echo "• Geo Service:     http://localhost:$GEO_SERVICE_PORT"
echo "• Matching Service: http://localhost:$MATCHING_SERVICE_PORT"
echo "• Trip Service:    http://localhost:$TRIP_SERVICE_PORT"
echo "• Payment Service: http://localhost:$PAYMENT_SERVICE_PORT"
echo "• Pricing Service: http://localhost:$PRICING_SERVICE_PORT"
echo "• API Gateway:     http://localhost:$API_GATEWAY_PORT"
echo ""
echo "🔍 To check service logs:"
echo "  tail -f services/*//*.log"
echo ""
echo "⚡ To test the platform:"
echo "  make test-all"
echo ""
echo "🛑 To stop all services:"
echo "  pkill -f 'user-service|vehicle-service|geo-service|matching-service|trip-service|payment-service|pricing-service|api-gateway'"
