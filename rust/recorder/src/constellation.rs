//! A speed-invariant fingerprint of recently captured audio: its spectral
//! peaks ("constellation").
//!
//! Chromaprint stops matching once a record plays about half a percent off
//! its original speed. Peaks do better: playing s times too fast multiplies
//! every frequency by s and divides every time distance by s, so for three
//! peaks the frequency *ratios* and the *ratio* of the two time distances
//! stay what they were. A client hashes those, and gets the speed itself out
//! of every match. Forming and matching the hashes is the client's business;
//! the recorder only finds the peaks, which is the part that needs the audio.
//!
//! This is a port of the reference in the DiscQL addon
//! (`discql_addon_chromaprintMatching/speed_invariant/sifp.py`,
//! `log_spectrogram` and `peaks`), which is also what computes the library
//! side. The two have to find the same peaks in the same audio, so every
//! constant and every rounding below follows that file - including the ones
//! that look arbitrary. Change them only together with it.

use realfft::RealFftPlanner;

/// Everything is computed on the audio resampled to this rate, mono.
pub const RATE: u32 = 11025;
const NFFT: usize = 2048;
/// One frame - the time unit of a peak - is this many samples at `RATE`.
pub const HOP: usize = 256;
/// Log-frequency axis: bin b is `FMIN * 2^(b / BINS_PER_OCTAVE)`.
pub const FMIN: f64 = 300.0;
pub const BINS_PER_OCTAVE: usize = 48;
pub const NBINS: usize = 192;
/// A peak is the maximum of this many frames x bins around it ...
const PEAK_FRAMES: usize = 35;
const PEAK_BINS: usize = 31;
/// ... and at most this far below the loud end (95th percentile) of the window.
const PEAK_FLOOR_DB: f32 = 45.0;

const MAGIC: &[u8; 4] = b"SIC1";

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Peak {
    pub frame: u32,
    pub bin: u8,
}

/// The peaks of `samples` (interleaved PCM, `channels` channels at
/// `sample_rate` Hz) in the wire format "SIC1": the magic, a uint32 count,
/// all frame numbers (uint32), then all bins (uint8), little endian, sorted
/// by frame. Empty if the audio is too short to hold a peak.
pub fn compute_constellation(samples: &[i16], sample_rate: u32, channels: u32) -> Vec<u8> {
    let peaks = constellation(samples, sample_rate, channels);
    if peaks.is_empty() {
        return Vec::new();
    }
    pack(&peaks)
}

pub fn pack(peaks: &[Peak]) -> Vec<u8> {
    let mut out = Vec::with_capacity(8 + peaks.len() * 5);
    out.extend_from_slice(MAGIC);
    out.extend_from_slice(&(peaks.len() as u32).to_le_bytes());
    for peak in peaks {
        out.extend_from_slice(&peak.frame.to_le_bytes());
    }
    out.extend(peaks.iter().map(|peak| peak.bin));
    out
}

pub fn constellation(samples: &[i16], sample_rate: u32, channels: u32) -> Vec<Peak> {
    if channels == 0 || sample_rate == 0 {
        return Vec::new();
    }
    let mono = downmix(samples, channels as usize);
    let audio = resample(&mono, sample_rate, RATE);
    let (spectrogram, frames) = log_spectrogram(&audio);
    find_peaks(&spectrogram, frames)
}

/// Mean of the channels, kept on the int16 scale (the reference decodes to
/// s16 mono; the scale matters for the small constant inside the logarithm).
fn downmix(samples: &[i16], channels: usize) -> Vec<f32> {
    samples
        .chunks_exact(channels)
        .map(|frame| frame.iter().map(|&s| s as f32).sum::<f32>() / channels as f32)
        .collect()
}

/// Windowed-sinc resampling to `to` Hz. Rational ratios give a small table
/// of filter phases (147 for 48 kHz -> 11025 Hz), built once per call.
fn resample(input: &[f32], from: u32, to: u32) -> Vec<f32> {
    if from == to {
        return input.to_vec();
    }
    let g = gcd(from, to);
    let (step, phases) = ((from / g) as usize, (to / g) as usize); // output n sits at input n * step / phases
    let ratio = from as f64 / to as f64;
    // Pass everything up to the top of the log axis (4.8 kHz), stop by the new
    // Nyquist (5.5 kHz); when upsampling, just stay below the old Nyquist.
    let cutoff = if to < from {
        5150.0 / from as f64
    } else {
        0.45
    };
    let half = (ratio.max(1.0) * 28.0).ceil() as isize; // taps each side
    let taps = (2 * half + 1) as usize;
    let mut table = vec![0f32; phases * taps];
    for phase in 0..phases {
        let frac = phase as f64 / phases as f64;
        let mut sum = 0.0;
        for k in 0..taps {
            let x = k as f64 - half as f64 - frac; // distance from the output position, in input samples
            let sinc = if x.abs() < 1e-12 {
                2.0 * cutoff
            } else {
                (2.0 * std::f64::consts::PI * cutoff * x).sin() / (std::f64::consts::PI * x)
            };
            let w = 0.5 + 0.5 * (std::f64::consts::PI * x / (half as f64 + 1.0)).cos(); // Hann
            let v = sinc * w;
            table[phase * taps + k] = v as f32;
            sum += v;
        }
        for k in 0..taps {
            table[phase * taps + k] = (table[phase * taps + k] as f64 / sum) as f32;
        }
    }
    let out_len = (input.len() as u64 * to as u64 / from as u64) as usize;
    let mut out = Vec::with_capacity(out_len);
    for n in 0..out_len {
        let pos = n * step;
        let (center, phase) = ((pos / phases) as isize, pos % phases);
        let kernel = &table[phase * taps..(phase + 1) * taps];
        let first = center - half;
        let mut acc = 0f32;
        if first >= 0 && (first as usize + taps) <= input.len() {
            let window = &input[first as usize..first as usize + taps];
            for (x, k) in window.iter().zip(kernel) {
                acc += x * k;
            }
        } else {
            for (k, coefficient) in kernel.iter().enumerate() {
                let i = first + k as isize;
                if i >= 0 && (i as usize) < input.len() {
                    acc += input[i as usize] * coefficient;
                }
            }
        }
        out.push(acc);
    }
    out
}

fn gcd(a: u32, b: u32) -> u32 {
    if b == 0 {
        a
    } else {
        gcd(b, a % b)
    }
}

/// Round half to even, as numpy's `round` does.
fn round_half_even(x: f64) -> f64 {
    let r = x.round();
    if (x - x.trunc()).abs() == 0.5 && r % 2.0 != 0.0 {
        r - x.signum()
    } else {
        r
    }
}

/// dB magnitudes on the log-frequency axis, `frames` rows of `NBINS`.
fn log_spectrogram(audio: &[f32]) -> (Vec<f32>, usize) {
    if audio.len() < NFFT {
        return (Vec::new(), 0);
    }
    let frames = (audio.len() - NFFT) / HOP + 1;
    let fft_bin_hz = RATE as f64 / NFFT as f64;
    let log_freq = |k: usize| FMIN * 2f64.powf(k as f64 / BINS_PER_OCTAVE as f64);
    // A log bin narrower than an FFT bin is read off by interpolation, a wider
    // one takes the strongest FFT bin it covers.
    let edges: Vec<usize> = (0..=NBINS)
        .map(|k| (round_half_even(log_freq(k) / fft_bin_hz) as usize).clamp(1, NFFT / 2))
        .collect();
    let centers: Vec<f64> = (0..NBINS)
        .map(|k| log_freq(k) * 2f64.powf(0.5 / BINS_PER_OCTAVE as f64) / fft_bin_hz)
        .collect();
    // np.hanning: symmetric, zero at both ends.
    let window: Vec<f32> = (0..NFFT)
        .map(|n| {
            (0.5 - 0.5 * (2.0 * std::f64::consts::PI * n as f64 / (NFFT - 1) as f64).cos()) as f32
        })
        .collect();

    let fft = RealFftPlanner::<f32>::new().plan_fft_forward(NFFT);
    let mut frame = fft.make_input_vec();
    let mut spectrum = fft.make_output_vec();
    let mut magnitude = vec![0f32; NFFT / 2 + 1];
    let mut out = vec![0f32; frames * NBINS];
    for t in 0..frames {
        let start = t * HOP;
        for ((x, s), w) in frame
            .iter_mut()
            .zip(&audio[start..start + NFFT])
            .zip(&window)
        {
            *x = s * w;
        }
        fft.process(&mut frame, &mut spectrum)
            .expect("buffers come from the plan itself");
        for (m, c) in magnitude.iter_mut().zip(&spectrum) {
            *m = (c.re * c.re + c.im * c.im).sqrt();
        }
        let row = &mut out[t * NBINS..(t + 1) * NBINS];
        for b in 0..NBINS {
            let value = if edges[b + 1] - edges[b] > 1 {
                // The last bin runs to the end of the spectrum, like the
                // reference's `np.maximum.reduceat` does.
                let end = if b == NBINS - 1 {
                    magnitude.len()
                } else {
                    edges[b + 1]
                };
                magnitude[edges[b]..end]
                    .iter()
                    .copied()
                    .fold(0f32, f32::max)
            } else {
                let lo = centers[b].floor() as usize;
                let frac = (centers[b] - centers[b].floor()) as f32;
                magnitude[lo] * (1.0 - frac) + magnitude[lo + 1] * frac
            };
            row[b] = 20.0 * (value + 1e-3).log10();
        }
    }
    (out, frames)
}

fn find_peaks(spectrogram: &[f32], frames: usize) -> Vec<Peak> {
    if frames < PEAK_FRAMES {
        return Vec::new();
    }
    // numpy's percentile: linear interpolation between the two nearest ranks.
    let mut sorted = spectrogram.to_vec();
    sorted.sort_unstable_by(f32::total_cmp);
    let rank = 0.95 * (sorted.len() - 1) as f64;
    let (lo, frac) = (rank.floor() as usize, (rank - rank.floor()) as f32);
    let p95 = sorted[lo] + (sorted[(lo + 1).min(sorted.len() - 1)] - sorted[lo]) * frac;
    let floor = p95 - PEAK_FLOOR_DB;

    // The 2-D maximum is separable: along time first, then along frequency.
    let (half_t, half_f) = (PEAK_FRAMES / 2, PEAK_BINS / 2);
    let mut along_time = vec![f32::MIN; spectrogram.len()];
    for t in 0..frames {
        let (a, b) = (t.saturating_sub(half_t), (t + half_t + 1).min(frames));
        let row = &mut along_time[t * NBINS..(t + 1) * NBINS];
        for u in a..b {
            for (m, v) in row.iter_mut().zip(&spectrogram[u * NBINS..(u + 1) * NBINS]) {
                if *v > *m {
                    *m = *v;
                }
            }
        }
    }
    let mut peaks = Vec::new();
    for t in 0..frames {
        let row = &along_time[t * NBINS..(t + 1) * NBINS];
        for b in 0..NBINS {
            let value = spectrogram[t * NBINS + b];
            if value <= floor {
                continue;
            }
            let (a, e) = (b.saturating_sub(half_f), (b + half_f + 1).min(NBINS));
            if row[a..e].iter().all(|&m| m <= value) {
                peaks.push(Peak {
                    frame: t as u32,
                    bin: b as u8,
                });
            }
        }
    }
    peaks
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Three steady tones for the first half, three others for the second,
    /// plus a little noise so the floor is not degenerate.
    fn tones(sample_rate: u32, seconds: f64, speed: f64) -> Vec<i16> {
        let n = (sample_rate as f64 * seconds / speed) as usize;
        let mut state: u32 = 99;
        (0..n)
            .flat_map(|i| {
                let t = i as f64 / sample_rate as f64 * speed; // position in the original
                let set: [f64; 3] = if t < seconds / 2.0 {
                    [440.0, 1000.0, 2500.0]
                } else {
                    [600.0, 1500.0, 3300.0]
                };
                // each tone pulses, so there is something to be a maximum of in time too
                let pulse = 0.6 + 0.4 * (t * 2.0 * std::f64::consts::TAU).sin();
                let v: f64 = set
                    .iter()
                    .map(|f| (t * f * std::f64::consts::TAU).sin())
                    .sum::<f64>()
                    * 4000.0
                    * pulse;
                state = state.wrapping_mul(1664525).wrapping_add(1013904223);
                let noise = ((state >> 16) as i16 / 256) as f64;
                let s = (v + noise) as i16;
                [s, s]
            })
            .collect()
    }

    fn bin_of(freq: f64) -> i32 {
        (BINS_PER_OCTAVE as f64 * (freq / FMIN).log2()).floor() as i32
    }

    #[test]
    fn too_short_audio_has_no_peaks() {
        assert!(compute_constellation(&vec![0i16; 1000], 48000, 2).is_empty());
        assert!(compute_constellation(&[], 48000, 2).is_empty());
    }

    #[test]
    fn peaks_sit_on_the_tones_that_were_played() {
        let peaks = constellation(&tones(48000, 10.0, 1.0), 48000, 2);
        assert!(!peaks.is_empty());
        let half = (5.0 * RATE as f64 / HOP as f64) as u32;
        for (range, freqs) in [
            (0..half - 20, [440.0, 1000.0, 2500.0]),
            (half + 20..u32::MAX, [600.0, 1500.0, 3300.0]),
        ] {
            let here: Vec<&Peak> = peaks.iter().filter(|p| range.contains(&p.frame)).collect();
            for f in freqs {
                assert!(
                    here.iter().any(|p| (p.bin as i32 - bin_of(f)).abs() <= 1),
                    "no peak near {f} Hz (bin {})",
                    bin_of(f)
                );
            }
        }
    }

    #[test]
    fn a_faster_playback_moves_every_peak_by_the_same_number_of_bins() {
        // 6 % too fast is 4.03 log bins up; the tones' peaks must follow.
        let normal = constellation(&tones(48000, 10.0, 1.0), 48000, 2);
        let fast = constellation(&tones(48000, 10.0, 1.06), 48000, 2);
        let shift = (BINS_PER_OCTAVE as f64 * 1.06f64.log2()).round() as i32;
        for f in [440.0, 1000.0, 2500.0] {
            let expected = bin_of(f * 1.06);
            assert!((expected - bin_of(f) - shift).abs() <= 1);
            assert!(
                fast.iter().any(|p| (p.bin as i32 - expected).abs() <= 1),
                "no shifted peak for {f} Hz"
            );
        }
        assert!(!normal.is_empty());
    }

    #[test]
    fn the_sample_rate_of_the_capture_does_not_matter() {
        // Same tones captured at two rates: the peaks land in the same bins.
        // (Their exact frames are not compared - on steady tones those are
        // decided by the noise, which differs between the two signals.)
        let bins = |rate: u32| {
            let mut bins: Vec<u8> = constellation(&tones(rate, 8.0, 1.0), rate, 2)
                .iter()
                .map(|p| p.bin)
                .collect();
            bins.sort_unstable();
            bins.dedup();
            bins
        };
        let (a, b) = (bins(48000), bins(44100));
        for f in [440.0, 1000.0, 2500.0, 600.0, 1500.0, 3300.0] {
            let near = |set: &[u8]| set.iter().any(|&x| (x as i32 - bin_of(f)).abs() <= 1);
            assert!(near(&a) && near(&b), "{f} Hz missing at one of the rates");
        }
    }

    #[test]
    fn packed_form_is_magic_count_frames_bins() {
        let packed = pack(&[
            Peak { frame: 3, bin: 7 },
            Peak {
                frame: 0x01020304,
                bin: 191,
            },
        ]);
        assert_eq!(&packed[..4], b"SIC1");
        assert_eq!(&packed[4..8], &2u32.to_le_bytes());
        assert_eq!(&packed[8..16], &[3, 0, 0, 0, 4, 3, 2, 1]);
        assert_eq!(&packed[16..], &[7, 191]);
    }

    #[test]
    fn rounding_follows_numpy() {
        assert_eq!(round_half_even(2.5), 2.0);
        assert_eq!(round_half_even(3.5), 4.0);
        assert_eq!(round_half_even(2.4), 2.0);
        assert_eq!(round_half_even(2.6), 3.0);
    }
}
