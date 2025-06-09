# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Session Recorder is a distributed audio recording system with three main components:
- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams chunks via gRPC
- **Go Backend Server**: Receives audio chunks, manages sessions, provides API
- **Vue.js Web Interface**: User interface for managing recordings and sessions

## Common Commands

### Protocol Generation (Required First)
```bash
cd protocols/
npm install  # Install protobuf TypeScript plugin
make all     # Generate C++, Go, and TypeScript code from .proto files
```

### Go Backend
```bash
cd go/
make chunk_sink           # Build chunk sink server
./bin/chunk_sink          # Run server (requires S3 env vars)
```

### C++ Client
```bash
cd cpp/chunk-sink-client/
cmake .
make
```

### Web Interface
```bash
cd web/
npm install
npm start                 # Development server
npm test                  # Run tests with Vitest
npm run build            # Production build

# For session-waveform library development:
npx nx storybook --project session-waveform
```

### gRPC-Web Proxy (Required for Web)
```bash
cd grpc-web-proxy/
docker-compose up envoy   # Proxy on port 8080
```

## Architecture

### Service Communication
- **ChunkSink Service** (port 8779): Receives audio chunks from C++ clients
- **SessionSource Service** (port 8780): Provides session management API to web interface
- **mDNS Discovery**: Components auto-discover via `_session-recorder-chunksink._tcp`
- **gRPC-Web Proxy**: Envoy proxies gRPC to web clients on port 8080

### Data Flow
1. C++ client captures audio from ALSA devices
2. Audio chunks streamed to Go server via gRPC
3. Audio stored in MinIO S3-compatible storage
4. Web interface connects via gRPC-Web for session management
5. Session waveform visualization using Peaks.js

### Key Dependencies

**C++ (Fedora)**:
```bash
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel
```

**Environment Variables**:
- `S3_ENDPOINT`: MinIO server endpoint
- `S3_ACCESS_KEY`: S3 access key
- `S3_SECRET_KEY`: S3 secret key
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL (default: http://localhost:4200)
- `VITE_FILE_SERVER_URL`: File server URL (default: http://172.17.0.2:9090)

## Development Workflow

1. **Setup**: Install system dependencies and generate protocols
2. **Backend**: Start Go server with S3 configuration
3. **Proxy**: Run Envoy proxy for gRPC-Web
4. **Frontend**: Create `.env` from `.env.example` and start web server
5. **Client**: Build and run C++ client for audio capture

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