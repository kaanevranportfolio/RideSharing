#!/bin/bash

# Enhanced Docker Unit Test Runner with Explicit Reporting

echo "╔══════════════════════════════════════════════════════════════════════════════╗"
echo "║ 🚀 RIDESHARE PLATFORM - DOCKER UNIT TEST RUNNER                              ║"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Test Configuration:"
echo "  ℹ️ Container: rideshare-unit-tests"
echo "  ℹ️ Go Version: $(go version)"
echo "  ℹ️ Working Directory: /app"
echo "  ℹ️ Execution Mode: unit tests only"
echo "  ℹ️ Timestamp: $(date +%Y%m%d_%H%M%S)"
echo ""

# Install required packages
echo "┌─ ⚙️ Environment Setup ─────────────────────────────────────────────────────┐"
echo "   🔄 Installing dependencies..."
apk add --no-cache git make bash > /dev/null 2>&1
echo "   ✅ Dependencies installed"
echo "└─────────────────────────────────────────────────────────────────────────────┘"
echo ""

# Initialize counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo "┌─ 🧪 DOCKER UNIT TESTS ─────────────────────────────────────────────────────┐"

# Test foundation modules first
echo "   🔎 Running unit tests in: tests/testutils"
echo "      🔄 Executing: go test -v ./... -timeout=30s -cover"
cd tests/testutils
if go test -v ./... -timeout=30s -cover 2>&1; then
  echo "      ✅ testutils tests passed"
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo "      ❌ testutils tests failed"
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ""

# Test shared module
echo "   🔎 Running unit tests in: shared"
echo "      🔄 Executing: go test -v ./... -timeout=30s -cover"
cd shared
if go test -v ./... -timeout=30s -cover 2>&1; then
  echo "      ✅ shared module tests passed"
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo "      ❌ shared module tests failed"
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ""

# Test service modules
for service in services/*/; do
  if [ -d "$service" ]; then
    service_name=$(basename "$service")
    echo "   🔎 Running unit tests in: services/$service_name"
    echo "      🔄 Executing: go test -v ./... -timeout=30s -cover"
    cd "$service"
    if go test -v ./... -timeout=30s -cover 2>&1; then
      echo "      ✅ $service_name tests passed"
      PASSED_TESTS=$((PASSED_TESTS + 1))
    else
      echo "      ❌ $service_name tests failed"
      FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    cd /app
    echo ""
  fi
done

# Test performance tests (unit mode)
echo "   ⚡ Running performance tests in: tests/performance"
echo "      🔄 Executing: go test -v ./... -timeout=60s -cover"
cd tests/performance
if go test -v ./... -timeout=60s -cover 2>&1; then
  echo "      ✅ performance tests passed"
  PASSED_TESTS=$((PASSED_TESTS + 1))
else
  echo "      ❌ performance tests failed"
  FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
cd /app
echo ""

echo "└─────────────────────────────────────────────────────────────────────────────┘"
echo ""

# Summary
echo "📊 DOCKER UNIT TESTS SUMMARY:"
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
echo "║                   🎯 DOCKER TEST RESULTS SUMMARY                           ║"
echo "╠══════════════════════════════════════════════════════════════════════════════╣"
printf "║ Test Type    │ Status      │ Pass │ Fail │ Total │ Success Rate │ Environment ║\n"
echo "╠══════════════════════════════════════════════════════════════════════════════╣"
printf "║ Unit         │ %-10s │ %-4s │ %-4s │ %-5s │ %-11s │ Docker      ║\n" "$STATUS" "$PASSED_TESTS" "$FAILED_TESTS" "$TOTAL_TESTS" "$SUCCESS_RATE%"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo ""

if [ $FAILED_TESTS -gt 0 ]; then
  echo "❌ Tests failed: $FAILED_TESTS failure(s) detected in Docker environment."
  exit 1
else
  echo "✅ All Docker unit tests completed successfully!"
  exit 0
fi
