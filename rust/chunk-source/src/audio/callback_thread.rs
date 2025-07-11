use crate::audio::channels::AudioChannels;
use crate::audio::generic_buffer::AudioBuffer;
use crate::audio::utils::deinterleave_and_convert_to_float;
use anyhow::Result;
use core_affinity;
use log::{error, info, warn};
use ringbuf::traits::*;
use std::sync::Arc;
use std::sync::atomic::AtomicBool;

pub fn start_callback_thread(
    num_input_channels: usize,
    num_output_channels: usize,
    period_size: usize,
    capture_pcm: Option<alsa::pcm::PCM>,
    playback_pcm: Option<alsa::pcm::PCM>,
    mut audio_channels: AudioChannels,
    shutdown_signal: Arc<AtomicBool>,
) -> std::thread::JoinHandle<()> {
    //TODO:
    let dedicated_core_id = 1;

    let mut buffer = vec![0; num_input_channels * period_size];
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
            let mut output_buffer = AudioBuffer::new(num_output_channels, period_size);

            let capture_pcm = capture_pcm.as_ref().unwrap();
            let playback_pcm = playback_pcm.as_ref().unwrap();

            {
                let capture_period_size = capture_pcm.hw_params_current()?.get_period_size()?;
                let playback_period_size = playback_pcm.hw_params_current()?.get_period_size()?;
                let playback_buffer_size = playback_pcm.hw_params_current()?.get_buffer_size()?;

                let swp = playback_pcm.sw_params_current()?;
                swp.set_avail_min(playback_period_size)?;

                let start_threshold = playback_buffer_size;
                swp.set_start_threshold(start_threshold)?;

                playback_pcm.sw_params(&swp)?;

                let swp = capture_pcm.sw_params_current()?;
                swp.set_avail_min(capture_period_size)?;

                swp.set_start_threshold(1)?;
                capture_pcm.sw_params(&swp)?;
            }

            let silence = vec![0i32; playback_pcm.hw_params_current()?.get_buffer_size()? as usize];
            capture_pcm.prepare()?;

            let io_handle = playback_pcm.io_i32()?;
            io_handle.writei(&silence)?;
            drop(io_handle);

            capture_pcm.start()?;

            info!("Starting audio processing loop");
            'main: loop {
                if shutdown_signal.load(std::sync::atomic::Ordering::Relaxed) {
                    break 'main;
                }

                // Wait for capture data to be available
                match capture_pcm.wait(Some(100)) {
                    Ok(true) => {
                        // Try to read from capture device
                        match capture_pcm.io_i32()?.readi(&mut buffer) {
                            Ok(frames) if frames > 0 => {
                                let samples_read = frames * num_input_channels;
                                let interleaved_input = &buffer[..samples_read];
                                let noninterleaved_input =
                                    &mut input_buffer.get_all_mut()[..samples_read];

                                deinterleave_and_convert_to_float(
                                    interleaved_input,
                                    noninterleaved_input,
                                    num_input_channels,
                                );

                                // Push to input channel
                                let samples_pushed = audio_channels
                                    .input_producer
                                    .push_slice(&input_buffer.get_all()[..samples_read]);

                                if samples_pushed != samples_read {
                                    warn!(
                                        "Could not push all samples to input channel: {} of {}",
                                        samples_pushed, samples_read
                                    );
                                }

                                // Try to get data from output channel
                                let samples_received = audio_channels
                                    .output_consumer
                                    .pop_slice(output_buffer.get_all_mut());

                                if samples_received > 0 {
                                    // Convert back to interleaved format and write to playback
                                    let mut interleaved_output = vec![0i32; samples_received];
                                    crate::audio::utils::interleave_and_convert_to_i32(
                                        &output_buffer.get_all()[..samples_received],
                                        &mut interleaved_output,
                                        num_output_channels,
                                    );

                                    match playback_pcm.io_i32()?.writei(&interleaved_output) {
                                        Ok(_) => {}
                                        Err(e) => {
                                            error!("Playback error: {}", e);
                                            if e.errno() == libc::EPIPE {
                                                info!("Playback underrun detected, recovering...");
                                                match playback_pcm.try_recover(e, true) {
                                                    Ok(_) => {
                                                        info!("Playback recovered successfully")
                                                    }
                                                    Err(recovery_err) => {
                                                        error!(
                                                            "Failed to recover playback: {}",
                                                            recovery_err
                                                        );
                                                        playback_pcm.drain()?;
                                                        playback_pcm.prepare()?;
                                                        if playback_pcm.state()
                                                            != alsa::pcm::State::Running
                                                        {
                                                            playback_pcm.start()?;
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
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
                                            error!(
                                                "Failed to recover capture PCM: {}",
                                                recovery_err
                                            );
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
    });
    audio_thread
}
