# Recorder Display

A gRPC server that receives and logs recorder status updates from session-recorder chunk-sink-clients. The server automatically announces itself via mDNS for service discovery.

## Overview

This Rust application implements a ChunkSink gRPC server that:
- Automatically announces itself via mDNS as `_session-recorder-chunksink._tcp.local.`
- Receives recorder status updates (signal level, clipping detection, etc.)
- Receives audio chunks from recorders
- Provides a command stream interface for sending commands to recorders
- Logs all received data for monitoring and debugging
- Enables automatic discovery by C++ chunk-sink-clients

## Building

Make sure you have Rust installed, then build the project:

```bash
cargo build
```

## Running

Start the recorder display server:

```bash
cargo run
```

By default, the server listens on `0.0.0.0:50051`. You can specify a different address:

```bash
cargo run -- --address 127.0.0.1:8080
```

## Usage

### Command Line Options

- `-a, --address <ADDRESS>`: Address to bind the server to (default: `0.0.0.0:50051`)
- `-h, --help`: Print help information

### Example

```bash
# Start server on default port
cargo run

# Start server on custom address
cargo run -- --address 192.168.1.100:9090
```

## gRPC Service

The server implements the `ChunkSink` service with three methods:

### SetRecorderStatus

Receives and logs recorder status updates including:
- Recorder ID and name
- Signal status (Unknown, No Signal, Signal)
- RMS percentage
- Clipping detection

### SetChunks

Receives audio chunks from recorders and logs basic information about them.

### GetCommands

Provides a streaming interface for sending commands to recorders. Currently returns an empty stream but can be extended to send commands like:
- CutSession
- Reboot

## Integration

This server is designed to work with the C++ chunk-sink-client found in `../../cpp/chunk-sink-client/`. The C++ client will automatically discover this server via mDNS and connect to send status updates.

### Service Discovery

The server announces itself via mDNS with:
- Service type: `_session-recorder-chunksink._tcp.local.`
- Instance name: `session-recorder-chunksink`
- Port: The port specified in the `--address` parameter

Clients can discover the service using standard mDNS browsers or the service discovery mechanisms built into the session-recorder ecosystem.

## Development

### Testing

Run the tests:

```bash
cargo test
```

### Proto Files

The gRPC service definitions are in `../../protocols/proto/`:
- `chunksink.proto`: Main service definition
- `common.proto`: Common message types

## Example Output

When receiving status updates, the server logs:

```
Starting recorder display server on: 0.0.0.0:50051
Announced mDNS service: _session-recorder-chunksink._tcp.local. on port 50051
Starting ChunkSink gRPC server on 0.0.0.0:50051
=== Recorder Status Update ===
Recorder ID: studio-1
Recorder Name: Studio 1 Recorder
Signal Status: Signal
RMS Percent: 75.23%
Clipping: No
==============================
Received 1024 chunks from recorder: studio-1
```

## Integration Example

To test the complete integration with the Go chunk sink client:

1. Start the Rust recorder-display server:
```bash
cargo run -- --address 0.0.0.0:8779
```

2. In another terminal, run the Go test client:
```bash
cd ../../go/cmd/chunk_sink_client
go run main.go --host 127.0.0.1
```

3. You should see status updates and chunk data being logged in real-time:
```
=== Recorder Status Update ===
Recorder ID: 2bb00a6a-c468-41b0-b8b8-40e3cd22450e
Recorder Name: Test Recorder 1
Signal Status: Signal
RMS Percent: 0.50%
Clipping: No
==============================
Received 0 chunks from recorder: 2bb00a6a-c468-41b0-b8b8-40e3cd22450e
```

The C++ chunk-sink-client will automatically discover the server via mDNS and connect without any manual configuration.
