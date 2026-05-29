// WS2812 ring controller — I2C slave, stereo (two LED outputs).
//
// Linux (or any I2C master) writes 8-bit values into an 8-bit register file
// (see registers.h). The per-channel registers form a block that repeats once
// per LED output: channel 0 at base 0x00, channel 1 at base CHANNEL_STRIDE.
// Each channel renders its own animation from its own block, so the two rings
// are fully independent (e.g. left/right peak meters).
//
// Concurrency: the register file is written from the TWI (Wire) interrupt and
// read from loop(). On the 8-bit AVR, single-byte loads/stores are atomic, so
// reading individual registers needs no locking. A multi-byte value (e.g. RGB)
// could in theory tear across a frame, which is cosmetically irrelevant here.

#include <Arduino.h>
#include <Wire.h>
#include <FastLED.h>
#include "registers.h"

#ifndef LED_PIN
#define LED_PIN 6       // channel 0 (e.g. left) WS2812 DIN
#endif
#ifndef LED_PIN2
#define LED_PIN2 7      // channel 1 (e.g. right) WS2812 DIN
#endif
#ifndef NUM_LEDS
#define NUM_LEDS 15
#endif
#ifndef I2C_ADDRESS
#define I2C_ADDRESS 0x20
#endif

static CRGB leds[NUM_CHANNELS][NUM_LEDS];

// The register file. `volatile` because it is touched from the TWI ISR.
static volatile uint8_t regs[REG_COUNT];
static volatile uint8_t regPtr = 0; // current register pointer for I2C read/write

// Convenience: absolute address of a channel's register, and a reader for it.
static inline uint8_t regAddr(uint8_t ch, uint8_t off) { return ch * CHANNEL_STRIDE + off; }
static inline uint8_t reg(uint8_t ch, uint8_t off)      { return regs[regAddr(ch, off)]; }

// ---------------------------------------------------------------------------
// I2C (TWI) slave callbacks — run in interrupt context, keep them short.
// ---------------------------------------------------------------------------

static bool isReadOnly(uint8_t addr) {
    return addr == REG_NUM_LEDS || addr == REG_NUM_CHANNELS || addr == REG_VERSION;
}

// Master wrote bytes to us. First byte = register pointer, the rest are values
// stored into consecutive registers (pointer auto-increments).
static void onI2CReceive(int numBytes) {
    if (numBytes <= 0) return;

    regPtr = Wire.read() & 0x7F; // first byte sets the pointer
    numBytes--;

    while (numBytes-- > 0) {
        uint8_t value = Wire.read();
        if (regPtr < REG_COUNT && !isReadOnly(regPtr)) {
            regs[regPtr] = value;
        }
        regPtr = (regPtr + 1) & 0x7F; // auto-increment, wrap within file
    }
}

// Master wants to read. Return the register at the pointer, then advance it so
// sequential reads walk the register file.
static void onI2CRequest() {
    uint8_t value = (regPtr < REG_COUNT) ? regs[regPtr] : 0;
    Wire.write(value);
    regPtr = (regPtr + 1) & 0x7F;
}

// ---------------------------------------------------------------------------
// Animations. Each renders into leds[ch] from channel ch's register block and
// is advanced by one step per rendered frame. Per-channel animation state lives
// in arrays indexed by channel.
// ---------------------------------------------------------------------------

static uint16_t stepCounter[NUM_CHANNELS]; // phase accumulator per channel
static uint16_t cometPhase[NUM_CHANNELS];  // sub-pixel comet position (Q8.8 in pixel units)
static uint8_t  meterLevel[NUM_CHANNELS];  // current decayed meter level per channel

static CRGB primaryColor(uint8_t ch) {
    return CRGB(reg(ch, REG_COLOR_R), reg(ch, REG_COLOR_G), reg(ch, REG_COLOR_B));
}

// Add an anti-aliased point of light at fractional position `q8` (Q8.8 pixel
// units around the ring), pre-scaled by `bright`. The fractional part spreads
// the light across the two adjacent pixels, so motion looks continuous.
static void addAA(CRGB* px, uint16_t q8, const CRGB& color, uint8_t bright) {
    const uint16_t span = (uint16_t)NUM_LEDS << 8;
    q8 %= span;
    const uint8_t i = q8 >> 8;       // integer pixel
    const uint8_t f = q8 & 0xFF;     // fraction toward the next pixel
    CRGB c = color; c.nscale8(bright);
    CRGB lo = c; lo.nscale8(255 - f);
    CRGB hi = c; hi.nscale8(f);
    px[i]                  += lo;
    px[(i + 1) % NUM_LEDS] += hi;
}

// Comet / "nightrider": a head with a fading tail gliding smoothly around the
// ring. Position is tracked sub-pixel so the wrap at pixel 0 is seamless.
static void renderCylon(uint8_t ch) {
    CRGB* px = leds[ch];
    fill_solid(px, NUM_LEDS, CRGB::Black);
    const CRGB color = primaryColor(ch);
    uint8_t length = reg(ch, REG_LENGTH);
    if (length < 1) length = 1;
    if (length > NUM_LEDS) length = NUM_LEDS;
    const bool ccw = reg(ch, REG_DIRECTION) != 0;

    const uint16_t span = (uint16_t)NUM_LEDS << 8;
    const uint16_t head = cometPhase[ch] % span;

    // Head plus a tail of `length` samples trailing opposite the motion. The
    // shared sub-pixel fraction shifts every sample together for smooth glide.
    for (uint8_t i = 0; i < length; i++) {
        const uint16_t back = (uint16_t)i << 8;
        const uint16_t pos = ccw ? (head + back) % span
                                 : (head + span - back) % span;
        const uint8_t fade = 255 - (uint16_t)i * 255 / length;
        addAA(px, pos, color, fade);
    }

    // Advance sub-pixel position; SPEED sets how far per (fixed-rate) frame.
    const uint16_t step = map(reg(ch, REG_SPEED), 0, 255, 2, 256);
    cometPhase[ch] = ccw ? (head + span - (step % span)) % span
                         : (head + step) % span;
}

static void renderRainbow(uint8_t ch) {
    const bool ccw = reg(ch, REG_DIRECTION) != 0;
    const uint8_t hue = ccw ? -(uint8_t)stepCounter[ch] : (uint8_t)stepCounter[ch];
    fill_rainbow(leds[ch], NUM_LEDS, hue, 255 / NUM_LEDS);
}

static void renderBreathe(uint8_t ch) {
    uint8_t phase = stepCounter[ch] & 0xFF;
    uint8_t level = (phase < 128) ? (phase * 2) : ((255 - phase) * 2);
    CRGB color = primaryColor(ch);
    color.nscale8(level);
    fill_solid(leds[ch], NUM_LEDS, color);
}

// Peak level meter: a bargraph whose length tracks the most recent level
// sample, snapping up instantly and decaying down on its own. Pixels are
// colored by their position on the ring: green up to REG_METER_GREEN, amber up
// to REG_METER_RED, red beyond.
static void renderMeter(uint8_t ch) {
    CRGB* px = leds[ch];
    uint8_t input = reg(ch, REG_METER_LEVEL);
    if (input > 100) input = 100;
    regs[regAddr(ch, REG_METER_LEVEL)] = 0; // consume the sample so it decays

    if (input > meterLevel[ch]) {
        meterLevel[ch] = input; // fast attack to new peaks
    } else {
        uint8_t decay = reg(ch, REG_METER_DECAY);
        meterLevel[ch] = (meterLevel[ch] > decay) ? (meterLevel[ch] - decay) : 0;
    }

    const uint8_t greenTo = reg(ch, REG_METER_GREEN);
    const uint8_t redFrom = reg(ch, REG_METER_RED);
    const bool ccw = reg(ch, REG_DIRECTION) != 0;

    fill_solid(px, NUM_LEDS, CRGB::Black);
    const uint8_t lit = ((uint16_t)meterLevel[ch] * NUM_LEDS + 50) / 100;
    for (uint8_t i = 0; i < lit; i++) {
        uint8_t pp = ((uint16_t)(i + 1) * 100) / NUM_LEDS; // this pixel's percent
        CRGB c;
        if (pp >= redFrom)     c = CRGB::Red;
        else if (pp > greenTo) c = CRGB::Orange; // amber transition zone
        else                   c = CRGB::Green;
        uint8_t idx = ccw ? (NUM_LEDS - 1 - i) : i;
        px[idx] = c;
    }
}

static void renderChannel(uint8_t ch) {
    switch (reg(ch, REG_MODE)) {
        case MODE_SOLID:   fill_solid(leds[ch], NUM_LEDS, primaryColor(ch)); break;
        case MODE_CYLON:   renderCylon(ch);   break;
        case MODE_RAINBOW: renderRainbow(ch); break;
        case MODE_BREATHE: renderBreathe(ch); break;
        case MODE_METER:   renderMeter(ch);   break;
        case MODE_OFF:
        default:           fill_solid(leds[ch], NUM_LEDS, CRGB::Black); break;
    }
    // Per-channel brightness applied here (global FastLED brightness stays 255).
    const uint8_t b = reg(ch, REG_BRIGHTNESS);
    if (b != 255) {
        for (uint8_t i = 0; i < NUM_LEDS; i++) leds[ch][i].nscale8(b);
    }
    stepCounter[ch]++;
}

// Map REG_SPEED (0..255) to a per-frame delay: 0 -> 120ms (slow), 255 -> 4ms.
// The meter and comet run at a fixed fast rate instead: the meter so its decay
// stays smooth, the comet so its sub-pixel motion glides (SPEED drives the
// per-frame step distance, not the frame delay).
static uint16_t frameInterval(uint8_t ch) {
    const uint8_t mode = reg(ch, REG_MODE);
    if (mode == MODE_METER || mode == MODE_CYLON) return 16;
    return map(reg(ch, REG_SPEED), 0, 255, 120, 4);
}

// ---------------------------------------------------------------------------

// Initialise one channel's register block with sensible defaults.
static void initChannelDefaults(uint8_t ch) {
    regs[regAddr(ch, REG_MODE)]        = MODE_OFF;
    regs[regAddr(ch, REG_BRIGHTNESS)]  = 64;
    regs[regAddr(ch, REG_COLOR_R)]     = 32;
    regs[regAddr(ch, REG_COLOR_G)]     = 32;
    regs[regAddr(ch, REG_COLOR_B)]     = 32;
    regs[regAddr(ch, REG_SPEED)]       = 160;
    regs[regAddr(ch, REG_LENGTH)]      = NUM_LEDS / 3 > 0 ? NUM_LEDS / 3 : 1;
    regs[regAddr(ch, REG_DIRECTION)]   = 0;
    regs[regAddr(ch, REG_METER_GREEN)] = 60; // green up to 60%
    regs[regAddr(ch, REG_METER_RED)]   = 85; // red from 85% up
    regs[regAddr(ch, REG_METER_DECAY)] = 4;  // ~4%/frame (~15ms) -> full fall ~0.4s
}

void setup() {
    for (uint8_t i = 0; i < REG_COUNT; i++) regs[i] = 0;
    for (uint8_t ch = 0; ch < NUM_CHANNELS; ch++) initChannelDefaults(ch);
    regs[REG_NUM_LEDS]     = NUM_LEDS;
    regs[REG_NUM_CHANNELS] = NUM_CHANNELS;
    regs[REG_VERSION]      = FIRMWARE_VERSION;

    // One FastLED controller per output pin. Pins are compile-time template args.
    FastLED.addLeds<WS2812B, LED_PIN,  GRB>(leds[0], NUM_LEDS);
    FastLED.addLeds<WS2812B, LED_PIN2, GRB>(leds[1], NUM_LEDS);
    FastLED.setBrightness(255); // per-channel brightness handled in renderChannel
    FastLED.clear(true);

    // Do NOT drive the internal pull-ups to 5V — a 3.3V master / level shifter
    // provides them. (Wire enables them by default; turn them back off.)
    Wire.begin(I2C_ADDRESS);
    digitalWrite(SDA, LOW);
    digitalWrite(SCL, LOW);
    Wire.onReceive(onI2CReceive);
    Wire.onRequest(onI2CRequest);

    Serial.begin(115200);
    Serial.print(F("WS2812 ring I2C slave @0x"));
    Serial.print(I2C_ADDRESS, HEX);
    Serial.print(F(", "));
    Serial.print(NUM_CHANNELS);
    Serial.print(F(" channels x "));
    Serial.print(NUM_LEDS);
    Serial.println(F(" LEDs"));
}

void loop() {
    static uint32_t lastFrame[NUM_CHANNELS] = {0};
    uint32_t now = millis();
    bool dirty = false;
    for (uint8_t ch = 0; ch < NUM_CHANNELS; ch++) {
        if (now - lastFrame[ch] >= frameInterval(ch)) {
            lastFrame[ch] = now;
            renderChannel(ch);
            dirty = true;
        }
    }
    if (dirty) FastLED.show(); // outputs all channels at once
}
