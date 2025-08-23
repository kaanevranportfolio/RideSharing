#!/bin/bash

# Simple test script for the geo-service
echo "🌍 Starting Geospatial/ETA Service Test..."

# Set environment variables
export SERVICE_NAME="geo-service"
export ENVIRONMENT="development"
export LOG_LEVEL="info"
export GRPC_PORT="50053"
export HTTP_PORT="8053"

# Database configuration
export DB_HOST="localhost"
export DB_PORT="27017"
export DB_NAME="rideshare_geo"
export DB_USERNAME=""
export DB_PASSWORD=""

echo "Environment variables set:"
echo "  Service: $SERVICE_NAME"
echo "  Environment: $ENVIRONMENT"
echo "  Log Level: $LOG_LEVEL"
echo "  gRPC Port: $GRPC_PORT"
echo "  HTTP Port: $HTTP_PORT"
echo "  Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo

echo "🚀 Running geo-service..."
cd /home/kaan/Projects/rideshare-platform/services/geo-service
go run main.go
