#!/bin/bash

# Session Recorder Build Script
# Builds all necessary components in the correct order

set -e  # Exit on any error

# Parse command line arguments
SKIP_CPP=false
SKIP_WEB=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-cpp)
            SKIP_CPP=true
            shift
            ;;
        --skip-web)
            SKIP_WEB=true
            shift
            ;;
        --clean)
            echo "🧹 Cleaning build artifacts..."
            rm -rf protocols/node_modules protocols/cpp protocols/ts protocols/go
            rm -rf go/bin cpp/chunk-sink-client/CMakeFiles cpp/chunk-sink-client/CMakeCache.txt cpp/chunk-sink-client/Makefile cpp/chunk-sink-client/cmake_install.cmake cpp/chunk-sink-client/chunk-sink-client
            rm -rf web/node_modules web/dist web/.nx
            echo "✅ Clean complete"
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --skip-cpp    Skip C++ client build"
            echo "  --skip-web    Skip web interface build"
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
if [ ! -f "README.md" ] || [ ! -d "protocols" ] || [ ! -d "go" ] || [ ! -d "cpp" ] || [ ! -d "web" ]; then
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

# 2. Build Go Backend
print_status "Step 2/4: Building Go Backend"
cd go/

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is required but not installed. Please install Go."
    exit 1
fi

# Build the chunk sink server
print_status "Building chunk sink server..."
mkdir -p bin
make clean
make chunk_sink

print_success "Go backend build completed"
cd ..

# 3. Build C++ Client
if [ "$SKIP_CPP" = true ]; then
    print_warning "Skipping C++ client build (--skip-cpp flag)"
else
    print_status "Step 3/4: Building C++ Client"
    cd cpp/chunk-sink-client/

# Check if CMake is installed
if ! command -v cmake &> /dev/null; then
    print_error "CMake is required but not installed. Please install CMake."
    exit 1
fi

# Check if required system dependencies are available (Fedora)
print_status "Checking system dependencies..."
missing_deps=()

# Check for pkg-config first
if ! command -v pkg-config &> /dev/null; then
    missing_deps+=("pkg-config")
fi

# Check for required libraries using pkg-config
for lib in alsa avahi-client avahi-core; do
    if ! pkg-config --exists $lib 2>/dev/null; then
        missing_deps+=("$lib-devel")
    fi
done

# Check for gRPC
if ! pkg-config --exists grpc++ 2>/dev/null; then
    missing_deps+=("grpc-devel grpc-cpp grpc-plugins")
fi

if [ ${#missing_deps[@]} -ne 0 ]; then
    print_error "Missing system dependencies. Please install:"
    print_error "dnf install ${missing_deps[*]}"
    exit 1
fi

# Clean any existing build files first
if [ -f "Makefile" ] || [ -d "CMakeFiles" ]; then
    print_status "Cleaning previous build files..."
    rm -rf CMakeFiles CMakeCache.txt Makefile cmake_install.cmake
fi

# Build the C++ client
print_status "Configuring CMake build..."
if ! cmake . -Wno-dev; then
    print_error "CMake configuration failed. Check system dependencies."
    exit 1
fi

print_status "Building C++ client..."
if ! cmake --build . --parallel; then
    print_warning "Parallel build failed, trying sequential build..."
    if ! cmake --build .; then
        print_error "C++ build failed. Check protobuf/gRPC installation."
        print_error "Try: sudo dnf reinstall protobuf-devel grpc-devel"
        exit 1
    fi
fi

# Make the binary executable
chmod +x chunk-sink-client

    print_success "C++ client build completed"
    cd ../..
fi

# 4. Build Web Interface
if [ "$SKIP_WEB" = true ]; then
    print_warning "Skipping web interface build (--skip-web flag)"
else
    print_status "Step 4/4: Building Web Interface"
    cd web/

# Install web dependencies
print_status "Installing web dependencies..."
npm install

# Build the web application
print_status "Building web application..."
npm run build

    print_success "Web interface build completed"
    cd ..
fi

echo ""
echo "🎉 Build Complete!"
echo "=================="
print_success "All components have been built successfully:"
echo "  ✅ Protocol Buffers generated"
echo "  ✅ Go backend: ./go/bin/chunk_sink"
if [ "$SKIP_CPP" = true ]; then
    echo "  ⚠️  C++ client: Skipped (use --skip-cpp to build)"
else
    echo "  ✅ C++ client: ./cpp/chunk-sink-client/chunk-sink-client"
fi
if [ "$SKIP_WEB" = true ]; then
    echo "  ⚠️  Web interface: Skipped"
else
    echo "  ✅ Web interface: ./web/dist/"
fi
echo ""
print_warning "Note: Generated files are excluded from git (.gitignore)"
print_status "To clean build artifacts: ./build.sh --clean"
echo ""
print_status "Next steps:"
echo "  1. Start MinIO storage container"
echo "  2. Configure S3 environment variables"
echo "  3. Run the Go backend services"
echo "  4. Start the gRPC-Web proxy"
echo "  5. Serve the web interface"
echo "  6. Execute the C++ client for audio capture"
echo ""
print_status "See README.md for detailed runtime instructions."