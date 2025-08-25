#!/bin/bash

# =============================================================================
# 🔧 PROJECT GO VERSION UPDATE SCRIPT
# =============================================================================
# Updates all project files to use the new Go version consistently
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Icons
CHECKMARK="✅"
CROSS="❌"
WARNING="⚠️"
INFO="ℹ️"
GEAR="⚙️"

# Configuration
NEW_GO_VERSION="1.25"
NEW_GO_FULL_VERSION="1.25.0"

log() {
    echo -e "$1"
}

print_header() {
    log ""
    log "${CYAN}================================================================================================${NC}"
    log "${CYAN} $1${NC}"
    log "${CYAN}================================================================================================${NC}"
}

# Update all go.mod files
update_go_mod_files() {
    print_header "${GEAR} UPDATING GO.MOD FILES"
    
    local files_updated=0
    
    # Find all go.mod files
    while IFS= read -r -d '' file; do
        log "${INFO} Updating $file..."
        
        # Update go version directive
        if sed -i "s/^go [0-9]*\.[0-9]*/go $NEW_GO_VERSION/" "$file"; then
            log "${GREEN}${CHECKMARK} Updated $file${NC}"
            ((files_updated++))
        else
            log "${RED}${CROSS} Failed to update $file${NC}"
        fi
        
    done < <(find . -name "go.mod" -print0)
    
    log "${GREEN}${CHECKMARK} Updated $files_updated go.mod files${NC}"
}

# Update Dockerfiles
update_dockerfiles() {
    print_header "${GEAR} UPDATING DOCKERFILES"
    
    local files_updated=0
    
    # Find all Dockerfiles
    while IFS= read -r -d '' file; do
        log "${INFO} Updating $file..."
        
        # Update golang base image version
        if sed -i "s/FROM golang:[0-9]*\.[0-9]*[0-9]*-alpine/FROM golang:$NEW_GO_VERSION-alpine/" "$file"; then
            log "${GREEN}${CHECKMARK} Updated $file${NC}"
            ((files_updated++))
        else
            log "${RED}${CROSS} Failed to update $file${NC}"
        fi
        
    done < <(find . -name "Dockerfile*" -print0)
    
    log "${GREEN}${CHECKMARK} Updated $files_updated Dockerfile(s)${NC}"
}

# Update GitHub Actions workflow
update_github_actions() {
    print_header "${GEAR} UPDATING GITHUB ACTIONS WORKFLOW"
    
    local workflow_file=".github/workflows/ci-cd.yml"
    
    if [ -f "$workflow_file" ]; then
        log "${INFO} Updating GitHub Actions workflow..."
        
        # Update GO_VERSION environment variable
        sed -i "s/GO_VERSION: '[0-9]*\.[0-9]*'/GO_VERSION: '$NEW_GO_VERSION'/" "$workflow_file"
        
        # Update setup-go action version references
        sed -i "s/go-version: ['\"][0-9]*\.[0-9]*['\"/go-version: '$NEW_GO_VERSION'/g" "$workflow_file"
        
        log "${GREEN}${CHECKMARK} Updated GitHub Actions workflow${NC}"
    else
        log "${YELLOW}${WARNING} GitHub Actions workflow not found${NC}"
    fi
}

# Update docker-compose files
update_docker_compose() {
    print_header "${GEAR} UPDATING DOCKER-COMPOSE FILES"
    
    local files_updated=0
    
    # Find all docker-compose files
    while IFS= read -r -d '' file; do
        if grep -q "golang:" "$file" 2>/dev/null; then
            log "${INFO} Updating $file..."
            
            # Update golang image version in docker-compose files
            if sed -i "s/golang:[0-9]*\.[0-9]*[0-9]*-alpine/golang:$NEW_GO_VERSION-alpine/g" "$file"; then
                log "${GREEN}${CHECKMARK} Updated $file${NC}"
                ((files_updated++))
            else
                log "${RED}${CROSS} Failed to update $file${NC}"
            fi
        fi
    done < <(find . -name "docker-compose*.yml" -print0)
    
    if [ $files_updated -eq 0 ]; then
        log "${INFO} No docker-compose files with golang images found${NC}"
    else
        log "${GREEN}${CHECKMARK} Updated $files_updated docker-compose file(s)${NC}"
    fi
}

# Clean and update dependencies
clean_and_update_deps() {
    print_header "${GEAR} CLEANING AND UPDATING DEPENDENCIES"
    
    log "${INFO} Cleaning Go module cache..."
    go clean -modcache
    
    log "${INFO} Updating shared module..."
    cd shared
    go mod tidy
    cd ..
    
    log "${INFO} Updating service modules..."
    for service_dir in services/*/; do
        if [ -f "${service_dir}go.mod" ]; then
            service=$(basename "$service_dir")
            log "${INFO} Updating $service..."
            cd "$service_dir"
            go mod tidy
            cd - > /dev/null
        fi
    done
    
    log "${INFO} Updating tests module..."
    if [ -f "tests/go.mod" ]; then
        cd tests
        go mod tidy
        cd ..
    fi
    
    log "${GREEN}${CHECKMARK} Dependencies updated${NC}"
}

# Verify Go version consistency
verify_consistency() {
    print_header "✅ VERIFYING GO VERSION CONSISTENCY"
    
    local errors=0
    
    # Check system Go version
    local system_version
    system_version=$(go version | grep -o 'go[0-9]*\.[0-9]*' | sed 's/go//')
    
    if [[ "$system_version" == "$NEW_GO_VERSION"* ]]; then
        log "${GREEN}${CHECKMARK} System Go version: $system_version${NC}"
    else
        log "${RED}${CROSS} System Go version mismatch: $system_version (expected $NEW_GO_VERSION.x)${NC}"
        ((errors++))
    fi
    
    # Check go.mod files
    local mod_errors=0
    while IFS= read -r -d '' file; do
        if ! grep -q "^go $NEW_GO_VERSION" "$file"; then
            log "${RED}${CROSS} $file has wrong Go version${NC}"
            ((mod_errors++))
        fi
    done < <(find . -name "go.mod" -print0)
    
    if [ $mod_errors -eq 0 ]; then
        log "${GREEN}${CHECKMARK} All go.mod files use Go $NEW_GO_VERSION${NC}"
    else
        log "${RED}${CROSS} $mod_errors go.mod files have wrong Go version${NC}"
        ((errors++))
    fi
    
    # Check Dockerfiles
    local dockerfile_errors=0
    while IFS= read -r -d '' file; do
        if ! grep -q "golang:$NEW_GO_VERSION-alpine" "$file"; then
            log "${RED}${CROSS} $file has wrong Go version${NC}"
            ((dockerfile_errors++))
        fi
    done < <(find . -name "Dockerfile*" -print0)
    
    if [ $dockerfile_errors -eq 0 ]; then
        log "${GREEN}${CHECKMARK} All Dockerfiles use Go $NEW_GO_VERSION${NC}"
    else
        log "${RED}${CROSS} $dockerfile_errors Dockerfiles have wrong Go version${NC}"
        ((errors++))
    fi
    
    return $errors
}

# Main execution
main() {
    print_header "🔧 PROJECT GO VERSION UPDATE TO $NEW_GO_VERSION"
    
    log "${INFO} Current directory: $(pwd)"
    log "${INFO} Target Go version: $NEW_GO_VERSION"
    log "${INFO} System Go version: $(go version 2>/dev/null || echo 'Not available')"
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ] && [ ! -d "services" ]; then
        log "${RED}${CROSS} This doesn't appear to be the rideshare platform root directory${NC}"
        log "${RED} Please run this script from the project root${NC}"
        exit 1
    fi
    
    # Update all configurations
    update_go_mod_files
    update_dockerfiles
    update_github_actions
    update_docker_compose
    clean_and_update_deps
    
    # Verify everything is consistent
    if verify_consistency; then
        log ""
        log "${GREEN}🎉 PROJECT GO VERSION UPDATE COMPLETED SUCCESSFULLY! 🎉${NC}"
        log ""
        log "${CYAN}ALL PROJECT FILES NOW USE GO $NEW_GO_VERSION${NC}"
        log ""
        log "${YELLOW}NEXT STEPS:${NC}"
        log "${YELLOW}1. Test the project: make test-all${NC}"
        log "${YELLOW}2. Rebuild containers: docker-compose build${NC}"
        log "${YELLOW}3. Generate protobuf files: ./scripts/generate-proto.sh${NC}"
        log ""
    else
        log ""
        log "${RED}❌ SOME INCONSISTENCIES FOUND${NC}"
        log "${RED}Please review the errors above and fix manually${NC}"
        exit 1
    fi
}

# Execute main function
main "$@"
