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

### Build Audio Client

```bash
./build-audio-client.sh
```

### Run Audio Client

**Option 1: Using Docker (Recommended)**
```bash
# Run with default settings
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client

# Run with custom parameters
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client \
  --recorder-id my-recorder \
  --recorder-name "My Audio Recorder" \
  --device pipewire \
  --rate 48000 \
  --channels 2
```

**Option 2: Native Binary (requires system dependencies)**
```bash
chmod +x ./cpp/chunk-sink-client/chunk-sink-client
./cpp/chunk-sink-client/chunk-sink-client
```

### Command Line Options

```
Generic:
  --help                             Print help message
  --recorder-id arg                  Unique ID of this recorder
  --recorder-name arg                Name of this recorder

Audio:
  --device arg (=pipewire)           ALSA device to read from
  --rate arg (=48000)                Sample rate
  --channels arg (=2)                Channels
  --latency arg (=1024)              Input latency in ms
  --format arg (=S16_LE)             Sample format

Detector:
  --rest-time arg (=5)               Time signal needs to rest before status change
  --window-time arg (=1)             Time window to analyze input signal
  --detector-threshold arg (=0.001)  RMS threshold to detect silence

Led:
  --led-detector arg                 LED for detector state
  --led-indexer arg                  LED for indexer state
```
