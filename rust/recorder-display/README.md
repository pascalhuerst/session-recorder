# Recorder Display

A modern GUI application for displaying real-time recorder status updates on a Raspberry Pi 7" Touch LCD (800x480). The application receives status updates via gRPC and displays them in an intuitive touch-friendly interface with full Wayland support for optimal performance.

## Overview

This Rust application provides a dual-function system:
1. **gRPC Server**: Receives recorder status updates and audio chunks from session-recorder clients
2. **GUI Display**: Shows real-time status information optimized for 7" touch screens

### Features

- **Modern Wayland Support**: Minimal compositor setup for optimal embedded performance
- **Touch-Optimized GUI**: Designed specifically for 800x480 resolution with large, touch-friendly elements
- **Real-time Updates**: Live display of recorder status including signal levels, RMS values, and clipping indicators
- **Multi-Recorder Support**: Displays status for multiple recorders simultaneously in a responsive card layout
- **Auto-Discovery**: gRPC server automatically announces itself via mDNS for service discovery
- **Connection Monitoring**: Visual indicators show when recorders haven't sent updates recently
- **Flexible Display Backends**: Supports Wayland (preferred), X11, and automatic detection

## Hardware Requirements

- Raspberry Pi (3B+ or newer recommended)
- 7" Touch LCD Display (800x480 resolution)
- Network connection for receiving gRPC status updates

## Quick Setup for Raspberry Pi

### Automated Wayland Installation

For the best performance on Raspberry Pi, use our automated Wayland setup:

```bash
# Run the installation script (one-time setup)
./install-wayland.sh

# Reboot to apply configuration
sudo reboot

# Run with test data
./run-wayland.sh --test-mode
```

### Manual Building

Make sure you have Rust installed, then build the project:

```bash
cargo build --release
```

## Running

### Quick Start Options

**Recommended: Auto-detecting runner (detects best backend)**
```bash
./run-auto.sh --test-mode
```

**Wayland (best performance for embedded)**
```bash
./run-wayland.sh --test-mode
```

**X11 (fallback option)**
```bash
./run-direct.sh --test-mode
```

**Use existing display server**
```bash
./run.sh --test-mode
```

### Manual Start

Start the recorder display application:

```bash
cargo run --release -- --test-mode
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

This application is designed to work with the recorder and other session-recorder components.

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

### Wayland Setup (Recommended)

For the best performance, use the automated Wayland setup:

```bash
./install-wayland.sh
sudo reboot
```

This script automatically configures:
- GPU acceleration with VC4/KMS
- 7" touchscreen support
- Optimal display settings
- Wayland compositors (Cage/Weston)
- User permissions
- Boot optimization

### Manual Display Configuration

For manual setup, add to `/boot/firmware/config.txt` (or `/boot/config.txt`):

```ini
# GPU acceleration (required for Wayland)
dtoverlay=vc4-kms-v3d
gpu_mem=128

# 7" touchscreen
dtoverlay=rpi-ft5406
lcd_rotate=0

# 800x480 resolution
hdmi_group=2
hdmi_mode=87
hdmi_cvt=800 480 60 6 0 0 0
disable_overscan=1

# Boot optimization
quiet
logo.nologo=1
disable_splash=1
```

### Auto-Start on Boot

The installation script creates a systemd service. To enable it:

```bash
# Enable auto-start
sudo systemctl enable recorder-display-wayland.service
sudo systemctl start recorder-display-wayland.service

# Check status
sudo systemctl status recorder-display-wayland.service

# View logs
journalctl -u recorder-display-wayland.service -f
```

Manual service creation:

```ini
# /etc/systemd/system/recorder-display.service
[Unit]
Description=Session Recorder Display (Wayland)
After=multi-user.target

[Service]
Type=simple
User=pi
Environment=XDG_RUNTIME_DIR=/tmp/runtime-pi
Environment=WAYLAND_DISPLAY=wayland-0
Environment=WINIT_UNIX_BACKEND=wayland
WorkingDirectory=/home/pi/session-recorder/rust/recorder-display
ExecStart=/home/pi/session-recorder/rust/recorder-display/run-wayland.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable the service:

```bash
sudo systemctl enable recorder-display.service
sudo systemctl start recorder-display.service
```

### Backend Selection

The application automatically detects the best display backend:

1. **Wayland with Cage** (preferred for kiosk mode)
2. **Wayland with Weston** (more features)
3. **X11 minimal** (fallback)

Force a specific backend:
```bash
# Force Wayland
WINIT_UNIX_BACKEND=wayland ./target/release/recorder-display

# Force X11
WINIT_UNIX_BACKEND=x11 ./target/release/recorder-display
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

### Display Issues

**Wayland not working:**
```bash
# Check DRM devices
ls -la /dev/dri/

# Check GPU memory
vcgencmd get_mem gpu

# Test with simple app
cage -- weston-simple-egl

# Check compositor availability
which cage weston
```

**X11 fallback:**
```bash
# Check X11
echo $DISPLAY
xset q

# Test X11 backend
WINIT_UNIX_BACKEND=x11 ./target/release/recorder-display --test-mode
```

### No Recorder Data

1. Verify network connectivity between recorders and display
2. Check gRPC server is listening: `netstat -tlnp | grep 50051`
3. Ensure mDNS is working: `avahi-browse -r _session-recorder-chunksink._tcp`

### Performance Issues

1. Use release build: `cargo build --release`
2. Ensure GPU acceleration: `vcgencmd get_mem gpu` should show 128MB+
3. Check compositor: Cage uses less resources than Weston
4. Monitor temperature: `vcgencmd measure_temp`
5. Check CPU governor: `cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor`

### Touch Issues

```bash
# List input devices
libinput list-devices

# Test touch events
sudo libinput debug-events

# Check permissions
ls -la /dev/input/
```

### Getting Help

```bash
# Check logs
journalctl -u recorder-display-wayland.service -f

# Debug mode
RUST_LOG=debug WAYLAND_DEBUG=1 ./run-wayland.sh --test-mode

# System info
./install-wayland.sh --check  # If you added a check option
```

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

## Documentation

- **[Wayland Setup Guide](SETUP_WAYLAND.md)** - Detailed Wayland configuration
- **[Raspberry Pi Setup](SETUP_RASPBERRY_PI.md)** - X11 and general Pi setup
- **[Running Options](run-auto.sh)** - Auto-detecting display backend

## Performance Notes

- **Wayland + Cage**: Lowest resource usage, perfect for kiosk mode
- **Wayland + Weston**: More features, slightly higher resource usage  
- **X11**: Fallback option, higher resource usage than Wayland
- **Frame Rate**: Targets 10 FPS for smooth updates with low CPU usage
- **Memory**: Efficient state management keeps usage under 50MB
- **GPU**: Hardware acceleration via VC4/KMS on Pi 4, legacy GPU on Pi 3

## License

This project is part of the session-recorder ecosystem. See the main project license for details.