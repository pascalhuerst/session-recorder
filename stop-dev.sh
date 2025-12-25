#!/bin/bash

# Development stop script for Session Recorder
# This script stops the development environment services

set -e

echo "🛑 Stopping Session Recorder development environment..."

# Stop services using dedicated development compose file
echo "🔧 Stopping MinIO and Envoy services..."
docker compose -f docker-compose.dev.yml down

echo "🧹 Cleaning up..."
docker compose -f docker-compose.dev.yml down --volumes --remove-orphans

echo ""
echo "✅ Development environment stopped!"
echo ""
echo "💡 To start again: ./start-dev.sh"
