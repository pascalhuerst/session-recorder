use chunk_source::audio::{
    alsa::{AudioSettings, configure_input_device, configure_output_device},
    callback_thread::start_callback_thread,
    channels::AudioChannelPair,
};
use chunk_source::grpc::chunk_sink_client::{
    ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
};
use chunk_source::io::{input_key::InputKey, led::Led};
use evdev::KeyCode;
use log::info;
use ringbuf::traits::{Consumer, Producer};
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::Duration;

#[tokio::main]
async fn main() {
    // Initialize logger
    env_logger::init();

    info!("Starting session recorder with audio, input key, and LED support");

    // Test LED functionality
    test_led_functionality();

    // Test input key functionality
    test_input_key_functionality();

    // Test gRPC client functionality
    test_grpc_client_functionality().await;

    // Create Audio Settings
    let audio_settings = AudioSettings {
        input_device: "default".to_string(),
        output_device: "default".to_string(),
        num_channels: 2,
        period_size: 512,
        buffer_size: 2048,
        sample_rate: 44100,
    };

    info!("Audio settings: {:?}", audio_settings);

    // Create Capture Device
    let capture_pcm =
        configure_input_device(&audio_settings).expect("Failed to configure input device");

    // Create Playback device
    let playback_pcm =
        configure_output_device(&audio_settings).expect("Failed to configure output device");

    // Create channels and shutdown signal for callback thread
    let channel_pair = AudioChannelPair::new(audio_settings.buffer_size as usize * 4);
    let shutdown_signal = Arc::new(AtomicBool::new(false));

    // Create callback thread and run it
    let callback_handle = start_callback_thread(
        audio_settings.num_channels as usize,
        audio_settings.num_channels as usize,
        audio_settings.period_size as usize,
        Some(capture_pcm),
        Some(playback_pcm),
        channel_pair.callback_channels,
        shutdown_signal.clone(),
    );

    info!("Audio callback thread started");

    // Run a loop in main thread, that copies data from producer channel to the consumer channel,
    // so we create a loopback, but with audio data
    let loopback_shutdown = shutdown_signal.clone();
    let mut main_channels = channel_pair.main_channels;

    let loopback_handle = thread::spawn(move || {
        let mut buffer = vec![
            0.0f32;
            audio_settings.num_channels as usize
                * audio_settings.period_size as usize
        ];

        info!("Starting loopback thread");
        let mut loop_count = 0;

        while !loopback_shutdown.load(Ordering::Relaxed) {
            // Try to read from input channel
            let samples_read = main_channels.output_consumer.pop_slice(&mut buffer);
            if samples_read > 0 {
                // Process the audio data here (currently just pass through)
                // In a real application, you would apply effects, filters, etc.

                // Write to output channel
                let mut attempts = 0;
                while main_channels
                    .input_producer
                    .push_slice(&buffer[..samples_read])
                    != samples_read
                    && attempts < 10
                {
                    thread::sleep(Duration::from_micros(100));
                    attempts += 1;
                }

                if attempts >= 10 {
                    log::warn!("Failed to push audio data to output channel after 10 attempts");
                }
            } else {
                // No data available, sleep briefly to avoid busy waiting
                thread::sleep(Duration::from_micros(100));
                loop_count += 1;
                if loop_count % 100000 == 0 {
                    info!("Loopback: Audio processing active... ({})", loop_count);
                }
            }
        }

        info!("Loopback thread shutting down");
    });

    // Wait for Ctrl+C or other shutdown signal
    info!("Audio loopback running. Press Ctrl+C to stop.");

    // Set up signal handler for graceful shutdown
    let shutdown_signal_clone = shutdown_signal.clone();
    ctrlc::set_handler(move || {
        info!("Received Ctrl+C signal, shutting down...");
        shutdown_signal_clone.store(true, Ordering::Relaxed);
    })
    .expect("Error setting Ctrl+C handler");

    // Wait for shutdown signal
    while !shutdown_signal.load(Ordering::Relaxed) {
        thread::sleep(Duration::from_millis(100));
    }

    info!("Shutting down...");
    shutdown_signal.store(true, Ordering::Relaxed);

    // Wait for threads to finish
    if let Err(e) = callback_handle.join() {
        eprintln!("Error joining callback thread: {:?}", e);
    }

    if let Err(e) = loopback_handle.join() {
        eprintln!("Error joining loopback thread: {:?}", e);
    }

    info!("Shutdown complete");
}

fn test_led_functionality() {
    info!("Testing LED functionality...");

    // List available LEDs
    match Led::list_available() {
        Ok(leds) => {
            info!("Available LEDs: {:?}", leds);

            // Try to control the first LED if available
            if let Some(led_name) = leds.first() {
                match Led::new(led_name) {
                    Ok(led) => {
                        info!("Successfully created LED controller for: {}", led_name);
                        info!("LED info: max_brightness={}", led.max_brightness());

                        // Test turning LED on and off
                        if let Err(e) = led.on() {
                            log::warn!("Failed to turn LED on: {}", e);
                        } else {
                            info!("LED turned on");
                        }

                        thread::sleep(Duration::from_millis(500));

                        if let Err(e) = led.off() {
                            log::warn!("Failed to turn LED off: {}", e);
                        } else {
                            info!("LED turned off");
                        }
                    }
                    Err(e) => {
                        log::warn!("Failed to create LED controller: {}", e);
                    }
                }
            } else {
                info!("No LEDs available for testing");
            }
        }
        Err(e) => {
            log::warn!("Failed to list LEDs: {}", e);
        }
    }
}

fn test_input_key_functionality() {
    info!("Testing input key functionality...");

    // List available input devices
    let devices = InputKey::list_devices();
    info!("Available input devices: {:?}", devices);

    // Try to create an input key handler for the first device
    if let Some((device_path, device_name)) = devices.first() {
        match InputKey::from_path(device_path) {
            Ok(mut input_key) => {
                info!(
                    "Successfully created input key handler for: {}",
                    device_name
                );

                // Register a key handler for the space key
                input_key.register_key(
                    KeyCode::KEY_SPACE,
                    || {
                        info!("Space key pressed!");
                    },
                    |duration| {
                        info!("Space key released after {:?}", duration);
                    },
                );

                // Start monitoring (but don't actually start since we're in a test)
                info!("Input key handler configured (not started for testing)");
            }
            Err(e) => {
                log::warn!("Failed to create input key handler: {}", e);
            }
        }
    } else {
        info!("No input devices available for testing");
    }
}

async fn test_grpc_client_functionality() {
    info!("Testing gRPC client functionality...");

    // Create gRPC client configuration
    let config = ChunkSinkConfig {
        server_address: "http://localhost:50051".to_string(),
        recorder_id: "test-recorder".to_string(),
        recorder_name: "Test Recorder".to_string(),
        connect_timeout: Duration::from_secs(5),
        request_timeout: Duration::from_secs(3),
        retry_interval: Duration::from_secs(2),
        max_retries: 2,
        audio_buffer_size: 4096,
        parameter_buffer_size: 32,
    };

    let mut grpc_client = ChunkSinkClientService::new(config);

    // Initialize channels
    grpc_client.initialize_channels();
    info!("gRPC client channels initialized");

    // Try to connect (this will likely fail in test environment)
    match grpc_client.connect().await {
        Ok(_) => {
            info!("Successfully connected to gRPC server");

            // Send test recorder status
            let status = RecorderStatusInfo {
                signal_status: chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                rms_percent: 0.75,
                clipping: false,
            };

            match grpc_client.set_recorder_status(status).await {
                Ok(success) => {
                    info!("Recorder status sent successfully: {}", success);
                }
                Err(e) => {
                    log::warn!("Failed to send recorder status: {}", e);
                }
            }

            // Test audio data sending
            let test_audio_data = vec![0.1, 0.2, 0.3, 0.4, 0.5];
            match grpc_client.send_audio_data(&test_audio_data) {
                Ok(samples_sent) => {
                    info!("Sent {} audio samples to gRPC pipeline", samples_sent);
                }
                Err(e) => {
                    log::warn!("Failed to send audio data: {}", e);
                }
            }

            // Start command listener
            match grpc_client.start_command_listener().await {
                Ok(_) => {
                    info!("Command listener started successfully");
                }
                Err(e) => {
                    log::warn!("Failed to start command listener: {}", e);
                }
            }

            // Test parameter checking
            use chunk_source::audio::channels::Parameters;
            let mut param_buffer = [Parameters::Cut(); 4];
            match grpc_client.receive_parameters(&mut param_buffer) {
                Ok(params_received) => {
                    info!("Received {} parameters from server", params_received);
                }
                Err(e) => {
                    log::warn!("Failed to receive parameters: {}", e);
                }
            }
        }
        Err(e) => {
            log::warn!("Failed to connect to gRPC server (expected in test): {}", e);
        }
    }

    info!("gRPC client test completed");
}
