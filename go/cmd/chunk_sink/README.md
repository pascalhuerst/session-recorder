# ChunkSink Server

Main Go backend server that provides both ChunkSink and SessionSource gRPC services.

## Services

- **ChunkSink**: Receives audio chunks from C++ clients (port 8779)
- **SessionSource**: Provides session management API (port 8780)
- **mDNS Advertising**: Broadcasts service discovery for audio clients

## Quick Start

### Docker (Recommended)
```bash
# From project root
./docker-build.sh up --build
```

### Local Development
```bash
# Run with source files
go run main.go session-source-handler.go chunk-sink-handler.go

# Or build and run binary
cd ../../
make chunk_sink
./bin/chunk_sink
```

## Prerequisites

### MinIO Storage
```bash
docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   -v ./data/minio:/data \
   -e "MINIO_ROOT_USER=admin" \
   -e "MINIO_ROOT_PASSWORD=password123" \
   quay.io/minio/minio server /data --console-address ":9090"
```

### Environment Variables
```bash
export S3_ENDPOINT=127.0.0.1:9000
export S3_ACCESS_KEY=admin
export S3_SECRET_KEY=password123
export S3_USE_SSL=false
```

### gRPC-Web Proxy (for web clients)
```bash
cd ../../../grpc-web-proxy/
docker-compose up envoy
```

## Architecture

This server handles:
1. **Audio Chunk Reception**: From C++ clients via ChunkSink gRPC
2. **Session Management**: Create, manage, export sessions via SessionSource gRPC
3. **Storage**: Audio data stored in MinIO S3-compatible storage
4. **Service Discovery**: mDNS advertising for client auto-discovery

See `.claude/architecture.md` for complete system design.
