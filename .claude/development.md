# Development Guide

## Automated Build (Recommended)

Use the build script for complete setup:

```bash
./build.sh                # Build all components
./build.sh --skip-cpp     # Skip C++ client (if dependencies missing)
./build.sh --skip-web     # Skip web interface
./build.sh --clean        # Clean all build artifacts
./build.sh --help         # Show options
```

## Manual Local Development Setup

### Prerequisites

**Required System Dependencies (Fedora):**
```bash
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel pkg-config cmake
```

**Required Tools:**
- Node.js & npm (for protocols and web)
- Go 1.21+ (for backend)
- CMake (for C++ client)
- protoc (Protocol Buffer compiler)

### Step 1: Generate Protocol Buffers

```bash
cd protocols/
npm install
make clean
make all
```

This generates:
- C++ headers in `cpp/`
- Go packages in `go/`
- TypeScript files in `ts/`

### Step 2: Build Go Backend

```bash
cd go/
make clean
make chunk_sink
```

Creates binary: `./bin/chunk_sink`

### Step 3: Build C++ Audio Client

```bash
cd cpp/chunk-sink-client/
cmake . -Wno-dev
cmake --build . --parallel
chmod +x chunk-sink-client
```

Creates binary: `./chunk-sink-client`

### Step 4: Build Web Interface

```bash
cd web/
npm install
npm run build
```

Creates distribution: `./dist/`

## Local Runtime Setup

### 1. Start MinIO Storage

```bash
docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   -v ./data/minio:/data \
   -e "MINIO_ROOT_USER=admin" \
   -e "MINIO_ROOT_PASSWORD=password123" \
   quay.io/minio/minio server /data --console-address ":9090"
```

### 2. Configure Environment Variables

```bash
export S3_ENDPOINT=127.0.0.1:9000
export S3_ACCESS_KEY=admin
export S3_SECRET_KEY=password123
export S3_USE_SSL=false
```

### 3. Run Go Backend Services

**Terminal 1 - Chunk Sink Server:**
```bash
cd go/
go run cmd/chunk_sink/main.go cmd/chunk_sink/session-source-handler.go cmd/chunk_sink/chunk-sink-handler.go
```

**Terminal 2 - Session Source Client:**
```bash
cd go/
go run cmd/session_source_client/main.go
```

Or use the built binary:
```bash
cd go/
./bin/chunk_sink
```

### 4. Start gRPC-Web Proxy

```bash
cd grpc-web-proxy/
docker-compose up envoy
```

### 5. Serve Web Interface

**Development server:**
```bash
cd web/
npm start
```

**Production serve:**
```bash
cd web/
npm run preview  # Serves ./dist/
```

### 6. Run Audio Client

```bash
cd cpp/chunk-sink-client/
./chunk-sink-client
```

## Docker Development

For development with live reloading:

```bash
./docker-build.sh up --build
```

This starts:
- **MinIO**: S3 storage on ports 9000/9090
- **Go Backend**: gRPC services on ports 8779/8780
- **gRPC Proxy**: Envoy on port 8080
- **Web Interface**: Nginx on port 3000

The go-backend service mounts source for development via `docker-compose.override.yml`.

## Service Ports & URLs

- **MinIO Console**: http://localhost:9090 (admin/password123)
- **Web Interface**: http://localhost:3000
- **ChunkSink gRPC**: localhost:8779
- **SessionSource gRPC**: localhost:8780
- **gRPC-Web Proxy**: localhost:8080
- **MinIO API**: localhost:9000

## Development Workflow

1. **Protocol Changes**: Run `make all` in `protocols/` to regenerate code
2. **Go Backend**: Restart services or use built binary
3. **C++ Client**: Rebuild with cmake after protocol changes
4. **Web Changes**: Use `npm start` for hot reload development
5. **Docker**: Use `./docker-build.sh up --build` for full rebuild

## Troubleshooting

**Missing Dependencies:**
```bash
# Fedora
dnf install alsa-lib-devel avahi-devel grpc-devel grpc-cpp grpc-plugins

# Check installed packages
pkg-config --exists alsa avahi-client grpc++
```

**CMake Configuration Issues:**
```bash
cd cpp/chunk-sink-client/
rm -rf CMakeFiles CMakeCache.txt Makefile
cmake . -Wno-dev
```

**Protocol Generation Issues:**
```bash
cd protocols/
npm install
make clean && make all
```

**Clean Everything:**
```bash
./build.sh --clean
```