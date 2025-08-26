#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="/home/kaan/Projects/rideshare-platform"
TEST_ROOT="${PROJECT_ROOT}/tests"
REPORTS_DIR="${PROJECT_ROOT}/test-reports"

source "${PROJECT_ROOT}/scripts/test-helpers.sh"

UNIT_PASS=0
UNIT_FAIL=0

echo "Testing with absolute paths..."

setup_test_dirs "$REPORTS_DIR"

# Test testutils
if [ -d "${TEST_ROOT}/testutils" ]; then
    echo "Testing testutils..."
    cd "${TEST_ROOT}"
    log_file="${REPORTS_DIR}/unit/testutils_test.log"
    if run_go_tests "./testutils/..." "30s" "${REPORTS_DIR}/unit/testutils_coverage.out" | tee "$log_file"; then
        read pass fail < <(count_test_results "$log_file")
        UNIT_PASS=$((UNIT_PASS + pass))
        UNIT_FAIL=$((UNIT_FAIL + fail))
        print_result "PASS" "testutils PASSED"
    else
        read pass fail < <(count_test_results "$log_file")
        UNIT_PASS=$((UNIT_PASS + pass))
        UNIT_FAIL=$((UNIT_FAIL + fail))
        print_result "FAIL" "testutils FAILED"
    fi
    cd "$PROJECT_ROOT"
else
    print_result "FAIL" "testutils directory not found"
fi

echo "UNIT_PASS: $UNIT_PASS"
echo "UNIT_FAIL: $UNIT_FAIL"
echo "Done!"
