#!/bin/bash

# Development startup script for Session Recorder
# This script starts only MinIO and Envoy for development purposes

set -e

echo "🚀 Starting Session Recorder development environment..."
echo "📦 Services: MinIO (S3 storage) + Envoy (gRPC-Web proxy)"

# Create data directory if it doesn't exist
mkdir -p ./data/minio

# Start services using dedicated development compose file
echo "🔧 Starting MinIO and Envoy services..."
docker-compose -f docker-compose.dev.yml up -d

echo "⏳ Waiting for services to be ready..."

# Wait for MinIO to be healthy
echo "🪣 Checking MinIO health..."
until curl -s http://localhost:9000/minio/health/live > /dev/null 2>&1; do
    echo "   Waiting for MinIO to be ready..."
    sleep 2
done

# Wait for Envoy to be ready
echo "🌐 Checking Envoy health..."
until curl -s http://localhost:9901/ready > /dev/null 2>&1; do
    echo "   Waiting for Envoy to be ready..."
    sleep 2
done

echo ""
echo "✅ Development environment is ready!"
echo ""
echo "📋 Available services:"
echo "   • MinIO S3 API:     http://localhost:9000"
echo "   • MinIO Console:    http://localhost:9090"
echo "   • Envoy Proxy:      http://localhost:8080"
echo "   • Envoy Admin:      http://localhost:9901"
echo ""
echo "🔑 MinIO credentials:"
echo "   Username: admin"
echo "   Password: password123"
echo ""
echo "🛑 To stop services: ./stop-dev.sh"
echo "📝 To view logs: docker-compose -f docker-compose.dev.yml logs -f"
