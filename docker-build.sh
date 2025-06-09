#!/bin/bash

# Docker Build Script for Session Recorder
# Builds and manages Docker containers for all components

set -e  # Exit on any error

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

# Parse command line arguments
ACTION="up"
BUILD_FLAG=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD_FLAG="--build"
            shift
            ;;
        up|down|logs|ps|clean)
            ACTION="$1"
            shift
            ;;
        --help)
            echo "Usage: $0 [ACTION] [OPTIONS]"
            echo ""
            echo "Actions:"
            echo "  up      Start all backend services (default)"
            echo "  down    Stop all services"
            echo "  logs    Show logs from all services"
            echo "  ps      Show running containers"
            echo "  clean   Remove all containers, networks, and volumes"
            echo ""
            echo "Options:"
            echo "  --build           Force rebuild of images"
            echo "  --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 up --build                    # Start all services with fresh build"
            echo "  $0 logs                          # Show all logs"
            echo "  $0 down                          # Stop all services"
            echo "  $0 clean                         # Clean up everything"
            echo ""
            echo "Note: Audio clients run separately and connect via mDNS discovery"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Check if docker and docker-compose are installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is required but not installed. Please install Docker."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "docker-compose is required but not installed. Please install docker-compose."
    exit 1
fi

echo "🐳 Session Recorder Docker Management"
echo "====================================="

case $ACTION in
    "up")
        print_status "Starting Session Recorder backend services..."
        docker-compose up $BUILD_FLAG -d
        
        print_status "Waiting for services to be ready..."
        sleep 5
        
        print_success "Backend services started successfully!"
        echo ""
        print_status "Service URLs:"
        echo "  📊 MinIO Console:    http://localhost:9090 (admin/password123)"
        echo "  🌐 Web Interface:    http://localhost:3000"
        echo "  🔧 Go Backend:       localhost:8779 (ChunkSink), localhost:8780 (SessionSource)"
        echo "  🌉 gRPC-Web Proxy:   localhost:8080"
        echo ""
        print_status "Audio clients will auto-discover via mDNS and connect to the backend"
        print_status "Use 'docker-compose logs -f' to follow logs"
        ;;
        
    "down")
        print_status "Stopping Session Recorder services..."
        docker-compose down
        print_success "Services stopped successfully!"
        ;;
        
    "logs")
        print_status "Showing logs from all services..."
        docker-compose logs -f
        ;;
        
    "ps")
        print_status "Running containers:"
        docker-compose ps
        ;;
        
    "clean")
        print_warning "This will remove ALL containers, networks, and volumes!"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Stopping services..."
            docker-compose down
            
            print_status "Removing containers, networks, and volumes..."
            docker-compose down --volumes --remove-orphans
            
            print_status "Removing images..."
            docker-compose down --rmi all --volumes --remove-orphans
            
            print_status "Pruning Docker system..."
            docker system prune -f
            
            print_success "Cleanup complete!"
        else
            print_status "Cleanup cancelled."
        fi
        ;;
esac