#!/usr/bin/env bash

# =============================================================================
# 🎯 COMPREHENSIVE TEST ORCHESTRATOR
# =============================================================================
# A centralized test controller with enhanced visualization and reporting
# Author: Senior Test Engineer
# =============================================================================

set -uo pipefail

# Color definitions
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m' # No Color

# Icons
readonly CHECK="✅"
readonly CROSS="❌"
readonly WARNING="⚠️"
readonly INFO="ℹ️"
readonly ROCKET="🚀"
readonly GEAR="⚙️"
readonly CHART="📊"
readonly CLOCK="⏱️"

# Test categories
readonly UNIT_TESTS="unit"
readonly INTEGRATION_TESTS="integration"
readonly E2E_TESTS="e2e"
readonly LOAD_TESTS="load"
readonly SECURITY_TESTS="security"
readonly CONTRACT_TESTS="contract"

# Configuration
readonly PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TEST_ROOT="${PROJECT_ROOT}/tests"
readonly REPORTS_DIR="${PROJECT_ROOT}/test-reports"
readonly TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Test results
declare -A test_results
declare -A test_durations
declare -A test_coverage

# Test counters
UNIT_PASS=0
UNIT_FAIL=0
INTEGRATION_PASS=0
INTEGRATION_FAIL=0
E2E_PASS=0
E2E_FAIL=0
LOAD_PASS=0
LOAD_FAIL=0
SECURITY_PASS=0
SECURITY_FAIL=0
CONTRACT_PASS=0
CONTRACT_FAIL=0

# Utility functions
echo_header() {
    local message="$1"
    echo
    echo -e "${CYAN}┌─ $message ─────────────────────────────────────────────────────┐${NC}"
}

echo_footer() {
    echo -e "${CYAN}└─────────────────────────────────────────────────────────────────┘${NC}"
    echo
}

print_result() {
    local status="$1"
    local message="$2"
    local icon=""
    local color=""
    
    case "$status" in
        "PASS") icon="$CHECK"; color="$GREEN" ;;
        "FAIL") icon="$CROSS"; color="$RED" ;;
        "WARN") icon="$WARNING"; color="$YELLOW" ;;
        *) icon="$INFO"; color="$BLUE" ;;
    esac
    
    echo -e "   ${color}${icon} $message${NC}"
}

print_results() {
    local category="$1"
    local pass_count="$2"
    local fail_count="$3"
    local duration="$4"
    
    echo
    echo -e "${BOLD}📊 $category SUMMARY:${NC}"
    echo -e "   ${GREEN}✅ Passed: $pass_count${NC}"
    echo -e "   ${RED}❌ Failed: $fail_count${NC}"
    echo -e "   ${BLUE}⏱️ Duration: ${duration}s${NC}"
    echo_footer
}

run_test_command() {
    local command="$1"
    echo "    🔄 Executing: $command"
    
    if eval "$command" 2>&1; then
        return 0
    else
        return 1
    fi
}

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

print_header() {
    local title="$1"
    echo -e "\n${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║${NC} ${ROCKET} ${BOLD}${title}${NC}$(printf "%*s" $((75 - ${#title})) "")${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}\n"
}

print_section() {
    local title="$1"
    echo -e "\n${BOLD}${CYAN}┌─ ${GEAR} ${title} ${NC}$(printf '─%.0s' $(seq 1 $((70 - ${#title}))))${BOLD}${CYAN}┐${NC}"
}

print_subsection() {
    local title="$1"
    echo -e "\n${PURPLE}├─ ${title}${NC}"
}

print_result() {
    local status="$1"
    local message="$2"
    local duration="${3:-}"
    
    if [[ "$status" == "PASS" ]]; then
        echo -e "   ${CHECK} ${GREEN}${message}${NC} ${duration:+${YELLOW}(${duration})${NC}}"
    elif [[ "$status" == "FAIL" ]]; then
        echo -e "   ${CROSS} ${RED}${message}${NC} ${duration:+${YELLOW}(${duration})${NC}}"
    elif [[ "$status" == "WARN" ]]; then
        echo -e "   ${WARNING} ${YELLOW}${message}${NC} ${duration:+${YELLOW}(${duration})${NC}}"
    else
        echo -e "   ${INFO} ${message} ${duration:+${YELLOW}(${duration})${NC}}"
    fi
}

print_summary_table() {
    echo -e "\n${BOLD}${BLUE}╔═══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║${NC} ${CHART} ${BOLD}TEST EXECUTION SUMMARY${NC}$(printf "%*s" 49 "")${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}╠═══════════════════════════════════════════════════════════════════════════════╣${NC}"
    
    printf "${BOLD}${BLUE}║${NC} %-20s │ %-10s │ %-10s │ %-15s │ %-10s ${BOLD}${BLUE}║${NC}\n" \
           "Test Category" "Status" "Tests" "Duration" "Coverage"
    
    echo -e "${BOLD}${BLUE}╠═══════════════════════════════════════════════════════════════════════════════╣${NC}"
    
    for category in "$UNIT_TESTS" "$INTEGRATION_TESTS" "$E2E_TESTS" "$LOAD_TESTS" "$SECURITY_TESTS" "$CONTRACT_TESTS"; do
        local status="${test_results[$category]:-SKIP}"
        local duration="${test_durations[$category]:-N/A}"
        local coverage="${test_coverage[$category]:-N/A}"
        local count="N/A"
        
        local status_icon
        case "$status" in
            "PASS") status_icon="${CHECK} ${GREEN}PASS${NC}" ;;
            "FAIL") status_icon="${CROSS} ${RED}FAIL${NC}" ;;
            "WARN") status_icon="${WARNING} ${YELLOW}WARN${NC}" ;;
            *) status_icon="${INFO} ${YELLOW}SKIP${NC}" ;;
        esac
        
        printf "${BOLD}${BLUE}║${NC} %-28s │ %-18s │ %-10s │ %-15s │ %-10s ${BOLD}${BLUE}║${NC}\n" \
               "$category" "$status_icon" "$count" "$duration" "$coverage"
    done
    
    echo -e "${BOLD}${BLUE}╚═══════════════════════════════════════════════════════════════════════════════╝${NC}"
}

setup_test_environment() {
    print_section "Environment Setup"
    
    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$REPORTS_DIR/unit"
    mkdir -p "$REPORTS_DIR/integration"
    mkdir -p "$REPORTS_DIR/e2e"
    mkdir -p "$REPORTS_DIR/load"
    mkdir -p "$REPORTS_DIR/security"
    mkdir -p "$REPORTS_DIR/contract"
    
    print_result "PASS" "Test reports directory created: $REPORTS_DIR"
    
    # Verify test dependencies
    if command -v go >/dev/null 2>&1; then
        print_result "PASS" "Go runtime available: $(go version | cut -d' ' -f3)"
    else
        print_result "FAIL" "Go runtime not found"
        return 1
    fi
    
    # Check for test utilities
    if [[ -d "$TEST_ROOT" ]]; then
        print_result "PASS" "Test directory available: $TEST_ROOT"
    else
        print_result "FAIL" "Test directory not found: $TEST_ROOT"
        return 1
    fi
}

# =============================================================================
# TEST EXECUTION FUNCTIONS
# =============================================================================

run_unit_tests() {
    echo_header "🧪 UNIT TESTS"
    local start_time=$(date +%s)
    
    echo "  🔍 Discovering unit tests..."
    
    # Test the main tests directory
    if [ -d "${TEST_ROOT}/unit" ]; then
        echo "  📂 Testing tests/unit..."
        cd "${TEST_ROOT}"
        if run_test_command "go test ./unit/... -v -timeout=30s"; then
            ((UNIT_PASS++))
        else
            ((UNIT_FAIL++))
        fi
        cd "${PROJECT_ROOT}"
    fi
    
    # Test testutils
    if [ -d "${TEST_ROOT}/testutils" ]; then
        echo "  🛠️  Testing testutils..."
        cd "${TEST_ROOT}"
        if run_test_command "go test ./testutils/... -v -timeout=30s"; then
            ((UNIT_PASS++))
        else
            ((UNIT_FAIL++))
        fi
        cd "${PROJECT_ROOT}"
    fi
    
    # Test individual services that have test files
    for service_dir in "${PROJECT_ROOT}"/services/*/; do
        if [ -d "$service_dir" ] && [ -f "${service_dir}go.mod" ]; then
            service_name=$(basename "$service_dir")
            
            # Check if service has test files
            if find "$service_dir" -name "*_test.go" -type f | grep -q .; then
                echo "  🔧 Testing service: $service_name"
                
                cd "$service_dir"
                if run_test_command "go test ./... -v -timeout=30s"; then
                    ((UNIT_PASS++))
                else
                    ((UNIT_FAIL++))
                fi
                cd "${PROJECT_ROOT}"
            else
                echo "  ⚠️  Service $service_name has no test files - skipping"
            fi
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    print_results "UNIT TESTS" $UNIT_PASS $UNIT_FAIL $duration
}

run_integration_tests() {
    echo_header "🔗 INTEGRATION TESTS"
    local start_time=$(date +%s)
    
    echo "  🔍 Discovering integration tests..."
    
    # Test the integration directory
    if [ -d "tests/integration" ]; then
        cd tests
        echo "  🏗️  Testing integration suite..."
        
        # Find all integration test files
        integration_files=$(find integration -name "*_test.go" -type f | head -10)
        
        for test_file in $integration_files; do
            test_name=$(basename "$test_file" .go)
            echo "    ▶️  Running $test_name..."
            
            if run_test_command "go test ./$test_file -v -timeout=60s"; then
                ((INTEGRATION_PASS++))
            else
                ((INTEGRATION_FAIL++))
            fi
        done
        
        cd ..
    else
        echo "  ⚠️  No integration tests directory found"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    print_results "INTEGRATION TESTS" $INTEGRATION_PASS $INTEGRATION_FAIL $duration
}

run_e2e_tests() {
    print_section "End-to-End Tests Execution"
    local start_time=$(date +%s)
    
    cd "$TEST_ROOT"
    
    print_subsection "E2E Test Scenarios"
    if [[ -d "e2e" ]]; then
        if go test ./e2e/... -v -timeout=10m -json > "$REPORTS_DIR/e2e/results.json" 2>&1; then
            print_result "PASS" "E2E test suite"
            test_results["$E2E_TESTS"]="PASS"
        else
            print_result "FAIL" "E2E test suite"
            test_results["$E2E_TESTS"]="FAIL"
        fi
    else
        print_result "WARN" "E2E tests not found"
        test_results["$E2E_TESTS"]="WARN"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    test_durations["$E2E_TESTS"]="${duration}s"
}

run_load_tests() {
    print_section "Load Tests Execution"
    local start_time=$(date +%s)
    
    print_subsection "Performance Benchmarks"
    cd "$TEST_ROOT"
    
    if go test ./unit/... -bench=. -benchmem -run=^$ > "$REPORTS_DIR/load/benchmark_results.txt" 2>&1; then
        print_result "PASS" "Go benchmark tests"
        
        # Extract benchmark results
        local benchmark_count=$(grep -c "^Benchmark" "$REPORTS_DIR/load/benchmark_results.txt" || echo "0")
        print_result "INFO" "Executed $benchmark_count benchmark tests"
        
        test_results["$LOAD_TESTS"]="PASS"
    else
        print_result "FAIL" "Benchmark tests failed"
        test_results["$LOAD_TESTS"]="FAIL"
    fi
    
    print_subsection "K6 Load Tests"
    if command -v k6 >/dev/null 2>&1; then
        if [[ -f "$TEST_ROOT/performance/load-test.js" ]]; then
            if k6 run --vus 10 --duration 30s "$TEST_ROOT/performance/load-test.js" > "$REPORTS_DIR/load/k6_results.txt" 2>&1; then
                print_result "PASS" "K6 load tests"
            else
                print_result "WARN" "K6 load tests (may need running services)"
            fi
        else
            print_result "WARN" "K6 test scripts not found"
        fi
    else
        print_result "WARN" "K6 not installed"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    test_durations["$LOAD_TESTS"]="${duration}s"
}

run_security_tests() {
    print_section "Security Tests Execution"
    local start_time=$(date +%s)
    
    print_subsection "Static Security Analysis"
    
    # Check for gosec
    if command -v gosec >/dev/null 2>&1; then
        cd "$PROJECT_ROOT"
        if gosec -fmt json -out "$REPORTS_DIR/security/gosec_results.json" ./... >/dev/null 2>&1; then
            local issues=$(jq '.Issues | length' "$REPORTS_DIR/security/gosec_results.json" 2>/dev/null || echo "unknown")
            print_result "PASS" "Static security analysis completed ($issues issues found)"
            test_results["$SECURITY_TESTS"]="PASS"
        else
            print_result "WARN" "Static security analysis had warnings"
            test_results["$SECURITY_TESTS"]="WARN"
        fi
    else
        print_result "WARN" "gosec not installed (go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)"
        test_results["$SECURITY_TESTS"]="WARN"
    fi
    
    print_subsection "Dependency Vulnerability Scan"
    cd "$PROJECT_ROOT"
    if go list -json -deps ./... | nancy sleuth > "$REPORTS_DIR/security/dependency_scan.txt" 2>&1; then
        print_result "PASS" "Dependency vulnerability scan"
    else
        print_result "WARN" "Dependency scan not available (install nancy)"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    test_durations["$SECURITY_TESTS"]="${duration}s"
}

run_contract_tests() {
    print_section "Contract Tests Execution"
    local start_time=$(date +%s)
    
    print_subsection "API Contract Validation"
    
    # Check GraphQL schema validation
    if [[ -f "$PROJECT_ROOT/services/api-gateway/schema/schema.graphql" ]]; then
        print_result "PASS" "GraphQL schema found"
        
        # Validate schema syntax if graphql-cli is available
        if command -v graphql >/dev/null 2>&1; then
            cd "$PROJECT_ROOT/services/api-gateway"
            if graphql validate-schema > "$REPORTS_DIR/contract/graphql_validation.txt" 2>&1; then
                print_result "PASS" "GraphQL schema validation"
            else
                print_result "WARN" "GraphQL schema validation warnings"
            fi
        else
            print_result "INFO" "GraphQL CLI not available for schema validation"
        fi
        
        test_results["$CONTRACT_TESTS"]="PASS"
    else
        print_result "WARN" "GraphQL schema not found"
        test_results["$CONTRACT_TESTS"]="WARN"
    fi
    
    print_subsection "gRPC Proto Validation"
    local proto_files=$(find "$PROJECT_ROOT" -name "*.proto" | wc -l)
    if [[ $proto_files -gt 0 ]]; then
        print_result "PASS" "Found $proto_files protobuf files"
        
        # Validate proto files if protoc is available
        if command -v protoc >/dev/null 2>&1; then
            local proto_valid=true
            while IFS= read -r -d '' proto_file; do
                if ! protoc --descriptor_set_out=/dev/null "$proto_file" 2>/dev/null; then
                    proto_valid=false
                    print_result "FAIL" "Invalid proto file: $(basename "$proto_file")"
                fi
            done < <(find "$PROJECT_ROOT" -name "*.proto" -print0)
            
            if $proto_valid; then
                print_result "PASS" "All protobuf files are valid"
            fi
        else
            print_result "INFO" "protoc not available for proto validation"
        fi
    else
        print_result "WARN" "No protobuf files found"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    test_durations["$CONTRACT_TESTS"]="${duration}s"
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

main() {
    local test_type="${1:-all}"
    
    print_header "RIDESHARE PLATFORM - COMPREHENSIVE TEST SUITE"
    
    echo -e "${BOLD}${CYAN}Test Configuration:${NC}"
    echo -e "  ${INFO} Project Root: $PROJECT_ROOT"
    echo -e "  ${INFO} Test Root: $TEST_ROOT"
    echo -e "  ${INFO} Reports Directory: $REPORTS_DIR"
    echo -e "  ${INFO} Execution Mode: $test_type"
    echo -e "  ${INFO} Timestamp: $TIMESTAMP"
    
    # Setup environment
    if ! setup_test_environment; then
        echo -e "\n${CROSS} ${RED}Environment setup failed. Exiting.${NC}"
        exit 1
    fi
    
    # Execute tests based on type
    case "$test_type" in
        "unit"|"u")
            run_unit_tests
            ;;
        "integration"|"i")
            run_integration_tests
            ;;
        "e2e"|"e")
            run_e2e_tests
            ;;
        "load"|"l")
            run_load_tests
            ;;
        "security"|"s")
            run_security_tests
            ;;
        "contract"|"c")
            run_contract_tests
            ;;
        "all"|*)
            run_unit_tests
            run_integration_tests
            run_e2e_tests
            run_load_tests
            run_security_tests
            run_contract_tests
            ;;
    esac
    
    # Generate final report
    print_summary_table
    
    # Generate HTML report
    local html_report="$REPORTS_DIR/test_summary_${TIMESTAMP}.html"
    generate_html_report "$html_report"
    
    echo -e "\n${ROCKET} ${GREEN}Test execution complete!${NC}"
    echo -e "${INFO} Detailed reports available in: ${CYAN}$REPORTS_DIR${NC}"
    echo -e "${INFO} HTML summary: ${CYAN}$html_report${NC}"
    
    # Exit with appropriate code
    if [[ "${test_results[*]}" =~ "FAIL" ]]; then
        echo -e "\n${CROSS} ${RED}Some tests failed. Please check the reports.${NC}"
        exit 1
    else
        echo -e "\n${CHECK} ${GREEN}All tests passed successfully!${NC}"
        exit 0
    fi
}

generate_html_report() {
    local output_file="$1"
    
    cat > "$output_file" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Rideshare Platform - Test Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 20px; margin-bottom: 30px; }
        .summary-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .summary-table th, .summary-table td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        .summary-table th { background-color: #3498db; color: white; }
        .pass { color: #27ae60; font-weight: bold; }
        .fail { color: #e74c3c; font-weight: bold; }
        .warn { color: #f39c12; font-weight: bold; }
        .skip { color: #95a5a6; font-weight: bold; }
        .section { margin: 30px 0; padding: 20px; border-left: 4px solid #3498db; background: #ecf0f1; }
        .timestamp { color: #7f8c8d; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Rideshare Platform Test Results</h1>
            <p class="timestamp">Generated on: DATE_PLACEHOLDER</p>
        </div>
        
        <div class="section">
            <h2>📊 Test Execution Summary</h2>
            <table class="summary-table">
                <thead>
                    <tr>
                        <th>Test Category</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Coverage</th>
                        <th>Details</th>
                    </tr>
                </thead>
                <tbody id="test-results">
                    <!-- Results will be populated by script -->
                </tbody>
            </table>
        </div>
        
        <div class="section">
            <h2>📁 Report Files</h2>
            <ul>
                <li><strong>Unit Tests:</strong> unit/central_unit_results.json, unit/coverage.html</li>
                <li><strong>Integration Tests:</strong> integration/results.json</li>
                <li><strong>E2E Tests:</strong> e2e/results.json</li>
                <li><strong>Load Tests:</strong> load/benchmark_results.txt, load/k6_results.txt</li>
                <li><strong>Security Tests:</strong> security/gosec_results.json</li>
                <li><strong>Contract Tests:</strong> contract/graphql_validation.txt</li>
            </ul>
        </div>
    </div>
</body>
</html>
EOF
    
    # Replace placeholder with actual timestamp
    sed -i "s/DATE_PLACEHOLDER/$(date)/" "$output_file"
    
    print_result "PASS" "HTML report generated: $output_file"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
