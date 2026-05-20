#!/bin/bash

# Auto-detecting Session Recorder Display Runner
# This script automatically detects the best display backend and runs the application

set -e

echo "Session Recorder Display - Auto Runner"
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

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if a service is running
service_running() {
    pgrep -f "$1" >/dev/null 2>&1
}

# Function to detect display backend
detect_display_backend() {
    local backend=""
    local method=""

    # Check for existing Wayland session
    if [ -n "$WAYLAND_DISPLAY" ] && [ -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ]; then
        backend="wayland-existing"
        method="Existing Wayland session detected"

    # Check for existing X11 session
    elif [ -n "$DISPLAY" ] && xset q >/dev/null 2>&1; then
        backend="x11-existing"
        method="Existing X11 session detected"

    # Check if we can start minimal Wayland (preferred)
    elif command_exists cage; then
        backend="wayland-cage"
        method="Cage compositor available"

    elif command_exists weston; then
        backend="wayland-weston"
        method="Weston compositor available"

    # Check if we can start minimal X11
    elif command_exists startx && command_exists X; then
        backend="x11-minimal"
        method="X11 server available"

    # Fallback to trying Wayland anyway
    else
        backend="wayland-fallback"
        method="No display server detected, trying Wayland"
    fi

    echo "Display Backend: $backend"
    echo "Detection Method: $method"
    echo ""

    echo "$backend"
}

# Function to run with existing Wayland
run_wayland_existing() {
    echo "Using existing Wayland session"
    export WINIT_UNIX_BACKEND=wayland
    exec ./target/release/recorder-display "$@"
}

# Function to run with existing X11
run_x11_existing() {
    echo "Using existing X11 session"
    export WINIT_UNIX_BACKEND=x11

    # Optimize X11 for embedded use
    xset s off 2>/dev/null || true
    xset -dpms 2>/dev/null || true
    xset s noblank 2>/dev/null || true

    exec ./target/release/recorder-display "$@"
}

# Function to run with minimal Wayland (cage)
run_wayland_cage() {
    echo "Starting minimal Wayland session with Cage..."

    # Set up environment
    export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
    export WAYLAND_DISPLAY=wayland-0
    export WINIT_UNIX_BACKEND=wayland
    export RUST_LOG=info

    # Create runtime directory
    mkdir -p "$XDG_RUNTIME_DIR"
    chmod 700 "$XDG_RUNTIME_DIR"

    # Stop conflicting display servers
    sudo systemctl stop lightdm 2>/dev/null || true
    sudo systemctl stop gdm 2>/dev/null || true
    pkill -f "X\|weston" 2>/dev/null || true
    sleep 1

    # Cleanup function
    cleanup() {
        echo "Cleaning up Cage session..."
        pkill -f "cage\|recorder-display" 2>/dev/null || true
        rm -rf "$XDG_RUNTIME_DIR" 2>/dev/null || true
        sudo chvt 1 2>/dev/null || true
        exit 0
    }
    trap cleanup INT TERM EXIT

    # Switch to clean VT and run
    sudo chvt 2 2>/dev/null || true
    exec cage -d -- ./target/release/recorder-display "$@"
}

# Function to run with minimal Wayland (weston)
run_wayland_weston() {
    echo "Starting minimal Wayland session with Weston..."

    # Set up environment
    export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
    export WAYLAND_DISPLAY=wayland-0
    export WINIT_UNIX_BACKEND=wayland

    # Create runtime directory
    mkdir -p "$XDG_RUNTIME_DIR"
    chmod 700 "$XDG_RUNTIME_DIR"

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

    # Stop conflicting display servers
    sudo systemctl stop lightdm 2>/dev/null || true
    sudo systemctl stop gdm 2>/dev/null || true
    pkill -f "X\|cage" 2>/dev/null || true
    sleep 1

    # Cleanup function
    cleanup() {
        echo "Cleaning up Weston session..."
        pkill -f "weston\|recorder-display" 2>/dev/null || true
        rm -rf "$XDG_RUNTIME_DIR" 2>/dev/null || true
        sudo chvt 1 2>/dev/null || true
        exit 0
    }
    trap cleanup INT TERM EXIT

    # Start weston and wait for it
    sudo chvt 2 2>/dev/null || true
    weston --config="$XDG_RUNTIME_DIR/weston.ini" --tty=2 --idle-time=0 &
    sleep 3

    # Run application
    exec ./target/release/recorder-display "$@"
}

# Function to run with minimal X11
run_x11_minimal() {
    echo "Starting minimal X11 session..."

    export DISPLAY=:0
    export WINIT_UNIX_BACKEND=x11

    # Stop conflicting display servers
    sudo systemctl stop lightdm 2>/dev/null || true
    sudo systemctl stop gdm 2>/dev/null || true
    pkill -f "X.*:0\|cage\|weston" 2>/dev/null || true
    sleep 1

    # Cleanup function
    cleanup() {
        echo "Cleaning up X11 session..."
        pkill -f "recorder-display" 2>/dev/null || true
        pkill -f "X.*:0" 2>/dev/null || true
        sudo chvt 1 2>/dev/null || true
        exit 0
    }
    trap cleanup INT TERM EXIT

    # Start minimal X server
    startx /usr/bin/true -- :0 -nolisten tcp -s 0 -dpms -nocursor vt2 &
    sleep 3

    # Configure X11
    xset s off 2>/dev/null || true
    xset -dpms 2>/dev/null || true
    xset s noblank 2>/dev/null || true
    xrandr --output HDMI-1 --mode 800x480 2>/dev/null || true
    xrandr --output DSI-1 --mode 800x480 2>/dev/null || true

    # Run application
    exec ./target/release/recorder-display "$@"
}

# Function to run with Wayland fallback
run_wayland_fallback() {
    echo "Attempting Wayland fallback..."
    export WINIT_UNIX_BACKEND=wayland
    export XDG_RUNTIME_DIR="/tmp/runtime-$USER"
    export WAYLAND_DISPLAY=wayland-0

    mkdir -p "$XDG_RUNTIME_DIR"
    chmod 700 "$XDG_RUNTIME_DIR"

    exec ./target/release/recorder-display "$@"
}

# Main execution
echo "Detecting optimal display backend..."
BACKEND=$(detect_display_backend)

echo "Starting application with backend: $BACKEND"
echo "Arguments: $@"
echo ""
echo "Press Ctrl+C to exit"
echo ""

case "$BACKEND" in
    "wayland-existing")
        run_wayland_existing "$@"
        ;;
    "x11-existing")
        run_x11_existing "$@"
        ;;
    "wayland-cage")
        run_wayland_cage "$@"
        ;;
    "wayland-weston")
        run_wayland_weston "$@"
        ;;
    "x11-minimal")
        run_x11_minimal "$@"
        ;;
    "wayland-fallback")
        run_wayland_fallback "$@"
        ;;
    *)
        echo "Error: Unknown backend detected: $BACKEND"
        echo ""
        echo "Available manual options:"
        echo "  ./run-wayland.sh    - Force Wayland with cage/weston"
        echo "  ./run-direct.sh     - Force minimal X11"
        echo "  ./run.sh           - Use existing display server"
        exit 1
        ;;
esac
