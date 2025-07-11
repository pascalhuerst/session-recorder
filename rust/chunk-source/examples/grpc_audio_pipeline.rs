//! Example demonstrating integration of gRPC client with audio pipeline
//!
//! This example shows how to use the ChunkSinkClientService with the audio
//! processing pipeline to send audio data to a remote server and receive
//! commands back.

use chunk_source::audio::{
    alsa::{AudioSettings, configure_input_device, configure_output_device},
    callback_thread::start_callback_thread,
    channels::{AudioChannelPair, ParameterChannelPair, Parameters},
};
use chunk_source::grpc::chunk_sink_client::{
    ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
};
use log::{error, info, warn};
use ringbuf::traits::{Consumer, Producer};
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::Duration;
use tokio::time::sleep;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logger
    env_logger::init();

    info!("Starting gRPC audio pipeline example");

    // Audio configuration
    let audio_settings = AudioSettings {
        input_device: "default".to_string(),
        output_device: "default".to_string(),
        num_channels: 2,
        period_size: 512,
        buffer_size: 2048,
        sample_rate: 44100,
    };

    // gRPC client configuration
    let grpc_config = ChunkSinkConfig {
        server_address: "http://localhost:50051".to_string(),
        recorder_id: "audio-recorder-001".to_string(),
        recorder_name: "Audio Recorder Example".to_string(),
        connect_timeout: Duration::from_secs(10),
        request_timeout: Duration::from_secs(5),
        retry_interval: Duration::from_secs(3),
        max_retries: 5,
        audio_buffer_size: 8192,
        parameter_buffer_size: 64,
    };

    // Create and initialize gRPC client
    let mut grpc_client = ChunkSinkClientService::new(grpc_config);
    grpc_client.initialize_channels();

    // Try to connect to gRPC server
    match grpc_client.connect_with_retry().await {
        Ok(_) => {
            info!("Connected to gRPC server successfully");
        }
        Err(e) => {
            warn!("Failed to connect to gRPC server: {}", e);
            info!("Continuing with local audio processing only");
        }
    }

    // Create audio channels for the pipeline
    let audio_channel_pair = AudioChannelPair::new(audio_settings.buffer_size as usize * 4);
    let parameter_channel_pair = ParameterChannelPair::new(64);

    // Create audio devices
    let capture_pcm =
        configure_input_device(&audio_settings).expect("Failed to configure input device");
    let playback_pcm =
        configure_output_device(&audio_settings).expect("Failed to configure output device");

    // Shutdown signal
    let shutdown_signal = Arc::new(AtomicBool::new(false));

    // Start audio callback thread
    let callback_handle = start_callback_thread(
        audio_settings.num_channels as usize,
        audio_settings.num_channels as usize,
        audio_settings.period_size as usize,
        Some(capture_pcm),
        Some(playback_pcm),
        audio_channel_pair.callback_channels,
        shutdown_signal.clone(),
    );

    // Start gRPC processing task
    let grpc_shutdown = shutdown_signal.clone();
    let grpc_handle = tokio::spawn(async move {
        let mut audio_buffer = vec![0.0f32; 1024];
        let mut param_buffer = [Parameters::Cut(); 4];
        let mut chunk_counter = 0u32;

        info!("Starting gRPC processing loop");

        // Start command listener
        if let Err(e) = grpc_client.start_command_listener().await {
            error!("Failed to start command listener: {}", e);
        }

        while !grpc_shutdown.load(Ordering::Relaxed) {
            // Send recorder status periodically
            if chunk_counter % 100 == 0 {
                let status = RecorderStatusInfo {
                    signal_status:
                        chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                    rms_percent: 0.6,
                    clipping: false,
                };

                if let Err(e) = grpc_client.set_recorder_status_with_retry(status).await {
                    error!("Failed to send recorder status: {}", e);
                }
            }

            // Check for incoming parameters/commands
            match grpc_client.receive_parameters(&mut param_buffer) {
                Ok(params_received) => {
                    if params_received > 0 {
                        for i in 0..params_received {
                            match param_buffer[i] {
                                Parameters::Cut() => {
                                    info!("Received CUT command from server");
                                    // Handle cut command
                                }
                                Parameters::Shutdown() => {
                                    info!("Received SHUTDOWN command from server");
                                    grpc_shutdown.store(true, Ordering::Relaxed);
                                    break;
                                }
                            }
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to receive parameters: {}", e);
                }
            }

            // Process audio chunks (simulate getting audio data)
            // In a real implementation, this would come from the audio pipeline
            if chunk_counter % 10 == 0 {
                // Generate some test audio data
                for i in 0..audio_buffer.len() {
                    audio_buffer[i] = (chunk_counter as f32 * 0.001 + i as f32 * 0.01).sin() * 0.1;
                }

                // Send audio data to gRPC pipeline
                match grpc_client.send_audio_data(&audio_buffer) {
                    Ok(samples_sent) => {
                        if samples_sent > 0 {
                            info!("Sent {} audio samples to gRPC pipeline", samples_sent);
                        }
                    }
                    Err(e) => {
                        error!("Failed to send audio data: {}", e);
                    }
                }
            }

            chunk_counter += 1;
            sleep(Duration::from_millis(10)).await;
        }

        info!("gRPC processing loop finished");
    });

    // Audio processing loop in main thread
    let main_shutdown = shutdown_signal.clone();
    let mut main_channels = audio_channel_pair.main_channels;
    let audio_handle = thread::spawn(move || {
        let mut audio_buffer = vec![
            0.0f32;
            audio_settings.num_channels as usize
                * audio_settings.period_size as usize
        ];

        info!("Starting main audio processing loop");

        while !main_shutdown.load(Ordering::Relaxed) {
            // Process audio data from the callback thread
            let samples_read = main_channels.output_consumer.pop_slice(&mut audio_buffer);

            if samples_read > 0 {
                // Apply audio processing here (effects, filters, etc.)
                // For this example, we just pass the audio through

                // Send processed audio back to the callback thread
                let mut attempts = 0;
                while main_channels
                    .input_producer
                    .push_slice(&audio_buffer[..samples_read])
                    != samples_read
                    && attempts < 10
                {
                    thread::sleep(Duration::from_micros(100));
                    attempts += 1;
                }
            } else {
                // No audio data, yield CPU
                thread::sleep(Duration::from_micros(100));
            }
        }

        info!("Main audio processing loop finished");
    });

    // Set up graceful shutdown
    let shutdown_signal_clone = shutdown_signal.clone();
    ctrlc::set_handler(move || {
        info!("Received Ctrl+C signal, shutting down...");
        shutdown_signal_clone.store(true, Ordering::Relaxed);
    })
    .expect("Error setting Ctrl+C handler");

    info!("gRPC audio pipeline example running. Press Ctrl+C to stop.");

    // Wait for shutdown signal
    while !shutdown_signal.load(Ordering::Relaxed) {
        thread::sleep(Duration::from_millis(100));
    }

    info!("Shutting down...");

    // Wait for all threads to finish
    if let Err(e) = callback_handle.join() {
        error!("Error joining callback thread: {:?}", e);
    }

    if let Err(e) = audio_handle.join() {
        error!("Error joining audio thread: {:?}", e);
    }

    if let Err(e) = grpc_handle.await {
        error!("Error joining gRPC task: {:?}", e);
    }

    info!("Shutdown complete");
    Ok(())
}
