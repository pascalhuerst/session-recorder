//! Live Chromaprint fingerprinting of recently captured audio.
//!
//! Uses `rusty-chromaprint` directly on the PCM samples the recorder already
//! has in memory (no decode step needed, unlike fingerprinting a file from
//! disk) — validated as bit-compatible with libchromaprint/fpcalc and
//! Essentia's own Chromaprinter in
//! `discQL_Addons/discql_addon_chromaprintMatching/CHROMAPRINT_FINDINGS.md`,
//! whose companion Rust binary this module's API usage mirrors.

use rusty_chromaprint::{Configuration, Fingerprinter};

/// Compute a Chromaprint fingerprint over `samples` (interleaved PCM,
/// `channels` channels at `sample_rate` Hz) and return it as a compressed,
/// URL-safe base64 string. Returns `None` if fingerprinting produced no
/// output (e.g. the buffer was effectively silent/too short).
pub fn compute_fingerprint(samples: &[i16], sample_rate: u32, channels: u32) -> Option<String> {
    let config = Configuration::preset_test2();
    let mut printer = Fingerprinter::new(&config);
    printer.start(sample_rate, channels).ok()?;
    printer.consume(samples);
    printer.finish();

    let raw_fp = printer.fingerprint();
    if raw_fp.is_empty() {
        return None;
    }

    let compressor = rusty_chromaprint::FingerprintCompressor::from(&config);
    let compressed = compressor.compress(raw_fp);
    Some(base64_url_encode(&compressed))
}

/// URL-safe base64 (`-_` instead of `+/`), matching Chromaprint's own wire
/// format — copied from the validated encoder in
/// `discql_addon_chromaprintMatching/chromaprint_rust_helper`.
fn base64_url_encode(data: &[u8]) -> String {
    const CHARS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = *chunk.get(1).unwrap_or(&0) as u32;
        let b2 = *chunk.get(2).unwrap_or(&0) as u32;
        let n = (b0 << 16) | (b1 << 8) | b2;
        out.push(CHARS[((n >> 18) & 63) as usize] as char);
        out.push(CHARS[((n >> 12) & 63) as usize] as char);
        out.push(if chunk.len() > 1 { CHARS[((n >> 6) & 63) as usize] as char } else { '=' });
        out.push(if chunk.len() > 2 { CHARS[(n & 63) as usize] as char } else { '=' });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Deterministic pseudo-random noise (LCG), stereo interleaved - a stand-in
    /// for real music's varied spectral content. A pure tone was tried first
    /// and found to produce degenerate/empty chroma output from
    /// rusty-chromaprint far more easily than real audio does; noise is a
    /// closer proxy without needing a real audio fixture file.
    fn noise_samples(sample_rate: u32, seconds: f64) -> Vec<i16> {
        let n = (sample_rate as f64 * seconds) as usize;
        let mut samples = Vec::with_capacity(n * 2);
        let mut state: u32 = 12345;
        for _ in 0..n {
            state = state.wrapping_mul(1664525).wrapping_add(1013904223);
            let v = (state >> 16) as i16 / 2;
            samples.push(v); // left
            samples.push(v); // right
        }
        samples
    }

    #[test]
    fn silence_or_too_short_returns_none_but_never_panics() {
        let samples = vec![0i16; 100];
        assert!(compute_fingerprint(&samples, 48000, 2).is_none());
    }

    #[test]
    fn too_short_for_chromaprint_to_emit_anything_returns_none() {
        // Empirically, rusty-chromaprint needs well more than a couple of
        // seconds of input before it emits any fingerprint words at all
        // (observed: 0 words at 2s, 19 words at 5s, for the same noise
        // generator) - below that floor, None is the correct, expected
        // result, not a bug.
        let samples = noise_samples(48000, 2.0);
        assert!(compute_fingerprint(&samples, 48000, 2).is_none());
    }

    #[test]
    fn real_signal_produces_a_fingerprint() {
        // 10s, comfortably above the ~5s floor observed above, and far
        // short of production's actual 25s window.
        let samples = noise_samples(48000, 10.0);
        let fp = compute_fingerprint(&samples, 48000, 2);
        assert!(fp.is_some());
        assert!(!fp.unwrap().is_empty());
    }
}
