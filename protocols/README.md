# Session Recorder Protocol Definitions

Shared gRPC protocol definitions and code generation for all Session Recorder components.

## Protocols

- **ChunkSink**: Audio chunk streaming from C++ clients to Go backend
- **SessionSource**: Session management API for web interface
- **Common**: Shared types and definitions

## Quick Start

### Generate All Code
```bash
npm install
make clean
make all
```

This generates:
- **C++ headers**: `cpp/` directory
- **Go packages**: `go/` directory  
- **TypeScript**: `ts/` directory

## Build Requirements

### System Dependencies (Fedora)
```bash
dnf install grpc-plugins grpc-devel protobuf-devel
```

### Node.js Dependencies
```bash
npm install @protobuf-ts/runtime @protobuf-ts/runtime-rpc @protobuf-ts/grpcweb-transport
npm install --save-dev @protobuf-ts/plugin grpc-tools
```

## Manual Protocol Compiler Setup

To use a specific `protoc` version:
```bash
# Download from https://github.com/protocolbuffers/protobuf/releases
export PROTOC_INCLUDES=~/Downloads/protoc-23.4-linux-x86_64/include/google/protobuf
export PROTOC=~/Downloads/protoc-23.4-linux-x86_64/bin/protoc
```

## Generated Files

**Note**: Generated files are excluded from git (`.gitignore`) and must be built locally.

## Development Workflow

1. **Modify .proto files** in `proto/` directory
2. **Regenerate code** with `make all`
3. **Rebuild components** that use the protocols

See the main project README.md for complete build instructions.
