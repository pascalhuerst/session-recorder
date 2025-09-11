#!/bin/bash

# Wayland Installation Script for Raspberry Pi
# This script installs and configures Wayland components for the Session Recorder Display

set -e

echo "=========================================="
echo "Session Recorder Display - Wayland Setup"
echo "=========================================="
echo ""

# Check if running on Raspberry Pi
if ! grep -q "Raspberry Pi" /proc/cpuinfo && ! grep -q "BCM" /proc/cpuinfo; then
    echo "Warning: This script is optimized for Raspberry Pi"
    read -p "Continue anyway? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if running as non-root user
if [ "$EUID" -eq 0 ]; then
    echo "Error: Please run this script as a regular user (not root)"
    echo "The script will use sudo when needed"
    exit 1
fi

echo "Updating package lists..."
sudo apt update

echo ""
echo "Installing Wayland core components..."
sudo apt install -y \
    libwayland-dev \
    libwayland-client0 \
    libwayland-server0 \
    libwayland-cursor0 \
    libwayland-egl1 \
    libxkbcommon-dev \
    libxkbcommon0

echo ""
echo "Installing Wayland compositors..."
sudo apt install -y \
    weston \
    cage

echo ""
echo "Installing development tools..."
sudo apt install -y \
    build-essential \
    pkg-config \
    libssl-dev \
    git \
    curl

echo ""
echo "Installing graphics and input libraries..."
sudo apt install -y \
    libdrm-dev \
    libgbm-dev \
    libegl-dev \
    libgles2-mesa-dev \
    libinput-dev \
    libinput-tools \
    libudev-dev

echo ""
echo "Installing additional utilities..."
sudo apt install -y \
    mesa-utils \
    mesa-utils-extra \
    wayland-utils

# Check if Rust is installed
if ! command -v rustc >/dev/null 2>&1; then
    echo ""
    echo "Installing Rust..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source ~/.cargo/env
    echo "Rust installed successfully"
else
    echo ""
    echo "Rust is already installed ($(rustc --version))"
fi

# Add user to necessary groups
echo ""
echo "Adding user to required groups..."
sudo usermod -a -G video,input,render,audio $USER

# Configure boot settings
echo ""
echo "Configuring boot settings..."

BOOT_CONFIG="/boot/firmware/config.txt"
if [ ! -f "$BOOT_CONFIG" ]; then
    BOOT_CONFIG="/boot/config.txt"
fi

if [ -f "$BOOT_CONFIG" ]; then
    echo "Backing up $BOOT_CONFIG..."
    sudo cp "$BOOT_CONFIG" "${BOOT_CONFIG}.backup.$(date +%Y%m%d-%H%M%S)"

    # Check and add GPU settings
    if ! grep -q "dtoverlay=vc4-kms-v3d" "$BOOT_CONFIG"; then
        echo ""
        echo "Adding GPU acceleration settings..."
        sudo tee -a "$BOOT_CONFIG" >/dev/null << EOF

# GPU acceleration for Wayland (added by install-wayland.sh)
dtoverlay=vc4-kms-v3d
gpu_mem=128

EOF
    fi

    # Check and add touchscreen settings
    if ! grep -q "dtoverlay=rpi-ft5406" "$BOOT_CONFIG"; then
        echo "Adding touchscreen settings..."
        sudo tee -a "$BOOT_CONFIG" >/dev/null << EOF
# 7" touchscreen support (added by install-wayland.sh)
dtoverlay=rpi-ft5406
lcd_rotate=0

EOF
    fi

    # Check and add display settings
    if ! grep -q "hdmi_cvt=800 480" "$BOOT_CONFIG"; then
        echo "Adding display settings for 800x480..."
        sudo tee -a "$BOOT_CONFIG" >/dev/null << EOF
# Display settings for 800x480 (added by install-wayland.sh)
hdmi_group=2
hdmi_mode=87
hdmi_cvt=800 480 60 6 0 0 0
disable_overscan=1

EOF
    fi

    # Check and add boot optimization
    if ! grep -q "quiet" "$BOOT_CONFIG"; then
        echo "Adding boot optimization..."
        sudo tee -a "$BOOT_CONFIG" >/dev/null << EOF
# Boot optimization (added by install-wayland.sh)
quiet
logo.nologo=1
disable_splash=1

EOF
    fi
else
    echo "Warning: Boot config file not found at $BOOT_CONFIG"
fi

# Create udev rules for input devices
echo ""
echo "Creating udev rules for input devices..."
sudo tee /etc/udev/rules.d/99-recorder-display.rules >/dev/null << 'EOF'
# Input device permissions for recorder-display
SUBSYSTEM=="input", GROUP="input", MODE="0664"
KERNEL=="event*", SUBSYSTEM=="input", GROUP="input", MODE="0664"

# DRI device permissions
SUBSYSTEM=="drm", GROUP="video", MODE="0664"
KERNEL=="card*", SUBSYSTEM=="drm", GROUP="video", MODE="0664"

# Framebuffer permissions
KERNEL=="fb*", GROUP="video", MODE="0664"
EOF

# Create systemd service template
echo ""
echo "Creating systemd service template..."
sudo tee /etc/systemd/system/recorder-display-wayland.service >/dev/null << EOF
[Unit]
Description=Session Recorder Display (Wayland)
After=multi-user.target
Wants=multi-user.target
Documentation=man:weston(1) man:cage(1)

[Service]
Type=simple
User=$USER
Group=$USER
Environment=HOME=$HOME
Environment=XDG_RUNTIME_DIR=/tmp/runtime-$USER
Environment=WAYLAND_DISPLAY=wayland-0
Environment=WINIT_UNIX_BACKEND=wayland
Environment=RUST_LOG=info
WorkingDirectory=$HOME/session-recorder/rust/recorder-display
ExecStartPre=/bin/bash -c 'mkdir -p /tmp/runtime-$USER && chown $USER:$USER /tmp/runtime-$USER && chmod 700 /tmp/runtime-$USER'
ExecStart=$HOME/session-recorder/rust/recorder-display/run-wayland.sh
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
TimeoutStartSec=30
TimeoutStopSec=10

[Install]
WantedBy=multi-user.target
EOF

echo "Service template created at /etc/systemd/system/recorder-display-wayland.service"
echo "Note: You'll need to adjust paths after cloning the repository"

# Disable conflicting services
echo ""
echo "Disabling potentially conflicting services..."
sudo systemctl disable lightdm 2>/dev/null || echo "lightdm not found (OK)"
sudo systemctl disable gdm 2>/dev/null || echo "gdm not found (OK)"
sudo systemctl disable sddm 2>/dev/null || echo "sddm not found (OK)"

# Create runtime directory structure
echo ""
echo "Setting up runtime directories..."
sudo mkdir -p /tmp/runtime-$USER
sudo chown $USER:$USER /tmp/runtime-$USER
chmod 700 /tmp/runtime-$USER

# Test Wayland components
echo ""
echo "Testing Wayland installation..."

echo "Checking DRM devices..."
if ls /dev/dri/ >/dev/null 2>&1; then
    echo "✓ DRM devices found: $(ls /dev/dri/)"
else
    echo "✗ No DRM devices found - GPU acceleration may not work"
fi

echo "Checking input devices..."
if ls /dev/input/ >/dev/null 2>&1; then
    echo "✓ Input devices found: $(ls /dev/input/event* 2>/dev/null | wc -l) event devices"
else
    echo "✗ No input devices found"
fi

echo "Checking Wayland compositors..."
if command -v cage >/dev/null 2>&1; then
    echo "✓ Cage compositor: $(cage --version 2>&1 | head -n1 || echo 'installed')"
else
    echo "✗ Cage compositor not found"
fi

if command -v weston >/dev/null 2>&1; then
    echo "✓ Weston compositor: $(weston --version 2>&1 | head -n1 || echo 'installed')"
else
    echo "✗ Weston compositor not found"
fi

# Create test script
echo ""
echo "Creating test script..."
cat > ~/test-wayland.sh << 'EOF'
#!/bin/bash
# Quick Wayland test script

echo "Testing Wayland setup..."

export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
export WAYLAND_DISPLAY=wayland-0

mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

# Test with a simple wayland application
if command -v weston-simple-egl >/dev/null 2>&1; then
    echo "Starting Cage with test application..."
    echo "Press Ctrl+C to exit"
    cage -- weston-simple-egl
else
    echo "weston-simple-egl not found, testing with basic terminal"
    echo "Press Ctrl+C to exit"
    cage -- weston-terminal
fi
EOF

chmod +x ~/test-wayland.sh

echo ""
echo "=========================================="
echo "Installation Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Reboot to apply boot configuration changes:"
echo "   sudo reboot"
echo ""
echo "2. After reboot, test Wayland setup:"
echo "   ./test-wayland.sh"
echo ""
echo "3. Clone the session-recorder repository:"
echo "   git clone <your-repo-url> ~/session-recorder"
echo ""
echo "4. Build and run the recorder display:"
echo "   cd ~/session-recorder/rust/recorder-display"
echo "   ./run-wayland.sh --test-mode"
echo ""
echo "5. Enable auto-start (optional):"
echo "   sudo systemctl enable recorder-display-wayland.service"
echo "   sudo systemctl start recorder-display-wayland.service"
echo ""
echo "Troubleshooting:"
echo "- Check logs: journalctl -u recorder-display-wayland.service -f"
echo "- Manual test: WAYLAND_DEBUG=1 cage -- weston-simple-egl"
echo "- GPU check: vcgencmd get_mem gpu && vcgencmd measure_temp"
echo ""
echo "Note: You may need to log out and back in for group changes to take effect"

# Final system check
echo ""
echo "System information:"
echo "- OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "- Kernel: $(uname -r)"
echo "- GPU Memory: $(vcgencmd get_mem gpu 2>/dev/null || echo 'Unable to check')"
echo "- Temperature: $(vcgencmd measure_temp 2>/dev/null || echo 'Unable to check')"

echo ""
echo "Installation script completed successfully!"
