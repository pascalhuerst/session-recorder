//! Development aid: the peak constellation of a raw s16le file, as the
//! recorder would compute it. `constellation_dump <file> <rate> <channels> [repeat]`
//! writes the SIC1 bytes to stdout and the time taken to stderr.
use std::io::Write;
use std::time::Instant;

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let bytes = std::fs::read(&args[1]).expect("cannot read the file");
    let (rate, channels): (u32, u32) = (args[2].parse().unwrap(), args[3].parse().unwrap());
    let repeat: usize = args.get(4).map(|s| s.parse().unwrap()).unwrap_or(1);
    let samples: Vec<i16> = bytes
        .chunks_exact(2)
        .map(|b| i16::from_le_bytes([b[0], b[1]]))
        .collect();
    let mut packed = Vec::new();
    for _ in 0..repeat {
        let start = Instant::now();
        packed = recorder::constellation::compute_constellation(&samples, rate, channels);
        eprintln!(
            "{:.1} s of audio -> {} peaks in {} ms",
            samples.len() as f64 / (rate * channels) as f64,
            packed.len().saturating_sub(8) / 5,
            start.elapsed().as_millis()
        );
    }
    std::io::stdout().write_all(&packed).unwrap();
}
