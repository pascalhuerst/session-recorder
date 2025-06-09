# Session Recorder

A distributed audio recording system with real-time streaming and web-based session management.

## Quick Start with Docker

### 1. Build the components (Optional)

The build script generates protocols and builds the C++ client:

```bash
./build.sh
```

**Note**: This is optional since the Docker deployment handles all builds automatically.

### 2. Start the Services

```bash
./docker-build.sh up --build
```

This starts:

- **Web Interface**: http://localhost:3000
- **MinIO Console**: http://localhost:9090 (admin/password123)
- **Go Backend Services**:
    - ChunkSink gRPC: localhost:8779
    - SessionSource gRPC: localhost:8780
- **gRPC-Web Proxy**: http://localhost:8080
- **MinIO API**: http://localhost:9000


### 3. Connect Audio Sources

The system supports multiple audio sources that automatically discover the backend via mDNS:
Use the binary built in step 1 to start the recording.