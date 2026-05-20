use crate::audio::{
    alsa::{AudioSettings, configure_input_device},
    callback_thread::start_callback_thread,
    channels::{
        AudioChannels, AudioRingBufferConsumer, ParameterChannels, ParameterRingBufferConsumer,
        ParameterRingBufferProducer, Parameters,
    },
};
use crate::grpc::chunk_sink_client::{
    AudioChunk, ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo, ServerCommand,
    common::SignalStatus,
};
use log::{debug, error, info, warn};
use ringbuf::traits::{Consumer, Producer};
use std::collections::HashMap;

use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};

use std::thread;
use std::time::{Duration, SystemTime};
use tokio::sync::{
    Notify,
    mpsc::{self, Receiver, Sender},
};
use tokio::time::{interval, sleep};
use uuid::Uuid;
use zeroconf_tokio::prelude::*;
use zeroconf_tokio::{BrowserEvent, MdnsBrowser, MdnsBrowserAsync, ServiceType};

/// Core session recorder that manages discovery, audio handling, and status updates.
pub struct SessionRecorder {
    // gRPC client management
    grpc_clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,

    // Audio processing
    audio_settings: AudioSettings,
    audio_consumer: Option<AudioRingBufferConsumer>,
    recorder_id: Uuid,
    recorder_name: String,
    detector_threshold: f64,
    detector_succession: u32,
    callback_handle: Option<thread::JoinHandle<()>>,

    // Status management
    recorder_status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
    status_notify: Arc<Notify>,
    parameter_producer: Arc<tokio::sync::Mutex<ParameterRingBufferProducer>>,
    parameter_consumer: Option<ParameterRingBufferConsumer>,
    command_tx: Sender<Parameters>,
    command_rx: Option<Receiver<Parameters>>,

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
    addresses: Vec<String>,
    current_address_index: usize,
    connection_url: String,
    last_successful_send: Option<SystemTime>,
}

impl ClientInfo {
    fn new(
        client: ChunkSinkClientService,
        service_name: String,
        addresses: Vec<String>,
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

    fn get_next_address(&mut self) -> Option<String> {
        if self.addresses.is_empty() {
            return None;
        }

        let addr = self.addresses[self.current_address_index].clone();
        self.current_address_index = (self.current_address_index + 1) % self.addresses.len();
        Some(addr)
    }

    fn mark_successful_send(&mut self) {
        self.last_successful_send = Some(SystemTime::now());
    }
}

impl SessionRecorder {
    /// Create a new session recorder
    pub fn new(
        recorder_id: Uuid,
        recorder_name: String,
        detector_threshold: f64,
        detector_succession: u32,
    ) -> Self {
        let sanitized_threshold = detector_threshold.max(0.0);
        let sanitized_succession = detector_succession.max(1);

        if sanitized_threshold != detector_threshold || sanitized_succession != detector_succession
        {
            warn!(
                "Adjusted detector parameters: threshold {:.2}→{:.2}, succession {}→{}",
                detector_threshold, sanitized_threshold, detector_succession, sanitized_succession
            );
        }

        info!(
            "Creating new session recorder {} with ID: {} (threshold {:.2}, succession {})",
            recorder_name, recorder_id, sanitized_threshold, sanitized_succession
        );

        let audio_settings = AudioSettings {
            input_device: "pipewire".to_string(),
            num_channels: 2,
            period_size: 512,
            buffer_size: 16768,
            sample_rate: 48000,
        };

        let initial_status = RecorderStatusInfo {
            signal_status: SignalStatus::NoSignal,
            rms_percent: 0.0,
            clipping: false,
        };

        let ParameterChannels { producer, consumer } = ParameterChannels::new(64);
        let parameter_producer = Arc::new(tokio::sync::Mutex::new(producer));
        let parameter_consumer = Some(consumer);
        let (command_tx, command_rx) = mpsc::channel(32);

        Self {
            grpc_clients: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            audio_settings,
            audio_consumer: None,
            recorder_id,
            recorder_name,
            detector_threshold: sanitized_threshold,
            detector_succession: sanitized_succession,
            callback_handle: None,
            recorder_status: Arc::new(tokio::sync::Mutex::new(initial_status)),
            status_notify: Arc::new(Notify::new()),
            parameter_producer,
            parameter_consumer,
            command_tx,
            command_rx: Some(command_rx),
            shutdown_signal: Arc::new(AtomicBool::new(false)),
            discovery_handle: None,
            audio_handle: None,
            status_handle: None,
        }
    }

    /// Start the session recorder
    pub async fn start(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        info!("Starting Session Recorder");

        if let Some(command_rx) = self.command_rx.take() {
            let parameter_producer = Arc::clone(&self.parameter_producer);
            tokio::spawn(async move {
                let mut command_rx = command_rx;
                while let Some(command) = command_rx.recv().await {
                    let mut producer = parameter_producer.lock().await;
                    let _ = producer.push_slice(&[command]);
                }
            });
        } else {
            warn!("Command receiver not initialized");
        }

        // Start zeroconf service discovery
        self.start_discovery().await?;

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

    async fn start_discovery(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let service_type = ServiceType::new("session-recorder-chunksink", "tcp")?;
        let browser = MdnsBrowser::new(service_type);
        let mut async_browser = MdnsBrowserAsync::new(browser)?;
        async_browser.start().await?;
        info!("mDNS discovery started using zeroconf");

        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = Arc::clone(&self.shutdown_signal);
        let command_tx = self.command_tx.clone();
        let recorder_id = self.recorder_id;
        let recorder_name = self.recorder_name.clone();

        let handle = tokio::spawn(async move {
            Self::discovery_task(
                async_browser,
                clients,
                shutdown,
                command_tx,
                recorder_id,
                recorder_name,
            )
            .await;
        });

        self.discovery_handle = Some(handle);
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

    /// Start the audio processing task
    async fn start_audio_processing_task(&mut self) {
        let Some(consumer) = self.audio_consumer.take() else {
            warn!("Audio consumer not initialized");
            return;
        };
        let Some(parameter_consumer) = self.parameter_consumer.take() else {
            warn!("Parameter consumer not initialized");
            return;
        };
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);
        let status_notify = Arc::clone(&self.status_notify);
        let num_channels = self.audio_settings.num_channels as usize;
        let frames_per_chunk = self.audio_settings.period_size as usize;

        let detector_threshold = self.detector_threshold;
        let detector_succession = self.detector_succession;

        let handle = tokio::spawn(async move {
            Self::audio_processing_task(
                consumer,
                parameter_consumer,
                clients,
                shutdown,
                status,
                status_notify,
                num_channels,
                frames_per_chunk,
                detector_threshold,
                detector_succession,
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
        let status_notify = Arc::clone(&self.status_notify);

        let handle = tokio::spawn(async move {
            Self::status_update_task(clients, shutdown, status, status_notify).await;
        });

        self.status_handle = Some(handle);
    }

    /// Service discovery background task
    async fn discovery_task(
        mut browser: MdnsBrowserAsync,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        command_tx: Sender<Parameters>,
        recorder_id: Uuid,
        recorder_name: String,
    ) {
        info!("Service discovery task started");

        while !shutdown.load(Ordering::Relaxed) {
            tokio::select! {
                event = browser.next() => {
                    match event {
                        Some(Ok(BrowserEvent::Add(discovery))) => {
                            info!("Discovered service: {}", discovery.name());

                            let port = *discovery.port();
                            let host_name = discovery.host_name().to_string();
                            let mut addresses = Vec::new();

                            let primary_address = discovery.address().to_string();
                            if !primary_address.is_empty() {
                                addresses.push(primary_address);
                            }

                            if !host_name.is_empty() && !addresses.contains(&host_name) {
                                addresses.push(host_name.clone());
                            }

                            if addresses.is_empty() {
                                addresses.push(discovery.name().to_string());
                            }

                            let endpoint = addresses
                                .first()
                                .cloned()
                                .unwrap_or_else(|| host_name.clone());
                            let url = format!("http://{}:{}", endpoint, port);

                            info!(
                                "Discovered service: {} (endpoints: {:?})",
                                discovery.name(),
                                addresses
                            );

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
                                    info!("Connected to service {} via {}", discovery.name(), url);

                                    let sender = command_tx.clone();
                                    if let Err(e) = client
                                        .start_command_listener({
                                            let sender = sender.clone();
                                            move |command| match command {
                                                ServerCommand::CutSession => {
                                                    if let Err(err) =
                                                        sender.try_send(Parameters::Cut())
                                                    {
                                                        warn!("Failed to queue cut command: {}", err);
                                                    }
                                                }
                                                ServerCommand::Reboot => {
                                                    warn!("Received Reboot command from server");
                                                }
                                            }
                                        })
                                        .await
                                    {
                                        error!("Failed to start command listener: {}", e);
                                    }

                                    let client_info = ClientInfo::new(
                                        client,
                                        discovery.name().to_string(),
                                        addresses,
                                        url,
                                    );

                                    clients
                                        .lock()
                                        .await
                                        .insert(discovery.name().to_string(), client_info);
                                }
                                Err(e) => {
                                    error!("Failed to connect to {}: {}", discovery.name(), e);
                                }
                            }
                        }
                        Some(Ok(BrowserEvent::Remove(removal))) => {
                            info!("Service removed: {}", removal.name());

                            if let Some(mut client_info) = clients.lock().await.remove(removal.name()) {
                                client_info.client.disconnect().await;
                                info!(
                                    "Disconnected from service: {} at {}",
                                    client_info.service_name, client_info.connection_url
                                );
                            } else {
                                warn!("Tried to remove unknown service: {}", removal.name());
                            }
                        }
                        Some(Err(e)) => {
                            warn!("Service discovery error: {}", e);
                        }
                        None => {
                            info!("mDNS browser stream ended");
                            break;
                        }
                    }
                }
                _ = sleep(Duration::from_millis(500)) => {
                    if shutdown.load(Ordering::Relaxed) {
                        info!("Discovery shutdown requested");
                        break;
                    }
                }
            }
        }

        if let Err(e) = browser.shutdown().await {
            warn!("Failed to shut down mDNS browser: {}", e);
        }

        info!("Service discovery task stopped");
    }

    /// Audio processing background task
    async fn audio_processing_task(
        mut consumer: AudioRingBufferConsumer,
        mut parameter_consumer: ParameterRingBufferConsumer,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
        status_notify: Arc<Notify>,
        num_channels: usize,
        frames_per_chunk: usize,
        detector_threshold: f64,
        detector_succession: u32,
    ) {
        info!("Audio processing task started");

        let mut sample_buffer = vec![0i16; num_channels * frames_per_chunk];
        let mut chunk_counter = 0u32;
        let mut interval = interval(Duration::from_millis(100)); // 10 chunks per second
        let mut session_id = Uuid::new_v4();
        let mut rms_counter: u32 = 0;
        let mut recording = false;

        while !shutdown.load(Ordering::Relaxed) {
            interval.tick().await;

            let mut command_buf = [Parameters::Cut(); 8];
            loop {
                let count = parameter_consumer.pop_slice(&mut command_buf);
                if count == 0 {
                    break;
                }
                for &command in command_buf.iter().take(count) {
                    match command {
                        Parameters::Cut() => {
                            let new_session = Uuid::new_v4();
                            info!("Cut command received; starting new session {}", new_session);
                            session_id = new_session;
                            chunk_counter = 0;
                            status_notify.notify_waiters();
                        }
                        Parameters::Shutdown() => {
                            info!("Shutdown command received via parameter channel");
                            shutdown.store(true, Ordering::Relaxed);
                            status_notify.notify_waiters();
                        }
                    }
                }
            }

            if shutdown.load(Ordering::Relaxed) {
                break;
            }

            let samples_read = consumer.pop_slice(&mut sample_buffer);

            if samples_read == 0 {
                continue;
            }

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

            if rms_percent < detector_threshold {
                if rms_counter > 0 {
                    rms_counter -= 1;
                    if rms_counter == 0 && recording {
                        recording = false;

                        info!("Detector transitioned to SILENT (rms {:.2}%)", rms_percent);
                        rms_counter = 0;

                        status_notify.notify_waiters();
                    }
                }
            } else {
                if rms_counter < detector_succession {
                    rms_counter += 1;
                    if rms_counter >= detector_succession && !recording {
                        recording = true;

                        info!("Detector transitioned to SIGNAL (rms {:.2}%)", rms_percent);
                        session_id = Uuid::new_v4();
                        chunk_counter = 0;
                        rms_counter = detector_succession;

                        status_notify.notify_waiters();
                    }
                }
            }

            {
                let mut status_guard = status.lock().await;
                status_guard.signal_status = if recording {
                    SignalStatus::Signal
                } else {
                    SignalStatus::NoSignal
                };
                status_guard.rms_percent = rms_percent;
                status_guard.clipping = clipping;
            }

            if !recording {
                continue;
            }

            let chunk_template = AudioChunk {
                session_id: session_id.to_string(),
                chunk_count: chunk_counter,
                data: sample_buffer[..samples_read]
                    .iter()
                    .map(|&sample| (sample as u16) as u32)
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
        }

        if recording {
            {
                let mut status_guard = status.lock().await;
                status_guard.signal_status = SignalStatus::NoSignal;
                status_guard.rms_percent = 0.0;
                status_guard.clipping = false;
            }
            let final_status = {
                let status_guard = status.lock().await;
                status_guard.clone()
            };
            status_notify.notify_waiters();

            let mut clients_guard = clients.lock().await;
            let client_count = clients_guard.len();
            info!(
                "Dispatching final recorder status to {} client(s)",
                client_count
            );
            let mut failed_clients = Vec::new();

            for (name, client_info) in clients_guard.iter_mut() {
                if let Err(e) = client_info
                    .client
                    .set_recorder_status(final_status.clone())
                    .await
                {
                    warn!(
                        "Failed to send final status to {} at {}: {}",
                        client_info.service_name, client_info.connection_url, e
                    );
                    failed_clients.push(name.clone());
                }
            }

            for name in failed_clients {
                if let Some(mut client_info) = clients_guard.remove(&name) {
                    let service_name = client_info.service_name.clone();
                    let service_url = client_info.connection_url.clone();
                    client_info.client.disconnect().await;
                    warn!(
                        "Removed failed client {} at {} during final status update",
                        service_name, service_url
                    );
                }
            }
        }

        info!("Audio processing task stopped");
    }

    /// Status update background task
    async fn status_update_task(
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
        status_notify: Arc<Notify>,
    ) {
        info!("Status update task started");

        let mut interval = interval(Duration::from_secs(1)); // Update every second

        while !shutdown.load(Ordering::Relaxed) {
            tokio::select! {
                _ = interval.tick() => {},
                _ = status_notify.notified() => {},
            }

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
                debug!("Status update sent to {} client(s)", clients_guard.len());
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
