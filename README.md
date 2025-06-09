# Session Recorder

A distributed audio recording system with real-time streaming and web-based session management.

## Overview

Session Recorder consists of three main components:
- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams to backend
- **Go Backend Services**: Handles audio chunks, session management, and storage
- **Vue.js Web Interface**: Provides session visualization and management UI

## Quick Start

### Option 1: Docker Deployment (Recommended)

Use Docker Compose for the easiest setup:

```bash
# Start all services
./docker-build.sh up --build

# Start with audio client (requires audio devices)
./docker-build.sh up --profile audio --build

# View logs
./docker-build.sh logs

# Stop services
./docker-build.sh down

# Clean up everything
./docker-build.sh clean
```

**Service URLs:**
- 🌐 Web Interface: http://localhost:3000
- 📊 MinIO Console: http://localhost:9090 (admin/password123)
- 🔧 Go Backend: localhost:8779 (ChunkSink), localhost:8780 (SessionSource)
- 🌉 gRPC-Web Proxy: localhost:8080

### Option 2: Local Build Command

Use the automated build script to build all components:

```bash
./build.sh
```

**Build options:**
- `./build.sh --skip-cpp` - Skip C++ client build (if dependencies are missing)
- `./build.sh --skip-web` - Skip web interface build
- `./build.sh --clean` - Clean all build artifacts and generated files
- `./build.sh --help` - Show all available options

### Option 3: Manual Build Process

Follow these steps to run the complete system locally:

### 1. Build C++ Audio Client

```bash
cd cpp/chunk-sink-client/
cmake --build .
chmod +x chunk-sink-client
```

### 2. Start MinIO Storage Server

```bash
docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   -v ~/minio/data:/data \
   -e "MINIO_ROOT_USER=your_minio_username" \
   -e "MINIO_ROOT_PASSWORD=your_minio_password" \
   quay.io/minio/minio server /data --console-address ":9090"
```

### 3. Configure S3 Environment Variables

- Access MinIO console at http://localhost:9090
- Login with the credentials above
- Create access keys and set environment variables:

```bash
export S3_ENDPOINT=127.0.0.1:9000
export S3_ACCESS_KEY=your_s3_access_key
export S3_SECRET_KEY=your_s3_secret_key
```

### 4. Start Go Backend Services

In one terminal, start the chunk sink server:
```bash
cd go/
go run cmd/chunk_sink/main.go cmd/chunk_sink/session-source-handler.go cmd/chunk_sink/chunk-sink-handler.go
```

In another terminal, start the session source client:
```bash
cd go/
go run cmd/session_source_client/main.go
```

### 5. Start gRPC-Web Proxy

```bash
cd grpc-web-proxy/
docker-compose up envoy
```

### 6. Run Web Interface

First, install protocol dependencies:
```bash
cd protocols/
npm install
```

Then start the web application:
```bash
cd web/
npm install
npm start
```

### 7. Start Audio Recording

Once all services are running, execute the C++ client to begin audio capture:
```bash
cd cpp/chunk-sink-client/
./chunk-sink-client
```

## System Requirements

### Docker Deployment (Recommended)
- **Docker** (version 20.10+)
- **Docker Compose** (version 2.0+)
- **Audio devices** (for audio client - optional)

### Local Development
- **Dependencies (Fedora)**: `alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel`
- **Docker** (for MinIO storage and Envoy proxy)
- **Node.js** (for web interface and protocol generation)
- **Go** (version 1.21+)
- **CMake** and **C++ compiler**

## Architecture

The system uses a microservices architecture with the following data flow:

1. **Audio Capture**: C++ client captures audio from ALSA devices
2. **Service Discovery**: Components discover each other via mDNS
3. **Chunk Streaming**: Audio chunks streamed to Go server via gRPC (port 8779)
4. **Storage**: Audio data stored in MinIO S3-compatible storage
5. **Session Management**: Go server provides session API via gRPC (port 8780)
6. **Web Interface**: Vue.js app connects via gRPC-Web through Envoy proxy
7. **Visualization**: Real-time waveform rendering using Peaks.js

## Docker Deployment

### Quick Commands

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

### Service Architecture

The Docker deployment includes:

- **MinIO**: S3-compatible storage for audio files
- **Go Backend**: ChunkSink and SessionSource gRPC services
- **gRPC-Web Proxy**: Envoy proxy for web client communication
- **Web Interface**: Vue.js application served by Nginx
- **Audio Client**: C++ client for audio capture (optional)

### Configuration

Environment variables can be configured in `.env.docker`:

- `MINIO_ROOT_USER`: MinIO admin username
- `MINIO_ROOT_PASSWORD`: MinIO admin password
- `S3_ENDPOINT`: S3 server endpoint
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL for web client

### Volumes and Data

- `minio-data`: Persistent storage for audio files
- `protocol-data`: Generated protocol buffer files shared between services

## Development

### Protocol Buffer Generation
```bash
cd protocols/
make all  # Generates C++, Go, and TypeScript code from .proto files
```

### Testing
```bash
cd web/
npm test  # Run web interface tests

# Storybook for component development:
npx nx storybook --project session-waveform
```

### Building for Production
```bash
cd go/
make chunk_sink  # Build Go binary

cd web/
npm run build    # Build web assets
```