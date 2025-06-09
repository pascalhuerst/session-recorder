# Session Recorder Audio Client

C++ audio capture client that streams audio chunks to the Session Recorder backend via gRPC.

## Features

- **ALSA Audio Capture**: Records from ALSA audio devices
- **Automatic Discovery**: Finds backend services via mDNS (`_session-recorder-chunksink._tcp`)
- **Real-time Streaming**: Streams audio chunks over gRPC
- **Signal Detection**: Only records when audio signal is detected
- **Multiple Clients**: Supports multiple simultaneous recording sources

## Quick Start

### Prerequisites (Fedora)
```bash
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel cmake
```

### Build
```bash
cmake --build .
chmod +x chunk-sink-client
```

### Run
```bash
# Start Docker backend first (see main README.md)
./docker-build.sh up --build

# Run audio client (auto-discovers backend)
./chunk-sink-client
```

## Usage

1. **Start Backend**: Use Docker deployment from project root
2. **Run Client**: Execute `./chunk-sink-client` 
3. **Auto-Connect**: Client discovers and connects to backend via mDNS
4. **Record**: Audio automatically streams when signal detected
5. **Multiple Sources**: Run multiple clients for distributed recording

## Architecture

See the main project README.md for complete system design and data flow.
