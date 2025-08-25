#!/usr/bin/env bash

set -euo pipefail

# Test counters
UNIT_PASS=0
UNIT_FAIL=0

echo "Testing testutils specifically..."

if [ -d "tests/testutils" ]; then
    cd tests
    echo "Testing testutils..."
    if go test ./testutils/... -v -timeout=30s; then
        ((UNIT_PASS++))
        echo "testutils PASSED"
    else
        ((UNIT_FAIL++))
        echo "testutils FAILED"
    fi
    cd ..
else
    echo "testutils directory not found"
fi

echo "UNIT_PASS: $UNIT_PASS"
echo "UNIT_FAIL: $UNIT_FAIL"
echo "Done!"
