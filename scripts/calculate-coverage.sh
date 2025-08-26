#!/usr/bin/env bash

# Real coverage calculation script
# This script calculates actual test coverage from go test -cover output

set -euo pipefail

PROJECT_ROOT="/home/kaan/Projects/rideshare-platform"
COVERAGE_DIR="${PROJECT_ROOT}/coverage-reports"
REPORTS_DIR="${PROJECT_ROOT}/test-reports"

source "${PROJECT_ROOT}/scripts/test-helpers.sh"

# Function to calculate actual coverage percentage from coverage files
calculate_actual_coverage() {
    local service_name="$1"
    local coverage_file="${COVERAGE_DIR}/${service_name}_coverage.out"
    if [ ! -f "$coverage_file" ]; then
        echo "0.0"
        return
    fi
    if command -v go >/dev/null 2>&1; then
        local coverage=$(go tool cover -func="$coverage_file" 2>/dev/null | tail -1 | awk '{print $3}' | sed 's/%//' || echo "0.0")
        echo "$coverage"
    else
        echo "0.0"
    fi
}

# Function to run tests with coverage for a service (uses shared helper)
run_service_tests_with_coverage() {
    local service_path="$1"
    local service_name="$2"
    echo "Running tests with coverage for $service_name..."
    if [ -d "$service_path" ]; then
        cd "$service_path"
        run_go_tests "./..." "30s" "${COVERAGE_DIR}/${service_name}_coverage.out" | tee "${COVERAGE_DIR}/${service_name}_test.log"
        if [ -f "${COVERAGE_DIR}/${service_name}_coverage.out" ]; then
            go tool cover -html="${COVERAGE_DIR}/${service_name}_coverage.out" -o "${COVERAGE_DIR}/${service_name}_coverage.html"
            go tool cover -func="${COVERAGE_DIR}/${service_name}_coverage.out" > "${COVERAGE_DIR}/${service_name}_functions.txt"
        fi
        cd "$PROJECT_ROOT"
    else
        print_result "FAIL" "Service directory not found: $service_path"
    fi
}

# Main function to calculate comprehensive coverage
calculate_comprehensive_coverage() {
    echo "Calculating real test coverage..."
    
    # Ensure coverage directory exists
    mkdir -p "$COVERAGE_DIR"
    
    # Services to test
    local services=(
        "services/api-gateway:api-gateway"
        "services/user-service:user-service"
        "services/vehicle-service:vehicle-service"
        "services/geo-service:geo-service"
        "services/matching-service:matching-service"
        "services/trip-service:trip-service"
        "services/pricing-service:pricing-service"
        "services/payment-service:payment-service"
        "shared:shared"
    )
    
    local total_coverage=0
    local service_count=0
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r service_path service_name <<< "$service_info"
        
        # Run tests with coverage
        run_service_tests_with_coverage "${PROJECT_ROOT}/${service_path}" "$service_name"
        
        # Calculate coverage
        local coverage=$(calculate_actual_coverage "$service_name")
        echo "$service_name coverage: $coverage%"
        
        # Add to total (handle decimal)
        total_coverage=$(echo "$total_coverage + $coverage" | bc -l)
        ((service_count++))
    done
    
    # Calculate average coverage
    local average_coverage
    if [ "$service_count" -gt 0 ]; then
        average_coverage=$(echo "scale=1; $total_coverage / $service_count" | bc -l)
    else
        average_coverage="0.0"
    fi
    
    echo "Overall coverage: $average_coverage%"
    
    # Store results for test orchestrator
    echo "$average_coverage" > "${COVERAGE_DIR}/overall_coverage.txt"
    
    # Generate combined coverage report
    generate_combined_coverage_report "$average_coverage"
}

# Generate combined coverage report
generate_combined_coverage_report() {
    local overall_coverage="$1"
    
    cat > "${COVERAGE_DIR}/coverage_summary.txt" << EOF
# Test Coverage Summary
Generated: $(date)

## Overall Coverage: ${overall_coverage}%

## Service Coverage:
EOF
    
    # Add individual service coverage
    for service_info in "api-gateway" "user-service" "vehicle-service" "geo-service" "matching-service" "trip-service" "pricing-service" "payment-service" "shared"; do
        local coverage=$(calculate_actual_coverage "$service_info")
        echo "- $service_info: $coverage%" >> "${COVERAGE_DIR}/coverage_summary.txt"
    done
    
    cat >> "${COVERAGE_DIR}/coverage_summary.txt" << EOF

## Coverage Status:
$(if (( $(echo "$overall_coverage >= 70" | bc -l) )); then echo "✅ MEETS 70% REQUIREMENT"; else echo "❌ BELOW 70% REQUIREMENT"; fi)

## Next Steps:
- Review functions with 0% coverage in *_functions.txt files
- Add comprehensive unit tests for core algorithms
- Implement integration tests for business logic
- Add end-to-end tests for complete workflows
EOF
}

# Function to get coverage for use in test orchestrator
get_coverage_for_orchestrator() {
    local test_type="$1"  # unit, integration, e2e
    
    case "$test_type" in
        "unit")
            # Calculate unit test coverage from services
            local unit_coverage=0
            local count=0
            for service in "user-service" "vehicle-service" "geo-service" "matching-service"; do
                local coverage=$(calculate_actual_coverage "$service")
                unit_coverage=$(echo "$unit_coverage + $coverage" | bc -l)
                ((count++))
            done
            if [ "$count" -gt 0 ]; then
                echo "scale=1; $unit_coverage / $count" | bc -l
            else
                echo "0.0"
            fi
            ;;
        "integration")
            # Calculate integration test coverage
            local integration_coverage=0
            local count=0
            for service in "trip-service" "pricing-service" "payment-service"; do
                local coverage=$(calculate_actual_coverage "$service")
                integration_coverage=$(echo "$integration_coverage + $coverage" | bc -l)
                ((count++))
            done
            if [ "$count" -gt 0 ]; then
                echo "scale=1; $integration_coverage / $count" | bc -l
            else
                echo "0.0"
            fi
            ;;
        "overall")
            if [ -f "${COVERAGE_DIR}/overall_coverage.txt" ]; then
                cat "${COVERAGE_DIR}/overall_coverage.txt"
            else
                echo "0.0"
            fi
            ;;
        *)
            echo "0.0"
            ;;
    esac
}

# Main execution
if [ "${1:-}" = "calculate" ]; then
    calculate_comprehensive_coverage
elif [ "${1:-}" = "get" ]; then
    get_coverage_for_orchestrator "${2:-overall}"
else
    echo "Usage: $0 {calculate|get [unit|integration|overall]}"
    exit 1
fi
