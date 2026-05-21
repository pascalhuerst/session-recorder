use crate::audio::channels::CaptureProducer;
use anyhow::Result;
use core_affinity;
use log::{error, info, warn};
use ringbuf::traits::*;
use std::sync::Arc;
use std::sync::atomic::AtomicBool;

const I16_INV_SCALE: f32 = 1.0 / 32768.0;

// wait() poll timeout. At 48 kHz / 512-frame periods a period is ready roughly
// every ~10 ms, so a 100 ms timeout already means the stream produced nothing.
const WAIT_TIMEOUT_MS: u32 = 100;
// If the stream is RUNNING but produces no data for this many consecutive
// timeouts (~2 s), treat it as wedged and reinitialize.
const STALL_TIMEOUTS: u32 = 20;

// Bring the capture PCM back to a started state from XRUN / SETUP / SUSPENDED /
// PREPARED. Safe to call whenever the stream is not actively RUNNING.
fn reinit_capture(pcm: &alsa::pcm::PCM) -> Result<()> {
    pcm.prepare()?;
    pcm.start()?;
    Ok(())
}

pub fn start_callback_thread(
    num_input_channels: usize,
    period_size: usize,
    capture_pcm: alsa::pcm::PCM,
    mut capture: CaptureProducer,
    shutdown_signal: Arc<AtomicBool>,
) -> std::thread::JoinHandle<()> {
    //TODO:
    let dedicated_core_id = 1;

    let mut buffer = vec![0i16; num_input_channels * period_size];

    std::thread::spawn(move || {
        if !core_affinity::set_for_current(core_affinity::CoreId {
            id: dedicated_core_id,
        }) {
            warn!(
                "Failed to set affinity for callback thread to core {}",
                dedicated_core_id
            );
        } else {
            info!("Set callback thread affinity to core {}", dedicated_core_id);
        }

        let mut callback_thread = move || -> Result<()> {
            // Per-period f32 scratch buffer, kept interleaved (LRLRLR…) to
            // match the wire format the Go ChunkSink server expects.
            let mut float_buffer = vec![0.0f32; num_input_channels * period_size];

            {
                let capture_period_size = capture_pcm.hw_params_current()?.get_period_size()?;
                let swp = capture_pcm.sw_params_current()?;
                swp.set_avail_min(capture_period_size)?;
                swp.set_start_threshold(1)?;
                capture_pcm.sw_params(&swp)?;
            }

            capture_pcm.prepare()?;
            capture_pcm.start()?;

            info!("Starting audio processing loop");
            // Counts consecutive wait() timeouts while the stream claims to be
            // RUNNING; a sustained run means the stream is wedged.
            let mut stall_timeouts: u32 = 0;

            'main: loop {
                if shutdown_signal.load(std::sync::atomic::Ordering::Relaxed) {
                    break 'main;
                }

                match capture_pcm.wait(Some(WAIT_TIMEOUT_MS)) {
                    Ok(true) => {
                        stall_timeouts = 0;

                        let io = match capture_pcm.io_i16() {
                            Ok(io) => io,
                            Err(e) => {
                                warn!("Cannot obtain capture IO handle: {}; reinitializing", e);
                                if let Err(re) = reinit_capture(&capture_pcm) {
                                    error!("Capture reinit failed: {}", re);
                                }
                                continue;
                            }
                        };

                        match io.readi(&mut buffer) {
                            Ok(frames) if frames > 0 => {
                                let samples_read = frames * num_input_channels;

                                // Convert in-place, preserving interleaved layout.
                                for (dst, &src) in float_buffer[..samples_read]
                                    .iter_mut()
                                    .zip(buffer[..samples_read].iter())
                                {
                                    *dst = src as f32 * I16_INV_SCALE;
                                }

                                let samples_pushed =
                                    capture.producer.push_slice(&float_buffer[..samples_read]);

                                if samples_pushed != samples_read {
                                    warn!(
                                        "Could not push all samples to capture ring: {} of {}",
                                        samples_pushed, samples_read
                                    );
                                }
                            }
                            Ok(_) => {
                                // No frames read, continue
                            }
                            Err(e) => {
                                // try_recover handles EPIPE (overrun) and
                                // ESTRPIPE (suspend). For anything else, fall
                                // back to a full prepare/start. Never return —
                                // a dead capture thread silently freezes the
                                // whole recorder.
                                warn!("Capture read error: {} (errno {})", e, e.errno());
                                if capture_pcm.try_recover(e, true).is_err()
                                    && let Err(re) = reinit_capture(&capture_pcm)
                                {
                                    error!("Capture reinit failed: {}", re);
                                }
                            }
                        }
                    }
                    Ok(false) => {
                        // Timeout: no data within WAIT_TIMEOUT_MS. If the stream
                        // is no longer RUNNING it has fallen over without a read
                        // error ever surfacing (the classic silent stall) —
                        // reinitialize immediately. If it still claims RUNNING,
                        // tolerate a brief run before forcing a reinit.
                        let state = capture_pcm.state();
                        if state == alsa::pcm::State::Running {
                            stall_timeouts += 1;
                            if stall_timeouts >= STALL_TIMEOUTS {
                                warn!(
                                    "Capture produced no data for ~{} ms while RUNNING; reinitializing",
                                    STALL_TIMEOUTS * WAIT_TIMEOUT_MS
                                );
                                if let Err(re) = reinit_capture(&capture_pcm) {
                                    error!("Capture reinit failed: {}", re);
                                }
                                stall_timeouts = 0;
                            }
                        } else {
                            warn!("Capture stalled (state={:?}); reinitializing", state);
                            if let Err(re) = reinit_capture(&capture_pcm) {
                                error!("Capture reinit failed: {}", re);
                            }
                            stall_timeouts = 0;
                        }
                    }
                    Err(e) => {
                        warn!("Capture wait error: {}; reinitializing", e);
                        if let Err(re) = reinit_capture(&capture_pcm) {
                            error!("Capture reinit failed: {}", re);
                        }
                        stall_timeouts = 0;
                    }
                }
            }
            info!("Callback thread is shutting down");
            Ok(())
        };
        if let Err(e) = callback_thread() {
            error!("Audio processing thread error: {}", e);
        }
    })
}
