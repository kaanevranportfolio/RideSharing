#!/bin/bash

# =============================================================================
# 🚀 GO VERSION UPGRADE SCRIPT
# =============================================================================
# Upgrades system Go to latest version and updates all project configurations
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
ROCKET="🚀"

# Configuration
GO_VERSION="1.25.0"
GO_MINOR_VERSION="1.25"
INSTALL_DIR="/usr/local"
GO_ROOT="$INSTALL_DIR/go"

log() {
    echo -e "$1"
}

print_header() {
    log ""
    log "${CYAN}================================================================================================${NC}"
    log "${CYAN} $1${NC}"
    log "${CYAN}================================================================================================${NC}"
}

# Check if running as root for system installation
check_permissions() {
    if [ "$EUID" -ne 0 ]; then
        log "${YELLOW}${WARNING} This script needs sudo privileges to install Go system-wide${NC}"
        log "${INFO} Re-running with sudo..."
        exec sudo "$0" "$@"
    fi
}

# Backup current Go installation
backup_current_go() {
    print_header "📦 BACKING UP CURRENT GO INSTALLATION"
    
    if [ -d "$GO_ROOT" ]; then
        log "${INFO} Backing up current Go installation..."
        mv "$GO_ROOT" "${GO_ROOT}.backup.$(date +%Y%m%d_%H%M%S)"
        log "${GREEN}${CHECKMARK} Current Go installation backed up${NC}"
    else
        log "${INFO} No existing Go installation found in $GO_ROOT${NC}"
    fi
}

# Download and install Go
install_go() {
    print_header "⬇️ DOWNLOADING AND INSTALLING GO $GO_VERSION"
    
    local go_archive="go${GO_VERSION}.linux-amd64.tar.gz"
    local download_url="https://go.dev/dl/${go_archive}"
    local temp_dir="/tmp/go-install"
    
    # Create temp directory
    mkdir -p "$temp_dir"
    cd "$temp_dir"
    
    log "${INFO} Downloading Go $GO_VERSION from $download_url..."
    if wget -q --show-progress "$download_url"; then
        log "${GREEN}${CHECKMARK} Go $GO_VERSION downloaded successfully${NC}"
    else
        log "${RED}${CROSS} Failed to download Go $GO_VERSION${NC}"
        exit 1
    fi
    
    log "${INFO} Installing Go to $INSTALL_DIR..."
    tar -C "$INSTALL_DIR" -xzf "$go_archive"
    
    if [ -d "$GO_ROOT" ]; then
        log "${GREEN}${CHECKMARK} Go $GO_VERSION installed successfully${NC}"
    else
        log "${RED}${CROSS} Go installation failed${NC}"
        exit 1
    fi
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
}

# Update PATH and environment
update_environment() {
    print_header "🔧 UPDATING ENVIRONMENT VARIABLES"
    
    # Update system-wide profile
    echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh
    chmod +x /etc/profile.d/go.sh
    
    # Update current session
    export PATH=$PATH:/usr/local/go/bin
    
    log "${GREEN}${CHECKMARK} Environment variables updated${NC}"
    log "${INFO} Please run 'source /etc/profile.d/go.sh' or restart your terminal${NC}"
}

# Verify installation
verify_installation() {
    print_header "✅ VERIFYING GO INSTALLATION"
    
    # Source the new PATH
    export PATH=$PATH:/usr/local/go/bin
    
    local installed_version
    installed_version=$(/usr/local/go/bin/go version 2>/dev/null || echo "failed")
    
    if [[ "$installed_version" == *"$GO_VERSION"* ]]; then
        log "${GREEN}${CHECKMARK} Go $GO_VERSION installed and verified successfully${NC}"
        log "${INFO} Installed version: $installed_version${NC}"
        return 0
    else
        log "${RED}${CROSS} Go installation verification failed${NC}"
        log "${RED} Expected version containing: $GO_VERSION${NC}"
        log "${RED} Got: $installed_version${NC}"
        return 1
    fi
}

# Main installation process
main() {
    print_header "${ROCKET} GO $GO_VERSION SYSTEM UPGRADE"
    
    log "${INFO} Current Go version: $(go version 2>/dev/null || echo 'Not installed')"
    log "${INFO} Target Go version: $GO_VERSION"
    log "${INFO} Installation directory: $GO_ROOT"
    
    # Run installation steps
    backup_current_go
    install_go
    update_environment
    
    if verify_installation; then
        log ""
        log "${GREEN}🎉 GO $GO_VERSION INSTALLATION COMPLETED SUCCESSFULLY! 🎉${NC}"
        log ""
        log "${CYAN}NEXT STEPS:${NC}"
        log "${YELLOW}1. Restart your terminal or run: source /etc/profile.d/go.sh${NC}"
        log "${YELLOW}2. Verify with: go version${NC}"
        log "${YELLOW}3. Run the project configuration update script${NC}"
        log ""
    else
        log ""
        log "${RED}❌ GO INSTALLATION FAILED${NC}"
        log "${RED}Please check the logs above and try again${NC}"
        exit 1
    fi
}

# Check permissions and run
check_permissions "$@"
main "$@"
