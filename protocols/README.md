# Protocols

Protocol Buffer definitions for the session-recorder system. Generates typed stubs for Go, TypeScript, and C++.

## Proto Files

| File | Services / Messages |
|------|-------------------|
| `proto/common.proto` | `SignalStatus`, `RecorderStatus`, `Response` |
| `proto/chunksink.proto` | `ChunkSink` service (SetChunks, SetRecorderStatus, GetCommands, CutSession) |
| `proto/sessionsource.proto` | `SessionSource` service (StreamRecorders, StreamSessions, CutSession, etc.) |

## Generate Stubs

```bash
npm install    # First time only (installs protobuf-ts plugin)
make all       # Generate all (Go, TypeScript, C++)
make ts        # TypeScript only
make go        # Go only
make cpp       # C++ only
make clean     # Remove generated directories
```

Output directories: `go/`, `ts/`, `cpp/` (all gitignored, regenerated from proto).

## Dependencies

### System (Fedora)

```bash
dnf install grpc-plugins golang-google-protobuf golang-google-grpc grpc-data grpc grpc-cpp grpc-devel protobuf-devel
```

### TypeScript (npm)

```bash
npm install @protobuf-ts/runtime @protobuf-ts/runtime-rpc @protobuf-ts/grpcweb-transport
npm install --save-dev @protobuf-ts/plugin grpc-tools
```

### Custom protoc version

```bash
export PROTOC=~/Downloads/protoc-23.4-linux-x86_64/bin/protoc
export PROTOC_INCLUDES=~/Downloads/protoc-23.4-linux-x86_64/include/google/protobuf
```
