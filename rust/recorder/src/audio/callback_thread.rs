use crate::audio::channels::CaptureProducer;
use anyhow::Result;
use core_affinity;
use log::{error, info, warn};
use ringbuf::traits::*;
use std::sync::Arc;
use std::sync::atomic::AtomicBool;

const I16_INV_SCALE: f32 = 1.0 / 32768.0;

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
            'main: loop {
                if shutdown_signal.load(std::sync::atomic::Ordering::Relaxed) {
                    break 'main;
                }

                match capture_pcm.wait(Some(100)) {
                    Ok(true) => match capture_pcm.io_i16()?.readi(&mut buffer) {
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
                            error!("Capture error: {}", e);
                            if e.errno() == libc::EPIPE {
                                info!("Capture overrun detected, attempting recovery...");
                                match capture_pcm.try_recover(e, true) {
                                    Ok(_) => {
                                        info!("Capture PCM recovered successfully");
                                        continue;
                                    }
                                    Err(recovery_err) => {
                                        error!("Failed to recover capture PCM: {}", recovery_err);
                                        capture_pcm.drain()?;
                                        capture_pcm.prepare()?;
                                        capture_pcm.start()?;
                                        continue;
                                    }
                                }
                            }
                            return Err(anyhow::anyhow!("Capture error: {}", e));
                        }
                    },
                    Ok(false) => {
                        // Timeout, continue loop
                    }
                    Err(e) => {
                        error!("Wait error: {}", e);
                        return Err(anyhow::anyhow!("Wait error: {}", e));
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
