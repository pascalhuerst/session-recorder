use crate::audio::{
    alsa::{AudioSettings, configure_input_device},
    callback_thread::start_callback_thread,
    channels::{AudioChannels, AudioRingBufferConsumer},
};
use crate::grpc::chunk_sink_client::{
    AudioChunk, ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo, common::SignalStatus,
};
use crate::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::{error, info, warn};
use ringbuf::traits::Consumer;
use std::collections::HashMap;
use std::net::Ipv4Addr;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::Receiver;
use std::thread;
use std::time::{Duration, SystemTime};
use tokio::time::interval;
use uuid::Uuid;

/// Core session recorder that manages discovery, audio handling, and status updates.
pub struct SessionRecorder {
    // mDNS service discovery
    service_tracker: Option<ServiceTracker>,
    service_event_receiver: Option<Receiver<ServiceEvent>>,

    // gRPC client management
    grpc_clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,

    // Audio processing
    audio_settings: AudioSettings,
    audio_consumer: Option<AudioRingBufferConsumer>,
    recorder_id: Uuid,
    recorder_name: String,
    session_id: Uuid,
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
    pub fn new(recorder_id: Uuid, recorder_name: String) -> Self {
        let session_id = Uuid::new_v4();

        info!(
            "Creating new session recorder {} with ID: {} (session {})",
            recorder_name, recorder_id, session_id
        );

        let audio_settings = AudioSettings {
            input_device: "default".to_string(),
            num_channels: 2,
            period_size: 512,
            buffer_size: 2048,
            sample_rate: 48000,
        };

        let initial_status = RecorderStatusInfo {
            signal_status: SignalStatus::NoSignal,
            rms_percent: 0.0,
            clipping: false,
        };

        Self {
            service_tracker: None,
            service_event_receiver: None,
            grpc_clients: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            audio_settings,
            audio_consumer: None,
            recorder_id,
            recorder_name,
            session_id,
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

        // Start service discovery before configuring audio
        self.start_discovery_task().await;

        // Set up audio processing
        self.setup_audio_processing().await?;

        // Start remaining background tasks
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

        let buffer_capacity =
            self.audio_settings.buffer_size as usize * self.audio_settings.num_channels as usize;
        let mut channels = AudioChannels::new(buffer_capacity);
        let producer = channels.take_producer().ok_or_else(|| {
            Box::new(std::io::Error::new(
                std::io::ErrorKind::Other,
                "failed to acquire audio producer",
            )) as Box<dyn std::error::Error>
        })?;
        let consumer = channels.consumer;
        self.audio_consumer = Some(consumer);

        // Start audio callback thread
        let callback_handle = start_callback_thread(
            self.audio_settings.num_channels as usize,
            self.audio_settings.period_size as usize,
            Some(capture_pcm),
            self.shutdown_signal.clone(),
            producer,
        );

        self.callback_handle = Some(callback_handle);

        info!("Audio processing initialized");
        Ok(())
    }

    /// Start the service discovery task
    async fn start_discovery_task(&mut self) {
        if self.service_event_receiver.is_none() {
            error!("Discovery task requested before service tracker produced a receiver");
            return;
        }

        info!("Preparing discovery task startup");

        let event_receiver = self.service_event_receiver.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let recorder_id = self.recorder_id;
        let recorder_name = self.recorder_name.clone();

        info!(
            "Spawning discovery task with recorder_id={} recorder_name={}",
            recorder_id, recorder_name
        );

        let handle = tokio::spawn(async move {
            info!("Discovery task future started");
            Self::discovery_task(
                event_receiver,
                clients,
                shutdown,
                recorder_id,
                recorder_name,
            )
            .await;
            info!("Discovery task future completed");
        });

        self.discovery_handle = Some(handle);
    }

    /// Start the audio processing task
    async fn start_audio_processing_task(&mut self) {
        let Some(consumer) = self.audio_consumer.take() else {
            warn!("Audio consumer not initialized");
            return;
        };
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);
        let num_channels = self.audio_settings.num_channels as usize;
        let frames_per_chunk = self.audio_settings.period_size as usize;
        let session_id = self.session_id;

        let handle = tokio::spawn(async move {
            Self::audio_processing_task(
                consumer,
                clients,
                shutdown,
                status,
                num_channels,
                frames_per_chunk,
                session_id,
            )
            .await;
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
        event_receiver: Receiver<ServiceEvent>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        recorder_id: Uuid,
        recorder_name: String,
    ) {
        info!("Service discovery task started");

        while !shutdown.load(Ordering::Relaxed) {
            match event_receiver.recv_timeout(Duration::from_secs(1)) {
                Ok(event) => {
                    info!("Discovery task received event: {:?}", event);
                    match event {
                        ServiceEvent::ServiceDiscovered(service_info) => {
                            info!("Discovered service: {}", service_info.instance_name);

                            if let Some(url) = service_info.connection_url() {
                                let config = ChunkSinkConfig {
                                    server_address: url.clone(),
                                    recorder_id,
                                    recorder_name: recorder_name.clone(),
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
                                            "Connected to service {} at {}",
                                            service_info.instance_name, url
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
                                let service_name = client_info.service_name.clone();
                                let service_url = client_info.connection_url.clone();
                                client_info.client.disconnect().await;
                                info!(
                                    "Disconnected from service {} at {}",
                                    service_name, service_url
                                );
                            }
                        }
                        ServiceEvent::ServiceUpdated(service_info) => {
                            info!("Service updated: {}", service_info.instance_name);
                        }
                    }
                }
                Err(std::sync::mpsc::RecvTimeoutError::Timeout) => {
                    continue;
                }
                Err(std::sync::mpsc::RecvTimeoutError::Disconnected) => {
                    error!("Discovery task channel disconnected");
                    break;
                }
            }
        }

        info!("Service discovery task stopped");
    }

    /// Audio processing background task
    async fn audio_processing_task(
        mut consumer: AudioRingBufferConsumer,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
        num_channels: usize,
        frames_per_chunk: usize,
        session_id: Uuid,
    ) {
        info!("Audio processing task started");

        let mut sample_buffer = vec![0i16; num_channels * frames_per_chunk];
        let mut chunk_counter = 0u32;
        let mut interval = interval(Duration::from_millis(100)); // 10 chunks per second

        while !shutdown.load(Ordering::Relaxed) {
            interval.tick().await;

            let samples_read = consumer.pop_slice(&mut sample_buffer);

            info!("Read {} audio samples to be sent for storage", samples_read);

            if samples_read > 0 {
                let rms = sample_buffer[..samples_read]
                    .iter()
                    .map(|&x| {
                        let normalized = i32::from(x) as f64 / i16::MAX as f64;
                        normalized * normalized
                    })
                    .sum::<f64>()
                    / samples_read as f64;
                let rms_percent = (rms.sqrt() * 100.0).min(100.0);

                let clipping = sample_buffer[..samples_read]
                    .iter()
                    .any(|&x| i32::from(x).abs() >= i32::from(i16::MAX));

                {
                    let mut status_guard = status.lock().await;
                    status_guard.signal_status = SignalStatus::Signal;
                    status_guard.rms_percent = rms_percent;
                    status_guard.clipping = clipping;
                }

                let chunk_template = AudioChunk {
                    session_id: session_id.to_string(),
                    chunk_count: chunk_counter,
                    data: sample_buffer[..samples_read]
                        .iter()
                        .map(|&sample| (i32::from(sample) - i32::from(i16::MIN)) as u32)
                        .collect(),
                    timestamp: SystemTime::now(),
                };

                let mut clients_guard = clients.lock().await;
                let mut failed_clients = Vec::new();

                for (name, client_info) in clients_guard.iter_mut() {
                    if chunk_counter == 0 {
                        let client_config = client_info.client.config();
                        warn!(
                            "Sending first chunk to service {} at {} (recorder_id={}, recorder_name={}, session_id={}, samples={})",
                            client_info.service_name,
                            client_info.connection_url,
                            client_config.recorder_id,
                            client_config.recorder_name,
                            session_id,
                            samples_read
                        );
                    }

                    match client_info
                        .client
                        .set_chunks_with_retry(chunk_template.clone())
                        .await
                    {
                        Ok(true) => {
                            client_info.mark_successful_send();
                        }
                        Ok(false) => {
                            warn!(
                                "Server reported error for {} at {}",
                                client_info.service_name, client_info.connection_url
                            );
                            failed_clients.push(name.clone());
                        }
                        Err(e) => {
                            warn!(
                                "Failed to send chunk to {} at {}: {}",
                                client_info.service_name, client_info.connection_url, e
                            );

                            if let Some(_next_addr) = client_info.get_next_address() {
                                warn!(
                                    "Could implement reconnection with next address for {} at {}",
                                    client_info.service_name, client_info.connection_url
                                );
                            } else {
                                failed_clients.push(name.clone());
                            }
                        }
                    }
                }

                for name in failed_clients {
                    if let Some(mut client_info) = clients_guard.remove(&name) {
                        let service_name = client_info.service_name.clone();
                        let service_url = client_info.connection_url.clone();
                        client_info.client.disconnect().await;
                        warn!("Removed failed client {} at {}", service_name, service_url);
                    }
                }

                chunk_counter += 1;
            } else {
                let mut status_guard = status.lock().await;
                status_guard.signal_status = SignalStatus::NoSignal;
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
                    Ok(_) => {}
                    Err(e) => {
                        warn!(
                            "Failed to send status to {} at {}: {}",
                            client_info.service_name, client_info.connection_url, e
                        );
                        failed_clients.push(name.clone());
                    }
                }
            }

            for name in failed_clients {
                if let Some(mut client_info) = clients_guard.remove(&name) {
                    let service_name = client_info.service_name.clone();
                    let service_url = client_info.connection_url.clone();
                    client_info.client.disconnect().await;
                    warn!(
                        "Removed failed client {} at {} during status update",
                        service_name, service_url
                    );
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

    pub fn shutdown_signal(&self) -> Arc<AtomicBool> {
        Arc::clone(&self.shutdown_signal)
    }
}
