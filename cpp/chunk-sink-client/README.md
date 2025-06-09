# Session Recorder Audio Client

C++ audio capture client that streams audio chunks to the Session Recorder backend via gRPC.

## Features

- **ALSA Audio Capture**: Records from ALSA audio devices
- **Automatic Discovery**: Finds backend services via mDNS (`_session-recorder-chunksink._tcp`)
- **Real-time Streaming**: Streams audio chunks over gRPC
- **Signal Detection**: Only records when audio signal is detected
- **Multiple Clients**: Supports multiple simultaneous recording sources

## Quick Start

### Build
From the project root directory:
```bash
./build-audio-client.sh
```

### Run

**Option 1: Using Docker (Recommended)**
```bash
# Start Docker backend first (see main README.md)
./docker-build.sh up --build

# Run audio client with default settings
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client

# Run with custom parameters
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client \
  --recorder-id my-studio \
  --recorder-name "Studio Microphone" \
  --device hw:0 \
  --rate 44100
```

**Option 2: Native Binary**
Requires system dependencies:
```bash
# Prerequisites (Fedora)
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel cmake boost-program-options

# Run binary directly
./chunk-sink-client/chunk-sink-client
```

## Usage

1. **Start Backend**: Use Docker deployment from project root
2. **Run Client**: Execute via Docker or native binary
3. **Auto-Connect**: Client discovers and connects to backend via mDNS
4. **Record**: Audio automatically streams when signal detected
5. **Multiple Sources**: Run multiple clients for distributed recording

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

### Examples

```bash
# Basic usage with default settings
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client

# Custom recorder with specific audio device
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client \
  --recorder-id bedroom-mic \
  --recorder-name "Bedroom Microphone" \
  --device hw:1

# High-quality recording setup
docker run --rm --device /dev/snd session-recorder-cpp ./chunk-sink-client \
  --recorder-id studio-main \
  --recorder-name "Studio Main Mic" \
  --rate 96000 \
  --channels 1 \
  --detector-threshold 0.0005
```

## Architecture

See the main project README.md for complete system design and data flow.
