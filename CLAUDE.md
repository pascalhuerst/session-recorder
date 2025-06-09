# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Claude Code Guidelines

### CLI Operations
- **ALWAYS** add timeouts to CLI operations using the timeout parameter (default 120s, max 600s)
- Use appropriate timeouts based on operation type:
  - Quick operations (ls, git status): 30s
  - Build operations: 300s (5 minutes)
  - Long-running tests: 600s (10 minutes)
  - Network operations: 180s (3 minutes)

### Cost Optimization
- **Batch tool calls** whenever possible to reduce API overhead
- Use **parallel tool execution** for independent operations
- Prefer **targeted searches** (Grep/Glob) over broad exploration
- Use the **Task tool** for complex searches requiring multiple rounds
- **Read specific files** rather than exploring entire directories
- **Minimize context usage** by reading only necessary file sections

## Project Overview

Session Recorder is a distributed audio recording system with three main components:
- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams chunks via gRPC
- **Go Backend Server**: Receives audio chunks, manages sessions, provides API
- **Vue.js Web Interface**: User interface for managing recordings and sessions

## Docker Deployment (Recommended)

Use Docker Compose for complete containerized deployment:
```bash
./docker-build.sh up --build    # Start all services
./docker-build.sh up --profile audio --build  # Include audio client
./docker-build.sh logs          # View logs
./docker-build.sh down          # Stop services
./docker-build.sh clean         # Clean up everything
```

## Build All Components

Use the automated build script for local development:
```bash
./build.sh                # Build all components
./build.sh --skip-cpp     # Skip C++ client (if dependencies missing)
./build.sh --skip-web     # Skip web interface
./build.sh --clean        # Clean all build artifacts
./build.sh --help         # Show options
```

## Local Development Setup

### 1. Build C++ Client
```bash
cd cpp/chunk-sink-client/
cmake --build .
chmod +x chunk-sink-client
```

### 2. Start MinIO Storage
```bash
docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   -v ~/minio/data:/data \
   -e "MINIO_ROOT_USER=your_minio_username" \
   -e "MINIO_ROOT_PASSWORD=your_minio_password" \
   quay.io/minio/minio server /data --console-address ":9090"
```

### 3. Configure S3 Environment
Access MinIO console at http://localhost:9090 with credentials above, then set:
```bash
export S3_ENDPOINT=127.0.0.1:9000
export S3_ACCESS_KEY=your_s3_access_key
export S3_SECRET_KEY=your_s3_secret_key
```

### 4. Run Go Backend Services
```bash
# Chunk sink server (receives audio chunks)
cd go/
go run cmd/chunk_sink/main.go cmd/chunk_sink/session-source-handler.go cmd/chunk_sink/chunk-sink-handler.go

# Session source client (in separate terminal)
cd go/
go run cmd/session_source_client/main.go
```

### 5. Start gRPC-Web Proxy
```bash
cd grpc-web-proxy/
docker-compose up envoy
```

### 6. Run Web Interface
```bash
# Install protocol dependencies first
cd protocols/
npm install

# Start web app
cd ../web/
npm install
npm start
```

## Additional Commands

### Protocol Generation
```bash
cd protocols/
make all     # Generate C++, Go, and TypeScript code from .proto files
```

### Go Build Commands
```bash
cd go/
make chunk_sink           # Build chunk sink server binary
./bin/chunk_sink          # Run built binary
```

### Web Development
```bash
cd web/
npm test                  # Run tests with Vitest
npm run build            # Production build

# For session-waveform library development:
npx nx storybook --project session-waveform
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
- `S3_ENDPOINT`: MinIO server endpoint (127.0.0.1:9000 for local)
- `S3_ACCESS_KEY`: S3 access key (get from MinIO console)
- `S3_SECRET_KEY`: S3 secret key (get from MinIO console)
- `VITE_GRPC_SERVER_URL`: gRPC-Web proxy URL (default: http://localhost:4200)
- `VITE_FILE_SERVER_URL`: File server URL (default: http://172.17.0.2:9090)

## Development Workflow

1. **Build C++ Client**: Compile chunk-sink-client executable
2. **Storage**: Start MinIO container for S3-compatible storage
3. **Environment**: Configure S3 credentials from MinIO console
4. **Backend Services**: Run chunk sink server and session source client
5. **Proxy**: Start Envoy proxy for gRPC-Web communication
6. **Web Interface**: Install dependencies and start development server
7. **Audio Capture**: Run C++ client to begin recording

**Service Startup Order**: MinIO → Go backends → Envoy proxy → Web interface → C++ client

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