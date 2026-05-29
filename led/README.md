# WS2812 Controller (I2C slave)

An Arduino Uno (ATmega328P) acts as an I2C slave that drives a WS2812 / NeoPixel
ring. A Linux host (e.g. Raspberry Pi) configures animations by writing 8-bit
values into an 8-bit **register file** — like a normal I2C peripheral. No
reflashing needed to change color, speed, effect, etc.

## Wiring

| Uno pin | Connects to |
|---------|-------------|
| `D6`    | ring **0** (e.g. left) WS2812 `DIN` (add a ~330 Ω series resistor if you can) |
| `D7`    | ring **1** (e.g. right) WS2812 `DIN` |
| `5V`    | WS2812 `+5V` (small rings only — otherwise power the rings from a separate 5V supply) |
| `GND`   | both WS2812 `GND` **and** master `GND` (common ground is mandatory) |
| `A4` (SDA) | I2C SDA (via level shifter, see below) |
| `A5` (SCL) | I2C SCL (via level shifter, see below) |

Both rings share the same I2C address and 5V/GND; only the data pin differs.

> ⚠️ **Level shifting.** A Raspberry Pi (and most SBCs) use **3.3 V** I2C; the Uno
> is **5 V**. Connecting SDA/SCL directly risks back-feeding 5 V into the Pi. Use a
> bidirectional I2C level shifter (BSS138 type). The firmware disables the Uno's
> internal pull-ups, so the shifter board / master must provide pull-ups
> (the Pi already has 1.8 kΩ pull-ups to 3.3 V on its I2C bus).
>
> If the master is itself 5 V, you can wire SDA/SCL directly.

> ⚠️ **Power.** A WS2812 can draw ~60 mA at full white. More than ~5–6 LEDs at full
> brightness exceeds what's safe through the Uno's regulator — power the ring from a
> dedicated 5 V supply and just share ground.

## Build & flash

Requires [PlatformIO](https://platformio.org/install/cli):

```bash
pip install platformio          # or: pipx install platformio
cd led
pio run -t upload               # compile + flash the Uno
pio device monitor              # 115200 baud, prints address & LED count
```

Hardware config lives in `platformio.ini` (`build_flags`): `LED_PIN`, `LED_PIN2`,
`NUM_LEDS`, `I2C_ADDRESS`. Change there and re-flash.

## Register map

The master writes a register address byte first, then one or more value bytes;
the register pointer auto-increments, so a multi-byte write fills consecutive
registers. See `src/registers.h` for the authoritative definition.

**Stereo.** The per-channel registers below are a *block* repeated once per ring.
Channel 0 lives at base `0x00`, channel 1 at base `0x20` (`CHANNEL_STRIDE`). The
absolute address is `channel*0x20 + offset` — so channel 1's `MODE` is `0x20`,
its `COLOR_R` is `0x22`, etc. The table lists per-channel **offsets**; the
`NUM_LEDS`/`NUM_CHANNELS`/`VERSION` registers are global (absolute addresses).

Per-channel registers (offset within a channel block):

| Off  | Name         | R/W | Meaning |
|------|--------------|-----|---------|
| 0x00 | `MODE`        | RW  | 0=off, 1=solid, 2=cylon/nightrider, 3=rainbow, 4=breathe, 5=meter |
| 0x01 | `BRIGHTNESS`  | RW  | channel brightness 0..255 |
| 0x02 | `COLOR_R`     | RW  | primary color red |
| 0x03 | `COLOR_G`     | RW  | primary color green |
| 0x04 | `COLOR_B`     | RW  | primary color blue |
| 0x05 | `SPEED`       | RW  | 0=slow .. 255=fast |
| 0x06 | `LENGTH`      | RW  | comet/tail length in pixels (1..NUM_LEDS) |
| 0x07 | `DIRECTION`   | RW  | 0=clockwise, non-zero=counter-clockwise (also flips meter direction) |
| 0x08 | `METER_LEVEL` | RW  | peak meter: write a sample 0..100%; consumed each frame |
| 0x09 | `METER_GREEN` | RW  | meter: pixels up to this percent are green |
| 0x0A | `METER_RED`   | RW  | meter: pixels at/above this percent are red (between = amber) |
| 0x0B | `METER_DECAY` | RW  | meter: fall rate in percent-per-frame (frame ≈ 15 ms) |

Global registers (absolute address):

| Addr | Name           | R/W | Meaning |
|------|----------------|-----|---------|
| 0x70 | `NUM_LEDS`     | R   | compiled-in pixel count (per channel) |
| 0x71 | `NUM_CHANNELS` | R   | number of channel blocks (2) |
| 0x7F | `VERSION`      | R   | firmware version |

## Driving it from Linux

Using `i2c-tools` (bus 1 is typical on a Pi; check with `i2cdetect -y 1`):

```bash
# Nightrider: cyan comet, length 5, fast, clockwise
i2cset -y 1 0x20 0x00 0x02            # MODE = cylon
i2cset -y 1 0x20 0x02 0x00 0xC0 0xFF i # COLOR_R/G/B in one block write
i2cset -y 1 0x20 0x05 0xC0            # SPEED
i2cset -y 1 0x20 0x06 0x05            # LENGTH
i2cset -y 1 0x20 0x07 0x00            # DIRECTION = CW

# Solid warm white at low brightness
i2cset -y 1 0x20 0x00 0x01            # MODE = solid
i2cset -y 1 0x20 0x01 0x20            # BRIGHTNESS

# Read back (global registers)
i2cget -y 1 0x20 0x70                 # NUM_LEDS
i2cget -y 1 0x20 0x71                 # NUM_CHANNELS
i2cget -y 1 0x20 0x7f                 # VERSION

# Channel 1 (right): same registers, offset by 0x20
i2cset -y 1 0x20 0x20 0x05            # ch1 MODE = meter
i2cset -y 1 0x20 0x28 0x40            # ch1 METER_LEVEL = 64%
```

### Peak level meter

A bargraph that snaps up to each sample and decays on its own. You **write a
percentage** repeatedly; pixels are colored by position — green up to
`METER_GREEN`, amber up to `METER_RED`, red beyond.

```bash
i2cset -y 1 0x20 0x00 0x05            # MODE = meter
i2cset -y 1 0x20 0x09 0x3c            # METER_GREEN = 60%
i2cset -y 1 0x20 0x0a 0x55            # METER_RED   = 85%
i2cset -y 1 0x20 0x0b 0x04            # METER_DECAY = 4%/frame
while :; do
  i2cset -y 1 0x20 0x08 "$level"      # push current level (0..100) as fast as you like
done
```

A single write rises instantly then falls; stop writing and it decays to zero.
Drive `METER_LEVEL` continuously (e.g. from your audio RMS/peak) for a live meter.

A convenience wrapper is in [`tools/ring.sh`](tools/ring.sh):

```bash
tools/ring.sh cylon 0 192 255 --speed 200 --length 5 --dir cw
tools/ring.sh solid 255 64 0
tools/ring.sh meter --green 60 --red 85 --decay 4
tools/ring.sh level 73                 # push one sample; repeat from your meter source
tools/ring.sh off

# Stereo: pick the channel with CH (0 or 1)
CH=0 tools/ring.sh meter --green 60 --red 85    # left
CH=1 tools/ring.sh meter --green 60 --red 85    # right
CH=1 tools/ring.sh level 80                       # push right-channel sample
```
