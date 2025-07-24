//! Session Recorder - Essential Implementation
//!
//! This implementation focuses on the core functionality:
//! 1. mDNS service discovery running in background
//! 2. Automatic gRPC client management for discovered services
//! 3. Audio processing with chunk transmission to all connected servers
//! 4. Status updates to all connected clients

use chunk_source::audio::{
    alsa::{AudioSettings, configure_input_device, configure_output_device},
    callback_thread::start_callback_thread,
    channels::{AudioChannelPair, Parameters},
};
use chunk_source::grpc::chunk_sink_client::{
    AudioChunk, ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
};
use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::{error, info, warn};
use ringbuf::traits::{Consumer, Producer};
use std::collections::HashMap;
use std::net::Ipv4Addr;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::{Duration, SystemTime};
use tokio::time::interval;

/// Core session recorder that manages all components
pub struct SessionRecorder {
    // mDNS service discovery
    service_tracker: Option<ServiceTracker>,
    service_event_receiver: Option<std::sync::mpsc::Receiver<ServiceEvent>>,

    // gRPC client management
    grpc_clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,

    // Audio processing
    audio_settings: AudioSettings,
    audio_channels: Option<AudioChannelPair>,
    callback_handle: Option<thread::JoinHandle<()>>,

    // Status management
    recorder_status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,

    // Control
    shutdown_signal: Arc<AtomicBool>,

    // Runtime handles
    discovery_handle: Option<tokio::task::JoinHandle<()>>,
    audio_handle: Option<tokio::task::JoinHandle<()>>,
    status_handle: Option<tokio::task::JoinHandle<()>>,
}

/// Information about a connected gRPC client
struct ClientInfo {
    client: ChunkSinkClientService,
    service_name: String,
    addresses: Vec<Ipv4Addr>,
    current_address_index: usize,
    connection_url: String,
    last_successful_send: Option<SystemTime>,
}

impl ClientInfo {
    fn new(
        client: ChunkSinkClientService,
        service_name: String,
        addresses: Vec<Ipv4Addr>,
        connection_url: String,
    ) -> Self {
        Self {
            client,
            service_name,
            addresses,
            current_address_index: 0,
            connection_url,
            last_successful_send: None,
        }
    }

    fn get_next_address(&mut self) -> Option<Ipv4Addr> {
        if self.addresses.is_empty() {
            return None;
        }

        let addr = self.addresses[self.current_address_index];
        self.current_address_index = (self.current_address_index + 1) % self.addresses.len();
        Some(addr)
    }

    fn mark_successful_send(&mut self) {
        self.last_successful_send = Some(SystemTime::now());
    }
}

impl SessionRecorder {
    /// Create a new session recorder
    pub fn new() -> Self {
        let audio_settings = AudioSettings {
            input_device: "default".to_string(),
            output_device: "default".to_string(),
            num_channels: 2,
            period_size: 512,
            buffer_size: 2048,
            sample_rate: 48000,
        };

        let initial_status = RecorderStatusInfo {
            signal_status: chunk_source::grpc::chunk_sink_client::common::SignalStatus::NoSignal,
            rms_percent: 0.0,
            clipping: false,
        };

        Self {
            service_tracker: None,
            service_event_receiver: None,
            grpc_clients: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            audio_settings,
            audio_channels: None,
            callback_handle: None,
            recorder_status: Arc::new(tokio::sync::Mutex::new(initial_status)),
            shutdown_signal: Arc::new(AtomicBool::new(false)),
            discovery_handle: None,
            audio_handle: None,
            status_handle: None,
        }
    }

    /// Start the session recorder
    pub async fn start(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        info!("Starting Session Recorder");

        // Set up service discovery
        self.setup_service_discovery().await?;

        // Set up audio processing
        self.setup_audio_processing().await?;

        // Start background tasks
        self.start_discovery_task().await;
        self.start_audio_processing_task().await;
        self.start_status_update_task().await;

        info!("Session Recorder started successfully");
        Ok(())
    }

    /// Stop the session recorder
    pub async fn stop(&mut self) {
        info!("Stopping Session Recorder");

        // Signal shutdown
        self.shutdown_signal.store(true, Ordering::Relaxed);

        // Stop service tracker
        if let Some(mut tracker) = self.service_tracker.take() {
            tracker.stop();
        }

        // Wait for tasks to complete
        if let Some(handle) = self.discovery_handle.take() {
            let _ = handle.await;
        }

        if let Some(handle) = self.audio_handle.take() {
            let _ = handle.await;
        }

        if let Some(handle) = self.status_handle.take() {
            let _ = handle.await;
        }

        // Stop audio callback
        if let Some(handle) = self.callback_handle.take() {
            let _ = handle.join();
        }

        // Disconnect all clients
        let mut clients = self.grpc_clients.lock().await;
        for (name, mut client_info) in clients.drain() {
            client_info.client.disconnect().await;
            info!("Disconnected from service: {}", name);
        }

        info!("Session Recorder stopped");
    }

    /// Set up mDNS service discovery
    async fn setup_service_discovery(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let tracker_config = ServiceTrackerConfig {
            service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
            service_timeout: Duration::from_secs(30),
            cleanup_interval: Duration::from_secs(3),
            max_services: 50,
        };

        let mut tracker = ServiceTracker::new(tracker_config)?;
        let event_receiver = tracker.start()?;

        self.service_tracker = Some(tracker);
        self.service_event_receiver = Some(event_receiver);

        info!("mDNS service discovery initialized");
        Ok(())
    }

    /// Set up audio processing
    async fn setup_audio_processing(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        // Create audio devices
        let capture_pcm = configure_input_device(&self.audio_settings)?;
        let playback_pcm = configure_output_device(&self.audio_settings)?;

        // Create audio channels
        let audio_channels = AudioChannelPair::new(self.audio_settings.buffer_size as usize * 4);

        // Start audio callback thread
        let callback_handle = start_callback_thread(
            self.audio_settings.num_channels as usize,
            self.audio_settings.num_channels as usize,
            self.audio_settings.period_size as usize,
            Some(capture_pcm),
            Some(playback_pcm),
            audio_channels.callback_channels,
            self.shutdown_signal.clone(),
        );

        self.audio_channels = Some(audio_channels);
        self.callback_handle = Some(callback_handle);

        info!("Audio processing initialized");
        Ok(())
    }

    /// Start the service discovery task
    async fn start_discovery_task(&mut self) {
        let event_receiver = self.service_event_receiver.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();

        let handle = tokio::spawn(async move {
            Self::discovery_task(event_receiver, clients, shutdown).await;
        });

        self.discovery_handle = Some(handle);
    }

    /// Start the audio processing task
    async fn start_audio_processing_task(&mut self) {
        let audio_channels = self.audio_channels.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);

        let handle = tokio::spawn(async move {
            Self::audio_processing_task(audio_channels, clients, shutdown, status).await;
        });

        self.audio_handle = Some(handle);
    }

    /// Start the status update task
    async fn start_status_update_task(&mut self) {
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);

        let handle = tokio::spawn(async move {
            Self::status_update_task(clients, shutdown, status).await;
        });

        self.status_handle = Some(handle);
    }

    /// Service discovery background task
    async fn discovery_task(
        event_receiver: std::sync::mpsc::Receiver<ServiceEvent>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
    ) {
        info!("Service discovery task started");

        while !shutdown.load(Ordering::Relaxed) {
            match event_receiver.recv_timeout(Duration::from_secs(1)) {
                Ok(event) => {
                    match event {
                        ServiceEvent::ServiceDiscovered(service_info) => {
                            info!("Discovered service: {}", service_info.instance_name);

                            if let Some(url) = service_info.connection_url() {
                                let config = ChunkSinkConfig {
                                    server_address: url.clone(),
                                    recorder_id: format!(
                                        "recorder-{}",
                                        gethostname::gethostname().to_string_lossy()
                                    ),
                                    recorder_name: "Session Recorder".to_string(),
                                    connect_timeout: Duration::from_secs(10),
                                    request_timeout: Duration::from_secs(5),
                                    retry_interval: Duration::from_secs(3),
                                    max_retries: 3,
                                    audio_buffer_size: 8192,
                                    parameter_buffer_size: 64,
                                };

                                let mut client = ChunkSinkClientService::new(config);
                                client.initialize_channels();

                                match client.connect().await {
                                    Ok(_) => {
                                        info!(
                                            "Connected to service: {}",
                                            service_info.instance_name
                                        );

                                        let client_info = ClientInfo::new(
                                            client,
                                            service_info.instance_name.clone(),
                                            service_info.addresses.clone(),
                                            url,
                                        );

                                        clients.lock().await.insert(
                                            service_info.instance_name.clone(),
                                            client_info,
                                        );
                                    }
                                    Err(e) => {
                                        error!(
                                            "Failed to connect to {}: {}",
                                            service_info.instance_name, e
                                        );
                                    }
                                }
                            }
                        }
                        ServiceEvent::ServiceRemoved(instance_name) => {
                            info!("Service removed: {}", instance_name);

                            if let Some(mut client_info) =
                                clients.lock().await.remove(&instance_name)
                            {
                                client_info.client.disconnect().await;
                                info!("Disconnected from service: {}", instance_name);
                            }
                        }
                        ServiceEvent::ServiceUpdated(service_info) => {
                            info!("Service updated: {}", service_info.instance_name);
                            // Handle service updates if needed
                        }
                    }
                }
                Err(_) => {
                    // Timeout - normal behavior
                }
            }
        }

        info!("Service discovery task stopped");
    }

    /// Audio processing background task
    async fn audio_processing_task(
        audio_channels: AudioChannelPair,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
    ) {
        info!("Audio processing task started");

        let mut main_channels = audio_channels.main_channels;
        let mut audio_buffer = vec![0.0f32; 1024]; // Buffer for audio chunks
        let mut chunk_counter = 0u32;
        let mut interval = interval(Duration::from_millis(100)); // 10 chunks per second

        while !shutdown.load(Ordering::Relaxed) {
            interval.tick().await;

            // Read audio data from the audio pipeline
            let samples_read = main_channels.output_consumer.pop_slice(&mut audio_buffer);

            if samples_read > 0 {
                // Calculate RMS for status
                let rms = audio_buffer[..samples_read]
                    .iter()
                    .map(|&x| x * x)
                    .sum::<f32>()
                    / samples_read as f32;
                let rms_percent = (rms.sqrt() * 100.0).min(100.0);

                // Check for clipping
                let clipping = audio_buffer[..samples_read].iter().any(|&x| x.abs() > 0.95);

                // Update status
                {
                    let mut status_guard = status.lock().await;
                    status_guard.signal_status =
                        chunk_source::grpc::chunk_sink_client::common::SignalStatus::Signal;
                    status_guard.rms_percent = rms_percent as f64;
                    status_guard.clipping = clipping;
                }

                // Send audio chunks to all connected clients
                let mut clients_guard = clients.lock().await;
                let mut failed_clients = Vec::new();

                for (name, client_info) in clients_guard.iter_mut() {
                    let chunk = AudioChunk {
                        session_id: format!("session_{}", chunk_counter / 1000),
                        chunk_count: chunk_counter,
                        data: audio_buffer[..samples_read]
                            .iter()
                            .map(|&sample| {
                                let scaled = (sample + 1.0) / 2.0;
                                (scaled.clamp(0.0, 1.0) * u32::MAX as f32) as u32
                            })
                            .collect(),
                        timestamp: SystemTime::now(),
                    };

                    match client_info.client.set_chunks(chunk).await {
                        Ok(_) => {
                            client_info.mark_successful_send();
                        }
                        Err(e) => {
                            warn!("Failed to send chunk to {}: {}", name, e);

                            // Try next IP address if available
                            if let Some(_next_addr) = client_info.get_next_address() {
                                // In a more sophisticated implementation, we would
                                // reconnect using the next address
                                warn!(
                                    "Could implement reconnection with next address for {}",
                                    name
                                );
                            } else {
                                failed_clients.push(name.clone());
                            }
                        }
                    }
                }

                // Remove failed clients
                for name in failed_clients {
                    if let Some(mut client_info) = clients_guard.remove(&name) {
                        client_info.client.disconnect().await;
                        warn!("Removed failed client: {}", name);
                    }
                }

                chunk_counter += 1;

                // Send processed audio back to the audio pipeline
                let samples_pushed = main_channels
                    .input_producer
                    .push_slice(&audio_buffer[..samples_read]);
                if samples_pushed != samples_read {
                    warn!("Could not push all audio samples back to pipeline");
                }
            } else {
                // No audio data available
                let mut status_guard = status.lock().await;
                status_guard.signal_status =
                    chunk_source::grpc::chunk_sink_client::common::SignalStatus::NoSignal;
                status_guard.rms_percent = 0.0;
                status_guard.clipping = false;
            }
        }

        info!("Audio processing task stopped");
    }

    /// Status update background task
    async fn status_update_task(
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
    ) {
        info!("Status update task started");

        let mut interval = interval(Duration::from_secs(5)); // Update every 5 seconds

        while !shutdown.load(Ordering::Relaxed) {
            interval.tick().await;

            let current_status = {
                let status_guard = status.lock().await;
                status_guard.clone()
            };

            let mut clients_guard = clients.lock().await;
            let mut failed_clients = Vec::new();

            for (name, client_info) in clients_guard.iter_mut() {
                match client_info
                    .client
                    .set_recorder_status(current_status.clone())
                    .await
                {
                    Ok(_) => {
                        // Status sent successfully
                    }
                    Err(e) => {
                        warn!("Failed to send status to {}: {}", name, e);
                        failed_clients.push(name.clone());
                    }
                }
            }

            // Remove failed clients
            for name in failed_clients {
                if let Some(mut client_info) = clients_guard.remove(&name) {
                    client_info.client.disconnect().await;
                    warn!("Removed failed client during status update: {}", name);
                }
            }

            if !clients_guard.is_empty() {
                info!("Status update sent to {} client(s)", clients_guard.len());
            }
        }

        info!("Status update task stopped");
    }

    /// Check if the recorder is running
    pub fn is_running(&self) -> bool {
        !self.shutdown_signal.load(Ordering::Relaxed)
    }

    /// Get current status
    pub async fn get_status(&self) -> RecorderStatusInfo {
        let status_guard = self.recorder_status.lock().await;
        status_guard.clone()
    }

    /// Get connected client count
    pub async fn get_client_count(&self) -> usize {
        let clients = self.grpc_clients.lock().await;
        clients.len()
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logger
    env_logger::init();

    info!("Starting Session Recorder");

    // Create and start session recorder
    let mut recorder = SessionRecorder::new();
    recorder.start().await?;

    // Set up graceful shutdown
    let shutdown_signal = recorder.shutdown_signal.clone();
    ctrlc::set_handler(move || {
        info!("Received shutdown signal");
        shutdown_signal.store(true, Ordering::Relaxed);
    })?;

    info!("Session Recorder running. Press Ctrl+C to stop.");

    // Main loop
    while recorder.is_running() {
        tokio::time::sleep(Duration::from_secs(1)).await;

        // Periodic status logging
        let status = recorder.get_status().await;
        let client_count = recorder.get_client_count().await;

        if client_count > 0 {
            info!(
                "Connected to {} server(s), RMS: {:.1}%, Signal: {:?}",
                client_count, status.rms_percent, status.signal_status
            );
        }
    }

    // Stop the recorder
    recorder.stop().await;

    info!("Session Recorder stopped");
    Ok(())
}
