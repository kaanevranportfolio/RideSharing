#!/bin/bash
# =============================================================================
# 🎯 COMPREHENSIVE TEST ORCHESTRATOR
# =============================================================================
# Runs all test categories, summarizes results, outputs a checklist, and exits non-zero if any test fails.
# References: docs/testing-infrastructure.md, copilot-analysis/03-TESTING-INFRASTRUCTURE.md
# =============================================================================

set -euo pipefail

declare -A results
categories=("unit" "integration" "e2e" "load" "security" "contract")
failures=0

echo "🚀 Starting Comprehensive Test Suite..."
for cat in "${categories[@]}"; do
	echo "\n🔹 Running $cat tests..."
	output=$(./scripts/test-orchestrator.sh "$cat" 2>&1)
	if echo "$output" | grep -q "FAIL" || echo "$output" | grep -q "build failed"; then
		results[$cat]="❌ FAIL"
		((failures++))
	elif echo "$output" | grep -q "PASS"; then
		results[$cat]="✅ PASS"
	else
		results[$cat]="⚠️ UNKNOWN"
		((failures++))
	fi
done

echo "\n====================="
echo "Test Results Checklist"
echo "====================="
for cat in "${categories[@]}"; do
	echo "- $cat: ${results[$cat]}"
done

if [ "$failures" -eq 0 ]; then
	echo "\n🎉 All test categories passed!"
	exit 0
else
	echo "\n⚠️  $failures test category(ies) failed. See above for details."
	exit 1
fi
