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
