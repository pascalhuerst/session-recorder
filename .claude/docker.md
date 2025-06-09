# Docker Deployment Guide

## Quick Commands

```bash
# Start all services
./docker-build.sh up --build

# Start with audio recording capability
./docker-build.sh up --profile audio --build

# View service status
./docker-build.sh ps

# Follow logs
./docker-build.sh logs

# Stop services
./docker-build.sh down

# Complete cleanup
./docker-build.sh clean
```

## Service Architecture

The Docker deployment includes:

- **MinIO**: S3-compatible storage for audio files
- **Go Backend**: ChunkSink and SessionSource gRPC services
- **gRPC-Web Proxy**: Envoy proxy for web client communication
- **Web Interface**: Vue.js application served by Nginx
- **Audio Client**: C++ client for audio capture (optional)

## Configuration

Environment variables can be configured in `.env.docker`:

- `MINIO_ROOT_USER`: MinIO admin username
- `MINIO_ROOT_PASSWORD`: MinIO admin password
- `S3_ENDPOINT`: S3 server endpoint
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL for web client

## Volumes and Data

- `minio-data`: Persistent storage for audio files
- `protocol-data`: Generated protocol buffer files shared between services

## Service URLs

- 🌐 Web Interface: http://localhost:3000
- 📊 MinIO Console: http://localhost:9090 (admin/password123)
- 🔧 Go Backend: localhost:8779 (ChunkSink), localhost:8780 (SessionSource)
- 🌉 gRPC-Web Proxy: localhost:8080