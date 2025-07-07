# Session Recorder

## System Overview

Session Recorder is a distributed audio recording system with three main components:

- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams chunks via gRPC
- **Go Backend Server**: Receives audio chunks, manages sessions, provides API
- **Vue.js Web Interface**: User interface for managing recordings and sessions

## Running Backend Server and Web Interface

### With Docker

```bash
# Start all backend services
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

This starts:

- **Web Interface**: http://localhost:3000
- **MinIO Console**: http://localhost:9090 (admin/password123)
- **Go Backend Services**:
    - ChunkSink gRPC: localhost:8779
    - SessionSource gRPC: localhost:8780
- **gRPC-Web Proxy**: http://localhost:8080
- **MinIO API**: http://localhost:9000


## Connecting Audio Sources

### Build the chunk sink (recording device)

Before building the components, ensure you have the required dependencies installed:

```bash
dnf install cmake make gcc-c++ alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel protobuf-devel boost-devel
```

#### 1. Generate Protocol Buffers

```bash
cd protocols
npm install
make clean
make all
cd ..
```

#### 2. Build C++ Client

```bash
cd ./cpp/chunk-sink-client
cmake --build .
chmod +x ./chunk-sink-client
cd ../..
```

### Execute

```bash
# Run from project root directory
./cpp/chunk-sink-client/chunk-sink-client --recorder-id <uuid> --recorder-name <name>
# ./cpp/chunk-sink-client/chunk-sink-client --recorder-id 10b26ce0-75ff-4548-84e1-c91d955b1151 --recorder-name "Living Room"
```
