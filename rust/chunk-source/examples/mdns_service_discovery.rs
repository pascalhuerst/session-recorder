//! Example demonstrating mDNS service discovery with gRPC client integration
//!
//! This example shows how to use the ServiceTracker to discover chunk-sink servers
//! on the local network and automatically connect to them using the gRPC client.

use chunk_source::grpc::chunk_sink_client::{
    ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
};
use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::{error, info, warn};
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::Duration;
use tokio::time::sleep;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logger
    env_logger::init();

    info!("Starting mDNS service discovery example");

    // Create service tracker configuration
    let tracker_config = ServiceTrackerConfig {
        service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
        service_timeout: Duration::from_secs(30),
        cleanup_interval: Duration::from_secs(5),
        max_services: 50,
    };

    // Create and start the service tracker
    let mut tracker = ServiceTracker::new(tracker_config)?;
    let event_receiver = tracker.start()?;

    // Shutdown signal
    let shutdown_signal = Arc::new(AtomicBool::new(false));

    // Set up graceful shutdown
    let shutdown_clone = shutdown_signal.clone();
    ctrlc::set_handler(move || {
        info!("Received Ctrl+C signal, shutting down...");
        shutdown_clone.store(true, Ordering::Relaxed);
    })
    .expect("Error setting Ctrl+C handler");

    // Track active gRPC clients
    let grpc_clients = Arc::new(tokio::sync::Mutex::new(HashMap::<
        String,
        ChunkSinkClientService,
    >::new()));

    // Main event loop
    let main_shutdown = shutdown_signal.clone();
    let main_clients = Arc::clone(&grpc_clients);
    let main_handle = tokio::spawn(async move {
        info!("Starting main service discovery loop");

        while !main_shutdown.load(Ordering::Relaxed) {
            // Check for new service events
            match event_receiver.recv_timeout(Duration::from_millis(100)) {
                Ok(event) => {
                    match event {
                        ServiceEvent::ServiceDiscovered(service_info) => {
                            info!(
                                "🔍 Discovered new chunk-sink server: {} at {}:{}",
                                service_info.instance_name,
                                service_info.hostname,
                                service_info.port
                            );

                            // Display service details
                            info!("   📍 Addresses: {:?}", service_info.addresses);
                            info!("   🏷️  Properties: {:?}", service_info.properties);

                            if let Some(url) = service_info.connection_url() {
                                info!("   🌐 Connection URL: {}", url);

                                // Create and configure gRPC client
                                let grpc_config = ChunkSinkConfig {
                                    server_address: url,
                                    recorder_id: format!(
                                        "recorder-{}",
                                        gethostname::gethostname().to_string_lossy()
                                    ),
                                    recorder_name: "mDNS Discovery Client".to_string(),
                                    connect_timeout: Duration::from_secs(5),
                                    request_timeout: Duration::from_secs(3),
                                    retry_interval: Duration::from_secs(2),
                                    max_retries: 3,
                                    audio_buffer_size: 4096,
                                    parameter_buffer_size: 32,
                                };

                                let mut grpc_client = ChunkSinkClientService::new(grpc_config);
                                grpc_client.initialize_channels();

                                // Attempt to connect to the discovered server
                                match grpc_client.connect().await {
                                    Ok(_) => {
                                        info!(
                                            "✅ Successfully connected to {}",
                                            service_info.instance_name
                                        );

                                        // Send initial status
                                        let status = RecorderStatusInfo {
                                            signal_status: chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                                            rms_percent: 0.0,
                                            clipping: false,
                                        };

                                        if let Err(e) =
                                            grpc_client.set_recorder_status(status).await
                                        {
                                            warn!("Failed to send initial status: {}", e);
                                        }

                                        // Start command listener
                                        if let Err(e) = grpc_client.start_command_listener().await {
                                            warn!("Failed to start command listener: {}", e);
                                        }

                                        // Store the client for later use
                                        main_clients.lock().await.insert(
                                            service_info.instance_name.clone(),
                                            grpc_client,
                                        );
                                    }
                                    Err(e) => {
                                        error!(
                                            "❌ Failed to connect to {}: {}",
                                            service_info.instance_name, e
                                        );
                                    }
                                }
                            }
                        }
                        ServiceEvent::ServiceUpdated(service_info) => {
                            info!(
                                "🔄 Updated chunk-sink server: {} at {}:{}",
                                service_info.instance_name,
                                service_info.hostname,
                                service_info.port
                            );

                            // You could reconnect here if needed
                        }
                        ServiceEvent::ServiceRemoved(instance_name) => {
                            info!("🗑️  Removed chunk-sink server: {}", instance_name);

                            // Remove the gRPC client
                            if let Some(mut client) =
                                main_clients.lock().await.remove(&instance_name)
                            {
                                client.disconnect().await;
                                info!("Disconnected from removed server: {}", instance_name);
                            }
                        }
                    }
                }
                Err(std::sync::mpsc::RecvTimeoutError::Timeout) => {
                    // Normal timeout, continue
                }
                Err(std::sync::mpsc::RecvTimeoutError::Disconnected) => {
                    warn!("Service discovery event receiver disconnected");
                    break;
                }
            }

            // Periodically send status updates to all connected servers
            for (instance_name, client) in main_clients.lock().await.iter_mut() {
                let status = RecorderStatusInfo {
                    signal_status:
                        chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal,
                    rms_percent: (std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs() as f64
                        * 0.1)
                        .sin()
                        .abs()
                        * 100.0, // Simulate changing RMS
                    clipping: false,
                };

                if let Err(e) = client.set_recorder_status(status).await {
                    warn!("Failed to send status to {}: {}", instance_name, e);
                }
            }

            // Small delay to prevent busy waiting
            sleep(Duration::from_millis(50)).await;
        }

        info!("Main service discovery loop finished");
    });

    // Status reporting loop
    let status_shutdown = shutdown_signal.clone();
    let status_handle = thread::spawn(move || {
        info!("Starting status reporting loop");

        while !status_shutdown.load(Ordering::Relaxed) {
            // Simple status message
            info!("📊 Service discovery is active");
            thread::sleep(Duration::from_secs(10));
        }

        info!("Status reporting loop finished");
    });

    info!("🚀 mDNS service discovery is running. Press Ctrl+C to stop.");

    // Wait for shutdown signal
    while !shutdown_signal.load(Ordering::Relaxed) {
        thread::sleep(Duration::from_millis(100));
    }

    info!("Shutting down...");

    // Disconnect all gRPC clients
    let mut clients = grpc_clients.lock().await;
    for (instance_name, mut client) in clients.drain() {
        client.disconnect().await;
        info!("Disconnected from {}", instance_name);
    }
    drop(clients);

    // Stop the service tracker
    tracker.stop();

    // Wait for threads to finish
    if let Err(e) = main_handle.await {
        error!("Error joining main handle: {:?}", e);
    }

    if let Err(e) = status_handle.join() {
        error!("Error joining status handle: {:?}", e);
    }

    info!("Shutdown complete");
    Ok(())
}
