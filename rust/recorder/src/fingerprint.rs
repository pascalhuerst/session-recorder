//! Live Chromaprint fingerprinting of recently captured audio.
//!
//! Uses `rusty-chromaprint` directly on the PCM samples the recorder already
//! has in memory (no decode step needed, unlike fingerprinting a file from
//! disk) — validated as bit-compatible with libchromaprint/fpcalc and
//! Essentia's own Chromaprinter in
//! `discQL_Addons/discql_addon_chromaprintMatching/CHROMAPRINT_FINDINGS.md`,
//! whose companion Rust binary this module's API usage mirrors.

use rusty_chromaprint::{Configuration, Fingerprinter};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{sync_channel, SyncSender, TrySendError};
use std::sync::{Arc, Mutex};

/// Most speed variants one window may be fingerprinted for. Each costs about
/// as much as the plain fingerprint (roughly 0.23 s on a Raspberry Pi 3 for a
/// 25 s window), and they are computed one after another.
pub const MAX_SPEED_VARIANTS: usize = 64;

/// Compute a Chromaprint fingerprint over `samples` (interleaved PCM,
/// `channels` channels at `sample_rate` Hz) and return it as a compressed,
/// URL-safe base64 string. Returns `None` if fingerprinting produced no
/// output (e.g. the buffer was effectively silent/too short).
pub fn compute_fingerprint(samples: &[i16], sample_rate: u32, channels: u32) -> Option<String> {
    let config = Configuration::preset_test2();
    let raw_fp = raw_fingerprint(&config, samples, sample_rate, channels)?;
    let compressor = rusty_chromaprint::FingerprintCompressor::from(&config);
    let compressed = compressor.compress(&raw_fp);
    Some(base64_url_encode(&compressed))
}

/// Like [`compute_fingerprint`], for audio assumed to be playing
/// `speed_percent` % too fast (negative: too slow) - a turntable's pitch
/// fader changes tempo and pitch together, and Chromaprint only tolerates
/// about half a percent of that. The audio is brought back to its original
/// speed before fingerprinting, so the result matches a fingerprint of the
/// unshifted recording if the assumption is right.
pub fn compute_fingerprint_at_speed(
    samples: &[i16],
    sample_rate: u32,
    channels: u32,
    speed_percent: f64,
) -> Option<String> {
    compute_fingerprint(
        samples,
        corrected_sample_rate(sample_rate, speed_percent),
        channels,
    )
}

/// The sample rate to declare for audio captured at `sample_rate` while
/// playing `speed_percent` % too fast: the same samples, stretched back to
/// the original duration. Chromaprint resamples from whatever rate it is told,
/// so no resampling of our own is needed.
fn corrected_sample_rate(sample_rate: u32, speed_percent: f64) -> u32 {
    (sample_rate as f64 / (1.0 + speed_percent / 100.0)).round() as u32
}

fn raw_fingerprint(
    config: &Configuration,
    samples: &[i16],
    sample_rate: u32,
    channels: u32,
) -> Option<Vec<u32>> {
    let mut printer = Fingerprinter::new(config);
    printer.start(sample_rate, channels).ok()?;
    printer.consume(samples);
    printer.finish();

    let raw_fp = printer.fingerprint();
    if raw_fp.is_empty() {
        return None;
    }
    Some(raw_fp.to_vec())
}

/// The playback speeds, in percent deviation, to fingerprint a window for in
/// addition to the unshifted one: every multiple of `step_percent` from
/// `-range_percent` to `+range_percent`, without 0, ordered by distance from
/// the original speed (+1, -1, +2, -2, ...) - the order they are computed in,
/// small deviations being the likelier ones. A range of 0 means none.
pub fn speed_grid(range_percent: f64, step_percent: f64) -> Result<Vec<f64>, String> {
    if !(0.0..=50.0).contains(&range_percent) {
        return Err(format!(
            "speed range must be between 0 and 50 percent, got {range_percent}"
        ));
    }
    if range_percent == 0.0 {
        return Ok(Vec::new());
    }
    if !(step_percent > 0.0) {
        return Err(format!(
            "speed step must be greater than 0 percent, got {step_percent}"
        ));
    }
    // A hair of tolerance so that e.g. 8 / 0.1 does not lose its last step
    // to floating point.
    let steps = (range_percent / step_percent + 1e-9).floor() as usize;
    if steps * 2 > MAX_SPEED_VARIANTS {
        return Err(format!(
            "a range of +-{range_percent} % in steps of {step_percent} % is {} speeds, more than the {MAX_SPEED_VARIANTS} allowed",
            steps * 2
        ));
    }
    let mut speeds = Vec::with_capacity(steps * 2);
    for i in 1..=steps {
        speeds.push(i as f64 * step_percent);
        speeds.push(-(i as f64) * step_percent);
    }
    Ok(speeds)
}

/// A window's fingerprint for one assumed playback speed.
#[derive(Debug, Clone, PartialEq)]
pub struct SpeedFingerprint {
    pub speed_percent: f64,
    pub fingerprint: String,
}

/// What the worker has published so far for the current session.
#[derive(Debug, Clone, Default, PartialEq)]
pub struct FingerprintOutput {
    /// The window as captured (speed 0), or empty if none was computed yet.
    pub fingerprint: String,
    /// The same window's speed variants, as far as they are computed yet:
    /// empty when `fingerprint` is new, then growing one by one.
    pub speed_fingerprints: Vec<SpeedFingerprint>,
    /// The same window's peak constellation ("SIC1", see
    /// `crate::constellation`), if the worker computes one; replaced a
    /// moment after `fingerprint`.
    pub peak_constellation: Vec<u8>,
    /// The same window's tempo, if the worker estimates one: BPM and the
    /// model's probability for it; both 0 while there is none.
    pub tempo_bpm: f32,
    pub tempo_probability: f32,
    /// Counts the windows: goes up by one with every new `fingerprint`.
    pub window: u32,
}

struct Job {
    samples: Vec<i16>,
    sample_rate: u32,
    channels: u32,
    generation: u64,
}

#[derive(Default)]
struct Published {
    /// Bumped by `reset`; a job only publishes into the generation it was
    /// submitted in, so a result that arrives after its session ended is
    /// dropped instead of showing up in the next session.
    generation: u64,
    output: FingerprintOutput,
    /// What `take_new` has not handed out yet: the same as `output`, minus
    /// everything already taken.
    unsent: FingerprintOutput,
}

struct Shared {
    published: Mutex<Published>,
    busy: AtomicBool,
}

/// Fingerprints windows on one dedicated thread, strictly one computation
/// after another: the unshifted fingerprint first, then each speed variant
/// in turn, every one published as soon as it is done. A window submitted while the
/// previous one is still being worked on is skipped rather than queued, so
/// fingerprinting never uses more than one core however many variants are
/// configured.
pub struct FingerprintWorker {
    shared: Arc<Shared>,
    jobs: SyncSender<Job>,
    speed_count: usize,
}

impl FingerprintWorker {
    /// `speeds`: the assumed playback speeds to fingerprint each window for
    /// besides the original one. `constellation`: also compute the window's
    /// speed-invariant peak constellation. `spread`: instead of computing
    /// everything in one go, pace the computations evenly over this long, so
    /// the load is level rather than a burst after every window - at the
    /// price of the last results arriving that much later. `tempo`: also
    /// estimate the window's tempo with this model.
    pub fn spawn(
        speeds: Vec<f64>,
        constellation: bool,
        spread: Option<std::time::Duration>,
        tempo: Option<crate::tempo::TempoEstimator>,
    ) -> std::io::Result<Self> {
        let shared = Arc::new(Shared {
            published: Mutex::new(Published::default()),
            busy: AtomicBool::new(false),
        });
        let speed_count = speeds.len();
        let (jobs, queue) = sync_channel::<Job>(1);
        let worker_shared = Arc::clone(&shared);
        std::thread::Builder::new()
            .name("fingerprint".into())
            .spawn(move || {
                // Lowest priority: fingerprinting is steady CPU work with no
                // deadline, on a host whose audio threads (the capture here,
                // PipeWire, Bluetooth) may be running without realtime
                // priority - at equal priority it made a Bluetooth stream
                // crackle on the Raspberry Pi 3. Linux applies a nice value
                // to the calling thread only.
                // SAFETY: plain syscalls on this thread's own id.
                unsafe {
                    let tid = libc::syscall(libc::SYS_gettid) as libc::id_t;
                    libc::setpriority(libc::PRIO_PROCESS, tid, 19);
                }
                while let Ok(job) = queue.recv() {
                    Self::run(
                        &worker_shared,
                        &speeds,
                        constellation,
                        spread,
                        tempo.as_ref(),
                        job,
                    );
                    worker_shared.busy.store(false, Ordering::Release);
                }
            })?;
        Ok(Self {
            shared,
            jobs,
            speed_count,
        })
    }

    /// Hand over a window. Returns false, without taking it, if the previous
    /// window is still being fingerprinted.
    pub fn submit(&self, samples: Vec<i16>, sample_rate: u32, channels: u32) -> bool {
        if self.shared.busy.swap(true, Ordering::AcqRel) {
            return false;
        }
        let generation = self
            .shared
            .published
            .lock()
            .map(|p| p.generation)
            .unwrap_or(0);
        match self.jobs.try_send(Job {
            samples,
            sample_rate,
            channels,
            generation,
        }) {
            Ok(()) => true,
            Err(TrySendError::Full(_)) | Err(TrySendError::Disconnected(_)) => {
                self.shared.busy.store(false, Ordering::Release);
                false
            }
        }
    }

    /// Forget everything published and make any window still being worked on
    /// publish nothing - called when a session starts or ends.
    pub fn reset(&self) {
        if let Ok(mut published) = self.shared.published.lock() {
            published.generation = published.generation.wrapping_add(1);
            // The window number survives: the next window must not look like
            // more of this one.
            let window = published.output.window;
            published.output = FingerprintOutput {
                window,
                ..Default::default()
            };
            published.unsent = published.output.clone();
        }
    }

    /// How many speed variants a window gets in total.
    pub fn speed_count(&self) -> usize {
        self.speed_count
    }

    /// Everything published since the last call, and nothing twice: the
    /// plain fingerprint, constellation and speed variants not handed out
    /// yet, with the window they belong to. For whoever sends them on.
    pub fn take_new(&self) -> FingerprintOutput {
        let Ok(mut published) = self.shared.published.lock() else {
            return FingerprintOutput::default();
        };
        let window = published.unsent.window;
        std::mem::replace(
            &mut published.unsent,
            FingerprintOutput {
                window,
                ..Default::default()
            },
        )
    }

    pub fn output(&self) -> FingerprintOutput {
        self.shared
            .published
            .lock()
            .map(|p| p.output.clone())
            .unwrap_or_default()
    }

    fn run(
        shared: &Shared,
        speeds: &[f64],
        constellation: bool,
        spread: Option<std::time::Duration>,
        tempo: Option<&crate::tempo::TempoEstimator>,
        job: Job,
    ) {
        // Each result goes out the moment it exists. Whenever the session
        // has moved on in the meantime, the rest of the window is dropped.
        let publish = |update: &dyn Fn(&mut FingerprintOutput)| match shared.published.lock() {
            Ok(mut published) if published.generation == job.generation => {
                let published = &mut *published;
                update(&mut published.output);
                update(&mut published.unsent);
                true
            }
            _ => false,
        };
        // A new window: nothing of the previous one belongs to it.
        if !publish(&|output| {
            *output = FingerprintOutput {
                window: output.window.wrapping_add(1),
                ..Default::default()
            };
        }) {
            return;
        }

        // With `spread`, computation k of n may not start before k/n of that
        // time has passed since the window came in. The tempo is not one of
        // the n: it goes first and shares the plain fingerprint's turn.
        let started = std::time::Instant::now();
        let steps = 1 + constellation as u32 + speeds.len() as u32;
        let mut step = 0u32;
        let mut wait_for_turn = || {
            if let Some(spread) = spread {
                let due = started + spread * step / steps;
                std::thread::sleep(due.saturating_duration_since(std::time::Instant::now()));
            }
            step += 1;
        };

        // Constellation and tempo both work on the audio at 11025 Hz mono.
        let mono = (constellation || tempo.is_some()).then(|| {
            crate::constellation::mono_at_rate(&job.samples, job.sample_rate, job.channels)
        });
        // The tempo first, before any fingerprint: a client can use it to
        // narrow down what it compares the fingerprints with (a track's own
        // tempo times the playback speed must give this one), and for that
        // it has to be there when the fingerprints arrive.
        if let Some(estimator) = tempo {
            let found = estimator.estimate(mono.as_deref().unwrap_or_default());
            let (bpm, probability) = found.map(|t| (t.bpm, t.probability)).unwrap_or((0.0, 0.0));
            if !publish(&|output| (output.tempo_bpm, output.tempo_probability) = (bpm, probability))
            {
                return;
            }
        }

        wait_for_turn();
        let Some(fingerprint) = compute_fingerprint(&job.samples, job.sample_rate, job.channels)
        else {
            return;
        };
        if !publish(&|output| output.fingerprint = fingerprint.clone()) {
            return;
        }
        // Before the speeds: it is cheaper than a single one of them and
        // covers what all of them together are for.
        if constellation {
            wait_for_turn();
            let peaks = crate::constellation::compute_constellation_from_mono(
                mono.as_deref().unwrap_or_default(),
            );
            if !publish(&|output| output.peak_constellation = peaks.clone()) {
                return;
            }
        }
        for &speed_percent in speeds {
            wait_for_turn();
            let Some(fingerprint) = compute_fingerprint_at_speed(
                &job.samples,
                job.sample_rate,
                job.channels,
                speed_percent,
            ) else {
                continue;
            };
            if !publish(&|output| {
                output.speed_fingerprints.push(SpeedFingerprint {
                    speed_percent,
                    fingerprint: fingerprint.clone(),
                })
            }) {
                return;
            }
        }
    }
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
        out.push(if chunk.len() > 1 {
            CHARS[((n >> 6) & 63) as usize] as char
        } else {
            '='
        });
        out.push(if chunk.len() > 2 {
            CHARS[(n & 63) as usize] as char
        } else {
            '='
        });
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

    /// A made-up melody: three-note chords changing a few times a second,
    /// plus a little noise - enough moving pitch content for Chromaprint to
    /// tell a transposition apart, which plain noise is not.
    fn melody_samples(sample_rate: u32, seconds: f64) -> Vec<i16> {
        let n = (sample_rate as f64 * seconds) as usize;
        let mut samples = Vec::with_capacity(n * 2);
        let mut state: u32 = 777;
        let mut notes = [0.0f64; 3];
        let note_len = (sample_rate as f64 * 0.37) as usize;
        for i in 0..n {
            if i % note_len == 0 {
                for note in notes.iter_mut() {
                    state = state.wrapping_mul(1664525).wrapping_add(1013904223);
                    let semitone = (state >> 16) % 36;
                    *note = 110.0 * 2f64.powf(semitone as f64 / 12.0);
                }
            }
            state = state.wrapping_mul(1664525).wrapping_add(1013904223);
            let noise = ((state >> 16) as i16 / 64) as f64;
            let t = i as f64 / sample_rate as f64;
            let tone: f64 = notes
                .iter()
                .map(|f| (t * f * std::f64::consts::TAU).sin())
                .sum::<f64>()
                * 5000.0;
            let v = (tone + noise) as i16;
            samples.push(v);
            samples.push(v);
        }
        samples
    }

    /// `samples` (stereo, interleaved) as heard when played `factor` times
    /// too fast: linear interpolation, tempo and pitch move together.
    fn played_faster(samples: &[i16], factor: f64) -> Vec<i16> {
        let frames = samples.len() / 2;
        let out_frames = ((frames - 1) as f64 / factor) as usize;
        let mut out = Vec::with_capacity(out_frames * 2);
        for i in 0..out_frames {
            let pos = i as f64 * factor;
            let (a, frac) = (pos as usize, pos.fract());
            for ch in 0..2 {
                let (x, y) = (samples[a * 2 + ch] as f64, samples[(a + 1) * 2 + ch] as f64);
                out.push((x + (y - x) * frac) as i16);
            }
        }
        out
    }

    fn bit_error_rate(a: &[u32], b: &[u32]) -> f64 {
        let n = a.len().min(b.len());
        assert!(n > 50, "fingerprints too short to compare ({n} words)");
        let wrong: u32 = a
            .iter()
            .zip(b)
            .take(n)
            .map(|(x, y)| (x ^ y).count_ones())
            .sum();
        wrong as f64 / (32 * n) as f64
    }

    #[test]
    fn speed_zero_is_the_plain_fingerprint() {
        let samples = melody_samples(48000, 10.0);
        assert_eq!(
            compute_fingerprint_at_speed(&samples, 48000, 2, 0.0),
            compute_fingerprint(&samples, 48000, 2)
        );
    }

    #[test]
    fn the_right_speed_assumption_undoes_a_pitched_playback() {
        let config = Configuration::preset_test2();
        let original = melody_samples(48000, 25.0);
        let pitched = played_faster(&original, 1.04);
        let reference = raw_fingerprint(&config, &original, 48000, 2).unwrap();

        let uncorrected = raw_fingerprint(&config, &pitched, 48000, 2).unwrap();
        let corrected =
            raw_fingerprint(&config, &pitched, corrected_sample_rate(48000, 4.0), 2).unwrap();
        let wrong_way =
            raw_fingerprint(&config, &pitched, corrected_sample_rate(48000, -4.0), 2).unwrap();

        let (plain, fixed, worse) = (
            bit_error_rate(&reference, &uncorrected),
            bit_error_rate(&reference, &corrected),
            bit_error_rate(&reference, &wrong_way),
        );
        assert!(fixed < 0.08, "corrected fingerprint still differs: {fixed}");
        assert!(
            plain > 0.25,
            "a 4 % pitch should break the plain fingerprint: {plain}"
        );
        assert!(
            worse > 0.25,
            "correcting the wrong way must not match: {worse}"
        );
    }

    #[test]
    fn speed_grid_lists_every_step_without_zero() {
        assert_eq!(speed_grid(0.0, 1.0).unwrap(), Vec::<f64>::new());
        assert_eq!(speed_grid(2.0, 1.0).unwrap(), vec![1.0, -1.0, 2.0, -2.0]);
        assert_eq!(speed_grid(1.0, 0.5).unwrap(), vec![0.5, -0.5, 1.0, -1.0]);
        assert_eq!(speed_grid(8.0, 1.0).unwrap().len(), 16);
        // A range that is not a multiple of the step stops at the last full step.
        assert_eq!(speed_grid(2.5, 1.0).unwrap(), vec![1.0, -1.0, 2.0, -2.0]);
    }

    #[test]
    fn speed_grid_refuses_nonsense() {
        assert!(speed_grid(-1.0, 1.0).is_err());
        assert!(speed_grid(60.0, 1.0).is_err());
        assert!(speed_grid(8.0, 0.0).is_err());
        assert!(speed_grid(8.0, -1.0).is_err());
        assert!(speed_grid(8.0, f64::NAN).is_err());
        assert!(
            speed_grid(16.0, 0.1).is_err(),
            "320 speeds is more than the limit"
        );
    }

    fn wait_for(
        worker: &FingerprintWorker,
        done: impl Fn(&FingerprintOutput) -> bool,
    ) -> FingerprintOutput {
        for _ in 0..600 {
            let output = worker.output();
            if done(&output) {
                return output;
            }
            std::thread::sleep(std::time::Duration::from_millis(50));
        }
        panic!("the fingerprint worker did not publish in time");
    }

    #[test]
    fn worker_publishes_the_plain_fingerprint_and_every_speed() {
        let samples = melody_samples(48000, 10.0);
        let speeds = speed_grid(2.0, 1.0).unwrap();
        let worker = FingerprintWorker::spawn(speeds.clone(), false, None, None).unwrap();
        assert!(worker.submit(samples.clone(), 48000, 2));

        let output = wait_for(&worker, |o| o.speed_fingerprints.len() == speeds.len());
        assert_eq!(
            Some(output.fingerprint),
            compute_fingerprint(&samples, 48000, 2)
        );
        for (variant, speed) in output.speed_fingerprints.iter().zip(&speeds) {
            assert_eq!(variant.speed_percent, *speed);
            assert_eq!(
                Some(variant.fingerprint.clone()),
                compute_fingerprint_at_speed(&samples, 48000, 2, *speed)
            );
        }
    }

    #[test]
    fn worker_publishes_each_speed_as_soon_as_it_is_done_and_starts_over_per_window() {
        let worker =
            FingerprintWorker::spawn(speed_grid(8.0, 1.0).unwrap(), false, None, None).unwrap();
        assert_eq!(worker.speed_count(), 16);
        assert!(worker.submit(melody_samples(48000, 25.0), 48000, 2));

        // Seen growing: some, but not all, of the speeds are out already.
        let partial = wait_for(&worker, |o| !o.speed_fingerprints.is_empty());
        assert!(
            partial.speed_fingerprints.len() < 16,
            "the speeds came as one set"
        );
        assert_eq!(partial.speed_fingerprints[0].speed_percent, 1.0);
        let first_window = wait_for(&worker, |o| o.speed_fingerprints.len() == 16);

        // The next window's plain fingerprint takes the old variants away.
        let other = played_faster(&melody_samples(48000, 26.0), 1.03);
        while !worker.submit(other.clone(), 48000, 2) {
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
        let second = wait_for(&worker, |o| o.fingerprint != first_window.fingerprint);
        assert!(second.speed_fingerprints.len() < 16);
        for variant in &second.speed_fingerprints {
            assert!(!first_window.speed_fingerprints.contains(variant));
        }
    }

    #[test]
    fn worker_publishes_the_peak_constellation_only_when_asked() {
        let samples = melody_samples(48000, 10.0);
        let worker = FingerprintWorker::spawn(Vec::new(), true, None, None).unwrap();
        assert!(worker.submit(samples.clone(), 48000, 2));
        let output = wait_for(&worker, |o| !o.peak_constellation.is_empty());
        assert_eq!(
            output.peak_constellation,
            crate::constellation::compute_constellation(&samples, 48000, 2)
        );
        assert!(!output.fingerprint.is_empty());
        worker.reset();
        assert!(worker.output().peak_constellation.is_empty());

        let plain = FingerprintWorker::spawn(Vec::new(), false, None, None).unwrap();
        assert!(plain.submit(samples, 48000, 2));
        wait_for(&plain, |o| !o.fingerprint.is_empty());
        std::thread::sleep(std::time::Duration::from_millis(200));
        assert!(plain.output().peak_constellation.is_empty());
    }

    #[test]
    fn take_new_hands_out_every_result_exactly_once_with_its_window() {
        let speeds = speed_grid(2.0, 1.0).unwrap();
        let worker = FingerprintWorker::spawn(speeds.clone(), true, None, None).unwrap();
        let collect_until_done = |worker: &FingerprintWorker| {
            let (mut plain, mut constellations, mut variants, mut window) = (0, 0, Vec::new(), 0);
            for _ in 0..600 {
                let new = worker.take_new();
                plain += !new.fingerprint.is_empty() as usize;
                constellations += !new.peak_constellation.is_empty() as usize;
                variants.extend(new.speed_fingerprints.iter().map(|v| v.speed_percent));
                window = new.window;
                if variants.len() == speeds.len() {
                    break;
                }
                std::thread::sleep(std::time::Duration::from_millis(20));
            }
            (plain, constellations, variants, window)
        };

        assert!(worker.submit(melody_samples(48000, 10.0), 48000, 2));
        let (plain, constellations, variants, first_window) = collect_until_done(&worker);
        assert_eq!((plain, constellations), (1, 1));
        assert_eq!(variants, speeds);
        // Nothing is handed out a second time ...
        assert_eq!(
            worker.take_new(),
            FingerprintOutput {
                window: first_window,
                ..Default::default()
            }
        );
        // ... while the full picture is still there for whoever wants it.
        assert_eq!(worker.output().speed_fingerprints.len(), speeds.len());

        while !worker.submit(melody_samples(48000, 11.0), 48000, 2) {
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
        let (plain, _, variants, second_window) = collect_until_done(&worker);
        assert_eq!((plain, variants.len()), (1, speeds.len()));
        assert_eq!(second_window, first_window + 1);

        // A reset keeps the count going, so the next window cannot be
        // mistaken for more of this one.
        worker.reset();
        assert_eq!(worker.take_new().window, first_window + 1);
    }

    #[test]
    fn spread_paces_the_computations_over_the_given_time() {
        let speeds = speed_grid(2.0, 1.0).unwrap(); // 4 speeds + the plain one = 5 steps
        let worker = FingerprintWorker::spawn(
            speeds,
            false,
            Some(std::time::Duration::from_millis(1500)),
            None,
        )
        .unwrap();
        let started = std::time::Instant::now();
        assert!(worker.submit(melody_samples(48000, 10.0), 48000, 2));
        wait_for(&worker, |o| !o.fingerprint.is_empty());
        assert!(
            started.elapsed().as_millis() < 600,
            "the plain fingerprint must not wait"
        );
        wait_for(&worker, |o| o.speed_fingerprints.len() == 4);
        let took = started.elapsed().as_millis();
        // The last of 5 steps starts at 4/5 of 1.5 s.
        assert!((1200..2500).contains(&took), "all speeds after {took} ms");
    }

    #[test]
    fn worker_without_speeds_publishes_only_the_plain_fingerprint() {
        let worker = FingerprintWorker::spawn(Vec::new(), false, None, None).unwrap();
        assert!(worker.submit(melody_samples(48000, 10.0), 48000, 2));
        let output = wait_for(&worker, |o| !o.fingerprint.is_empty());
        assert!(output.speed_fingerprints.is_empty());
    }

    #[test]
    fn worker_skips_a_window_while_busy_and_drops_results_after_a_reset() {
        let worker =
            FingerprintWorker::spawn(speed_grid(8.0, 1.0).unwrap(), false, None, None).unwrap();
        assert!(worker.submit(melody_samples(48000, 25.0), 48000, 2));
        assert!(
            !worker.submit(melody_samples(48000, 25.0), 48000, 2),
            "second window must be skipped"
        );

        // The session ends while that window is still being worked on.
        worker.reset();
        // The worker gets free again without ever publishing the stale result.
        for _ in 0..600 {
            let output = worker.output();
            assert!(output.fingerprint.is_empty() && output.speed_fingerprints.is_empty());
            if worker.submit(melody_samples(48000, 10.0), 48000, 2) {
                let output = wait_for(&worker, |o| o.speed_fingerprints.len() == 16);
                assert!(!output.fingerprint.is_empty());
                return;
            }
            std::thread::sleep(std::time::Duration::from_millis(50));
        }
        panic!("the worker never became free again");
    }
}
