#!/bin/bash

# Development backend startup script for Session Recorder
# This script starts the Go backend services for development

set -e

echo "🚀 Starting Session Recorder Go backend for development..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.19 or later."
    exit 1
fi

# Navigate to the Go directory
cd "$(dirname "$0")/go"

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found. Please ensure you're in the correct directory."
    exit 1
fi

# Build the Go backend
echo "🔨 Building Go backend..."
go build -o bin/session-recorder ./cmd/session-recorder

# Check if the binary was created
if [ ! -f "bin/session-recorder" ]; then
    echo "❌ Failed to build Go backend"
    exit 1
fi

# Set environment variables for development
export S3_ENDPOINT="localhost:9000"
export S3_PUBLIC_ENDPOINT="localhost:9000"
export S3_ACCESS_KEY="admin"
export S3_SECRET_KEY="password123"
export S3_USE_SSL="false"

echo "🎯 Starting Go backend services..."
echo "   • ChunkSink service will be available at: localhost:8779"
echo "   • SessionSource service will be available at: localhost:8780"
echo ""
echo "📋 Environment configuration:"
echo "   • S3_ENDPOINT: $S3_ENDPOINT"
echo "   • S3_PUBLIC_ENDPOINT: $S3_PUBLIC_ENDPOINT"
echo "   • S3_ACCESS_KEY: $S3_ACCESS_KEY"
echo "   • S3_USE_SSL: $S3_USE_SSL"
echo ""
echo "🛑 Press Ctrl+C to stop the backend"
echo ""

# Start the backend
./bin/session-recorder
