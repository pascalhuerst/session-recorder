#!/bin/bash

# Session Recorder Build Script
# Builds all necessary components in the correct order

set -e  # Exit on any error

# Parse command line arguments
SKIP_CPP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-cpp)
            SKIP_CPP=true
            shift
            ;;
        --clean)
            echo "🧹 Cleaning build artifacts..."
            rm -rf protocols/node_modules protocols/cpp protocols/ts protocols/go
            rm -rf cpp/chunk-sink-client/CMakeFiles cpp/chunk-sink-client/CMakeCache.txt cpp/chunk-sink-client/Makefile cpp/chunk-sink-client/cmake_install.cmake cpp/chunk-sink-client/chunk-sink-client
            rm -f cpp/Dockerfile
            echo "🐳 Cleaning Docker build images..."
            docker rmi session-recorder-protocols session-recorder-cpp 2>/dev/null || true
            echo "✅ Clean complete"
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --skip-cpp    Skip C++ client build"
            echo "  --clean       Clean all build artifacts"
            echo "  --help        Show this help message"
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

# 1. Generate Protocol Buffers (using Docker)
print_status "Step 1/4: Generating Protocol Buffers (via Docker)"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is required but not installed. Please install Docker."
    exit 1
fi

# Build protocols in container and extract generated files
print_status "Building protocols container and extracting generated files..."
if ! docker build -t session-recorder-protocols ./protocols/; then
    print_error "Failed to build protocols container"
    exit 1
fi

# Create temporary container to copy files
container_id=$(docker create session-recorder-protocols)
if [ $? -ne 0 ]; then
    print_error "Failed to create protocols container"
    exit 1
fi

# Copy generated files from container
print_status "Extracting generated protocol files..."
docker cp "$container_id:/app/cpp" ./protocols/
docker cp "$container_id:/app/ts" ./protocols/
docker cp "$container_id:/app/go" ./protocols/

# Clean up container
docker rm "$container_id" > /dev/null

print_success "Protocol generation completed (containerized)"

# 2. Build C++ Client (using Docker)
if [ "$SKIP_CPP" = true ]; then
    print_warning "Skipping C++ client build (--skip-cpp flag)"
else
    print_status "Step 2/2: Building C++ Client (via Docker)"
    
    # Create C++ client Dockerfile if it doesn't exist
    if [ ! -f "cpp/Dockerfile" ]; then
        print_status "Creating C++ client Dockerfile..."
        cat > cpp/Dockerfile << 'EOF'
# C++ Client Build Dockerfile
FROM fedora:39

# Install build dependencies
RUN dnf update -y && dnf install -y \
    cmake \
    gcc-c++ \
    make \
    pkg-config \
    alsa-lib-devel \
    avahi-devel \
    grpc-devel \
    grpc-cpp \
    grpc-plugins \
    protobuf-devel \
    openssl-devel \
    boost-devel \
    c-ares-devel \
    re2-devel \
    zlib-devel \
    && dnf clean all

WORKDIR /app

# Copy protocol files
COPY protocols/cpp/ protocols/cpp/

# Copy C++ source code
COPY cpp/chunk-sink-client/ chunk-sink-client/

# Build the application
WORKDIR /app/chunk-sink-client
RUN cmake . -Wno-dev && cmake --build .
RUN chmod +x chunk-sink-client

# This is a build-only container
EOF
    fi
    
    # Build C++ client in container
    print_status "Building C++ client container..."
    if ! docker build -t session-recorder-cpp -f cpp/Dockerfile .; then
        print_error "Failed to build C++ client container"
        exit 1
    fi
    
    # Extract built binary
    container_id=$(docker create session-recorder-cpp)
    if [ $? -ne 0 ]; then
        print_error "Failed to create C++ client container"
        exit 1
    fi
    
    print_status "Extracting C++ binary..."
    docker cp "$container_id:/app/chunk-sink-client/chunk-sink-client" ./cpp/chunk-sink-client/
    docker rm "$container_id" > /dev/null
    
    print_success "C++ client build completed (containerized)"
fi


echo ""
echo "🎉 Build Complete!"
echo "=================="
print_success "All components have been built successfully (using Docker):"
echo "  ✅ Protocol Buffers generated (containerized)"
if [ "$SKIP_CPP" = true ]; then
    echo "  ⚠️  C++ client: Skipped (use --skip-cpp to build)"
else
    echo "  ✅ C++ client: ./cpp/chunk-sink-client/chunk-sink-client (containerized)"
fi
echo ""
print_warning "Note: Generated files are excluded from git (.gitignore)"
print_status "To clean build artifacts: ./build-audio-client.sh --clean"
echo ""
print_status "Next steps:"
echo "  1. Start MinIO storage container"
echo "  2. Configure S3 environment variables"
echo "  3. Run the Go backend services"
echo "  4. Execute the C++ client for audio capture"
echo ""
print_status "See README.md for detailed runtime instructions."