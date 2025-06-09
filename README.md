# Session Recorder

A distributed audio recording system with real-time streaming and web-based session management.

## Start the Services

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

## Connect Audio Sources

The system supports multiple audio sources that automatically discover the backend via mDNS.

```bash
# Build the audio client binary with
./build-audio-client.sh

# Execute it
chmod +x ./cpp/chunk-sink-client
./cpp/chunk-sink-client
```
