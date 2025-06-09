# Session Recorder Go Backend

Backend services for the Session Recorder distributed audio recording system.

## Components

- **ChunkSink Service**: Receives audio chunks from C++ clients via gRPC (port 8779)
- **SessionSource Service**: Provides session management API to web interface (port 8780)
- **mDNS Advertising**: Broadcasts service discovery for audio clients

## Quick Start

### Docker (Recommended)
```bash
# From project root
./docker-build.sh up --build
```

### Local Development
```bash
# Build binary
make chunk_sink

# Run with source
go run cmd/chunk_sink/main.go cmd/chunk_sink/session-source-handler.go cmd/chunk_sink/chunk-sink-handler.go
```

## Prerequisites

- Go 1.21+
- MinIO S3 storage (configured in docker-compose.yml)
- Environment variables:
  - `S3_ENDPOINT`
  - `S3_ACCESS_KEY` 
  - `S3_SECRET_KEY`

## Architecture

See the main project README.md for complete system design.
