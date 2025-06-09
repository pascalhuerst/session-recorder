# Development Guide

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

## Development Workflow

1. **Build C++ Client**: Compile chunk-sink-client executable
2. **Storage**: Start MinIO container for S3-compatible storage
3. **Environment**: Configure S3 credentials from MinIO console
4. **Backend Services**: Run chunk sink server and session source client
5. **Proxy**: Start Envoy proxy for gRPC-Web communication
6. **Web Interface**: Install dependencies and start development server
7. **Audio Capture**: Run C++ client to begin recording

**Service Startup Order**: MinIO → Go backends → Envoy proxy → Web interface → C++ client