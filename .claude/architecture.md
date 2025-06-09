# Architecture Guide

## System Overview

Session Recorder is a distributed audio recording system with three main components:
- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams chunks via gRPC
- **Go Backend Server**: Receives audio chunks, manages sessions, provides API
- **Vue.js Web Interface**: User interface for managing recordings and sessions

## Service Communication
- **ChunkSink Service** (port 8779): Receives audio chunks from C++ clients
- **SessionSource Service** (port 8780): Provides session management API to web interface
- **mDNS Discovery**: Components auto-discover via `_session-recorder-chunksink._tcp`
- **gRPC-Web Proxy**: Envoy proxies gRPC to web clients on port 8080

## Data Flow
1. C++ client captures audio from ALSA devices
2. Audio chunks streamed to Go server via gRPC
3. Audio stored in MinIO S3-compatible storage
4. Web interface connects via gRPC-Web for session management
5. Session waveform visualization using Peaks.js

## Key Dependencies

**C++ (Fedora)**:
```bash
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel
```

**Environment Variables**:
- `S3_ENDPOINT`: MinIO server endpoint (127.0.0.1:9000 for local)
- `S3_ACCESS_KEY`: S3 access key (get from MinIO console)
- `S3_SECRET_KEY`: S3 secret key (get from MinIO console)
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL (default: http://localhost:4200)
- `VITE_FILE_SERVER_URL`: File server URL (default: http://172.17.0.2:9090)

## Project Structure

- `protocols/`: Shared gRPC definitions and code generation
- `go/`: Backend server with chunk sink and session source services
- `cpp/chunk-sink-client/`: Audio capture client with ALSA integration
- `web/`: Vue.js SPA with NX workspace
- `web/libs/session-waveform/`: Reusable waveform library with Storybook
- `grpc-web-proxy/`: Envoy configuration for gRPC-Web

## NX Workspace (Web)

The web interface uses NX for monorepo management:
- Main app: Session recorder web interface
- Library: `session-waveform` for audio visualization
- Storybook: Component development environment
- Vite: Build tool and dev server