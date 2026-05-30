//! Driver for the WS2812 pixel-LED controller (an Arduino acting as an I2C slave).
//!
//! The device exposes an 8-bit register file (see `led/src/registers.h`): the
//! master writes a register-pointer byte, then one or more value bytes, and the
//! pointer auto-increments — so a multi-byte write fills consecutive registers.
//! Per-channel registers form a block repeated once per output channel at
//! `CHANNEL_STRIDE` (channel 0 at base 0x00, channel 1 at base 0x20).
//!
//! It talks to `/dev/i2c-N` with raw ioctls via `libc` (already a dependency),
//! so it needs no extra crate and cross-compiles to arm64 with the recorder.

use libc::{c_ulong, c_void};
use log::info;
use std::ffi::CString;
use std::io;
use std::os::unix::io::RawFd;

// Linux i2c-dev ioctl: select the 7-bit slave address for following transfers.
const I2C_SLAVE: c_ulong = 0x0703;

// Per-channel register offsets (mirror of led/src/registers.h).
const REG_MODE: u8 = 0x00;
const REG_BRIGHTNESS: u8 = 0x01;
const REG_COLOR_R: u8 = 0x02; // R, G, B are consecutive (0x02..0x04)
const REG_SPEED: u8 = 0x05;
const REG_LENGTH: u8 = 0x06;
const REG_DIRECTION: u8 = 0x07;
const REG_METER_LEVEL: u8 = 0x08; // LEVEL and PEAK are consecutive (0x08..0x09)
const REG_METER_GREEN: u8 = 0x0A; // GREEN, RED, DECAY, PEAK_DECAY consecutive (0x0A..0x0D)

// Global, read-only registers (absolute addresses).
const REG_NUM_LEDS: u8 = 0x70;
const REG_NUM_CHANNELS: u8 = 0x71;
const REG_VERSION: u8 = 0x7F;

const CHANNEL_STRIDE: u8 = 0x20;

/// Animation modes (mirror of the firmware's `Mode` enum).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Mode {
    Off = 0,
    Solid = 1,
    Cylon = 2,
    Rainbow = 3,
    Breathe = 4,
    Meter = 5,
}

/// An open handle to the WS2812 I2C controller.
pub struct Ws2812 {
    fd: RawFd,
}

impl Ws2812 {
    /// Open an I2C bus — a bus number like `"1"` or a path like `"/dev/i2c-1"` —
    /// and select the controller's slave address for subsequent transfers.
    pub fn open(bus: &str, address: u16) -> io::Result<Self> {
        let path = if bus.starts_with('/') {
            bus.to_string()
        } else {
            format!("/dev/i2c-{bus}")
        };
        let c_path = CString::new(path.clone())
            .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "bus path contains NUL"))?;

        let fd = unsafe { libc::open(c_path.as_ptr(), libc::O_RDWR) };
        if fd < 0 {
            return Err(io::Error::last_os_error());
        }

        let dev = Ws2812 { fd };
        if unsafe { libc::ioctl(fd, I2C_SLAVE, address as c_ulong) } < 0 {
            return Err(io::Error::last_os_error()); // fd closed by Drop
        }

        info!("Opened WS2812 controller on {path} @ {address:#04x}");
        Ok(dev)
    }

    fn write_bytes(&mut self, bytes: &[u8]) -> io::Result<()> {
        let n = unsafe { libc::write(self.fd, bytes.as_ptr() as *const c_void, bytes.len()) };
        if n < 0 {
            return Err(io::Error::last_os_error());
        }
        if (n as usize) != bytes.len() {
            return Err(io::Error::other("short I2C write"));
        }
        Ok(())
    }

    /// Write one register (pointer byte + value).
    fn write_reg(&mut self, addr: u8, val: u8) -> io::Result<()> {
        self.write_bytes(&[addr, val])
    }

    /// Write consecutive registers starting at `addr` (pointer auto-increments).
    fn write_block(&mut self, addr: u8, vals: &[u8]) -> io::Result<()> {
        let mut buf = Vec::with_capacity(vals.len() + 1);
        buf.push(addr);
        buf.extend_from_slice(vals);
        self.write_bytes(&buf)
    }

    /// Read one register: set the pointer, then read a byte back.
    fn read_reg(&mut self, addr: u8) -> io::Result<u8> {
        self.write_bytes(&[addr])?;
        let mut b = [0u8; 1];
        let n = unsafe { libc::read(self.fd, b.as_mut_ptr() as *mut c_void, 1) };
        if n != 1 {
            return Err(io::Error::last_os_error());
        }
        Ok(b[0])
    }

    pub fn version(&mut self) -> io::Result<u8> {
        self.read_reg(REG_VERSION)
    }

    pub fn num_leds(&mut self) -> io::Result<u8> {
        self.read_reg(REG_NUM_LEDS)
    }

    pub fn num_channels(&mut self) -> io::Result<u8> {
        self.read_reg(REG_NUM_CHANNELS)
    }

    fn ch_addr(ch: u8, off: u8) -> u8 {
        ch * CHANNEL_STRIDE + off
    }

    pub fn set_mode(&mut self, ch: u8, mode: Mode) -> io::Result<()> {
        self.write_reg(Self::ch_addr(ch, REG_MODE), mode as u8)
    }

    pub fn set_brightness(&mut self, ch: u8, brightness: u8) -> io::Result<()> {
        self.write_reg(Self::ch_addr(ch, REG_BRIGHTNESS), brightness)
    }

    pub fn set_color(&mut self, ch: u8, r: u8, g: u8, b: u8) -> io::Result<()> {
        self.write_block(Self::ch_addr(ch, REG_COLOR_R), &[r, g, b])
    }

    pub fn set_speed(&mut self, ch: u8, speed: u8) -> io::Result<()> {
        self.write_reg(Self::ch_addr(ch, REG_SPEED), speed)
    }

    pub fn set_length(&mut self, ch: u8, length: u8) -> io::Result<()> {
        self.write_reg(Self::ch_addr(ch, REG_LENGTH), length)
    }

    pub fn set_direction(&mut self, ch: u8, counter_clockwise: bool) -> io::Result<()> {
        self.write_reg(Self::ch_addr(ch, REG_DIRECTION), counter_clockwise as u8)
    }

    /// Push the bar and peak-hold inputs in a single block write (the firmware
    /// consumes both each frame). Drive continuously for a live PPM-style meter:
    /// `level` typically the analysis-window RMS, `peak` the max-|sample|.
    pub fn push_meter_and_peak(&mut self, ch: u8, level: u8, peak: u8) -> io::Result<()> {
        self.write_block(
            Self::ch_addr(ch, REG_METER_LEVEL),
            &[level.min(100), peak.min(100)],
        )
    }

    /// Configure color zones + fall rates as one block: pixels up to `green`
    /// are green, at/above `red` are red (amber between); `bar_decay` and
    /// `peak_decay` are percent/frame.
    pub fn set_meter_zones(
        &mut self,
        ch: u8,
        green: u8,
        red: u8,
        bar_decay: u8,
        peak_decay: u8,
    ) -> io::Result<()> {
        self.write_block(
            Self::ch_addr(ch, REG_METER_GREEN),
            &[green, red, bar_decay, peak_decay],
        )
    }

    pub fn off(&mut self, ch: u8) -> io::Result<()> {
        self.set_mode(ch, Mode::Off)
    }
}

impl Drop for Ws2812 {
    fn drop(&mut self) {
        unsafe {
            libc::close(self.fd);
        }
    }
}
