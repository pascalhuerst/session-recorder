# Raspberry Pi Setup Guide for Session Recorder Display

This guide explains how to run the Session Recorder Display GUI directly on a Raspberry Pi 7" touch LCD (800x480) without a full desktop environment.

## Hardware Requirements

- Raspberry Pi 3B+ or newer (4B recommended)
- 7" Touch LCD Display (800x480 resolution)
- MicroSD card (16GB+ recommended)
- Network connection (WiFi or Ethernet)

## Software Installation

### 1. Base System Setup

Start with a fresh Raspberry Pi OS Lite installation:

```bash
# Update the system
sudo apt update
sudo apt upgrade -y

# Install minimal X11 components
sudo apt install -y \
    xorg-dev \
    xserver-xorg \
    xinit \
    x11-xserver-utils \
    libxrandr-dev \
    libxcursor-dev \
    libxi-dev \
    libxinerama-dev

# Install development tools
sudo apt install -y \
    build-essential \
    pkg-config \
    libssl-dev \
    git
```

### 2. Install Rust

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Verify installation
rustc --version
cargo --version
```

### 3. Display Configuration

Edit `/boot/config.txt`:

```bash
sudo nano /boot/config.txt
```

Add these lines for the 7" touchscreen:

```ini
# Enable 7" touchscreen
dtoverlay=rpi-ft5406

# Set display rotation if needed (0, 90, 180, 270)
lcd_rotate=0

# GPU memory allocation
gpu_mem=128

# Disable overscan for exact fit
disable_overscan=1

# Force 800x480 resolution
hdmi_group=2
hdmi_mode=87
hdmi_cvt=800 480 60 6 0 0 0

# Disable rainbow splash
disable_splash=1

# Reduce boot messages
quiet
```

### 4. Audio Configuration (if using audio feedback)

```bash
# Enable audio
sudo nano /boot/config.txt
```

Add:
```ini
dtparam=audio=on
```

### 5. Touch Calibration

Install touchscreen calibration tools:

```bash
sudo apt install -y xinput-calibrator

# Run calibration (after X11 is running)
xinput_calibrator
```

## Running Methods

### Method 1: Direct X11 (Recommended)

This is the simplest and most reliable method:

```bash
# Clone the project
git clone https://github.com/your-repo/session-recorder.git
cd session-recorder/rust/recorder-display

# Build the application
cargo build --release

# Use the direct runner script
./run-direct.sh --test-mode
```

#### What the direct runner does:
- Starts a minimal X11 server on the framebuffer
- Disables screen blanking and power management
- Sets optimal display settings
- Runs the application in fullscreen mode
- Handles cleanup on exit

### Method 2: Manual X11 Setup

If you prefer manual control:

```bash
# Start X server manually
sudo systemctl stop lightdm  # Stop any existing display manager
startx /usr/bin/true -- :0 -nolisten tcp -s 0 -dpms -nocursor vt1 &

# Wait for X to start
sleep 3

# Configure display
export DISPLAY=:0
xset s off
xset -dpms
xset s noblank

# Run the application
./target/release/recorder-display --test-mode
```

### Method 3: Auto-Start on Boot

Create a systemd service for automatic startup:

```bash
sudo nano /etc/systemd/system/recorder-display.service
```

```ini
[Unit]
Description=Session Recorder Display
After=multi-user.target
Wants=multi-user.target

[Service]
Type=simple
User=pi
Group=pi
Environment=HOME=/home/pi
Environment=XDG_RUNTIME_DIR=/tmp/runtime-pi
WorkingDirectory=/home/pi/session-recorder/rust/recorder-display
ExecStartPre=/bin/bash -c 'mkdir -p /tmp/runtime-pi && chown pi:pi /tmp/runtime-pi'
ExecStart=/home/pi/session-recorder/rust/recorder-display/run-direct.sh
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable recorder-display.service
sudo systemctl start recorder-display.service

# Check status
sudo systemctl status recorder-display.service
```

## Performance Optimization

### CPU Governor
Set performance mode for consistent frame rates:

```bash
# Temporary
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Permanent - add to /boot/config.txt
sudo nano /boot/config.txt
```

Add:
```ini
# Set CPU frequency
force_turbo=1
arm_freq=1400  # Adjust based on your Pi model
```

### GPU Memory
Increase GPU memory split in `/boot/config.txt`:

```ini
gpu_mem=128  # or 256 for Pi 4
```

### Disable Unnecessary Services

```bash
# Disable services you don't need
sudo systemctl disable bluetooth
sudo systemctl disable hciuart
sudo systemctl disable wifi-country
sudo systemctl disable triggerhappy
```

## Troubleshooting

### Common Issues

#### 1. Display Not Working
```bash
# Check framebuffer devices
ls -la /dev/fb*

# Check display info
fbset -fb /dev/fb0

# Test display
cat /dev/urandom > /dev/fb0
```

#### 2. Touch Not Working
```bash
# Check input devices
ls /dev/input/
cat /proc/bus/input/devices

# Test touch events
sudo evtest /dev/input/event0  # Adjust event number
```

#### 3. Application Won't Start
```bash
# Check X11 is running
ps aux | grep X

# Check display variable
echo $DISPLAY

# Check permissions
ls -la /tmp/.X11-unix/

# Check logs
journalctl -u recorder-display.service -f
```

#### 4. Performance Issues
```bash
# Check CPU usage
htop

# Check GPU usage (Pi 4)
vcgencmd measure_temp
vcgencmd get_mem gpu

# Monitor frame rate
# The application logs frame timing information
```

### Debug Mode

Run with verbose logging:

```bash
RUST_LOG=debug ./target/release/recorder-display --test-mode
```

## Network Configuration

### Static IP (Optional)
Set a static IP for reliable network access:

```bash
sudo nano /etc/dhcpcd.conf
```

Add:
```
interface eth0
static ip_address=192.168.1.100/24
static routers=192.168.1.1
static domain_name_servers=192.168.1.1 8.8.8.8
```

### Firewall
Allow gRPC traffic:

```bash
sudo ufw allow 50051/tcp
sudo ufw enable
```

## Monitoring

### System Health
Create a simple monitoring script:

```bash
#!/bin/bash
# monitor.sh
while true; do
    echo "$(date): CPU: $(vcgencmd measure_temp) GPU: $(vcgencmd get_mem gpu)"
    sleep 60
done
```

### Application Health
The application logs connection status and frame rates. Monitor with:

```bash
journalctl -u recorder-display.service -f | grep -E "(Frame|Connection|Error)"
```

## Backup and Recovery

### Create System Image
Once configured, create a backup:

```bash
# On your computer
sudo dd if=/dev/sdX of=recorder-display-backup.img bs=4M status=progress
```

### Configuration Backup
Backup important configuration files:

```bash
tar -czf recorder-config-backup.tar.gz \
    /boot/config.txt \
    /etc/systemd/system/recorder-display.service \
    ~/.cargo \
    ~/session-recorder
```

## Security Considerations

### Disable SSH Password Auth (Optional)
```bash
sudo nano /etc/ssh/sshd_config
```

Set:
```
PasswordAuthentication no
PubkeyAuthentication yes
```

### Update Strategy
```bash
# Create update script
#!/bin/bash
cd ~/session-recorder
git pull
cd rust/recorder-display
cargo build --release
sudo systemctl restart recorder-display.service
```

## Advanced Configuration

### Custom Touch Gestures
The application supports basic touch input. For advanced gestures, you can modify the input handling in `src/ui.rs`.

### Multiple Displays
For multi-display setups, modify the X11 configuration:

```bash
# Create xorg.conf
sudo nano /etc/X11/xorg.conf
```

### Custom Themes
Modify colors and fonts in `src/ui.rs`:

```rust
// Change theme colors
ui.visuals_mut().override_text_color = Some(egui::Color32::WHITE);
ui.visuals_mut().window_fill = egui::Color32::from_gray(30);
```

This setup provides a robust, kiosk-like experience perfect for monitoring recording sessions in a professional environment.