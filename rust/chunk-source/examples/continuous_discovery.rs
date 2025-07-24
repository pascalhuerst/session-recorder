//! Continuous mDNS service discovery example
//!
//! This example demonstrates how the service discovery works in practice:
//! - Starts discovery and waits indefinitely for services
//! - Automatically connects to discovered chunk-sink servers
//! - Sends periodic status updates and audio data
//! - Handles server disconnections gracefully
//! - Runs continuously until interrupted

use chunk_source::grpc::chunk_sink_client::{
    ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
};
use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::{error, info, warn};
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, SystemTime};
use tokio::time::interval;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logger
    env_logger::init();

    info!("🚀 Starting continuous mDNS service discovery for chunk-sink servers");
    info!("This will run indefinitely - press Ctrl+C to stop");

    // Create service tracker configuration
    let tracker_config = ServiceTrackerConfig {
        service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
        service_timeout: Duration::from_secs(120), // 2 minutes timeout
        cleanup_interval: Duration::from_secs(30), // Check every 30 seconds
        max_services: 100,
    };

    // Create and start the service tracker
    let mut tracker = ServiceTracker::new(tracker_config)?;
    let event_receiver = tracker.start()?;

    // Shutdown signal
    let shutdown_signal = Arc::new(AtomicBool::new(false));
    let shutdown_clone = shutdown_signal.clone();

    // Set up graceful shutdown
    ctrlc::set_handler(move || {
        info!("📡 Received shutdown signal, stopping discovery...");
        shutdown_clone.store(true, Ordering::Relaxed);
    })
    .expect("Error setting Ctrl+C handler");

    // Track active gRPC clients
    let grpc_clients = Arc::new(tokio::sync::Mutex::new(HashMap::<
        String,
        ChunkSinkClientService,
    >::new()));

    // Service event handler task
    let event_shutdown = shutdown_signal.clone();
    let event_clients = Arc::clone(&grpc_clients);
    let event_handle = std::thread::spawn(move || {
        let rt = tokio::runtime::Runtime::new().unwrap();

        rt.block_on(async move {
            info!("🔍 mDNS service discovery is now active");
            info!("Waiting for '_session-recorder-chunksink._tcp.local.' services...");

            while !event_shutdown.load(Ordering::Relaxed) {
                // Check for new service events with timeout
                match event_receiver.recv_timeout(Duration::from_millis(500)) {
                    Ok(event) => {
                        match event {
                            ServiceEvent::ServiceDiscovered(service_info) => {
                                info!(
                                    "🔍 DISCOVERED: {} at {}:{}",
                                    service_info.instance_name,
                                    service_info.hostname,
                                    service_info.port
                                );

                                info!("   📍 IP addresses: {:?}", service_info.addresses);
                                if !service_info.properties.is_empty() {
                                    info!("   🏷️  Properties: {:?}", service_info.properties);
                                }

                                if let Some(url) = service_info.connection_url() {
                                    info!("   🌐 Connecting to: {}", url);

                                    // Create gRPC client configuration
                                    let grpc_config = ChunkSinkConfig {
                                        server_address: url.clone(),
                                        recorder_id: format!(
                                            "recorder-{}",
                                            gethostname::gethostname().to_string_lossy()
                                        ),
                                        recorder_name: format!(
                                            "Continuous Discovery Client ({})",
                                            service_info.hostname
                                        ),
                                        connect_timeout: Duration::from_secs(10),
                                        request_timeout: Duration::from_secs(5),
                                        retry_interval: Duration::from_secs(3),
                                        max_retries: 3,
                                        audio_buffer_size: 8192,
                                        parameter_buffer_size: 64,
                                    };

                                    let mut grpc_client = ChunkSinkClientService::new(grpc_config);
                                    grpc_client.initialize_channels();

                                    // Attempt to connect
                                    match grpc_client.connect_with_retry().await {
                                        Ok(_) => {
                                            info!("   ✅ Connected to {}", service_info.instance_name);

                                            // Send initial status
                                            let status = RecorderStatusInfo {
                                                signal_status: chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                                                rms_percent: 0.0,
                                                clipping: false,
                                            };

                                            match grpc_client
                                                .set_recorder_status_with_retry(status)
                                                .await
                                            {
                                                Ok(_) => {
                                                    info!("   📊 Initial status sent successfully");
                                                }
                                                Err(e) => {
                                                    warn!(
                                                        "   ⚠️  Failed to send initial status: {}",
                                                        e
                                                    );
                                                }
                                            }

                                            // Start command listener
                                            if let Err(e) = grpc_client.start_command_listener().await {
                                                warn!("   ⚠️  Failed to start command listener: {}", e);
                                            }

                                            // Store the client
                                            event_clients.lock().await.insert(
                                                service_info.instance_name.clone(),
                                                grpc_client,
                                            );

                                            info!(
                                                "   🎯 Ready to send audio data and receive commands"
                                            );
                                        }
                                        Err(e) => {
                                            error!("   ❌ Failed to connect: {}", e);
                                        }
                                    }
                                }
                            }
                            ServiceEvent::ServiceUpdated(service_info) => {
                                info!(
                                    "🔄 UPDATED: {} at {}:{}",
                                    service_info.instance_name,
                                    service_info.hostname,
                                    service_info.port
                                );
                            }
                            ServiceEvent::ServiceRemoved(instance_name) => {
                                info!("🗑️  REMOVED: {}", instance_name);

                                // Disconnect and remove the gRPC client
                                if let Some(mut client) =
                                    event_clients.lock().await.remove(&instance_name)
                                {
                                    client.disconnect().await;
                                    info!("   🔌 Disconnected from {}", instance_name);
                                }
                            }
                        }
                    }
                    Err(_) => {
                        // Normal timeout - no events received
                    }
                }
            }
        });

        info!("🔍 Service discovery event handler stopped");
    });

    // Status update and audio data sender task
    let status_shutdown = shutdown_signal.clone();
    let status_clients = Arc::clone(&grpc_clients);
    let status_handle = tokio::spawn(async move {
        let mut status_interval = interval(Duration::from_secs(10));
        let mut audio_interval = interval(Duration::from_secs(1));
        let mut chunk_counter = 0u32;

        info!("📊 Status and audio data sender started");

        while !status_shutdown.load(Ordering::Relaxed) {
            tokio::select! {
                _ = status_interval.tick() => {
                    // Send status updates to all connected servers
                    let mut clients = status_clients.lock().await;

                    if !clients.is_empty() {
                        info!("📡 Sending status updates to {} server(s)", clients.len());

                        for (instance_name, client) in clients.iter_mut() {
                            // Simulate varying RMS values
                            let time_factor = SystemTime::now()
                                .duration_since(SystemTime::UNIX_EPOCH)
                                .unwrap()
                                .as_secs() as f64 * 0.1;

                            let status = RecorderStatusInfo {
                                signal_status: chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                                rms_percent: (time_factor.sin().abs() * 80.0) + 10.0, // 10-90% range
                                clipping: time_factor.sin() > 0.9, // Occasional clipping
                            };

                            match client.set_recorder_status_with_retry(status).await {
                                Ok(_) => {
                                    info!("   📊 Status sent to {}", instance_name);
                                }
                                Err(e) => {
                                    warn!("   ⚠️  Failed to send status to {}: {}", instance_name, e);
                                }
                            }
                        }
                    }
                }

                _ = audio_interval.tick() => {
                    // Send audio data to all connected servers
                    let mut clients = status_clients.lock().await;

                    if !clients.is_empty() {
                        // Generate some test audio data (sine wave)
                        let audio_data: Vec<f32> = (0..512)
                            .map(|i| {
                                let t = (chunk_counter * 512 + i as u32) as f32 * 0.001;
                                (t * 440.0 * 2.0 * std::f32::consts::PI).sin() * 0.1
                            })
                            .collect();

                        for (instance_name, client) in clients.iter_mut() {
                            match client.send_audio_data(&audio_data) {
                                Ok(samples_sent) => {
                                    if chunk_counter % 100 == 0 { // Log every 100 chunks
                                        info!("   🎵 Sent {} audio samples to {}", samples_sent, instance_name);
                                    }
                                }
                                Err(e) => {
                                    warn!("   ⚠️  Failed to send audio to {}: {}", instance_name, e);
                                }
                            }
                        }

                        chunk_counter += 1;
                    }
                }
            }
        }

        info!("📊 Status and audio data sender stopped");
    });

    // Main monitoring loop
    let monitor_shutdown = shutdown_signal.clone();
    let monitor_clients = Arc::clone(&grpc_clients);
    let monitor_handle = tokio::spawn(async move {
        let mut monitor_interval = interval(Duration::from_secs(60));

        while !monitor_shutdown.load(Ordering::Relaxed) {
            monitor_interval.tick().await;

            let client_count = monitor_clients.lock().await.len();
            if client_count > 0 {
                info!(
                    "🔗 Currently connected to {} chunk-sink server(s)",
                    client_count
                );
            } else {
                info!("⏳ No chunk-sink servers connected - waiting for discovery...");
            }
        }

        info!("🔍 Service monitor stopped");
    });

    info!("🎯 System ready! Discovery is running continuously...");
    info!("   • Chunk-sink servers will be discovered automatically");
    info!("   • Audio data and status updates will be sent to connected servers");
    info!("   • Press Ctrl+C to stop");

    // Wait for shutdown signal
    while !shutdown_signal.load(Ordering::Relaxed) {
        tokio::time::sleep(Duration::from_millis(100)).await;
    }

    info!("🛑 Shutting down...");

    // Disconnect all gRPC clients
    let mut clients = grpc_clients.lock().await;
    for (instance_name, mut client) in clients.drain() {
        client.disconnect().await;
        info!("🔌 Disconnected from {}", instance_name);
    }
    drop(clients);

    // Stop the service tracker
    tracker.stop();

    // Wait for tasks to complete
    let _ = event_handle.join();
    let _ = tokio::join!(status_handle, monitor_handle);

    info!("✅ Shutdown complete");
    Ok(())
}
