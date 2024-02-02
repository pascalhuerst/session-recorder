# nubbock

  > Nubbock (n.): The kind of person who has to leave before a party can relax and enjoy itself.

Simple Wayland compositor, written in Qt.

# Description

This is a very basic Wayland compositor, implemented in Qt using QtCompositor.

There is no window management but basic input handling.

The code is based on the examples provided by Qt.

# Configuration

This program is configurable through some environment variables.

## Background image

If `NUBBOCK_BACKGROUND_IMAGE` is present, the image it points to will be displayed at coordinates `0, 0`. No scaling or tiling is done.

## Accelerometer

If `NUBBOCK_ACCELEROMETER_DEV` is present, the input device node it is pointing will be opened. Incoming events will be parsed to detect two positions of the device, standing and laying. The Wayland output is then rotated accordingly.



