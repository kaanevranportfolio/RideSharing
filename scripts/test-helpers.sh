#!/usr/bin/env bash

# Shared test helper functions for DRY compliance

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Print result
print_result() {
    local status="$1"
    local message="$2"
    local icon=""
    local color=""
    case "$status" in
        "PASS") icon="✅"; color="$GREEN" ;;
        "FAIL") icon="❌"; color="$RED" ;;
        "WARN") icon="⚠️"; color="$YELLOW" ;;
        *) icon="ℹ️"; color="$BLUE" ;;
    esac
    echo -e "   ${color}${icon} $message${NC}"
}

# Run go tests with timeout and coverage
run_go_tests() {
    local test_path="$1"
    local timeout="$2"
    local coverage_file="$3"
    go test "$test_path" -v -timeout="$timeout" -coverprofile="$coverage_file"
}

# Setup test directories
setup_test_dirs() {
    local reports_dir="$1"
    mkdir -p "$reports_dir"
    mkdir -p "$reports_dir/unit"
    mkdir -p "$reports_dir/integration"
    mkdir -p "$reports_dir/e2e"
}

# Count test results
count_test_results() {
    local log_file="$1"
    local pass_count=$(grep -c "PASS:" "$log_file")
    local fail_count=$(grep -c "FAIL:" "$log_file")
    echo "$pass_count $fail_count"
}

# ...add more shared helpers as needed...
