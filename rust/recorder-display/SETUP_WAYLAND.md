# Wayland Setup Guide for Session Recorder Display

This guide explains how to run the Session Recorder Display GUI with a minimal Wayland setup on Raspberry Pi 7" touch LCD (800x480).

## Why Wayland?

Wayland offers several advantages over X11 for embedded systems:
- **Better Performance**: Direct hardware access, no legacy overhead
- **Lower Resource Usage**: More efficient memory and CPU utilization
- **Better Security**: Isolated applications, no global input access
- **Modern Architecture**: Designed for current hardware capabilities
- **Touch-First Design**: Native multi-touch support

## Prerequisites

- Raspberry Pi 4B (recommended) or 3B+
- 7" Touch LCD Display (800x480)
- Raspberry Pi OS (64-bit recommended)
- Network connection

## Installation

### 1. System Preparation

```bash
# Update system
sudo apt update
sudo apt upgrade -y

# Install Wayland components
sudo apt install -y \
    libwayland-dev \
    libwayland-client0 \
    libwayland-server0 \
    libwayland-cursor0 \
    libwayland-egl1 \
    libxkbcommon-dev \
    libxkbcommon0 \
    weston \
    cage

# Install development tools
sudo apt install -y \
    build-essential \
    pkg-config \
    libssl-dev \
    git \
    curl
```

### 2. GPU and Display Configuration

Edit `/boot/firmware/config.txt` (or `/boot/config.txt` on older systems):

```ini
# Enable GPU
dtoverlay=vc4-kms-v3d
gpu_mem=128

# 7" touchscreen support
dtoverlay=rpi-ft5406
lcd_rotate=0

# Force 800x480 resolution
hdmi_group=2
hdmi_mode=87
hdmi_cvt=800 480 60 6 0 0 0

# Disable overscan
disable_overscan=1

# Reduce boot messages
quiet
logo.nologo=1
```

### 3. User Permissions

Add your user to the necessary groups:

```bash
sudo usermod -a -G video,input,render,audio $USER
```

Log out and back in for group changes to take effect.

## Compositor Options

### Option 1: Cage (Recommended for Kiosk Mode)

Cage is a minimal Wayland compositor that runs a single application in fullscreen:

```bash
# Install cage
sudo apt install cage

# Run the recorder display
./run-wayland.sh --test-mode
```

**Advantages:**
- Minimal resource usage
- Perfect for kiosk applications
- No desktop environment needed
- Automatic fullscreen mode

### Option 2: Weston (More Features)

Weston is the reference Wayland compositor with more features:

```bash
# Install weston
sudo apt install weston

# Run with weston
COMPOSITOR=weston ./run-wayland.sh --test-mode
```

**Advantages:**
- More configuration options
- Better debugging tools
- Support for multiple applications
- More mature codebase

## Running the Application

### Quick Start

```bash
# Clone the repository
git clone <your-repo-url>
cd session-recorder/rust/recorder-display

# Install Rust (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Run with test data
./run-wayland.sh --test-mode

# Run for production (waiting for real recorder data)
./run-wayland.sh
```

### Manual Wayland Session

If you prefer manual control:

```bash
# Set environment
export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
export WAYLAND_DISPLAY=wayland-0
export WINIT_UNIX_BACKEND=wayland

# Create runtime directory
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

# Start compositor and application
cage -- ./target/release/recorder-display --test-mode
```

## Auto-Start Configuration

### Systemd Service

Create `/etc/systemd/system/recorder-display.service`:

```ini
[Unit]
Description=Session Recorder Display (Wayland)
After=multi-user.target
Wants=multi-user.target

[Service]
Type=simple
User=pi
Group=pi
Environment=HOME=/home/pi
Environment=XDG_RUNTIME_DIR=/tmp/runtime-pi
Environment=WAYLAND_DISPLAY=wayland-0
Environment=WINIT_UNIX_BACKEND=wayland
WorkingDirectory=/home/pi/session-recorder/rust/recorder-display
ExecStartPre=/bin/bash -c 'mkdir -p /tmp/runtime-pi && chown pi:pi /tmp/runtime-pi && chmod 700 /tmp/runtime-pi'
ExecStart=/home/pi/session-recorder/rust/recorder-display/run-wayland.sh
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable recorder-display.service
sudo systemctl start recorder-display.service

# Check status
sudo systemctl status recorder-display.service

# View logs
journalctl -u recorder-display.service -f
```

### Boot Configuration

To start automatically without login, add to `/etc/rc.local` (before `exit 0`):

```bash
# Start recorder display as pi user
sudo -u pi /home/pi/session-recorder/rust/recorder-display/run-wayland.sh &
```

## Touch Configuration

### Calibration

The 7" touchscreen usually works out of the box, but you can calibrate if needed:

```bash
# Install calibration tool
sudo apt install xinput-calibrator

# For Wayland, you might need to use libinput tools
sudo apt install libinput-tools

# List input devices
libinput list-devices

# Test touch input
sudo libinput debug-events
```

### Touch Gestures

The application supports basic touch input. Modify `/etc/libinput/local-overrides.quirks` for custom touch behavior:

```
[Raspberry Pi Touch]
MatchUdevType=touchscreen
MatchName=*Raspberry Pi*
AttrTouchSizeRange=10:8
AttrPalmSizeThreshold=40
```

## Performance Optimization

### 1. GPU Acceleration

Ensure hardware acceleration is working:

```bash
# Check GPU info
vcgencmd get_mem gpu
vcgencmd measure_temp

# Verify DRM/KMS
ls -la /dev/dri/

# Test OpenGL ES
sudo apt install mesa-utils-extra
es2_info
```

### 2. CPU Performance

```bash
# Set performance governor
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Make permanent by adding to /boot/config.txt
echo "force_turbo=1" | sudo tee -a /boot/firmware/config.txt
```

### 3. Memory Optimization

```bash
# Increase GPU memory in /boot/firmware/config.txt
gpu_mem=256  # For Pi 4 with 4GB+ RAM

# Disable unnecessary services
sudo systemctl disable bluetooth
sudo systemctl disable wifi-country
sudo systemctl disable triggerhappy
```

## Troubleshooting

### Common Issues

#### 1. Application Won't Start

```bash
# Check Wayland compositor
ps aux | grep -E 'cage|weston'

# Check runtime directory
ls -la $XDG_RUNTIME_DIR

# Verify permissions
groups $USER

# Check logs
journalctl -xe | grep -i wayland
```

#### 2. No Display Output

```bash
# Check display connection
vcgencmd display_power

# Verify DRM
ls -la /dev/dri/
dmesg | grep drm

# Test basic display
cage -- weston-simple-egl
```

#### 3. Touch Not Working

```bash
# List input devices
cat /proc/bus/input/devices

# Test raw touch events
sudo evtest /dev/input/event0  # Adjust event number

# Check libinput
sudo libinput debug-events --device /dev/input/event0
```

#### 4. Performance Issues

```bash
# Monitor CPU usage
htop

# Check GPU temperature
vcgencmd measure_temp

# Monitor frame rate (application logs this)
journalctl -u recorder-display.service -f | grep -i frame

# Check memory usage
free -h
```

### Debug Mode

Run with verbose logging:

```bash
RUST_LOG=debug WAYLAND_DEBUG=1 ./run-wayland.sh --test-mode
```

### Alternative Compositors

If cage/weston don't work, try other minimal compositors:

```bash
# Install alternatives
sudo apt install labwc  # Lightweight Wayland compositor
sudo apt install river  # Tiling Wayland compositor
sudo apt install sway   # i3-compatible Wayland compositor (more complex)

# Test with labwc
labwc & 
sleep 2
./target/release/recorder-display --test-mode
```

## Network Configuration

### Firewall

Allow gRPC traffic:

```bash
sudo ufw allow 50051/tcp
sudo ufw --force enable
```

### mDNS

Enable service discovery:

```bash
sudo apt install avahi-daemon
sudo systemctl enable avahi-daemon
```

## Security Considerations

### User Isolation

Wayland provides better security than X11:
- Applications can't spy on each other
- No global keylogger access
- Secure clipboard handling

### Network Security

```bash
# Limit SSH access (optional)
sudo ufw limit ssh

# Disable unnecessary network services
sudo systemctl disable cups
sudo systemctl disable avahi-daemon  # If not needed for discovery
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup system configuration
sudo tar -czf wayland-config-backup.tar.gz \
    /boot/firmware/config.txt \
    /etc/systemd/system/recorder-display.service \
    ~/.cargo \
    ~/session-recorder
```

### System Image

Create a full system backup once configured:

```bash
# On your computer, create image of SD card
sudo dd if=/dev/sdX of=recorder-display-wayland.img bs=4M status=progress
```

## Advanced Configuration

### Custom Wayland Protocol

For advanced features, you can implement custom Wayland protocols. This requires modifying the application code and is beyond basic setup.

### Multiple Displays

Configure multiple displays by editing the Wayland compositor configuration and adjusting the application's display logic.

### Hardware Buttons

Map hardware buttons to application functions by configuring libinput and handling key events in the application.

## Performance Monitoring

### Real-time Monitoring

```bash
# Create monitoring script
cat > ~/monitor.sh << 'EOF'
#!/bin/bash
while true; do
    echo "$(date): $(vcgencmd measure_temp) | $(vcgencmd get_mem gpu) | $(uptime | cut -d',' -f3-)"
    sleep 30
done
EOF

chmod +x ~/monitor.sh
./monitor.sh
```

### Application Metrics

The application logs performance metrics. Monitor with:

```bash
journalctl -u recorder-display.service -f | grep -E "(Frame|FPS|Latency)"
```

This Wayland setup provides a modern, efficient foundation for your touch-based recording display with minimal overhead and maximum performance.