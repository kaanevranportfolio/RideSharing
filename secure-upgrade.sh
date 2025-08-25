#!/bin/bash

# =============================================================================
# 🛡️ RIDESHARE PLATFORM SECURITY & GO UPGRADE SCRIPT
# =============================================================================
# This script:
# 1. Upgrades Go to the latest version
# 2. Regenerates protobuf files with new Go version
# 3. Secures environment configuration
# 4. Runs comprehensive tests
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project paths
PROJECT_ROOT=$(pwd)
SCRIPTS_DIR="$PROJECT_ROOT/scripts"

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

print_banner() {
    echo -e "${BLUE}"
    cat << 'EOF'
 ____  _     _           _                    
|  _ \(_) __| | ___  ___| |__   __ _ _ __ ___ 
| |_) | |/ _` |/ _ \/ __| '_ \ / _` | '__/ _ \
|  _ <| | (_| |  __/\__ \ | | | (_| | | |  __/
|_| \_\_|\__,_|\___||___/_| |_|\__,_|_|  \___|
                                             
🛡️ Security & Go Upgrade Script
EOF
    echo -e "${NC}"
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if running as non-root user
    if [[ $EUID -eq 0 ]]; then
        error "This script should not be run as root for security reasons"
        error "Run as a regular user (sudo will be prompted when needed)"
        exit 1
    fi
    
    # Check internet connectivity
    if ! ping -c 1 google.com &> /dev/null; then
        error "No internet connection available"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

backup_current_state() {
    log "Creating backup of current state..."
    
    BACKUP_DIR="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Backup Go installation info
    if command -v go &> /dev/null; then
        go version > "$BACKUP_DIR/go_version_before.txt"
        echo "$GOPATH" > "$BACKUP_DIR/gopath_before.txt"
        echo "$GOROOT" > "$BACKUP_DIR/goroot_before.txt"
    fi
    
    # Backup environment files
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        cp "$PROJECT_ROOT/.env" "$BACKUP_DIR/.env.backup"
    fi
    
    # Backup proto files
    find "$PROJECT_ROOT" -name "*.pb.go" -exec cp {} "$BACKUP_DIR/" \; 2>/dev/null || true
    
    success "Backup created at $BACKUP_DIR"
}

upgrade_go_version() {
    log "Upgrading Go to latest version..."
    
    # Check current Go version
    CURRENT_GO=""
    if command -v go &> /dev/null; then
        CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
        log "Current Go version: $CURRENT_GO"
    else
        log "Go not currently installed"
    fi
    
    # Get latest Go version
    LATEST_GO=$(curl -s https://golang.org/VERSION?m=text | sed 's/go//')
    log "Latest Go version: $LATEST_GO"
    
    if [[ "$CURRENT_GO" == "$LATEST_GO" ]]; then
        success "Go is already up to date ($LATEST_GO)"
        return 0
    fi
    
    # Download and install latest Go
    log "Downloading Go $LATEST_GO..."
    cd /tmp
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv7l) ARCH="armv6l" ;;
        *) error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    GO_TARBALL="go${LATEST_GO}.linux-${ARCH}.tar.gz"
    
    if ! wget -q "https://golang.org/dl/$GO_TARBALL"; then
        error "Failed to download Go $LATEST_GO"
        exit 1
    fi
    
    # Remove old Go installation
    if [[ -d "/usr/local/go" ]]; then
        log "Removing old Go installation..."
        sudo rm -rf /usr/local/go
    fi
    
    # Install new Go
    log "Installing Go $LATEST_GO..."
    sudo tar -C /usr/local -xzf "$GO_TARBALL"
    rm "$GO_TARBALL"
    
    # Update PATH in common shell configs
    for shell_config in ~/.bashrc ~/.zshrc ~/.profile; do
        if [[ -f "$shell_config" ]]; then
            # Remove old Go paths
            sed -i '/export PATH.*\/usr\/local\/go\/bin/d' "$shell_config"
            sed -i '/export GOPATH/d' "$shell_config"
            sed -i '/export GOROOT/d' "$shell_config"
            
            # Add new Go paths
            echo 'export PATH=$PATH:/usr/local/go/bin' >> "$shell_config"
            echo 'export GOPATH=$HOME/go' >> "$shell_config"
            echo 'export GOROOT=/usr/local/go' >> "$shell_config"
        fi
    done
    
    # Update current session
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export GOROOT=/usr/local/go
    
    # Verify installation
    if command -v go &> /dev/null; then
        NEW_VERSION=$(go version)
        success "Go upgraded successfully: $NEW_VERSION"
    else
        error "Go installation failed"
        exit 1
    fi
    
    cd "$PROJECT_ROOT"
}

update_go_modules() {
    log "Updating Go modules to use new Go version..."
    
    # Update go.mod files
    find "$PROJECT_ROOT" -name "go.mod" -type f | while read -r go_mod; do
        dir=$(dirname "$go_mod")
        log "Updating $go_mod"
        
        cd "$dir"
        
        # Update Go version in go.mod
        if grep -q "^go " go.mod; then
            sed -i "s/^go .*/go $(go version | awk '{print $3}' | sed 's/go//')/" go.mod
        fi
        
        # Clean and update dependencies
        go mod tidy
        go mod download
        
        cd "$PROJECT_ROOT"
    done
    
    success "Go modules updated"
}

regenerate_protobuf() {
    log "Regenerating protobuf files with new Go version..."
    
    if [[ -f "$PROJECT_ROOT/generate-proto.sh" ]]; then
        chmod +x "$PROJECT_ROOT/generate-proto.sh"
        ./generate-proto.sh
        success "Protobuf files regenerated"
    else
        warning "generate-proto.sh not found, skipping protobuf regeneration"
    fi
}

secure_environment() {
    log "Securing environment configuration..."
    
    # Check for hardcoded secrets
    warning "Scanning for potential hardcoded secrets..."
    
    # Common patterns to check
    PATTERNS=(
        "password.*=.*[^CHANGE_ME]"
        "secret.*=.*[^CHANGE_ME]"
        "key.*=.*[^CHANGE_ME]"
        "token.*=.*[^CHANGE_ME]"
    )
    
    for pattern in "${PATTERNS[@]}"; do
        if grep -r -i --include="*.yml" --include="*.yaml" --include="*.env" --exclude=".env.example" "$pattern" "$PROJECT_ROOT" 2>/dev/null; then
            warning "Found potential hardcoded secrets matching: $pattern"
        fi
    done
    
    # Create .env from .env.example if it doesn't exist
    if [[ ! -f "$PROJECT_ROOT/.env" ]] && [[ -f "$PROJECT_ROOT/.env.example" ]]; then
        log "Creating .env from .env.example..."
        cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
        warning "IMPORTANT: Please update the credentials in .env file!"
        warning "Default passwords are set to CHANGE_ME_* - these MUST be changed!"
    fi
    
    # Generate random secrets
    log "Generating secure random values..."
    
    if command -v openssl &> /dev/null; then
        echo "# Generated secure values - $(date)" > "$PROJECT_ROOT/generated-secrets.txt"
        echo "JWT_SECRET=$(openssl rand -base64 32)" >> "$PROJECT_ROOT/generated-secrets.txt"
        echo "ENCRYPTION_KEY=$(openssl rand -hex 32)" >> "$PROJECT_ROOT/generated-secrets.txt"
        echo "POSTGRES_PASSWORD=$(openssl rand -base64 16 | tr -d /=+ | cut -c -16)" >> "$PROJECT_ROOT/generated-secrets.txt"
        echo "MONGODB_PASSWORD=$(openssl rand -base64 16 | tr -d /=+ | cut -c -16)" >> "$PROJECT_ROOT/generated-secrets.txt"
        
        success "Secure random values generated in generated-secrets.txt"
        warning "Copy these values to your .env file and then DELETE generated-secrets.txt"
    else
        warning "OpenSSL not found, please manually generate secure passwords"
    fi
    
    success "Environment security check completed"
}

run_comprehensive_tests() {
    log "Running comprehensive tests..."
    
    # Build all services first
    log "Building all services..."
    if [[ -f "$PROJECT_ROOT/Makefile" ]]; then
        make build || warning "Build failed - some tests may not work"
    fi
    
    # Run tests service by service
    services=("user-service" "vehicle-service" "geo-service" "matching-service" "pricing-service" "trip-service" "payment-service")
    
    for service in "${services[@]}"; do
        if [[ -d "$PROJECT_ROOT/services/$service" ]]; then
            log "Testing $service..."
            cd "$PROJECT_ROOT/services/$service"
            
            # Run tests with coverage
            if go test -v -race -coverprofile=coverage.out ./... 2>&1; then
                success "$service tests passed"
                
                # Show coverage
                if [[ -f coverage.out ]]; then
                    coverage=$(go tool cover -func=coverage.out | tail -n 1 | awk '{print $3}')
                    log "$service coverage: $coverage"
                fi
            else
                error "$service tests failed"
            fi
            
            cd "$PROJECT_ROOT"
        fi
    done
    
    # Run integration tests
    if [[ -d "$PROJECT_ROOT/tests" ]]; then
        log "Running integration tests..."
        cd "$PROJECT_ROOT/tests"
        
        if go test -v -race ./... 2>&1; then
            success "Integration tests passed"
        else
            error "Integration tests failed"
        fi
        
        cd "$PROJECT_ROOT"
    fi
}

cleanup() {
    log "Cleaning up temporary files..."
    
    # Remove generated secrets file if it exists
    if [[ -f "$PROJECT_ROOT/generated-secrets.txt" ]]; then
        warning "SECURITY: Removing generated-secrets.txt"
        warning "Make sure you copied the values to your .env file first!"
        read -p "Have you copied the secrets to .env? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm "$PROJECT_ROOT/generated-secrets.txt"
            success "Temporary secrets file removed"
        else
            warning "Keeping generated-secrets.txt - remember to delete it after copying!"
        fi
    fi
    
    # Clean up any .orig files
    find "$PROJECT_ROOT" -name "*.orig" -delete 2>/dev/null || true
    
    success "Cleanup completed"
}

main() {
    print_banner
    
    log "Starting Rideshare Platform Security & Go Upgrade..."
    
    check_prerequisites
    backup_current_state
    upgrade_go_version
    update_go_modules
    regenerate_protobuf
    secure_environment
    run_comprehensive_tests
    cleanup
    
    success "🎉 Security & Go upgrade completed successfully!"
    echo
    echo -e "${GREEN}Next steps:${NC}"
    echo "1. Update credentials in .env file"
    echo "2. Review test results and fix any failures"
    echo "3. Deploy updated services"
    echo "4. Run end-to-end tests"
    
    # Show final Go version
    echo
    log "Final Go version: $(go version)"
}

# Run main function
main "$@"
