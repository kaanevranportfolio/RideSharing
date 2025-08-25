#!/bin/bash

# =============================================================================
# 🧬 PROTOBUF GENERATION SCRIPT
# =============================================================================
# Generates all gRPC and protobuf files for the rideshare platform
# This script should be run by developers after cloning the repository
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
DNA="🧬"

log() {
    echo -e "$1"
}

print_header() {
    log ""
    log "${CYAN}================================================================================================${NC}"
    log "${CYAN} $1${NC}"
    log "${CYAN}================================================================================================${NC}"
}

# Check if protoc is installed
check_protoc() {
    print_header "${GEAR} CHECKING PROTOBUF COMPILER"
    
    if command -v protoc >/dev/null 2>&1; then
        local version
        version=$(protoc --version)
        log "${GREEN}${CHECKMARK} protoc found: $version${NC}"
        return 0
    else
        log "${RED}${CROSS} protoc not found${NC}"
        return 1
    fi
}

# Install protoc if needed
install_protoc() {
    print_header "📦 INSTALLING PROTOBUF COMPILER"
    
    log "${INFO} Installing protoc..."
    
    # Detect OS
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch
    arch=$(uname -m)
    
    case $arch in
        x86_64) arch="x86_64" ;;
        aarch64|arm64) arch="aarch_64" ;;
        *) log "${RED}${CROSS} Unsupported architecture: $arch${NC}"; exit 1 ;;
    esac
    
    # Download and install protoc
    local protoc_version="25.1"
    local protoc_zip="protoc-${protoc_version}-${os}-${arch}.zip"
    local download_url="https://github.com/protocolbuffers/protobuf/releases/download/v${protoc_version}/${protoc_zip}"
    
    log "${INFO} Downloading protoc $protoc_version for $os/$arch..."
    
    local temp_dir="/tmp/protoc-install"
    mkdir -p "$temp_dir"
    cd "$temp_dir"
    
    if wget -q --show-progress "$download_url"; then
        log "${GREEN}${CHECKMARK} Downloaded $protoc_zip${NC}"
    else
        log "${RED}${CROSS} Failed to download protoc${NC}"
        exit 1
    fi
    
    # Extract and install
    unzip -q "$protoc_zip"
    
    # Install system-wide (requires sudo)
    if [ "$EUID" -eq 0 ]; then
        cp bin/protoc /usr/local/bin/
        cp -r include/* /usr/local/include/
        log "${GREEN}${CHECKMARK} protoc installed system-wide${NC}"
    else
        log "${INFO} Installing protoc to ~/.local/bin (you may need to add this to PATH)${NC}"
        mkdir -p ~/.local/bin ~/.local/include
        cp bin/protoc ~/.local/bin/
        cp -r include/* ~/.local/include/
        export PATH=$PATH:~/.local/bin
        log "${GREEN}${CHECKMARK} protoc installed to ~/.local/bin${NC}"
    fi
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
}

# Check and install Go protobuf plugins
check_go_plugins() {
    print_header "${GEAR} CHECKING GO PROTOBUF PLUGINS"
    
    local plugins_needed=("protoc-gen-go" "protoc-gen-go-grpc")
    local missing_plugins=()
    
    for plugin in "${plugins_needed[@]}"; do
        if command -v "$plugin" >/dev/null 2>&1; then
            log "${GREEN}${CHECKMARK} $plugin found${NC}"
        else
            log "${YELLOW}${WARNING} $plugin not found${NC}"
            missing_plugins+=("$plugin")
        fi
    done
    
    if [ ${#missing_plugins[@]} -gt 0 ]; then
        log "${INFO} Installing missing Go protobuf plugins..."
        
        for plugin in "${missing_plugins[@]}"; do
            case $plugin in
                "protoc-gen-go")
                    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
                    ;;
                "protoc-gen-go-grpc")
                    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
                    ;;
            esac
            log "${GREEN}${CHECKMARK} Installed $plugin${NC}"
        done
    fi
    
    # Ensure Go bin is in PATH
    local go_bin
    go_bin=$(go env GOPATH)/bin
    if [[ ":$PATH:" != *":$go_bin:"* ]]; then
        export PATH=$PATH:$go_bin
        log "${INFO} Added $go_bin to PATH for this session${NC}"
    fi
}

# Clean old generated files
clean_generated_files() {
    print_header "🧹 CLEANING OLD GENERATED FILES"
    
    local files_deleted=0
    
    # Find and delete old .pb.go files
    while IFS= read -r -d '' file; do
        log "${INFO} Removing $file"
        rm "$file"
        ((files_deleted++))
    done < <(find shared/proto -name "*.pb.go" -print0 2>/dev/null || true)
    
    log "${GREEN}${CHECKMARK} Deleted $files_deleted old generated files${NC}"
}

# Generate protobuf files
generate_proto_files() {
    print_header "${DNA} GENERATING PROTOBUF FILES"
    
    local proto_files=()
    local generated_files=0
    
    # Find all .proto files
    while IFS= read -r -d '' file; do
        proto_files+=("$file")
    done < <(find shared/proto -name "*.proto" -print0 2>/dev/null || true)
    
    if [ ${#proto_files[@]} -eq 0 ]; then
        log "${YELLOW}${WARNING} No .proto files found in shared/proto${NC}"
        return 0
    fi
    
    log "${INFO} Found ${#proto_files[@]} proto files to generate${NC}"
    
    # Generate files for each proto
    for proto_file in "${proto_files[@]}"; do
        log "${INFO} Generating: $proto_file"
        
        # Run protoc with Go and gRPC plugins
        if protoc \
            --proto_path=. \
            --go_out=. \
            --go_opt=paths=source_relative \
            --go-grpc_out=. \
            --go-grpc_opt=paths=source_relative \
            "$proto_file"; then
            log "${GREEN}${CHECKMARK} Generated Go files for $proto_file${NC}"
            ((generated_files++))
        else
            log "${RED}${CROSS} Failed to generate files for $proto_file${NC}"
            return 1
        fi
    done
    
    log "${GREEN}${CHECKMARK} Generated files for $generated_files proto definitions${NC}"
}

# Verify generated files
verify_generated_files() {
    print_header "✅ VERIFYING GENERATED FILES"
    
    local pb_files=()
    local grpc_files=()
    
    # Count generated files
    while IFS= read -r -d '' file; do
        pb_files+=("$file")
    done < <(find shared/proto -name "*.pb.go" -print0 2>/dev/null || true)
    
    while IFS= read -r -d '' file; do
        grpc_files+=("$file")
    done < <(find shared/proto -name "*_grpc.pb.go" -print0 2>/dev/null || true)
    
    log "${INFO} Generated ${#pb_files[@]} .pb.go files${NC}"
    log "${INFO} Generated ${#grpc_files[@]} _grpc.pb.go files${NC}"
    
    # Test compilation
    log "${INFO} Testing compilation of generated files..."
    cd shared
    if go build ./proto/...; then
        log "${GREEN}${CHECKMARK} All generated files compile successfully${NC}"
        cd ..
        return 0
    else
        log "${RED}${CROSS} Generated files have compilation errors${NC}"
        cd ..
        return 1
    fi
}

# Create .gitignore entry for generated files (optional)
update_gitignore() {
    print_header "📝 UPDATING .GITIGNORE"
    
    local gitignore_entry="# Generated protobuf files
*.pb.go"
    
    if [ -f .gitignore ]; then
        if ! grep -q "*.pb.go" .gitignore; then
            echo "" >> .gitignore
            echo "$gitignore_entry" >> .gitignore
            log "${GREEN}${CHECKMARK} Added protobuf files to .gitignore${NC}"
        else
            log "${INFO} .gitignore already contains protobuf entries${NC}"
        fi
    else
        echo "$gitignore_entry" > .gitignore
        log "${GREEN}${CHECKMARK} Created .gitignore with protobuf entries${NC}"
    fi
}

# Create developer setup instructions
create_setup_instructions() {
    print_header "📋 CREATING DEVELOPER SETUP INSTRUCTIONS"
    
    cat > PROTO_SETUP.md << 'EOF'
# Protobuf Setup for Developers

## Quick Start

After cloning this repository, run:

```bash
./scripts/generate-proto.sh
```

This will:
1. Check for protoc installation (install if missing)
2. Install Go protobuf plugins
3. Generate all .pb.go files
4. Verify compilation

## Manual Setup

If you prefer manual setup:

### 1. Install protoc

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

**macOS:**
```bash
brew install protobuf
```

**Other systems:**
Download from: https://github.com/protocolbuffers/protobuf/releases

### 2. Install Go plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 3. Generate protobuf files

```bash
./scripts/generate-proto.sh
```

## Protobuf Schema Locations

- **User Service**: `shared/proto/user/`
- **Trip Service**: `shared/proto/trip/`
- **Geo Service**: `shared/proto/geo/`
- **Matching Service**: `shared/proto/matching/`
- **Payment Service**: `shared/proto/payment/`
- **Pricing Service**: `shared/proto/pricing/`

## Generated Files

Generated `.pb.go` files are created alongside `.proto` files and should be committed to the repository for consistency across development environments.

## Troubleshooting

### protoc not found
- Ensure protoc is installed and in your PATH
- Run: `protoc --version` to verify

### Go plugins not found
- Ensure `$(go env GOPATH)/bin` is in your PATH
- Reinstall plugins: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`

### Compilation errors
- Ensure Go version matches project requirements (check go.mod)
- Clean and regenerate: `./scripts/generate-proto.sh`

EOF

    log "${GREEN}${CHECKMARK} Created PROTO_SETUP.md with developer instructions${NC}"
}

# Main execution
main() {
    print_header "${DNA} PROTOBUF GENERATION FOR RIDESHARE PLATFORM"
    
    log "${INFO} Starting protobuf generation process..."
    log "${INFO} Working directory: $(pwd)"
    
    # Check if we're in the right directory
    if [ ! -d "shared/proto" ]; then
        log "${RED}${CROSS} shared/proto directory not found${NC}"
        log "${RED} Please run this script from the rideshare platform root directory${NC}"
        exit 1
    fi
    
    # Run setup steps
    if ! check_protoc; then
        install_protoc
    fi
    
    check_go_plugins
    clean_generated_files
    generate_proto_files
    
    if verify_generated_files; then
        update_gitignore
        create_setup_instructions
        
        log ""
        log "${GREEN}🎉 PROTOBUF GENERATION COMPLETED SUCCESSFULLY! 🎉${NC}"
        log ""
        log "${CYAN}GENERATED FILES:${NC}"
        find shared/proto -name "*.pb.go" | while read -r file; do
            log "${GREEN}  ✓ $file${NC}"
        done
        log ""
        log "${YELLOW}DEVELOPER NOTES:${NC}"
        log "${YELLOW}• Generated files are ready for development${NC}"
        log "${YELLOW}• See PROTO_SETUP.md for setup instructions${NC}"
        log "${YELLOW}• Re-run this script after modifying .proto files${NC}"
        log ""
    else
        log ""
        log "${RED}❌ PROTOBUF GENERATION FAILED${NC}"
        log "${RED}Please check the errors above and fix the issues${NC}"
        exit 1
    fi
}

# Execute main function
main "$@"
