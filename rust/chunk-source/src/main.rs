//! Session Recorder
//!
//! 1. mDNS service discovery running in background
//! 2. Automatic gRPC client management for discovered services
//! 3. Capture audio, detect signal level, transmit chunks only while recording
//! 4. Status updates to all connected clients

use chunk_source::audio::{
    alsa::{AudioSettings, configure_input_device},
    callback_thread::start_callback_thread,
    channels::{CaptureConsumer, new_capture_ring},
};
use chunk_source::grpc::chunk_sink_client::{
    AudioChunk, ChunkSinkClientService, ChunkSinkConfig, RecorderStatusInfo,
    common::SignalStatus,
};
use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::{error, info, warn};
use ringbuf::traits::Consumer;
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::{Duration, SystemTime};
use tokio::time::interval;
use uuid::Uuid;

const DETECTOR_THRESHOLD_PERCENT: f64 = 5.0;
const DETECTOR_SUCCESSION: u32 = 3;
const CHUNK_INTERVAL: Duration = Duration::from_millis(100);
const STATUS_INTERVAL: Duration = Duration::from_secs(5);
const CHUNK_BUFFER_SAMPLES: usize = 1024;

pub struct SessionRecorder {
    service_tracker: Option<ServiceTracker>,
    service_event_receiver: Option<std::sync::mpsc::Receiver<ServiceEvent>>,

    grpc_clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,

    audio_settings: AudioSettings,
    capture_consumer: Option<CaptureConsumer>,
    callback_handle: Option<thread::JoinHandle<()>>,

    recorder_status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,

    shutdown_signal: Arc<AtomicBool>,

    discovery_handle: Option<tokio::task::JoinHandle<()>>,
    audio_handle: Option<tokio::task::JoinHandle<()>>,
    status_handle: Option<tokio::task::JoinHandle<()>>,
}

struct ClientInfo {
    client: ChunkSinkClientService,
    last_successful_send: Option<SystemTime>,
}

impl ClientInfo {
    fn new(client: ChunkSinkClientService) -> Self {
        Self {
            client,
            last_successful_send: None,
        }
    }

    fn mark_successful_send(&mut self) {
        self.last_successful_send = Some(SystemTime::now());
    }
}

impl SessionRecorder {
    pub fn new() -> Self {
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
            capture_consumer: None,
            callback_handle: None,
            recorder_status: Arc::new(tokio::sync::Mutex::new(initial_status)),
            shutdown_signal: Arc::new(AtomicBool::new(false)),
            discovery_handle: None,
            audio_handle: None,
            status_handle: None,
        }
    }

    pub async fn start(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        info!("Starting Session Recorder");

        self.setup_service_discovery().await?;
        self.setup_audio_processing().await?;

        self.start_discovery_task().await;
        self.start_audio_processing_task().await;
        self.start_status_update_task().await;

        info!("Session Recorder started successfully");
        Ok(())
    }

    pub async fn stop(&mut self) {
        info!("Stopping Session Recorder");

        self.shutdown_signal.store(true, Ordering::Relaxed);

        if let Some(mut tracker) = self.service_tracker.take() {
            tracker.stop();
        }

        if let Some(handle) = self.discovery_handle.take() {
            let _ = handle.await;
        }
        if let Some(handle) = self.audio_handle.take() {
            let _ = handle.await;
        }
        if let Some(handle) = self.status_handle.take() {
            let _ = handle.await;
        }
        if let Some(handle) = self.callback_handle.take() {
            let _ = handle.join();
        }

        let mut clients = self.grpc_clients.lock().await;
        for (name, mut client_info) in clients.drain() {
            client_info.client.disconnect().await;
            info!("Disconnected from service: {}", name);
        }

        info!("Session Recorder stopped");
    }

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

    async fn setup_audio_processing(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let capture_pcm = configure_input_device(&self.audio_settings)?;

        let (capture_producer, capture_consumer) =
            new_capture_ring(self.audio_settings.buffer_size as usize * 4);

        let callback_handle = start_callback_thread(
            self.audio_settings.num_channels as usize,
            self.audio_settings.period_size as usize,
            capture_pcm,
            capture_producer,
            self.shutdown_signal.clone(),
        );

        self.capture_consumer = Some(capture_consumer);
        self.callback_handle = Some(callback_handle);

        info!("Audio processing initialized");
        Ok(())
    }

    async fn start_discovery_task(&mut self) {
        let event_receiver = self.service_event_receiver.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();

        let handle = tokio::spawn(async move {
            Self::discovery_task(event_receiver, clients, shutdown).await;
        });

        self.discovery_handle = Some(handle);
    }

    async fn start_audio_processing_task(&mut self) {
        let capture_consumer = self.capture_consumer.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);

        let handle = tokio::spawn(async move {
            Self::audio_processing_task(capture_consumer, clients, shutdown, status).await;
        });

        self.audio_handle = Some(handle);
    }

    async fn start_status_update_task(&mut self) {
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);

        let handle = tokio::spawn(async move {
            Self::status_update_task(clients, shutdown, status).await;
        });

        self.status_handle = Some(handle);
    }

    async fn discovery_task(
        event_receiver: std::sync::mpsc::Receiver<ServiceEvent>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
    ) {
        info!("Service discovery task started");

        while !shutdown.load(Ordering::Relaxed) {
            match event_receiver.recv_timeout(Duration::from_secs(1)) {
                Ok(ServiceEvent::ServiceDiscovered(service_info)) => {
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
                        };

                        let mut client = ChunkSinkClientService::new(config);

                        match client.connect().await {
                            Ok(_) => {
                                info!("Connected to service: {}", service_info.instance_name);

                                let client_info = ClientInfo::new(client);

                                clients
                                    .lock()
                                    .await
                                    .insert(service_info.instance_name.clone(), client_info);
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
                Ok(ServiceEvent::ServiceRemoved(instance_name)) => {
                    info!("Service removed: {}", instance_name);

                    if let Some(mut client_info) = clients.lock().await.remove(&instance_name) {
                        client_info.client.disconnect().await;
                        info!("Disconnected from service: {}", instance_name);
                    }
                }
                Ok(ServiceEvent::ServiceUpdated(service_info)) => {
                    info!("Service updated: {}", service_info.instance_name);
                }
                Err(_) => {
                    // Timeout - normal behavior
                }
            }
        }

        info!("Service discovery task stopped");
    }

    async fn audio_processing_task(
        mut capture_consumer: CaptureConsumer,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
    ) {
        info!("Audio processing task started");

        let mut audio_buffer = vec![0.0f32; CHUNK_BUFFER_SAMPLES];
        let mut chunk_counter = 0u32;
        let mut tick = interval(CHUNK_INTERVAL);

        // Recording-state detector
        let mut above_count: u32 = 0;
        let mut below_count: u32 = 0;
        let mut recording = false;
        let mut session_id: Option<String> = None;

        while !shutdown.load(Ordering::Relaxed) {
            tick.tick().await;

            let samples_read = capture_consumer.consumer.pop_slice(&mut audio_buffer);

            if samples_read == 0 {
                let mut status_guard = status.lock().await;
                status_guard.signal_status = SignalStatus::NoSignal;
                status_guard.rms_percent = 0.0;
                status_guard.clipping = false;
                continue;
            }

            let samples = &audio_buffer[..samples_read];

            let mean_square: f32 =
                samples.iter().map(|&x| x * x).sum::<f32>() / samples_read as f32;
            let rms_percent = (mean_square.sqrt() * 100.0).min(100.0) as f64;
            let clipping = samples.iter().any(|&x| x.abs() > 0.95);

            {
                let mut status_guard = status.lock().await;
                status_guard.signal_status = if rms_percent >= DETECTOR_THRESHOLD_PERCENT {
                    SignalStatus::Signal
                } else {
                    SignalStatus::NoSignal
                };
                status_guard.rms_percent = rms_percent;
                status_guard.clipping = clipping;
            }

            // State transitions
            if rms_percent >= DETECTOR_THRESHOLD_PERCENT {
                above_count = above_count.saturating_add(1);
                below_count = 0;
                if !recording && above_count >= DETECTOR_SUCCESSION {
                    recording = true;
                    let new_id = Uuid::new_v4().to_string();
                    info!("Recording started, session: {}", new_id);
                    session_id = Some(new_id);
                }
            } else {
                below_count = below_count.saturating_add(1);
                above_count = 0;
                if recording && below_count >= DETECTOR_SUCCESSION {
                    recording = false;
                    info!("Recording stopped");
                    session_id = None;
                }
            }

            if !recording {
                continue;
            }

            let Some(current_session) = session_id.as_ref() else {
                continue;
            };

            // Pre-encode chunk data once
            let data: Vec<u32> = samples
                .iter()
                .map(|&sample| {
                    let scaled = (sample + 1.0) / 2.0;
                    (scaled.clamp(0.0, 1.0) * u32::MAX as f32) as u32
                })
                .collect();

            let mut clients_guard = clients.lock().await;
            let mut failed_clients = Vec::new();

            for (name, client_info) in clients_guard.iter_mut() {
                let chunk = AudioChunk {
                    session_id: current_session.clone(),
                    chunk_count: chunk_counter,
                    data: data.clone(),
                    timestamp: SystemTime::now(),
                };

                match client_info.client.set_chunks(chunk).await {
                    Ok(_) => client_info.mark_successful_send(),
                    Err(e) => {
                        warn!("Failed to send chunk to {}: {}", name, e);
                        failed_clients.push(name.clone());
                    }
                }
            }

            for name in failed_clients {
                if let Some(mut client_info) = clients_guard.remove(&name) {
                    client_info.client.disconnect().await;
                    warn!("Removed failed client: {}", name);
                }
            }

            chunk_counter = chunk_counter.wrapping_add(1);
        }

        info!("Audio processing task stopped");
    }

    async fn status_update_task(
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
    ) {
        info!("Status update task started");

        let mut tick = interval(STATUS_INTERVAL);

        while !shutdown.load(Ordering::Relaxed) {
            tick.tick().await;

            let current_status = status.lock().await.clone();

            let mut clients_guard = clients.lock().await;
            let mut failed_clients = Vec::new();

            for (name, client_info) in clients_guard.iter_mut() {
                if let Err(e) = client_info
                    .client
                    .set_recorder_status(current_status.clone())
                    .await
                {
                    warn!("Failed to send status to {}: {}", name, e);
                    failed_clients.push(name.clone());
                }
            }

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

    pub fn is_running(&self) -> bool {
        !self.shutdown_signal.load(Ordering::Relaxed)
    }

    pub async fn get_status(&self) -> RecorderStatusInfo {
        self.recorder_status.lock().await.clone()
    }

    pub async fn get_client_count(&self) -> usize {
        self.grpc_clients.lock().await.len()
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    info!("Starting Session Recorder");

    let mut recorder = SessionRecorder::new();
    recorder.start().await?;

    let shutdown_signal = recorder.shutdown_signal.clone();
    ctrlc::set_handler(move || {
        info!("Received shutdown signal");
        shutdown_signal.store(true, Ordering::Relaxed);
    })?;

    info!("Session Recorder running. Press Ctrl+C to stop.");

    while recorder.is_running() {
        tokio::time::sleep(Duration::from_secs(1)).await;

        let status = recorder.get_status().await;
        let client_count = recorder.get_client_count().await;

        if client_count > 0 {
            info!(
                "Connected to {} server(s), RMS: {:.1}%, Signal: {:?}",
                client_count, status.rms_percent, status.signal_status
            );
        }
    }

    recorder.stop().await;
    info!("Session Recorder stopped");
    Ok(())
}
