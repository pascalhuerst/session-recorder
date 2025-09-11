# Recorder Display

A GUI application for displaying real-time recorder status updates on a Raspberry Pi 7" Touch LCD (800x480). The application receives status updates via gRPC and displays them in an intuitive touch-friendly interface.

## Overview

This Rust application provides a dual-function system:
1. **gRPC Server**: Receives recorder status updates and audio chunks from session-recorder clients
2. **GUI Display**: Shows real-time status information optimized for 7" touch screens

### Features

- **Touch-Optimized GUI**: Designed specifically for 800x480 resolution with large, touch-friendly elements
- **Real-time Updates**: Live display of recorder status including signal levels, RMS values, and clipping indicators
- **Multi-Recorder Support**: Displays status for multiple recorders simultaneously in a responsive card layout
- **Auto-Discovery**: gRPC server automatically announces itself via mDNS for service discovery
- **Connection Monitoring**: Visual indicators show when recorders haven't sent updates recently

## Hardware Requirements

- Raspberry Pi (3B+ or newer recommended)
- 7" Touch LCD Display (800x480 resolution)
- Network connection for receiving gRPC status updates

## Building

Make sure you have Rust installed, then build the project:

```bash
cargo build
```

For optimized performance on Raspberry Pi:

```bash
cargo build --release
```

## Running

### Quick Start

Use the provided run script:

```bash
./run.sh
```

### Manual Start

Start the recorder display application:

```bash
cargo run
```

By default, the gRPC server listens on `0.0.0.0:50051`. You can specify a different address:

```bash
cargo run -- --address 127.0.0.1:8080
```

## Usage

### Command Line Options

- `-a, --address <ADDRESS>`: Address to bind the gRPC server to (default: `0.0.0.0:50051`)
- `-h, --help`: Print help information

### GUI Interface

The GUI displays recorder status cards with the following information:

- **Recorder Name & ID**: Identifies each recorder
- **Connection Status**: Shows time since last update with color coding:
  - Green: Updated within 5 seconds
  - Yellow: Updated 5-10 seconds ago
  - Red: No updates for >10 seconds
- **Signal Status**: Visual indicator for signal presence:
  - 🟢 SIGNAL: Audio signal detected
  - 🔴 NO SIGNAL: No audio input
  - ⚪ UNKNOWN: Status unknown
- **RMS Level**: Real-time audio level with color-coded progress bar:
  - Green: 0-60%
  - Yellow: 60-80%
  - Red: >80%
- **Clipping Detection**: Shows if audio is clipping:
  - ✅ NO: No clipping detected
  - ⚠️ YES: Audio clipping detected

### Example Screenshots

```
┌─────────────────────────────────────────────────────┐
│          Session Recorder Status Display           │
├─────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐        │
│  │ Studio 1         │  │ Studio 2         │        │
│  │ ID: studio-1     │  │ ID: studio-2     │        │
│  │                  │  │                  │        │
│  │ Status: 🟢 SIGNAL│  │ Status: 🔴 NO SIG│        │
│  │ RMS: 75.2%       │  │ RMS: 2.1%        │        │
│  │ ████████░░ 75%   │  │ █░░░░░░░░░ 2%    │        │
│  │ Clipping: ✅ NO  │  │ Clipping: ✅ NO  │        │
│  └──────────────────┘  └──────────────────┘        │
└─────────────────────────────────────────────────────┘
```

## gRPC Service

The application implements the `ChunkSink` service with three methods:

### SetRecorderStatus

Receives recorder status updates including:
- Recorder ID and name
- Signal status (Unknown, No Signal, Signal)
- RMS percentage
- Clipping detection

Updates are immediately reflected in the GUI.

### SetChunks

Receives audio chunks from recorders and logs basic information.

### GetCommands

Provides a streaming interface for sending commands to recorders. Currently returns an empty stream but can be extended for future command functionality.

## Integration

This application is designed to work with the C++ chunk-sink-client and other session-recorder components.

### Service Discovery

The gRPC server announces itself via mDNS with:
- Service type: `_session-recorder-chunksink._tcp.local.`
- Instance name: `session-recorder-chunksink`
- Port: The port specified in the `--address` parameter

### Network Setup

Ensure your Raspberry Pi can receive network traffic on the configured port. For example, if using the default port 50051:

```bash
# Check if port is accessible
sudo netstat -tlnp | grep :50051
```

## Raspberry Pi Setup

### Display Configuration

For optimal performance on the 7" touch display, add to `/boot/config.txt`:

```ini
# Enable 7" touchscreen
dtoverlay=rpi-ft5406
lcd_rotate=2  # Rotate display if needed

# GPU memory split
gpu_mem=128

# Disable overscan for exact fit
disable_overscan=1
```

### Auto-Start on Boot

Create a systemd service to start the application on boot:

```ini
# /etc/systemd/system/recorder-display.service
[Unit]
Description=Session Recorder Display
After=graphical-session.target

[Service]
Type=simple
User=pi
Environment=DISPLAY=:0
WorkingDirectory=/home/pi/session-recorder/rust/recorder-display
ExecStart=/home/pi/session-recorder/rust/recorder-display/target/release/recorder-display
Restart=always
RestartSec=5

[Install]
WantedBy=graphical-session.target
```

Enable the service:

```bash
sudo systemctl enable recorder-display.service
sudo systemctl start recorder-display.service
```

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

### Dependencies

Key dependencies:
- `eframe`: GUI framework built on egui
- `egui`: Immediate mode GUI library
- `tonic`: gRPC implementation for Rust
- `tokio`: Async runtime
- `zeroconf`: mDNS service announcement

## Troubleshooting

### GUI Not Displaying

1. Check if X11 is running: `echo $DISPLAY`
2. Ensure proper permissions: `xhost +local:`
3. Try forcing X11 backend: `export WINIT_UNIX_BACKEND=x11`

### No Recorder Data

1. Verify network connectivity between recorders and display
2. Check gRPC server is listening: `netstat -tlnp | grep 50051`
3. Ensure mDNS is working: `avahi-browse -r _session-recorder-chunksink._tcp`

### Performance Issues

1. Use release build: `cargo build --release`
2. Increase GPU memory: Edit `gpu_mem` in `/boot/config.txt`
3. Close unnecessary applications

## Example Integration

To test with the Go test client:

1. Start the GUI application:
```bash
./run.sh
```

2. In another terminal, run a test client:
```bash
cd ../../go/cmd/chunk_sink_client
go run main.go --host YOUR_RPI_IP_ADDRESS
```

3. The GUI should display real-time status updates from the test client.

## Performance Notes

- The GUI updates at ~10 FPS to balance responsiveness with CPU usage
- Stale recorder entries are automatically removed after 30 seconds
- The interface is optimized for touch interaction with large tap targets
- Memory usage is kept low through efficient state management

## License

This project is part of the session-recorder ecosystem. See the main project license for details.