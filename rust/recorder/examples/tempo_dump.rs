//! Development aid: the tempo of raw s16le files, as the recorder would
//! estimate it. `tempo_dump <model.onnx> <rate> <channels> <file>...` prints
//! `<file>\t<bpm>\t<probability>\t<ms>` per file.
use std::time::Instant;

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let estimator = recorder::tempo::TempoEstimator::load(std::path::Path::new(&args[1])).unwrap();
    let (rate, channels): (u32, u32) = (args[2].parse().unwrap(), args[3].parse().unwrap());
    for path in &args[4..] {
        let bytes = std::fs::read(path).expect("cannot read the file");
        let samples: Vec<i16> = bytes
            .chunks_exact(2)
            .map(|b| i16::from_le_bytes([b[0], b[1]]))
            .collect();
        let start = Instant::now();
        let mono = recorder::constellation::mono_at_rate(&samples, rate, channels);
        match estimator.estimate(&mono) {
            Some(t) => println!(
                "{path}\t{}\t{:.4}\t{}",
                t.bpm,
                t.probability,
                start.elapsed().as_millis()
            ),
            None => println!("{path}\tnone\t0\t{}", start.elapsed().as_millis()),
        }
    }
}
