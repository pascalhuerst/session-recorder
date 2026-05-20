#!/bin/bash

# Direct Framebuffer Runner for Session Recorder Display
# This script runs the GUI application directly on the framebuffer without a window manager

set -e

echo "Starting Session Recorder Display in Direct Mode"
echo "Optimized for 7\" Touch LCD (800x480) - No Window Manager Required"
echo ""

# Change to the directory containing this script
cd "$(dirname "$0")"

# Build the application if needed
if [ ! -f "target/release/recorder-display" ] || [ "src/main.rs" -nt "target/release/recorder-display" ]; then
    echo "Building application in release mode..."
    cargo build --release
    echo ""
fi

# Check if we're running as root (needed for framebuffer access)
if [ "$EUID" -ne 0 ]; then
    echo "Note: You may need to run as root for direct framebuffer access"
    echo "Trying with current user first..."
fi

# Stop existing X11 sessions if any (optional)
if pgrep -x "X" > /dev/null; then
    echo "Warning: X11 is running. For best performance, consider stopping it:"
    echo "  sudo systemctl stop lightdm"
    echo "  sudo systemctl stop gdm"
    echo "  sudo pkill X"
    echo ""
fi

# Set console to graphics mode
if [ -w /dev/tty1 ]; then
    echo "Setting console to graphics mode..."
    echo 0 > /sys/class/vtconsole/vtcon1/bind 2>/dev/null || true
fi

# Set environment variables for direct rendering
export DISPLAY=:0
export XDG_RUNTIME_DIR=/tmp/runtime-$USER
export WAYLAND_DISPLAY=""
export WINIT_UNIX_BACKEND=x11

# Create minimal X11 session
echo "Starting minimal X11 server..."

# Kill existing X servers on display :0
pkill -f "X.*:0" 2>/dev/null || true
sleep 1

# Start X server in background
startx /usr/bin/true -- :0 -nolisten tcp -s 0 -dpms -nocursor vt1 &
X_PID=$!

# Wait for X to start
sleep 3

# Check if X started successfully
if ! pgrep -x "X" > /dev/null; then
    echo "Failed to start X server. Trying alternative method..."

    # Try xinit directly
    xinit /usr/bin/true -- /usr/bin/X :0 -nolisten tcp -s 0 -dpms -nocursor vt1 &
    X_PID=$!
    sleep 3
fi

# Disable screen blanking
xset s off 2>/dev/null || true
xset -dpms 2>/dev/null || true
xset s noblank 2>/dev/null || true

# Set display resolution if needed
xrandr --output HDMI-1 --mode 800x480 2>/dev/null || true
xrandr --output DSI-1 --mode 800x480 2>/dev/null || true

echo "X11 server started (PID: $X_PID)"
echo "Launching recorder display application..."
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."

    # Kill the recorder application
    pkill -f "recorder-display" 2>/dev/null || true

    # Kill X server
    if [ ! -z "$X_PID" ]; then
        kill $X_PID 2>/dev/null || true
    fi
    pkill -f "X.*:0" 2>/dev/null || true

    # Restore console
    echo 1 > /sys/class/vtconsole/vtcon1/bind 2>/dev/null || true

    echo "Cleanup complete"
    exit 0
}

# Set trap for cleanup on exit
trap cleanup INT TERM EXIT

# Run the application
echo "Application starting..."
echo "Press Ctrl+C to exit"
echo ""

# Run with nice priority for smooth performance
nice -n -10 ./target/release/recorder-display "$@" &
APP_PID=$!

# Wait for application to finish
wait $APP_PID

echo "Application exited"
