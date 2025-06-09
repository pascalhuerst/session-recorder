#!/bin/bash

# Session Recorder Build Script
# Builds all necessary components natively on the local system

set -e  # Exit on any error

# Parse command line arguments
SKIP_CPP=false
SKIP_PROTOCOLS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-cpp)
            SKIP_CPP=true
            shift
            ;;
        --skip-protocols)
            SKIP_PROTOCOLS=true
            shift
            ;;
        --clean)
            echo "🧹 Cleaning build artifacts..."
            rm -rf bin/
            rm -rf protocols/node_modules protocols/cpp protocols/ts protocols/go
            rm -rf cpp/chunk-sink-client/CMakeFiles cpp/chunk-sink-client/CMakeCache.txt cpp/chunk-sink-client/Makefile cpp/chunk-sink-client/cmake_install.cmake
            echo "✅ Clean complete"
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --skip-cpp        Skip C++ client build"
            echo "  --skip-protocols  Skip protocol generation"
            echo "  --clean           Clean all build artifacts"
            echo "  --help            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "🏗️  Building Session Recorder Components"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "README.md" ] || [ ! -d "protocols" ] || [ ! -d "go" ] || [ ! -d "cpp" ]; then
    print_error "Please run this script from the session-recorder root directory"
    exit 1
fi

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        print_error "$1 is required but not installed. Please install $1."
        exit 1
    fi
}

# Function to check for native build dependencies
check_dependencies() {
    print_status "Checking build dependencies..."
    
    if [ "$SKIP_PROTOCOLS" = false ]; then
        check_command "protoc"
        check_command "npm"
        check_command "go"
        
        # Check for protoc plugins
        if ! protoc --plugin=protoc-gen-go --version &> /dev/null; then
            print_error "protoc-gen-go plugin not found. Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
            exit 1
        fi
        
        if ! protoc --plugin=protoc-gen-go-grpc --version &> /dev/null; then
            print_error "protoc-gen-go-grpc plugin not found. Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
            exit 1
        fi
    fi
    
    if [ "$SKIP_CPP" = false ]; then
        check_command "cmake"
        check_command "make"
        check_command "g++"
        
        # Check for required libraries (these checks might vary by system)
        print_status "Note: Ensure you have the following development packages installed:"
        print_status "  - alsa-lib-devel (or libasound2-dev)"
        print_status "  - avahi-devel (or libavahi-client-dev)"
        print_status "  - grpc-devel (or libgrpc++-dev)"
        print_status "  - protobuf-devel (or libprotobuf-dev)"
        print_status "  - boost-devel (or libboost-all-dev)"
    fi
    
    print_success "Dependency check completed"
}

check_dependencies

# 1. Generate Protocol Buffers (native)
if [ "$SKIP_PROTOCOLS" = true ]; then
    print_warning "Skipping protocol generation (--skip-protocols flag)"
else
    print_status "Step 1/2: Generating Protocol Buffers (native)"
    
    cd protocols
    
    # Install Node.js dependencies
    print_status "Installing Node.js dependencies..."
    npm install
    
    # Use existing Makefile for protocol generation (most reliable)
    print_status "Generating protocol files using Makefile..."
    make clean
    make all
    
    cd ..
    print_success "Protocol generation completed (native)"
fi

# 2. Build C++ Client (native)
if [ "$SKIP_CPP" = true ]; then
    print_warning "Skipping C++ client build (--skip-cpp flag)"
else
    print_status "Step 2/2: Building C++ Client (native)"
    
    # Create bin directory for build artifacts
    mkdir -p bin
    
    cd cpp/chunk-sink-client
    
    # Copy protocol files to the build directory
    print_status "Copying protocol files..."
    if [ -d "../../protocols/cpp" ]; then
        cp -r ../../protocols/cpp/* .
    else
        print_error "Protocol files not found. Please run protocol generation first."
        exit 1
    fi
    
    # Configure build with CMake
    print_status "Configuring build with CMake..."
    cmake . -Wno-dev
    
    # Build the application
    print_status "Building C++ client..."
    cmake --build .
    
    # Move binary to bin directory and make executable
    print_status "Moving binary to bin directory..."
    mv chunk-sink-client ../../bin/
    chmod +x ../../bin/chunk-sink-client
    
    cd ../..
    print_success "C++ client build completed (native)"
fi


echo ""
echo "🎉 Build Complete!"
echo "=================="
print_success "All components have been built successfully (native):"

if [ "$SKIP_PROTOCOLS" = true ]; then
    echo "  ⚠️  Protocol Buffers: Skipped (--skip-protocols flag)"
else
    echo "  ✅ Protocol Buffers generated (native)"
fi

if [ "$SKIP_CPP" = true ]; then
    echo "  ⚠️  C++ client: Skipped (--skip-cpp flag)"
else
    echo "  ✅ C++ client: ./bin/chunk-sink-client (native)"
fi

echo ""
print_warning "Note: Generated files are excluded from git (.gitignore)"
print_status "To clean build artifacts: ./build-audio-client.sh --clean"
echo ""
print_status "Next steps:"
echo "  1. Start the backend services: ./docker-build.sh up --build"
echo "  2. Run the C++ client: ./bin/chunk-sink-client"
echo ""
print_status "See README.md for detailed runtime instructions."