//! Live tempo (BPM) of recently captured audio, with TempoCNN.
//!
//! TempoCNN (Schreiber & Müller, ISMIR 2018) is a small convolutional network
//! that classifies about twelve seconds of audio into one of 256 tempo
//! classes, 30 to 285 BPM. It is what DiscQL uses for its library (through
//! Essentia) because it also gets broken beats right, where onset- and
//! autocorrelation-based estimators land on half, double or two thirds of the
//! tempo: on 25 s excerpts of 130 tracks tagged breakbeat, drum'n'bass,
//! jungle and the like it agreed with the whole-track value in 96 %, a
//! tempogram-based estimator in 65 %.
//!
//! This module runs the same model (Essentia's `deeptemp-k16-3`, converted to
//! ONNX - see the recorder's README) with `tract`, a pure-Rust inference
//! engine, and rebuilds the input chain of Essentia's
//! `TensorflowPredictTempoCNN` around it:
//!
//!   audio at 11025 Hz mono
//!   -> frames of 1024 samples every 512, the first centred on sample 0,
//!      zeros outside the audio
//!   -> symmetric Hann window -> magnitude spectrum
//!   -> 40 triangular mel bands, 20 Hz - 5 kHz (Slaney's mel scale, each
//!      triangle scaled to unit area)
//!   -> patches of 256 frames every 128 (a rest at the end is dropped)
//!   -> each patch shifted and scaled to mean 0, standard deviation 1
//!   -> the model, one softmax over the 256 classes per patch
//!   -> the classes' mean over all patches; the largest is the tempo
//!
//! Checked against Essentia on 250 excerpts of 25 s: the same BPM in all of
//! them, probabilities equal to 5e-5. Every constant here is part of that
//! agreement.

use realfft::RealFftPlanner;
use std::path::Path;
use tract_onnx::prelude::*;

/// The model's input rate; the same as the peak constellation's.
pub const RATE: u32 = 11025;
const FRAME: usize = 1024;
const HOP: usize = 512;
const SPECTRUM: usize = FRAME / 2 + 1;
const BANDS: usize = 40;
const BAND_LOW_HZ: f64 = 20.0;
const BAND_HIGH_HZ: f64 = 5000.0;
const PATCH: usize = 256;
const PATCH_HOP: usize = 128;
const CLASSES: usize = 256;
/// Class 0 is this tempo, each further class one BPM more.
const MIN_BPM: f32 = 30.0;

/// The file name the model is looked for under when no path is given.
pub const MODEL_FILE_NAME: &str = "deeptemp-k16-3.onnx";

/// Where the model is looked for when no path is given, most specific
/// first: the user's data directory, then the machine's.
pub fn default_model_paths() -> Vec<std::path::PathBuf> {
    let user_data = std::env::var_os("XDG_DATA_HOME")
        .map(std::path::PathBuf::from)
        .filter(|p| p.is_absolute())
        .or_else(|| std::env::var_os("HOME").map(|home| Path::new(&home).join(".local/share")));
    user_data
        .into_iter()
        .chain([
            Path::new("/usr/local/share").to_path_buf(),
            Path::new("/usr/share").to_path_buf(),
        ])
        .map(|dir| dir.join("session-recorder").join(MODEL_FILE_NAME))
        .collect()
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub struct Tempo {
    pub bpm: f32,
    /// The model's (mean) probability for that tempo, 0..1.
    pub probability: f32,
}

type Model = SimplePlan<TypedFact, Box<dyn TypedOp>, Graph<TypedFact, Box<dyn TypedOp>>>;

pub struct TempoEstimator {
    model: Model,
    filterbank: Vec<f32>, // BANDS x SPECTRUM
    window: Vec<f32>,
}

impl TempoEstimator {
    /// Loads the TempoCNN model from an ONNX file with input
    /// `[1, 40 bands, 256 frames, 1]` and 256 class probabilities as output.
    pub fn load(path: &Path) -> Result<Self, String> {
        let model = tract_onnx::onnx()
            .model_for_path(path)
            .and_then(|m| m.with_input_fact(0, f32::fact([1, BANDS, PATCH, 1]).into()))
            .and_then(|m| m.into_optimized())
            .and_then(|m| m.into_runnable())
            .map_err(|e| format!("cannot load the tempo model {}: {e:#}", path.display()))?;
        Ok(Self {
            model,
            filterbank: mel_filterbank(),
            // Symmetric Hann: zero at both ends.
            window: (0..FRAME)
                .map(|n| {
                    (0.5 - 0.5 * (2.0 * std::f64::consts::PI * n as f64 / (FRAME - 1) as f64).cos())
                        as f32
                })
                .collect(),
        })
    }

    /// The tempo of `audio` (mono at `RATE` Hz, any scale). None if it is
    /// shorter than one patch (about 12 s) or silent throughout.
    pub fn estimate(&self, audio: &[f32]) -> Option<Tempo> {
        let mel = self.mel_frames(audio);
        let mut mean = [0f32; CLASSES];
        let mut patches = 0usize;
        let mut start = 0;
        while start + PATCH <= mel.len() {
            if let Some(input) = standardized_patch(&mel[start..start + PATCH]) {
                let output = self.model.run(tvec!(input.into())).ok()?;
                let probabilities = output[0].to_array_view::<f32>().ok()?;
                if probabilities.len() != CLASSES {
                    return None;
                }
                for (acc, p) in mean.iter_mut().zip(probabilities.iter()) {
                    *acc += p;
                }
                patches += 1;
            }
            start += PATCH_HOP;
        }
        if patches == 0 {
            return None;
        }
        let (class, sum) =
            mean.iter().enumerate().fold(
                (0, f32::MIN),
                |best, (i, &v)| if v > best.1 { (i, v) } else { best },
            );
        Some(Tempo {
            bpm: MIN_BPM + class as f32,
            probability: sum / patches as f32,
        })
    }

    fn mel_frames(&self, audio: &[f32]) -> Vec<[f32; BANDS]> {
        let fft = RealFftPlanner::<f32>::new().plan_fft_forward(FRAME);
        let (mut frame, mut spectrum) = (fft.make_input_vec(), fft.make_output_vec());
        let mut magnitude = [0f32; SPECTRUM];
        let mut out = Vec::with_capacity(audio.len() / HOP + 2);
        // Frame i covers samples [i * HOP - HOP, i * HOP + HOP).
        let mut i = 0usize;
        while i * HOP < audio.len() + HOP {
            let start = i as isize * HOP as isize - HOP as isize;
            for (n, x) in frame.iter_mut().enumerate() {
                let k = start + n as isize;
                *x = if k >= 0 && (k as usize) < audio.len() {
                    audio[k as usize] * self.window[n]
                } else {
                    0.0
                };
            }
            fft.process(&mut frame, &mut spectrum)
                .expect("buffers come from the plan itself");
            for (m, c) in magnitude.iter_mut().zip(&spectrum) {
                *m = (c.re * c.re + c.im * c.im).sqrt();
            }
            let mut bands = [0f32; BANDS];
            for (b, band) in bands.iter_mut().enumerate() {
                *band = self.filterbank[b * SPECTRUM..(b + 1) * SPECTRUM]
                    .iter()
                    .zip(&magnitude)
                    .map(|(w, m)| w * m)
                    .sum();
            }
            out.push(bands);
            i += 1;
        }
        out
    }
}

/// One patch as the model's input tensor `[1, bands, frames, 1]`, shifted and
/// scaled to mean 0 and (population) standard deviation 1 over all its
/// values. None for a patch without any variation - digital silence.
fn standardized_patch(patch: &[[f32; BANDS]]) -> Option<Tensor> {
    let n = (patch.len() * BANDS) as f64;
    let mean = patch.iter().flatten().map(|&v| v as f64).sum::<f64>() / n;
    let variance = patch
        .iter()
        .flatten()
        .map(|&v| (v as f64 - mean).powi(2))
        .sum::<f64>()
        / n;
    let deviation = variance.sqrt();
    if !(deviation > 0.0) {
        return None;
    }
    Some(Tensor::from(tract_ndarray::Array4::from_shape_fn(
        (1, BANDS, patch.len(), 1),
        |(_, band, frame, _)| ((patch[frame][band] as f64 - mean) / deviation) as f32,
    )))
}

/// Slaney's mel scale: linear below 1 kHz, logarithmic above.
fn hz_to_mel(hz: f64) -> f64 {
    if hz < 1000.0 {
        hz / (200.0 / 3.0)
    } else {
        15.0 + (hz / 1000.0).ln() / (6.4f64.ln() / 27.0)
    }
}

fn mel_to_hz(mel: f64) -> f64 {
    if mel < 15.0 {
        mel * (200.0 / 3.0)
    } else {
        1000.0 * ((mel - 15.0) * (6.4f64.ln() / 27.0)).exp()
    }
}

/// `BANDS` triangles over the spectrum's bins, edges equally spaced in mel,
/// each scaled to unit area (Essentia's `MelBands` with `slaneyMel` and
/// `unit_tri`; equal to its weights to 1e-7).
fn mel_filterbank() -> Vec<f32> {
    let (low, high) = (hz_to_mel(BAND_LOW_HZ), hz_to_mel(BAND_HIGH_HZ));
    let edges: Vec<f64> = (0..BANDS + 2)
        .map(|k| mel_to_hz(low + (high - low) * k as f64 / (BANDS + 1) as f64))
        .collect();
    let bin_hz = RATE as f64 / 2.0 / (SPECTRUM - 1) as f64;
    let mut filterbank = vec![0f32; BANDS * SPECTRUM];
    for b in 0..BANDS {
        let (l, c, h) = (edges[b], edges[b + 1], edges[b + 2]);
        for k in 0..SPECTRUM {
            let f = k as f64 * bin_hz;
            let weight = ((f - l) / (c - l)).min((h - f) / (h - c)).max(0.0);
            filterbank[b * SPECTRUM + k] = (weight * 2.0 / (h - l)) as f32;
        }
    }
    filterbank
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn filterbank_matches_essentias_weights() {
        let fb = mel_filterbank();
        let at = |band: usize, bin: usize| fb[band * SPECTRUM + bin];
        // Spot values and supports read from Essentia's MelBands (40 bands,
        // 11025 Hz, 20-5000 Hz, slaneyMel, unit_tri, 513 bins).
        assert!((at(0, 3) - 0.003_203_201).abs() < 1e-7);
        assert!((at(39, 440) - 0.002_829_433).abs() < 1e-7);
        assert_eq!(at(20, 150), 0.0);
        let support = |band: usize| {
            let nonzero: Vec<usize> = (0..SPECTRUM).filter(|&k| at(band, k) > 0.0).collect();
            (nonzero[0], *nonzero.last().unwrap())
        };
        assert_eq!(support(0), (2, 13));
        assert_eq!(support(1), (8, 19));
        assert_eq!(support(39), (409, 464));
    }

    #[test]
    fn a_patch_is_standardized_and_laid_out_bands_first() {
        let mut patch = vec![[0f32; BANDS]; PATCH];
        for (t, frame) in patch.iter_mut().enumerate() {
            for (b, v) in frame.iter_mut().enumerate() {
                *v = (t * 3 + b * 7 % 11) as f32 * 1000.0; // any scale
            }
        }
        let tensor = standardized_patch(&patch).unwrap();
        assert_eq!(tensor.shape(), &[1, BANDS, PATCH, 1]);
        let values = tensor.to_array_view::<f32>().unwrap();
        let n = values.len() as f64;
        let mean = values.iter().map(|&v| v as f64).sum::<f64>() / n;
        let variance = values
            .iter()
            .map(|&v| (v as f64 - mean).powi(2))
            .sum::<f64>()
            / n;
        assert!(mean.abs() < 1e-4 && (variance - 1.0).abs() < 1e-3);
        // [0, band, frame, 0] is frame `frame`'s band `band`.
        let expected_order = patch[5][2] < patch[6][2];
        assert_eq!(values[[0, 2, 5, 0]] < values[[0, 2, 6, 0]], expected_order);
    }

    #[test]
    fn the_model_is_looked_for_in_the_users_data_directory_first() {
        let paths = default_model_paths();
        assert!(paths.len() >= 2);
        assert!(paths
            .iter()
            .all(|p| p.ends_with("session-recorder/deeptemp-k16-3.onnx")));
        assert_eq!(
            paths[paths.len() - 1],
            Path::new("/usr/share/session-recorder/deeptemp-k16-3.onnx")
        );
        assert!(
            !paths[0].starts_with("/usr"),
            "the user's own directory comes before the machine's"
        );
    }

    #[test]
    fn a_patch_without_variation_is_refused() {
        assert!(standardized_patch(&vec![[0.25f32; BANDS]; PATCH]).is_none());
    }

    /// With the model at hand (`TEMPO_MODEL_TEST=/path/to/deeptemp-k16-3.onnx
    /// cargo test`): a click track is recognized at its tempo, at any level.
    /// Skipped otherwise - the model file is not part of the repository.
    #[test]
    fn a_click_track_is_recognized_at_its_tempo() {
        let Ok(path) = std::env::var("TEMPO_MODEL_TEST") else {
            eprintln!("TEMPO_MODEL_TEST not set - skipping the test that needs the model");
            return;
        };
        let estimator = TempoEstimator::load(Path::new(&path)).unwrap();
        for (bpm, level) in [(120.0f64, 0.5f32), (140.0, 12000.0), (100.0, 1.0)] {
            let period = RATE as f64 * 60.0 / bpm;
            let audio: Vec<f32> = (0..RATE as usize * 25)
                .map(|n| {
                    // a short decaying burst on every beat, a softer one in between
                    let phase = n as f64 % period;
                    let half = (n as f64 + period / 2.0) % period;
                    let burst = |p: f64, gain: f64| gain * (-p / 300.0).exp() * (p * 0.9).sin();
                    (burst(phase, 1.0) + burst(half, 0.4)) as f32 * level
                })
                .collect();
            let tempo = estimator.estimate(&audio).unwrap();
            assert!(
                (tempo.bpm as f64 - bpm).abs() <= 1.0,
                "{bpm} BPM clicks came out as {tempo:?}"
            );
        }
        assert!(
            estimator.estimate(&vec![0.0; RATE as usize * 25]).is_none(),
            "silence has no tempo"
        );
        assert!(
            estimator.estimate(&vec![0.1; RATE as usize * 5]).is_none(),
            "5 s is too short"
        );
    }
}
