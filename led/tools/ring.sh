#!/usr/bin/env bash
# ring.sh — drive the WS2812 ring controller over I2C from Linux.
#
# Usage:
#   ring.sh off
#   ring.sh solid   <r> <g> <b> [--brightness N]
#   ring.sh cylon   <r> <g> <b> [--speed N] [--length N] [--dir cw|ccw] [--brightness N]
#   ring.sh rainbow             [--speed N] [--dir cw|ccw] [--brightness N]
#   ring.sh breathe <r> <g> <b> [--speed N] [--brightness N]
#   ring.sh meter               [--green N] [--red N] [--decay N] [--dir cw|ccw] [--brightness N]
#   ring.sh level   <percent>   # push one meter sample (0..100); decays on its own
#   ring.sh get                 # dump readable registers for the selected channel
#
# Stereo: select the channel with CH (0 or 1). It just shifts the register
# addresses by CH*0x20.  e.g.   CH=1 ring.sh meter --green 60 --red 85
#
# Env overrides:  BUS (default 1), ADDR (default 0x20), CH (default 0)
set -euo pipefail

BUS="${BUS:-1}"
ADDR="${ADDR:-0x20}"
CH="${CH:-0}"
CHOFF=$(( CH * 0x20 ))   # channel block base (CHANNEL_STRIDE = 0x20)

# per-channel register offsets (must match src/registers.h)
REG_MODE=0x00 REG_BRIGHTNESS=0x01 REG_R=0x02 REG_G=0x03 REG_B=0x04
REG_SPEED=0x05 REG_LENGTH=0x06 REG_DIRECTION=0x07
REG_METER_LEVEL=0x08 REG_METER_GREEN=0x09 REG_METER_RED=0x0a REG_METER_DECAY=0x0b
# global (read-only) registers — not channel-offset
REG_NUM_LEDS=0x70 REG_NUM_CHANNELS=0x71 REG_VERSION=0x7f
MODE_OFF=0 MODE_SOLID=1 MODE_CYLON=2 MODE_RAINBOW=3 MODE_BREATHE=4 MODE_METER=5

addr()      { printf '0x%02x' $(( $1 + CHOFF )); }              # per-channel address
set_reg()   { i2cset -y "$BUS" "$ADDR" "$(addr "$1")" "$2"; }
set_color() { i2cset -y "$BUS" "$ADDR" "$(addr $REG_R)" "$1" "$2" "$3" i; } # block write R,G,B
get_reg()   { i2cget -y "$BUS" "$ADDR" "$(addr "$1")"; }        # per-channel read
get_glob()  { i2cget -y "$BUS" "$ADDR" "$1"; }                  # global read

# parse optional --flags after the positional args
parse_flags() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --speed)      set_reg "$REG_SPEED" "$2"; shift 2 ;;
            --length)     set_reg "$REG_LENGTH" "$2"; shift 2 ;;
            --brightness) set_reg "$REG_BRIGHTNESS" "$2"; shift 2 ;;
            --green)      set_reg "$REG_METER_GREEN" "$2"; shift 2 ;;
            --red)        set_reg "$REG_METER_RED" "$2"; shift 2 ;;
            --decay)      set_reg "$REG_METER_DECAY" "$2"; shift 2 ;;
            --dir)        [ "$2" = ccw ] && set_reg "$REG_DIRECTION" 1 || set_reg "$REG_DIRECTION" 0; shift 2 ;;
            *) echo "unknown flag: $1" >&2; exit 1 ;;
        esac
    done
}

cmd="${1:-}"; shift || true
case "$cmd" in
    off)     set_reg "$REG_MODE" "$MODE_OFF" ;;
    solid)   set_color "$1" "$2" "$3"; shift 3; set_reg "$REG_MODE" "$MODE_SOLID";   parse_flags "$@" ;;
    cylon)   set_color "$1" "$2" "$3"; shift 3; set_reg "$REG_MODE" "$MODE_CYLON";   parse_flags "$@" ;;
    breathe) set_color "$1" "$2" "$3"; shift 3; set_reg "$REG_MODE" "$MODE_BREATHE"; parse_flags "$@" ;;
    rainbow) set_reg "$REG_MODE" "$MODE_RAINBOW"; parse_flags "$@" ;;
    meter)   set_reg "$REG_MODE" "$MODE_METER"; parse_flags "$@" ;;
    level)   set_reg "$REG_METER_LEVEL" "$1" ;;
    get)
        printf 'ch=%s mode=%s brightness=%s rgb=%s,%s,%s speed=%s length=%s dir=%s | num_leds=%s num_channels=%s version=%s\n' \
            "$CH" \
            "$(get_reg "$REG_MODE")" \
            "$(get_reg "$REG_BRIGHTNESS")" \
            "$(get_reg "$REG_R")" \
            "$(get_reg "$REG_G")" \
            "$(get_reg "$REG_B")" \
            "$(get_reg "$REG_SPEED")" \
            "$(get_reg "$REG_LENGTH")" \
            "$(get_reg "$REG_DIRECTION")" \
            "$(get_glob "$REG_NUM_LEDS")" \
            "$(get_glob "$REG_NUM_CHANNELS")" \
            "$(get_glob "$REG_VERSION")" ;;
    *) sed -n '2,24p' "$0"; exit 1 ;;
esac
