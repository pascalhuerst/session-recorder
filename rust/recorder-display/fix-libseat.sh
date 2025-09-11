#!/bin/bash

# LibSeat Permission Fix Script
# This script fixes common libseat permission issues on Raspberry Pi for Wayland

set -e

echo "=========================================="
echo "LibSeat Permission Fix for Wayland"
echo "=========================================="
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "Error: Please run this script as a regular user (not root)"
    echo "The script will use sudo when needed"
    exit 1
fi

echo "Current user: $USER"
echo "Current groups: $(groups)"
echo ""

# Method 1: Add user to seat group and configure libseat
echo "Method 1: Configuring user permissions..."

# Add user to required groups
echo "Adding user to required groups..."
sudo usermod -a -G video,input,render,audio,tty,dialout $USER

# Create seat group if it doesn't exist
if ! getent group seat >/dev/null; then
    echo "Creating seat group..."
    sudo groupadd seat
fi
sudo usermod -a -G seat $USER

# Method 2: Configure seatd service
echo ""
echo "Method 2: Setting up seatd service..."

# Install seatd if not present
if ! command -v seatd >/dev/null 2>&1; then
    echo "Installing seatd..."
    sudo apt update
    sudo apt install -y seatd
fi

# Create seatd configuration
sudo tee /etc/seatd.conf >/dev/null << EOF
# Seatd configuration for recorder-display
# Allow users in seat group to access seat
group=seat
EOF

# Create seatd systemd service override
sudo mkdir -p /etc/systemd/system/seatd.service.d
sudo tee /etc/systemd/system/seatd.service.d/override.conf >/dev/null << EOF
[Service]
# Run seatd with proper permissions
User=root
Group=seat
ExecStart=
ExecStart=/usr/bin/seatd -g seat
EOF

# Enable and start seatd
echo "Enabling seatd service..."
sudo systemctl daemon-reload
sudo systemctl enable seatd.service
sudo systemctl restart seatd.service

# Method 3: Configure udev rules for device permissions
echo ""
echo "Method 3: Setting up udev rules..."

sudo tee /etc/udev/rules.d/99-seat-permissions.rules >/dev/null << 'EOF'
# Seat and input device permissions
# TTY devices
KERNEL=="tty[0-9]*", GROUP="tty", MODE="0664"
SUBSYSTEM=="tty", GROUP="tty", MODE="0664"

# Input devices
SUBSYSTEM=="input", GROUP="input", MODE="0664"
KERNEL=="event*", SUBSYSTEM=="input", GROUP="input", MODE="0664"
KERNEL=="mouse*", SUBSYSTEM=="input", GROUP="input", MODE="0664"

# DRI devices
SUBSYSTEM=="drm", GROUP="video", MODE="0664"
KERNEL=="card*", GROUP="video", MODE="0664"
KERNEL=="controlD*", GROUP="video", MODE="0664"

# Framebuffer devices
KERNEL=="fb*", GROUP="video", MODE="0664"

# Grant seat permissions to seat group
TAG=="seat", GROUP="seat", MODE="0664"
TAG=="uaccess", GROUP="seat", MODE="0664"
EOF

# Reload udev rules
sudo udevadm control --reload-rules
sudo udevadm trigger

# Method 4: Alternative - Use logind session (if available)
echo ""
echo "Method 4: Checking for systemd-logind..."

if systemctl is-active --quiet systemd-logind; then
    echo "systemd-logind is running - good for seat management"

    # Configure logind for auto-login (optional)
    sudo mkdir -p /etc/systemd/system/getty@tty1.service.d
    sudo tee /etc/systemd/system/getty@tty1.service.d/autologin.conf >/dev/null << EOF
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin $USER --noclear %I \$TERM
EOF

    echo "Configured auto-login for $USER on tty1"
else
    echo "systemd-logind not running - relying on seatd"
fi

# Method 5: Create runtime script with proper permissions
echo ""
echo "Method 5: Creating permission-aware runner..."

cat > ~/run-wayland-fixed.sh << 'EOF'
#!/bin/bash

# Wayland runner with libseat permission fixes

set -e

echo "Starting Wayland with libseat fixes..."

# Set environment
export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
export XDG_SESSION_TYPE=wayland
export WAYLAND_DISPLAY=wayland-0
export WINIT_UNIX_BACKEND=wayland

# Create runtime directory
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

# Check seatd is running
if ! systemctl is-active --quiet seatd; then
    echo "Starting seatd service..."
    sudo systemctl start seatd
    sleep 2
fi

# Try different seat backends in order of preference
run_with_seat() {
    local compositor="$1"
    shift

    echo "Attempting to run $compositor..."

    # Try with different libseat backends
    for backend in seatd logind; do
        echo "Trying libseat backend: $backend"
        export LIBSEAT_BACKEND=$backend

        if timeout 5 $compositor "$@" 2>/dev/null; then
            return 0
        fi

        # If that fails, try with sudo (not recommended but works)
        echo "Trying with elevated permissions..."
        if sudo -E $compositor "$@" 2>/dev/null; then
            return 0
        fi
    done

    return 1
}

# Change to recorder-display directory
cd "$(dirname "$0")"

# Build if needed
if [ ! -f "target/release/recorder-display" ]; then
    cargo build --release
fi

# Try cage first
if command -v cage >/dev/null 2>&1; then
    if run_with_seat cage -- ./target/release/recorder-display "$@"; then
        exit 0
    fi
fi

# Try weston as fallback
if command -v weston >/dev/null 2>&1; then
    cat > "$XDG_RUNTIME_DIR/weston.ini" << EOW
[core]
shell=kiosk-shell.so
require-input=false

[shell]
background-color=0xff000000
locking=false
animation=none

[output]
name=DSI-1
mode=800x480

[libinput]
enable-tap=true
EOW

    echo "Starting weston..."
    if run_with_seat weston --config="$XDG_RUNTIME_DIR/weston.ini"; then
        sleep 3
        ./target/release/recorder-display "$@" &
        wait
        exit 0
    fi
fi

echo "All Wayland attempts failed. Trying X11 fallback..."
export WINIT_UNIX_BACKEND=x11
export DISPLAY=:0

# Try to start minimal X11
if command -v startx >/dev/null 2>&1; then
    startx /usr/bin/true -- :0 -nolisten tcp &
    sleep 3
    ./target/release/recorder-display "$@"
else
    echo "No display server could be started"
    exit 1
fi
EOF

chmod +x ~/run-wayland-fixed.sh

# Method 6: Set up proper session management
echo ""
echo "Method 6: Session management setup..."

# Create session script
sudo tee /usr/local/bin/start-recorder-session >/dev/null << EOF
#!/bin/bash
# Start recorder display session with proper seat management

export XDG_RUNTIME_DIR="/run/user/\$(id -u $USER)"
export XDG_SESSION_TYPE=wayland
export WAYLAND_DISPLAY=wayland-0

# Ensure runtime directory exists
mkdir -p "\$XDG_RUNTIME_DIR"
chown $USER:$USER "\$XDG_RUNTIME_DIR"
chmod 700 "\$XDG_RUNTIME_DIR"

# Switch to user and run
cd /home/$USER/session-recorder/rust/recorder-display
sudo -u $USER -g seat /home/$USER/run-wayland-fixed.sh "\$@"
EOF

sudo chmod +x /usr/local/bin/start-recorder-session

# Set proper permissions on device files
echo ""
echo "Setting device permissions..."

# Make sure seat has access to required devices
for device in /dev/tty* /dev/input/event* /dev/dri/card* /dev/fb*; do
    if [ -e "$device" ]; then
        sudo chgrp seat "$device" 2>/dev/null || true
        sudo chmod g+rw "$device" 2>/dev/null || true
    fi
done

echo ""
echo "=========================================="
echo "LibSeat Fix Complete!"
echo "=========================================="
echo ""
echo "Applied fixes:"
echo "1. ✓ Added user to required groups (video, input, render, tty, seat)"
echo "2. ✓ Configured seatd service"
echo "3. ✓ Set up udev rules for device permissions"
echo "4. ✓ Configured logind (if available)"
echo "5. ✓ Created permission-aware runner script"
echo "6. ✓ Set up session management"
echo ""
echo "Next steps:"
echo ""
echo "1. Log out and back in (or reboot) for group changes:"
echo "   sudo reboot"
echo ""
echo "2. After reboot, test with the fixed runner:"
echo "   ~/run-wayland-fixed.sh --test-mode"
echo ""
echo "3. Alternative: Use the system session starter:"
echo "   sudo /usr/local/bin/start-recorder-session --test-mode"
echo ""
echo "4. Check seatd status:"
echo "   systemctl status seatd"
echo ""
echo "5. Debug if needed:"
echo "   journalctl -u seatd -f"
echo ""
echo "Troubleshooting:"
echo "- If still having issues, try: sudo loginctl show-session"
echo "- Check device permissions: ls -la /dev/tty* /dev/input/event*"
echo "- Verify groups: groups $USER"
echo ""
echo "The script created ~/run-wayland-fixed.sh with multiple fallback methods"
echo "This should resolve the libseat permission issues!"
