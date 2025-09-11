#!/bin/bash

# Session Recorder Display Runner
# This script runs the GUI application for displaying recorder statuses

set -e

echo "Starting Session Recorder Display..."
echo "Optimized for 7\" Touch LCD (800x480)"
echo ""

# Change to the directory containing this script
cd "$(dirname "$0")"

# Build the application if needed
if [ ! -f "target/debug/recorder-display" ] || [ "src/main.rs" -nt "target/debug/recorder-display" ]; then
    echo "Building application..."
    cargo build
    echo ""
fi

# Set environment variables for better performance on embedded systems
export RUST_LOG=info

# Auto-detect and prefer Wayland if available, fallback to X11
if [ -n "$WAYLAND_DISPLAY" ] || [ -n "$XDG_SESSION_TYPE" ] && [ "$XDG_SESSION_TYPE" = "wayland" ]; then
    echo "Wayland detected, using Wayland backend"
    export WINIT_UNIX_BACKEND=wayland
elif [ -n "$DISPLAY" ]; then
    echo "X11 detected, using X11 backend"
    export WINIT_UNIX_BACKEND=x11
else
    echo "No display server detected, trying Wayland first..."
    export WINIT_UNIX_BACKEND=wayland
fi

# Run the application
echo "Launching GUI..."
echo "gRPC server will listen on 0.0.0.0:50051"
echo "Backend: $WINIT_UNIX_BACKEND"
echo ""
echo "Options:"
echo "  --test-mode    Run with simulated recorder data for testing"
echo "  --address      Specify gRPC server address (default: 0.0.0.0:50051)"
echo ""
echo "Example: ./run.sh --test-mode"
echo "For Wayland: ./run-wayland.sh --test-mode"
echo ""
echo "Press Ctrl+C to exit"
echo ""

./target/debug/recorder-display "$@"
