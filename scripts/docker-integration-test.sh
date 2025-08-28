#!/bin/bash

# Enhanced Docker Integration Test Runnerecho '┌─ 🔗 DOCKER INTEGRATION TESTS ──────────────────────────────────────────────┐'

# Run integration tests with proper build tags
echo '   🔎 Running integration tests in: tests/integration'
echo '      🔄 Executing: go test -v ./tests/integration/... -tags=integration -timeout=300s'
cd tests
if go test -v ./integration/... -tags=integration -timeout=300s 2>&1; then
  echo '      ✅ Integration tests passed'
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo '      ❌ Integration tests failed'
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ''

# Run E2E tests with proper build tags and environment setup
echo '   🌐 Running E2E tests in: tests/e2e'
echo '      🔄 Executing: go test -v ./tests/e2e/... -tags=e2e -timeout=300s'
cd tests
if go test -v ./e2e/... -tags=e2e -timeout=300s 2>&1; then
  echo '      ✅ E2E tests passed'
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo '      ❌ E2E tests failed'
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ''

# Run performance tests 
echo '   ⚡ Running performance tests in: tests/performance'
echo '      🔄 Executing: go test -v ./tests/performance/... -timeout=120s'
cd tests
if go test -v ./performance/... -timeout=120s 2>&1; then
  echo '      ✅ Performance tests passed'
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo '      ❌ Performance tests failed'
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ''esults
echo ""
echo "╔══════════════════════════════════════════════════════════════════════════════╗"
echo "║ 🚀 RIDESHARE PLATFORM - DOCKER INTEGRATION TEST RUNNER                       ║"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Test Configuration:"
echo "  ℹ️ Container: rideshare-integration-tests"
echo "  ℹ️ Go Version: $(go version)"
echo "  ℹ️ Working Directory: /app"
echo "  ℹ️ Execution Mode: integration tests with full stack"
echo "  ℹ️ Timestamp: $(date +%Y%m%d_%H%M%S)"
echo ""

# Install required packages
echo "┌─ ⚙️ Environment Setup ─────────────────────────────────────────────────────┐"
echo "   🔄 Installing dependencies..."
apk add --no-cache git make bash postgresql-client redis curl > /dev/null 2>&1
echo "   ✅ Dependencies installed"
echo "└─────────────────────────────────────────────────────────────────────────────┘"
echo ""

# Service health verification
echo "┌─ 🔍 Service Health Verification ───────────────────────────────────────────┐"
echo "   ⏳ Waiting for services to be ready..."
sleep 15

# Check each service
for service in api-gateway:8080 user-service:8051 vehicle-service:8052 geo-service:8053; do
  service_name=$(echo $service | cut -d: -f1)
  service_port=$(echo $service | cut -d: -f2)
  echo "   🔍 Checking $service_name health..."
  
  for i in $(seq 1 30); do
    if curl -f http://$service/health > /dev/null 2>&1; then
      echo "   ✅ $service_name is ready"
      break
    fi
    if [ $i -eq 30 ]; then
      echo "   ⚠️ $service_name health check timeout"
    fi
    sleep 2
  done
done
echo "└─────────────────────────────────────────────────────────────────────────────┘"
echo ""

# Initialize counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo "┌─ 🔗 DOCKER INTEGRATION TESTS ──────────────────────────────────────────────┐"

# Run integration tests with proper build tags
echo "   🔎 Running integration tests in: tests/integration"
echo "      🔄 Executing: go test -v ./integration/... -tags=integration -timeout=300s"
cd tests
if go test -v ./integration/... -tags=integration -timeout=300s 2>&1; then
  echo "      ✅ Integration tests passed"
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo "      ❌ Integration tests failed"
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ""

echo "└─────────────────────────────────────────────────────────────────────────────┘"
echo ""

# Summary
echo "📊 DOCKER INTEGRATION TESTS SUMMARY:"
echo "   ✅ Passed: $PASSED_TESTS"
echo "   ❌ Failed: $FAILED_TESTS"
echo "   📊 Total: $TOTAL_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
  SUCCESS_RATE=100
  STATUS="✅ SUCCESS"
else
  SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
  STATUS="❌ PARTIAL"
fi

echo "   📈 Success Rate: $SUCCESS_RATE%"
echo "   🎯 Status: $STATUS"
echo ""

echo "╔══════════════════════════════════════════════════════════════════════════════╗"
echo "║                🎯 DOCKER INTEGRATION TEST RESULTS SUMMARY                   ║"
echo "╠══════════════════════════════════════════════════════════════════════════════╣"
printf "║ Test Type    │ Status      │ Pass │ Fail │ Total │ Success Rate │ Environment ║\n"
echo "╠══════════════════════════════════════════════════════════════════════════════╣"
printf "║ Integration  │ %-10s │ %-4s │ %-4s │ %-5s │ %-11s │ Docker+Stack║\n" "$STATUS" "$PASSED_TESTS" "$FAILED_TESTS" "$TOTAL_TESTS" "$SUCCESS_RATE%"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo ""

if [ $FAILED_TESTS -gt 0 ]; then
  echo "❌ Integration tests failed: $FAILED_TESTS failure(s) detected in Docker environment."
  exit 1
else
  echo "✅ All Docker integration tests completed successfully!"
  exit 0
fi
