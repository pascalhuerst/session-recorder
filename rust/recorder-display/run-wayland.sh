#!/bin/bash

# Minimal Wayland Runner for Session Recorder Display
# This script runs the GUI application directly on Wayland compositor without desktop environment

set -e

echo "Starting Session Recorder Display with Minimal Wayland"
echo "Optimized for 7\" Touch LCD (800x480) - Minimal Wayland Setup"
echo ""

# Change to the directory containing this script
cd "$(dirname "$0")"

# Build the application if needed
if [ ! -f "target/release/recorder-display" ] || [ "src/main.rs" -nt "target/release/recorder-display" ]; then
    echo "Building application in release mode..."
    cargo build --release
    echo ""
fi

# Check if we need to install wayland compositor
if ! command -v weston >/dev/null 2>&1 && ! command -v cage >/dev/null 2>&1; then
    echo "Installing minimal Wayland compositor..."
    sudo apt update
    sudo apt install -y weston cage
    echo ""
fi

# Stop any existing display servers
echo "Stopping existing display servers..."
sudo systemctl stop lightdm 2>/dev/null || true
sudo systemctl stop gdm 2>/dev/null || true
pkill -f "weston\|cage\|X" 2>/dev/null || true
sleep 2

# Set up environment for Wayland
export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
export XDG_SESSION_TYPE=wayland
export WAYLAND_DISPLAY=wayland-0
export QT_QPA_PLATFORM=wayland
export GDK_BACKEND=wayland
export WINIT_UNIX_BACKEND=wayland
export RUST_LOG=info

# Create runtime directory
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

# Set up display permissions
sudo usermod -a -G video,input,render "$USER" 2>/dev/null || true

# Choose compositor based on preference and availability
COMPOSITOR=""
if command -v cage >/dev/null 2>&1; then
    COMPOSITOR="cage"
    echo "Using cage compositor (kiosk mode)"
elif command -v weston >/dev/null 2>&1; then
    COMPOSITOR="weston"
    echo "Using weston compositor"
else
    echo "Error: No suitable Wayland compositor found"
    echo "Please install cage or weston:"
    echo "  sudo apt install cage"
    exit 1
fi

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up Wayland session..."

    # Kill the recorder application
    pkill -f "recorder-display" 2>/dev/null || true

    # Kill compositor
    pkill -f "$COMPOSITOR" 2>/dev/null || true

    # Clean up runtime directory
    rm -rf "$XDG_RUNTIME_DIR" 2>/dev/null || true

    # Restore console
    sudo chvt 1 2>/dev/null || true

    echo "Cleanup complete"
    exit 0
}

# Set trap for cleanup on exit
trap cleanup INT TERM EXIT

echo "Starting $COMPOSITOR compositor..."

if [ "$COMPOSITOR" = "cage" ]; then
    # Cage is perfect for kiosk mode - it runs a single fullscreen application
    echo "Launching application in kiosk mode..."
    echo "Press Ctrl+Alt+F1 to return to console, or Ctrl+C to exit"
    echo ""

    # Switch to virtual terminal 2 for clean display
    sudo chvt 2 2>/dev/null || true
    sleep 1

    # Run cage with our application
    cage -d -- ./target/release/recorder-display "$@" &
    COMPOSITOR_PID=$!

elif [ "$COMPOSITOR" = "weston" ]; then
    # Create weston configuration for kiosk mode
    cat > "$XDG_RUNTIME_DIR/weston.ini" << EOF
[core]
shell=kiosk-shell.so
require-input=false
backend=drm-backend.so

[shell]
background-image=/dev/null
background-type=centered
background-color=0xff000000
panel-color=0x00000000
locking=false
animation=none

[output]
name=DSI-1
mode=800x480
scale=1
transform=normal

[output]
name=HDMI-A-1
mode=800x480
scale=1
transform=normal

[libinput]
enable-tap=true
tap-and-drag=true
tap-and-drag-lock=false
natural-scroll=false
middle-emulation=false
left-handed=false
cursor-size=24

[keyboard]
keymap_rules=evdev
keymap_layout=us
keymap_variant=
keymap_options=

[terminal]
font=monospace
font-size=14

[launcher]
icon=/usr/share/pixmaps/weston.png
path=/usr/bin/weston-terminal

[screensaver]
path=/usr/libexec/weston-screensaver
duration=600
EOF

    echo "Starting weston compositor..."
    echo "Press Ctrl+Alt+F1 to return to console"
    echo ""

    # Switch to virtual terminal 2
    sudo chvt 2 2>/dev/null || true
    sleep 1

    # Start weston
    weston --config="$XDG_RUNTIME_DIR/weston.ini" --tty=2 --idle-time=0 &
    COMPOSITOR_PID=$!

    # Wait for weston to start
    sleep 3

    # Launch our application
    echo "Launching recorder display application..."
    ./target/release/recorder-display "$@" &
    APP_PID=$!
fi

# Wait for processes to finish
if [ "$COMPOSITOR" = "cage" ]; then
    wait $COMPOSITOR_PID
elif [ "$COMPOSITOR" = "weston" ]; then
    wait $APP_PID
fi

echo "Application exited"
