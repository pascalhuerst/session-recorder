// registers.h — I2C register map for the WS2812 ring controller.
//
// The device behaves like a typical I2C peripheral: the master first writes a
// 1-byte register address (the "pointer"), then any further bytes in the same
// transaction are written to consecutive registers (the pointer auto-increments).
// Reads return the register at the current pointer, also auto-incrementing.
//
// STEREO: the per-channel registers below are a *block* that repeats once per
// LED output. Channel 0 lives at base 0x00, channel 1 at base CHANNEL_STRIDE,
// etc. The absolute address of a register is  channel*CHANNEL_STRIDE + offset.
//
//   set channel 0 mode = meter:   i2cset -y 1 0x20 0x00 0x05
//   set channel 1 mode = meter:   i2cset -y 1 0x20 0x20 0x05   (0x20 = 1*stride)
//   read firmware version:        i2cget -y 1 0x20 0x7F
//
// All registers are 8-bit. Writes to read-only registers are ignored.

#pragma once
#include <stdint.h>

// Per-channel register offsets (relative to the channel's block base).
enum ChannelRegister : uint8_t {
    REG_MODE       = 0x00, // animation selector, see Mode enum below
    REG_BRIGHTNESS = 0x01, // channel brightness 0..255
    REG_COLOR_R    = 0x02, // primary color, red
    REG_COLOR_G    = 0x03, // primary color, green
    REG_COLOR_B    = 0x04, // primary color, blue
    REG_SPEED      = 0x05, // animation speed 0=slow .. 255=fast
    REG_LENGTH     = 0x06, // tail/comet length in pixels (1..NUM_LEDS)
    REG_DIRECTION  = 0x07, // 0 = clockwise, non-zero = counter-clockwise

    // Peak level meter (MODE_METER). Two independent inputs feed the display:
    //  * REG_METER_LEVEL drives the BAR — fast attack, REG_METER_DECAY fall rate.
    //  * REG_METER_PEAK  drives a single PEAK-HOLD indicator pixel painted on
    //    top of the bar — fast attack, slow REG_METER_PEAK_DECAY fall rate.
    // Both inputs are CONSUMED each frame (reset to 0 by the firmware), so the
    // master is expected to write them continuously (e.g. RMS into LEVEL and
    // max-|sample| into PEAK over the same short analysis window).
    REG_METER_LEVEL      = 0x08, // bar input  (0..100 %, write each frame)
    REG_METER_PEAK       = 0x09, // peak input (0..100 %, write each frame)
    REG_METER_GREEN      = 0x0A, // pixels up to this percent are GREEN
    REG_METER_RED        = 0x0B, // pixels at/above this percent are RED (between = amber)
    REG_METER_DECAY      = 0x0C, // bar fall rate in percent-per-frame
    REG_METER_PEAK_DECAY = 0x0D, // peak-hold fall rate in percent-per-frame
};

static const uint8_t CHANNEL_STRIDE = 0x20; // bytes reserved per channel block
static const uint8_t NUM_CHANNELS   = 2;     // stereo: ch0 @0x00, ch1 @0x20

// Global, read-only registers (shared across channels), placed above the
// channel blocks (channels occupy 0x00..0x3F for NUM_CHANNELS=2).
enum GlobalRegister : uint8_t {
    REG_NUM_LEDS     = 0x70, // compiled-in pixel count (per channel)
    REG_NUM_CHANNELS = 0x71, // number of channel blocks
    REG_VERSION      = 0x7F, // firmware version
};

static const uint8_t REG_COUNT = 0x80; // size of the register file (0x00..0x7F)

// Values for REG_MODE.
enum Mode : uint8_t {
    MODE_OFF     = 0, // all pixels dark
    MODE_SOLID   = 1, // whole ring = primary color
    MODE_CYLON   = 2, // "nightrider": comet with fading tail running around the ring
    MODE_RAINBOW = 3, // rotating rainbow (ignores color)
    MODE_BREATHE = 4, // primary color pulsing in brightness
    MODE_METER   = 5, // peak level meter (bargraph, green/amber/red zones)
    MODE_COUNT
};

static const uint8_t FIRMWARE_VERSION = 4;
