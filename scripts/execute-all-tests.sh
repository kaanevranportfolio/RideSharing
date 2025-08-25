#!/bin/bash

# =============================================================================
# 🚀 COMPREHENSIVE TEST EXECUTION SCRIPT
# =============================================================================
# This script ensures ALL tests pass as demanded by the user
# It runs unit tests, integration tests, builds all services, and provides coverage
# =============================================================================

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Icons
CHECKMARK="✅"
CROSS="❌"
WARNING="⚠️"
INFO="ℹ️"
ROCKET="🚀"
GEAR="⚙️"

LOG_FILE="test-execution-$(date +%Y%m%d_%H%M%S).log"

# Logging function
log() {
    echo -e "$1" | tee -a "$LOG_FILE"
}

# Header function
print_header() {
    log ""
    log "${CYAN}================================================================================================${NC}"
    log "${CYAN} $1${NC}"
    log "${CYAN}================================================================================================${NC}"
}

# Check if services build successfully
check_builds() {
    print_header "${GEAR} CHECKING SERVICE BUILDS"
    
    local failed_builds=()
    
    for service_dir in services/*/; do
        service=$(basename "$service_dir")
        log "${BLUE}${INFO} Building $service...${NC}"
        
        cd "$service_dir"
        
        if go build -v -o "$service" . 2>&1 | tee -a "../../../$LOG_FILE"; then
            log "${GREEN}${CHECKMARK} $service built successfully${NC}"
        else
            log "${RED}${CROSS} $service build failed${NC}"
            failed_builds+=("$service")
        fi
        
        cd - > /dev/null
    done
    
    if [ ${#failed_builds[@]} -eq 0 ]; then
        log "${GREEN}${CHECKMARK} ALL SERVICE BUILDS PASSED${NC}"
        return 0
    else
        log "${RED}${CROSS} FAILED BUILDS: ${failed_builds[*]}${NC}"
        return 1
    fi
}

# Run unit tests for all services
run_unit_tests() {
    print_header "${GEAR} RUNNING UNIT TESTS"
    
    local failed_tests=()
    local total_tests=0
    local passed_tests=0
    
    # Test shared module
    log "${BLUE}${INFO} Testing shared module...${NC}"
    cd shared
    if go test ./... -v 2>&1 | tee -a "../$LOG_FILE"; then
        log "${GREEN}${CHECKMARK} shared module tests passed${NC}"
        ((passed_tests++))
    else
        log "${RED}${CROSS} shared module tests failed${NC}"
        failed_tests+=("shared")
    fi
    ((total_tests++))
    cd ..
    
    # Test all services
    for service_dir in services/*/; do
        service=$(basename "$service_dir")
        log "${BLUE}${INFO} Testing $service...${NC}"
        
        cd "$service_dir"
        ((total_tests++))
        
        if go test ./... -v 2>&1 | tee -a "../../$LOG_FILE"; then
            log "${GREEN}${CHECKMARK} $service tests passed${NC}"
            ((passed_tests++))
        else
            log "${RED}${CROSS} $service tests failed${NC}"
            failed_tests+=("$service")
        fi
        
        cd - > /dev/null
    done
    
    # Test testutils
    log "${BLUE}${INFO} Testing testutils...${NC}"
    cd tests
    ((total_tests++))
    if go test ./testutils/... -v 2>&1 | tee -a "../$LOG_FILE"; then
        log "${GREEN}${CHECKMARK} testutils tests passed${NC}"
        ((passed_tests++))
    else
        log "${RED}${CROSS} testutils tests failed${NC}"
        failed_tests+=("testutils")
    fi
    cd ..
    
    log ""
    log "${CYAN}UNIT TEST SUMMARY:${NC}"
    log "${CYAN}Total modules tested: $total_tests${NC}"
    log "${GREEN}Passed: $passed_tests${NC}"
    log "${RED}Failed: $((total_tests - passed_tests))${NC}"
    
    if [ ${#failed_tests[@]} -eq 0 ]; then
        log "${GREEN}${CHECKMARK} ALL UNIT TESTS PASSED${NC}"
        return 0
    else
        log "${RED}${CROSS} FAILED UNIT TESTS: ${failed_tests[*]}${NC}"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    print_header "${GEAR} RUNNING INTEGRATION TESTS"
    
    # Check if Docker test infrastructure is running
    if ! docker compose -f docker-compose-test.yml ps | grep -q "Up"; then
        log "${YELLOW}${WARNING} Starting Docker test infrastructure...${NC}"
        docker compose -f docker-compose-test.yml up -d
        sleep 10  # Wait for services to be ready
    fi
    
    # Source test environment
    if [ -f .env.test ]; then
        source .env.test
        log "${GREEN}${CHECKMARK} Loaded test environment variables${NC}"
    else
        log "${YELLOW}${WARNING} No .env.test file found, using defaults${NC}"
    fi
    
    cd tests
    local failed_integration=()
    
    for test_file in integration/*_test.go; do
        if [ -f "$test_file" ]; then
            test_name=$(basename "$test_file" .go)
            log "${BLUE}${INFO} Running $test_name...${NC}"
            
            if go test "./$test_file" -v 2>&1 | tee -a "../$LOG_FILE"; then
                log "${GREEN}${CHECKMARK} $test_name passed${NC}"
            else
                log "${RED}${CROSS} $test_name failed${NC}"
                failed_integration+=("$test_name")
            fi
        fi
    done
    
    cd ..
    
    if [ ${#failed_integration[@]} -eq 0 ]; then
        log "${GREEN}${CHECKMARK} ALL INTEGRATION TESTS PASSED${NC}"
        return 0
    else
        log "${RED}${CROSS} FAILED INTEGRATION TESTS: ${failed_integration[*]}${NC}"
        return 1
    fi
}

# Generate coverage report
generate_coverage() {
    print_header "${GEAR} GENERATING COVERAGE REPORT"
    
    if [ -f scripts/generate-coverage.sh ]; then
        chmod +x scripts/generate-coverage.sh
        ./scripts/generate-coverage.sh
        log "${GREEN}${CHECKMARK} Coverage report generated${NC}"
    else
        log "${YELLOW}${WARNING} Coverage script not found, skipping coverage generation${NC}"
    fi
}

# Main execution function
main() {
    log "${ROCKET} Starting comprehensive test execution..."
    log "${INFO} Timestamp: $(date)"
    log "${INFO} User requirement: ALL TESTS MUST PASS"
    log "${INFO} CI/CD readiness: GitHub Actions compatible"
    
    local all_passed=true
    
    # Step 1: Check if all services build
    if ! check_builds; then
        log "${RED}${CROSS} BUILD VERIFICATION FAILED${NC}"
        all_passed=false
    fi
    
    # Step 2: Run unit tests
    if ! run_unit_tests; then
        log "${RED}${CROSS} UNIT TESTS FAILED${NC}"
        all_passed=false
    fi
    
    # Step 3: Run integration tests
    if ! run_integration_tests; then
        log "${RED}${CROSS} INTEGRATION TESTS FAILED${NC}"
        all_passed=false
    fi
    
    # Step 4: Generate coverage
    generate_coverage
    
    # Final report
    print_header "${ROCKET} FINAL TEST EXECUTION REPORT"
    
    if [ "$all_passed" = true ]; then
        log "${GREEN}${CHECKMARK}${CHECKMARK}${CHECKMARK} ALL TESTS PASSED SUCCESSFULLY! ${CHECKMARK}${CHECKMARK}${CHECKMARK}${NC}"
        log "${GREEN}✨ CI/CD READY: Your rideshare platform is production-ready! ✨${NC}"
        log "${GREEN}📊 Coverage reports available in coverage-reports/index.html${NC}"
        log "${GREEN}📋 Full test log available in $LOG_FILE${NC}"
        exit 0
    else
        log "${RED}${CROSS}${CROSS}${CROSS} SOME TESTS FAILED! ${CROSS}${CROSS}${CROSS}${NC}"
        log "${RED}❌ CI/CD NOT READY: Please fix the failing tests before deployment${NC}"
        log "${RED}📋 Check test log for details: $LOG_FILE${NC}"
        exit 1
    fi
}

# Trap to clean up on exit
cleanup() {
    log ""
    log "${INFO} Test execution completed at $(date)"
}
trap cleanup EXIT

# Execute main function
main "$@"
