# Docker Deployment Guide

## Quick Commands

```bash
# Start all backend services (recommended)
./docker-build.sh up --build

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

The Docker deployment includes backend services only:

- **MinIO**: S3-compatible storage for audio files
- **Go Backend**: ChunkSink and SessionSource gRPC services with mDNS advertising
- **gRPC-Web Proxy**: Envoy proxy for web client communication
- **Web Interface**: Vue.js application served by Nginx

Audio clients run separately and connect via mDNS discovery.

## Multiple Audio Sources

The system is designed for multiple distributed audio clients:

1. **Backend Discovery**: Go backend advertises via mDNS (`_session-recorder-chunksink._tcp`)
2. **Client Connection**: C++ clients scan for and auto-connect to the backend
3. **Distributed Recording**: Multiple clients can record simultaneously from different locations
4. **Unified Management**: All clients appear in the single web interface

## Configuration

Environment variables can be configured in `.env.docker`:

- `MINIO_ROOT_USER`: MinIO admin username (default: admin)
- `MINIO_ROOT_PASSWORD`: MinIO admin password (default: password123)
- `S3_ENDPOINT`: S3 server endpoint
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL for web client

## Volumes and Data

- Local data storage: `./data/minio` (persisted on host)
- No separate volumes needed - uses bind mounts for persistence

## Service URLs

- 🌐 Web Interface: http://localhost:3000
- 📊 MinIO Console: http://localhost:9090 (admin/password123)
- 🔧 Go Backend: localhost:8779 (ChunkSink), localhost:8780 (SessionSource)
- 🌉 gRPC-Web Proxy: localhost:8080