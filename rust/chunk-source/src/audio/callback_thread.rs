use crate::audio::generic_buffer::AudioBuffer;
use crate::channels::AudioRingBufferProducer;
use anyhow::Result;
use core_affinity;
use log::{error, info, warn};
use ringbuf::traits::*;
use std::sync::Arc;
use std::sync::atomic::AtomicBool;

pub fn start_callback_thread(
    num_input_channels: usize,
    period_size: usize,
    capture_pcm: Option<alsa::pcm::PCM>,
    shutdown_signal: Arc<AtomicBool>,
    mut producer: AudioRingBufferProducer,
) -> std::thread::JoinHandle<()> {
    let dedicated_core_id = 1;

    let audio_thread = std::thread::spawn(move || {
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
            let mut input_buffer = AudioBuffer::new(num_input_channels, period_size);

            let capture_pcm = capture_pcm.as_ref().unwrap();

            {
                let capture_period_size = capture_pcm.hw_params_current()?.get_period_size()?;
                let swp = capture_pcm.sw_params_current()?;
                swp.set_avail_min(capture_period_size)?;

                swp.set_start_threshold(1)?;
                capture_pcm.sw_params(&swp)?;
            }

            capture_pcm.prepare()?;
            capture_pcm.start()?;

            info!("Starting audio recording loop");
            loop {
                if shutdown_signal.load(std::sync::atomic::Ordering::Relaxed) {
                    break;
                }

                match capture_pcm.io_i16()?.readi(&mut input_buffer.get_all_mut()) {
                    Ok(frames) if frames > 0 => {
                        producer.push_slice(input_buffer.get_all());
                    }
                    Ok(_) => {}
                    Err(e) => {
                        if e.errno() == libc::EPIPE {
                            match capture_pcm.try_recover(e, true) {
                                Ok(_) => {
                                    continue;
                                }
                                Err(_) => {
                                    capture_pcm.drain()?;
                                    capture_pcm.prepare()?;
                                    capture_pcm.start()?;

                                    continue;
                                }
                            }
                        }
                        return Err(anyhow::anyhow!("Capture error: {}", e));
                    }
                }
            }
            info!("Callback thread is shutting down");
            Ok(())
        };
        if let Err(e) = callback_thread() {
            error!("Audio processing thread error: {}", e);
        }
    });
    audio_thread
}
