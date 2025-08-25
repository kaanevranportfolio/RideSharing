#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="/home/kaan/Projects/rideshare-platform"
TEST_ROOT="${PROJECT_ROOT}/tests"

UNIT_PASS=0
UNIT_FAIL=0

echo "Testing with absolute paths..."

# Test testutils
if [ -d "${TEST_ROOT}/testutils" ]; then
    echo "Testing testutils..."
    cd "${TEST_ROOT}"
    if go test ./testutils/... -v -timeout=30s; then
        ((UNIT_PASS++))
        echo "testutils PASSED"
    else
        ((UNIT_FAIL++))
        echo "testutils FAILED"
    fi
    cd "${PROJECT_ROOT}"
else
    echo "testutils directory not found"
fi

echo "UNIT_PASS: $UNIT_PASS"
echo "UNIT_FAIL: $UNIT_FAIL"
echo "Done!"
