#!/bin/bash

# Simple Wayland Runner with Sudo Workaround
# This script provides an immediate fix for libseat permission issues

set -e

echo "Session Recorder Display - Wayland (Sudo Workaround)"
echo "Optimized for 7\" Touch LCD (800x480)"
echo ""

# Change to the directory containing this script
cd "$(dirname "$0")"

# Build the application if needed
if [ ! -f "target/release/recorder-display" ] || [ "src/main.rs" -nt "target/release/recorder-display" ]; then
    echo "Building application in release mode..."
    cargo build --release
    echo ""
fi

# Check if we're already running as root
if [ "$EUID" -eq 0 ]; then
    echo "Running as root - setting up environment..."

    # Set up environment for root
    export XDG_RUNTIME_DIR="/tmp/runtime-root"
    export WAYLAND_DISPLAY=wayland-0
    export WINIT_UNIX_BACKEND=wayland
    export RUST_LOG=info

    # Create runtime directory
    mkdir -p "$XDG_RUNTIME_DIR"
    chmod 700 "$XDG_RUNTIME_DIR"

    # Run directly
    exec_app() {
        if command -v cage >/dev/null 2>&1; then
            echo "Using cage compositor (kiosk mode)"
            cage -d -- ./target/release/recorder-display "$@"
        elif command -v weston >/dev/null 2>&1; then
            echo "Using weston compositor"
            # Create minimal weston config
            cat > "$XDG_RUNTIME_DIR/weston.ini" << EOF
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

[output]
name=HDMI-A-1
mode=800x480

[libinput]
enable-tap=true
EOF
            weston --config="$XDG_RUNTIME_DIR/weston.ini" --idle-time=0 &
            sleep 3
            ./target/release/recorder-display "$@"
        else
            echo "No Wayland compositor found!"
            exit 1
        fi
    }

    exec_app "$@"
    exit 0
fi

# If not root, check for sudo and re-run
if ! command -v sudo >/dev/null 2>&1; then
    echo "Error: sudo not available and libseat permissions not configured"
    echo "Please run: ./fix-libseat.sh"
    exit 1
fi

echo "LibSeat permission issue detected. Using sudo workaround..."
echo ""
echo "Note: For a permanent fix without sudo, run: ./fix-libseat.sh"
echo ""

# Stop conflicting display servers
echo "Stopping conflicting display servers..."
sudo systemctl stop lightdm 2>/dev/null || true
sudo systemctl stop gdm 2>/dev/null || true
sudo pkill -f "X\|weston\|cage" 2>/dev/null || true
sleep 2

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    sudo pkill -f "recorder-display\|cage\|weston" 2>/dev/null || true
    sudo chvt 1 2>/dev/null || true
    echo "Cleanup complete"
    exit 0
}

# Set trap for cleanup on exit
trap cleanup INT TERM EXIT

echo "Starting Wayland compositor with sudo..."
echo "Press Ctrl+C to exit"
echo ""

# Switch to clean VT
sudo chvt 2 2>/dev/null || true
sleep 1

# Re-run this script as root
sudo -E "$0" "$@"
