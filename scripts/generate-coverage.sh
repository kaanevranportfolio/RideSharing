#!/usr/bin/env bash

# =============================================================================
# 📊 COMPREHENSIVE TEST COVERAGE ANALYZER
# =============================================================================
# Generates detailed coverage reports for CI/CD pipeline
# Author: Senior Software Engineer

set -euo pipefail

# Color definitions
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

# Icons
readonly CHECK="✅"
readonly CROSS="❌"
readonly WARNING="⚠️"
readonly INFO="ℹ️"
readonly CHART="📊"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo -e "${CYAN}${BOLD}📊 COMPREHENSIVE COVERAGE ANALYSIS${NC}"
echo "===================================="

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Function to generate coverage for a module
generate_module_coverage() {
    local module_path="$1"
    local module_name="$2"
    
    echo -e "${BLUE}${INFO} Analyzing coverage for $module_name...${NC}"
    
    cd "$module_path"
    
    # Generate coverage profile
    if go test ./... -coverprofile="${COVERAGE_DIR}/${module_name}_coverage.out" -covermode=atomic > "${COVERAGE_DIR}/${module_name}_test.log" 2>&1; then
        
        # Generate HTML report
        go tool cover -html="${COVERAGE_DIR}/${module_name}_coverage.out" -o="${COVERAGE_DIR}/${module_name}_coverage.html"
        
        # Get coverage percentage
        local coverage_pct=$(go tool cover -func="${COVERAGE_DIR}/${module_name}_coverage.out" | grep "total:" | awk '{print $3}')
        
        echo -e "${GREEN}${CHECK} $module_name coverage: $coverage_pct${NC}"
        echo "$module_name,$coverage_pct" >> "${COVERAGE_DIR}/coverage_summary.csv"
        
        # Generate detailed function coverage
        go tool cover -func="${COVERAGE_DIR}/${module_name}_coverage.out" > "${COVERAGE_DIR}/${module_name}_functions.txt"
        
    else
        echo -e "${RED}${CROSS} $module_name coverage generation failed${NC}"
        echo "$module_name,ERROR" >> "${COVERAGE_DIR}/coverage_summary.csv"
    fi
    
    cd "$PROJECT_ROOT"
}

# Initialize coverage summary
echo "Module,Coverage" > "${COVERAGE_DIR}/coverage_summary.csv"

echo -e "${CHART} Generating coverage reports..."

# Generate coverage for tests directory
if [ -d "tests" ]; then
    generate_module_coverage "$PROJECT_ROOT/tests" "tests"
fi

# Generate coverage for each service
for service_dir in services/*/; do
    if [ -d "$service_dir" ] && [ -f "${service_dir}go.mod" ]; then
        service_name=$(basename "$service_dir")
        
        # Check if service has test files
        if find "$service_dir" -name "*_test.go" -type f | grep -q .; then
            generate_module_coverage "$PROJECT_ROOT/$service_dir" "$service_name"
        else
            echo -e "${YELLOW}${WARNING} $service_name has no test files${NC}"
            echo "$service_name,NO_TESTS" >> "${COVERAGE_DIR}/coverage_summary.csv"
        fi
    fi
done

# Generate coverage for shared module
if [ -d "shared" ] && [ -f "shared/go.mod" ]; then
    generate_module_coverage "$PROJECT_ROOT/shared" "shared"
fi

# Generate comprehensive HTML report
cat > "${COVERAGE_DIR}/index.html" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🎯 Rideshare Platform - Test Coverage Report</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            padding: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 3px solid #667eea;
        }
        .header h1 {
            color: #667eea;
            margin: 0;
            font-size: 2.5em;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }
        .stat-card {
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
            margin-bottom: 10px;
        }
        .coverage-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
        }
        .coverage-table th,
        .coverage-table td {
            padding: 15px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        .coverage-table th {
            background: #667eea;
            color: white;
            font-weight: bold;
        }
        .coverage-table tr:hover {
            background: #f5f5f5;
        }
        .coverage-high { color: #28a745; font-weight: bold; }
        .coverage-medium { color: #ffc107; font-weight: bold; }
        .coverage-low { color: #dc3545; font-weight: bold; }
        .coverage-error { color: #6c757d; font-style: italic; }
        .timestamp {
            text-align: center;
            color: #666;
            margin-top: 30px;
            font-size: 0.9em;
        }
        .links {
            margin: 20px 0;
            text-align: center;
        }
        .links a {
            display: inline-block;
            margin: 5px 10px;
            padding: 10px 20px;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            transition: background 0.3s;
        }
        .links a:hover {
            background: #5a6fd8;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 Test Coverage Report</h1>
            <p>Comprehensive coverage analysis for Rideshare Platform</p>
        </div>
        
        <div class="stats" id="stats">
            <!-- Stats will be populated by JavaScript -->
        </div>

        <table class="coverage-table">
            <thead>
                <tr>
                    <th>📦 Module</th>
                    <th>📊 Coverage</th>
                    <th>🔗 Detailed Report</th>
                    <th>📋 Functions</th>
                </tr>
            </thead>
            <tbody id="coverage-data">
                <!-- Coverage data will be populated by JavaScript -->
            </tbody>
        </table>

        <div class="links">
            <h3>📁 Additional Reports</h3>
            <a href="../test-reports/">🧪 Test Reports</a>
            <a href="coverage_summary.csv">📄 CSV Export</a>
        </div>

        <div class="timestamp">
            <p>🕐 Generated on: $(date)</p>
            <p>🔄 Auto-refresh every test run</p>
        </div>
    </div>

    <script>
        // Load and display coverage data
        fetch('coverage_summary.csv')
            .then(response => response.text())
            .then(data => {
                const lines = data.trim().split('\\n');
                const tableBody = document.getElementById('coverage-data');
                let totalModules = 0;
                let totalCoverage = 0;
                let modulesWithTests = 0;
                let highCoverage = 0;
                
                lines.slice(1).forEach(line => { // Skip header
                    const [module, coverage] = line.split(',');
                    const row = document.createElement('tr');
                    
                    totalModules++;
                    
                    let coverageClass = 'coverage-error';
                    let coverageDisplay = coverage;
                    
                    if (coverage !== 'ERROR' && coverage !== 'NO_TESTS') {
                        modulesWithTests++;
                        const pct = parseFloat(coverage.replace('%', ''));
                        totalCoverage += pct;
                        
                        if (pct >= 80) {
                            coverageClass = 'coverage-high';
                            highCoverage++;
                        } else if (pct >= 60) {
                            coverageClass = 'coverage-medium';
                        } else {
                            coverageClass = 'coverage-low';
                        }
                    } else if (coverage === 'NO_TESTS') {
                        coverageDisplay = '⚠️ No Tests';
                    } else {
                        coverageDisplay = '❌ Error';
                    }
                    
                    row.innerHTML = \`
                        <td>\${module}</td>
                        <td class="\${coverageClass}">\${coverageDisplay}</td>
                        <td>
                            \${coverage !== 'ERROR' && coverage !== 'NO_TESTS' ? 
                                \`<a href="\${module}_coverage.html">📊 View Report</a>\` : 
                                'N/A'}
                        </td>
                        <td>
                            \${coverage !== 'ERROR' && coverage !== 'NO_TESTS' ? 
                                \`<a href="\${module}_functions.txt">📋 Functions</a>\` : 
                                'N/A'}
                        </td>
                    \`;
                    tableBody.appendChild(row);
                });
                
                // Update stats
                const avgCoverage = modulesWithTests > 0 ? (totalCoverage / modulesWithTests).toFixed(1) : 0;
                document.getElementById('stats').innerHTML = \`
                    <div class="stat-card">
                        <div class="stat-value">\${totalModules}</div>
                        <div>Total Modules</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">\${modulesWithTests}</div>
                        <div>With Tests</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">\${avgCoverage}%</div>
                        <div>Avg Coverage</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">\${highCoverage}</div>
                        <div>High Coverage (80%+)</div>
                    </div>
                \`;
            });
    </script>
</body>
</html>
EOF

echo ""
echo -e "${GREEN}${CHECK} Coverage analysis complete!${NC}"
echo -e "${INFO} Reports generated in: $COVERAGE_DIR${NC}"
echo -e "${INFO} Open: $COVERAGE_DIR/index.html${NC}"

# Create CI/CD friendly coverage badge
if [ -f "${COVERAGE_DIR}/coverage_summary.csv" ]; then
    # Calculate overall coverage
    total_coverage=0
    count=0
    
    while IFS=',' read -r module coverage; do
        if [[ "$coverage" != "Coverage" && "$coverage" != "ERROR" && "$coverage" != "NO_TESTS" ]]; then
            pct=$(echo "$coverage" | sed 's/%//')
            total_coverage=$(echo "$total_coverage + $pct" | bc -l)
            count=$((count + 1))
        fi
    done < "${COVERAGE_DIR}/coverage_summary.csv"
    
    if [ $count -gt 0 ]; then
        avg_coverage=$(echo "scale=1; $total_coverage / $count" | bc -l)
        echo "COVERAGE_PERCENTAGE=$avg_coverage" > "${COVERAGE_DIR}/coverage.env"
        echo -e "${CHART} Overall coverage: ${BOLD}$avg_coverage%${NC}"
        
        # Create coverage badge data
        if (( $(echo "$avg_coverage >= 80" | bc -l) )); then
            badge_color="brightgreen"
        elif (( $(echo "$avg_coverage >= 60" | bc -l) )); then
            badge_color="yellow"
        else
            badge_color="red"
        fi
        
        echo "COVERAGE_BADGE_COLOR=$badge_color" >> "${COVERAGE_DIR}/coverage.env"
    fi
fi

echo -e "${CYAN}🎉 Coverage analysis complete! All reports ready for CI/CD.${NC}"
