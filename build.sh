#!/bin/bash

# evilginx-build.sh - Multi-platform build script

set -e  # Exit on error

# Configuration
APP_NAME="evilginx"
BUILD_DIR="./build"
OUTPUT_NAME="$APP_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect platform
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64|x64)
            GOARCH="amd64"
            ;;
        arm64|aarch64)
            GOARCH="arm64"
            ;;
        armv7l|armv8l)
            GOARCH="arm"
            GOARM="7"
            ;;
        i386|i686)
            GOARCH="386"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac
    
    # Set GOOS
    case $OS in
        darwin)
            GOOS="darwin"
            PLATFORM="macOS"
            ;;
        linux)
            GOOS="linux"
            PLATFORM="Linux"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}Detected: $PLATFORM ($GOOS) with architecture $ARCH ($GOARCH)${NC}"
}

# Clean build directory
clean_build() {
    if [ -d "$BUILD_DIR" ]; then
        echo "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
    fi
    mkdir -p "$BUILD_DIR"
}

# Build the binary
build() {
    echo "Building $APP_NAME..."
    
    # Set environment variables
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    if [ -n "$GOARM" ]; then
        export GOARM=$GOARM
    fi
    
    # Build command
    go build -o "$BUILD_DIR/$OUTPUT_NAME" -mod=vendor
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Build successful!${NC}"
        echo -e "Binary location: ${YELLOW}$BUILD_DIR/$OUTPUT_NAME${NC}"
        chmod +x "$BUILD_DIR/$OUTPUT_NAME"
    else
        echo -e "${RED}✗ Build failed!${NC}"
        exit 1
    fi
}

# Run the application
run() {
    echo "Starting $APP_NAME..."
    clear
    sudo "$BUILD_DIR/$OUTPUT_NAME" -p ./phishlets -t ./redirectors -c . -developer -debug
}

# Main execution
main() {
    echo "========================================"
    echo "    $APP_NAME Build Script"
    echo "========================================"
    
    detect_platform
    clean_build
    build
    
    echo ""
    read -p "Do you want to run the application now? (y/n): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        run
    else
        echo -e "${YELLOW}You can run it later with:${NC}"
        echo -e "  ./$BUILD_DIR/$OUTPUT_NAME -p ./phishlets -t ./redirectors -developer -debug"
    fi
}

# Execute main function
main