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
    chunksink::{GetCommandRequest, chunk_sink_client::ChunkSinkClient, command::Command},
    common::SignalStatus,
};
use chunk_source::io::input_key::InputKey;
use chunk_source::io::led::Led;
use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use clap::Parser;
use evdev::KeyCode;
use log::{debug, error, info, warn};
use ringbuf::traits::Consumer;
use std::collections::{HashMap, HashSet, VecDeque};
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::{Duration, SystemTime};
use tokio::time::interval;
use tonic::Request;
use tonic::transport::Channel;
use uuid::Uuid;

const SAMPLE_RATE: u32 = 48000;
const NUM_CHANNELS: u32 = 2;
// Capture ring sized for ~5 s of audio. The producer is the ALSA callback
// thread (RT-priority); if the consumer task ever stalls on a slow gRPC send,
// this gives it plenty of headroom before any sample is dropped.
const CAPTURE_RING_SECONDS: usize = 5;
const RING_DRAIN_INTERVAL: Duration = Duration::from_millis(5);
const RING_POP_CHUNK_SAMPLES: usize = 4096;
const RMS_FLOOR_DB: f64 = -180.0;
const CLIPPING_THRESHOLD: f32 = 32767.0 / 32768.0;
// How often the discovery task re-checks that every known mDNS service still
// has a live gRPC client. Catches reconnects after backend restarts when no
// fresh mDNS event would otherwise wake us up.
const RECONCILE_INTERVAL: Duration = Duration::from_secs(5);

#[derive(Clone)]
struct DetectorConfig {
    sample_rate: u32,
    num_channels: u32,
    threshold_db: f64,
    window_time_s: f64,
    attack_time_s: f64,
    release_time_s: f64,
}

#[derive(Debug, Clone, Copy)]
#[allow(dead_code)] // produced by step 5 (input key) and step 6 (GetCommands)
enum CutTrigger {
    Local,
    Remote,
}

#[derive(Debug, Clone, Copy)]
enum LedRecEvent {
    State(bool),
    NewSession,
}

const LED_BLINK_COUNT: u32 = 10;
const LED_BLINK_HALF_MS: u64 = 80;
const LED_UPLOAD_PULSE_MS: u64 = 50;

#[derive(Parser, Debug, Clone)]
#[command(name = "chunk-source", about = "Session Recorder audio client")]
struct Args {
    /// Unique ID of this recorder (UUID)
    #[arg(long)]
    recorder_id: Uuid,

    /// Human-readable name of this recorder
    #[arg(long)]
    recorder_name: String,

    /// ALSA capture device
    #[arg(long, default_value = "default")]
    device: String,

    /// ALSA period size, in frames
    #[arg(long, default_value_t = 512)]
    period_size: u32,

    /// ALSA buffer size, in frames
    #[arg(long, default_value_t = 2048)]
    buffer_size: u32,

    /// Detector threshold in dB RMS (e.g. -45.0). Above → signal candidate.
    #[arg(long, default_value_t = -45.0)]
    detector_threshold_db: f64,

    /// RMS analysis window, in seconds. Defaults sized so attack requires ≥2 windows.
    #[arg(long, default_value_t = 0.25)]
    window_time: f64,

    /// Silence → signal transition time, in seconds
    #[arg(long, default_value_t = 5.0)]
    attack_time: f64,

    /// Signal → silence transition time, in seconds
    #[arg(long, default_value_t = 30.0)]
    release_time: f64,

    /// sysfs name of the recording-state LED (on = signal, off = silence;
    /// blinks 10× on new session)
    #[arg(long)]
    led_rec_state: Option<String>,

    /// sysfs name of the upload-pulse LED (brief flash per uploaded chunk)
    #[arg(long)]
    led_upload: Option<String>,

    /// `/dev/input/eventNN` number to read a local cut-session key from
    #[arg(long, requires = "input_keycode", requires = "input_hold_ms")]
    input_event: Option<u32>,

    /// Numeric Linux key code that triggers a local cut on release
    #[arg(long, requires = "input_event", requires = "input_hold_ms")]
    input_keycode: Option<u32>,

    /// Minimum hold-down duration (ms) before a release triggers a cut
    #[arg(long, requires = "input_event", requires = "input_keycode")]
    input_hold_ms: Option<u64>,

    /// Seconds of ready-to-send chunks to buffer when no backend is reachable.
    /// When the buffer fills, the oldest chunks are dropped.
    #[arg(long, default_value_t = 120.0)]
    send_buffer_secs: f64,
}

/// Bounded outbox of ready-to-send chunks. Drops the oldest entry when the
/// total buffered audio (counted in u32 samples) would exceed capacity.
/// Drained by the chunk sender task whenever ≥1 client is connected.
struct Outbox {
    deque: VecDeque<AudioChunk>,
    total_samples: usize,
    capacity_samples: usize,
}

impl Outbox {
    fn new(capacity_samples: usize) -> Self {
        Self {
            deque: VecDeque::new(),
            total_samples: 0,
            capacity_samples: capacity_samples.max(1),
        }
    }

    fn push(&mut self, chunk: AudioChunk) {
        self.total_samples += chunk.data.len();
        self.deque.push_back(chunk);
        while self.total_samples > self.capacity_samples {
            let Some(dropped) = self.deque.pop_front() else {
                break;
            };
            self.total_samples = self.total_samples.saturating_sub(dropped.data.len());
            warn!(
                "Outbox full ({}/{} samples), dropping oldest chunk ({} samples, session={})",
                self.total_samples,
                self.capacity_samples,
                dropped.data.len(),
                dropped.session_id,
            );
        }
    }

    fn pop_front(&mut self) -> Option<AudioChunk> {
        let chunk = self.deque.pop_front()?;
        self.total_samples = self.total_samples.saturating_sub(chunk.data.len());
        Some(chunk)
    }

    /// Restore a chunk that was popped but failed to send. Re-prepends without
    /// triggering drop-oldest (the sample count was already accounted for at
    /// original push time and zeroed at pop time).
    fn push_front_in_flight(&mut self, chunk: AudioChunk) {
        self.total_samples += chunk.data.len();
        self.deque.push_front(chunk);
    }

    fn len(&self) -> usize {
        self.deque.len()
    }
}

struct SessionRecorder {
    args: Args,
    service_tracker: Option<ServiceTracker>,
    service_event_receiver: Option<tokio::sync::mpsc::UnboundedReceiver<ServiceEvent>>,

    grpc_clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,

    audio_settings: AudioSettings,
    capture_consumer: Option<CaptureConsumer>,
    callback_handle: Option<thread::JoinHandle<()>>,

    recorder_status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,

    shutdown_signal: Arc<AtomicBool>,

    outbox: Arc<tokio::sync::Mutex<Outbox>>,

    cut_tx: Option<tokio::sync::mpsc::Sender<CutTrigger>>,
    led_rec_tx: Option<tokio::sync::mpsc::Sender<LedRecEvent>>,
    led_upload_tx: Option<tokio::sync::mpsc::Sender<()>>,

    input_key: Option<InputKey>,

    discovery_handle: Option<tokio::task::JoinHandle<()>>,
    audio_handle: Option<tokio::task::JoinHandle<()>>,
    drain_handle: Option<tokio::task::JoinHandle<()>>,
    sender_handle: Option<tokio::task::JoinHandle<()>>,
    led_rec_handle: Option<tokio::task::JoinHandle<()>>,
    led_upload_handle: Option<tokio::task::JoinHandle<()>>,
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
    fn new(args: Args) -> Self {
        let audio_settings = AudioSettings {
            input_device: args.device.clone(),
            num_channels: NUM_CHANNELS,
            period_size: args.period_size,
            buffer_size: args.buffer_size,
            sample_rate: SAMPLE_RATE,
        };

        let initial_status = RecorderStatusInfo {
            signal_status: SignalStatus::NoSignal,
            rms_percent: 0.0,
            clipping: false,
        };

        let capacity_samples = (audio_settings.sample_rate as f64
            * audio_settings.num_channels as f64
            * args.send_buffer_secs) as usize;
        info!(
            "Send-buffer outbox: {} samples (~{:.1} s of audio)",
            capacity_samples, args.send_buffer_secs
        );
        let outbox = Arc::new(tokio::sync::Mutex::new(Outbox::new(capacity_samples)));

        Self {
            args,
            service_tracker: None,
            service_event_receiver: None,
            grpc_clients: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            audio_settings,
            capture_consumer: None,
            callback_handle: None,
            recorder_status: Arc::new(tokio::sync::Mutex::new(initial_status)),
            shutdown_signal: Arc::new(AtomicBool::new(false)),
            outbox,
            cut_tx: None,
            led_rec_tx: None,
            led_upload_tx: None,
            input_key: None,
            discovery_handle: None,
            audio_handle: None,
            drain_handle: None,
            sender_handle: None,
            led_rec_handle: None,
            led_upload_handle: None,
        }
    }

    async fn start(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        info!(
            "Starting Session Recorder (id={}, name={:?})",
            self.args.recorder_id, self.args.recorder_name
        );

        self.setup_service_discovery().await?;
        self.setup_audio_processing().await?;

        self.start_led_tasks();
        self.start_audio_processing_task().await;
        self.start_chunk_sender_task();
        self.start_discovery_task().await;
        self.setup_input_key()?;

        info!("Session Recorder started successfully");
        Ok(())
    }

    async fn stop(&mut self) {
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
        if let Some(handle) = self.drain_handle.take() {
            let _ = handle.await;
        }
        if let Some(handle) = self.sender_handle.take() {
            let _ = handle.await;
        }
        // Dropping the InputKey stops its worker thread (see InputKey::Drop)
        self.input_key.take();
        // Drop LED senders so the LED tasks finish, then await them.
        self.led_rec_tx.take();
        self.led_upload_tx.take();
        if let Some(handle) = self.led_rec_handle.take() {
            let _ = handle.await;
        }
        if let Some(handle) = self.led_upload_handle.take() {
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

        let ring_capacity = CAPTURE_RING_SECONDS
            * self.audio_settings.sample_rate as usize
            * self.audio_settings.num_channels as usize;
        info!(
            "Capture ring: {} samples (~{} s of audio)",
            ring_capacity, CAPTURE_RING_SECONDS
        );

        let (capture_producer, capture_consumer) = new_capture_ring(ring_capacity);

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
        let recorder_id = self.args.recorder_id;
        let recorder_name = self.args.recorder_name.clone();
        let cut_tx = self.cut_tx.clone();

        let handle = tokio::spawn(async move {
            Self::discovery_task(
                event_receiver,
                clients,
                shutdown,
                recorder_id,
                recorder_name,
                cut_tx,
            )
            .await;
        });

        self.discovery_handle = Some(handle);
    }

    fn setup_input_key(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let (Some(event_nr), Some(keycode), Some(hold_ms)) = (
            self.args.input_event,
            self.args.input_keycode,
            self.args.input_hold_ms,
        ) else {
            return Ok(());
        };

        let Some(cut_tx) = self.cut_tx.clone() else {
            warn!("input-key configured but no cut channel available — skipping");
            return Ok(());
        };

        let mut input_key = match InputKey::new(event_nr) {
            Ok(ik) => ik,
            Err(e) => {
                warn!("Cannot open /dev/input/event{}: {}", event_nr, e);
                return Ok(());
            }
        };

        let hold_threshold = Duration::from_millis(hold_ms);
        let key = KeyCode::new(keycode as u16);

        input_key.register_key(
            key,
            || {}, // press: no-op; we act on release
            move |held| {
                if held >= hold_threshold {
                    info!("Input cut triggered (held {:?})", held);
                    let _ = cut_tx.try_send(CutTrigger::Local);
                } else {
                    info!("Input cut ignored (held {:?} < {:?})", held, hold_threshold);
                }
            },
        );

        if let Err(e) = input_key.start() {
            warn!("Failed to start InputKey worker: {}", e);
            return Ok(());
        }

        info!(
            "Listening for cut events on /dev/input/event{} keycode {} (hold ≥ {} ms)",
            event_nr, keycode, hold_ms
        );
        self.input_key = Some(input_key);
        Ok(())
    }

    fn start_led_tasks(&mut self) {
        if let Some(name) = self.args.led_rec_state.as_deref() {
            match Led::new(name) {
                Ok(led) => {
                    let (tx, rx) = tokio::sync::mpsc::channel::<LedRecEvent>(32);
                    let shutdown = self.shutdown_signal.clone();
                    self.led_rec_tx = Some(tx);
                    self.led_rec_handle = Some(tokio::spawn(Self::led_rec_task(led, rx, shutdown)));
                }
                Err(e) => warn!("Cannot open led-rec-state LED '{}': {}", name, e),
            }
        }
        if let Some(name) = self.args.led_upload.as_deref() {
            match Led::new(name) {
                Ok(led) => {
                    let (tx, rx) = tokio::sync::mpsc::channel::<()>(32);
                    let shutdown = self.shutdown_signal.clone();
                    self.led_upload_tx = Some(tx);
                    self.led_upload_handle =
                        Some(tokio::spawn(Self::led_upload_task(led, rx, shutdown)));
                }
                Err(e) => warn!("Cannot open led-upload LED '{}': {}", name, e),
            }
        }
    }

    async fn led_rec_task(
        led: Led,
        mut rx: tokio::sync::mpsc::Receiver<LedRecEvent>,
        shutdown: Arc<AtomicBool>,
    ) {
        let _ = led.off();
        let mut current_state = false;
        while !shutdown.load(Ordering::Relaxed) {
            let Some(event) = rx.recv().await else { break };
            match event {
                LedRecEvent::State(on) => {
                    current_state = on;
                    let _ = if on { led.on() } else { led.off() };
                }
                LedRecEvent::NewSession => {
                    for _ in 0..LED_BLINK_COUNT {
                        if shutdown.load(Ordering::Relaxed) {
                            return;
                        }
                        let _ = led.on();
                        tokio::time::sleep(Duration::from_millis(LED_BLINK_HALF_MS)).await;
                        let _ = led.off();
                        tokio::time::sleep(Duration::from_millis(LED_BLINK_HALF_MS)).await;
                    }
                    let _ = if current_state { led.on() } else { led.off() };
                }
            }
        }
        let _ = led.off();
    }

    async fn led_upload_task(
        led: Led,
        mut rx: tokio::sync::mpsc::Receiver<()>,
        shutdown: Arc<AtomicBool>,
    ) {
        let _ = led.off();
        while !shutdown.load(Ordering::Relaxed) {
            if rx.recv().await.is_none() {
                break;
            }
            // Coalesce queued pulses so we don't flicker forever after a burst
            while rx.try_recv().is_ok() {}
            let _ = led.on();
            tokio::time::sleep(Duration::from_millis(LED_UPLOAD_PULSE_MS)).await;
            let _ = led.off();
        }
        let _ = led.off();
    }

    async fn start_audio_processing_task(&mut self) {
        let capture_consumer = self.capture_consumer.take().unwrap();
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let status = Arc::clone(&self.recorder_status);
        let outbox = Arc::clone(&self.outbox);
        let detector = DetectorConfig {
            sample_rate: SAMPLE_RATE,
            num_channels: NUM_CHANNELS,
            threshold_db: self.args.detector_threshold_db,
            window_time_s: self.args.window_time,
            attack_time_s: self.args.attack_time,
            release_time_s: self.args.release_time,
        };

        let (cut_tx, cut_rx) = tokio::sync::mpsc::channel::<CutTrigger>(8);
        self.cut_tx = Some(cut_tx);

        // Dedicated drain task: pulls samples from the lock-free ring into a
        // tokio channel as fast as it can, so a slow gRPC send in the
        // processing task can never starve the ALSA producer.
        let (samples_tx, samples_rx) = tokio::sync::mpsc::channel::<Vec<f32>>(128);
        let drain_shutdown = self.shutdown_signal.clone();
        let drain_handle = tokio::spawn(async move {
            Self::ring_drain_task(capture_consumer, samples_tx, drain_shutdown).await;
        });
        self.drain_handle = Some(drain_handle);

        let led_rec_tx = self.led_rec_tx.clone();

        let handle = tokio::spawn(async move {
            Self::audio_processing_task(
                samples_rx, clients, shutdown, status, detector, cut_rx, led_rec_tx, outbox,
            )
            .await;
        });

        self.audio_handle = Some(handle);
    }

    fn start_chunk_sender_task(&mut self) {
        let outbox = Arc::clone(&self.outbox);
        let clients = Arc::clone(&self.grpc_clients);
        let shutdown = self.shutdown_signal.clone();
        let led_upload_tx = self.led_upload_tx.clone();

        let handle = tokio::spawn(async move {
            Self::chunk_sender_task(outbox, clients, shutdown, led_upload_tx).await;
        });

        self.sender_handle = Some(handle);
    }

    /// Drain the outbox to all currently connected clients. When zero clients
    /// are connected, idle without dropping chunks (the producer enforces the
    /// time-based capacity via drop-oldest). On full failure, restore the
    /// chunk at the front and back off.
    async fn chunk_sender_task(
        outbox: Arc<tokio::sync::Mutex<Outbox>>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        led_upload_tx: Option<tokio::sync::mpsc::Sender<()>>,
    ) {
        info!("Chunk sender task started");
        let idle_sleep = Duration::from_millis(20);
        let wait_sleep = Duration::from_millis(200);

        while !shutdown.load(Ordering::Relaxed) {
            let Some(chunk) = outbox.lock().await.pop_front() else {
                tokio::time::sleep(idle_sleep).await;
                continue;
            };

            if clients.lock().await.is_empty() {
                outbox.lock().await.push_front_in_flight(chunk);
                tokio::time::sleep(wait_sleep).await;
                continue;
            }

            let mut sent_to_any = false;
            {
                let mut g = clients.lock().await;
                let mut failed = Vec::new();
                for (name, ci) in g.iter_mut() {
                    match ci.client.set_chunks(chunk.clone()).await {
                        Ok(_) => {
                            sent_to_any = true;
                            ci.mark_successful_send();
                        }
                        Err(e) => {
                            warn!("Chunk send to {} failed: {}", name, e);
                            failed.push(name.clone());
                        }
                    }
                }
                for n in failed {
                    if let Some(mut ci) = g.remove(&n) {
                        ci.client.disconnect().await;
                        warn!("Removed failed client during chunk send: {}", n);
                    }
                }
            }

            if sent_to_any {
                if let Some(tx) = &led_upload_tx {
                    let _ = tx.try_send(());
                }
            } else {
                outbox.lock().await.push_front_in_flight(chunk);
                tokio::time::sleep(wait_sleep).await;
            }
        }
        info!("Chunk sender task stopped");
    }

    async fn ring_drain_task(
        mut consumer: CaptureConsumer,
        samples_tx: tokio::sync::mpsc::Sender<Vec<f32>>,
        shutdown: Arc<AtomicBool>,
    ) {
        info!("Ring drain task started");
        let mut buf = vec![0.0f32; RING_POP_CHUNK_SAMPLES];
        let mut tick = interval(RING_DRAIN_INTERVAL);

        while !shutdown.load(Ordering::Relaxed) {
            tick.tick().await;

            loop {
                let n = consumer.consumer.pop_slice(&mut buf);
                if n == 0 {
                    break;
                }
                let chunk = buf[..n].to_vec();
                if samples_tx.send(chunk).await.is_err() {
                    info!("Ring drain task: downstream closed, exiting");
                    return;
                }
                if n < buf.len() {
                    break;
                }
            }
        }
        info!("Ring drain task stopped");
    }

    async fn discovery_task(
        mut event_receiver: tokio::sync::mpsc::UnboundedReceiver<ServiceEvent>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        recorder_id: Uuid,
        recorder_name: String,
        cut_tx: Option<tokio::sync::mpsc::Sender<CutTrigger>>,
    ) {
        info!("Service discovery task started");

        // Local view of every service mdns-sd has told us about (Discovered or
        // Updated, until Removed). Used by the reconciler to re-attach when a
        // client got dropped (e.g. the backend restarted under us) without a
        // fresh mDNS event firing.
        let mut known_services: HashMap<String, chunk_source::mdns::service_tracker::ServiceInfo> =
            HashMap::new();
        let mut reconcile_tick = tokio::time::interval(RECONCILE_INTERVAL);
        // First tick fires immediately; skip it so we don't reconcile-before-we-discover.
        reconcile_tick.tick().await;

        while !shutdown.load(Ordering::Relaxed) {
            tokio::select! {
                maybe_event = event_receiver.recv() => {
                    let Some(event) = maybe_event else {
                        info!("Service tracker channel closed, discovery task exiting");
                        break;
                    };

                    match event {
                        ServiceEvent::ServiceDiscovered(service_info) => {
                            info!("Discovered service: {}", service_info.instance_name);
                            known_services.insert(service_info.instance_name.clone(), service_info.clone());
                            Self::try_attach_service(
                                &service_info,
                                &clients,
                                &shutdown,
                                recorder_id,
                                &recorder_name,
                                cut_tx.as_ref(),
                            )
                            .await;
                        }
                        ServiceEvent::ServiceUpdated(service_info) => {
                            known_services.insert(service_info.instance_name.clone(), service_info.clone());

                            let already_connected = clients
                                .lock()
                                .await
                                .contains_key(&service_info.instance_name);
                            if already_connected {
                                debug!(
                                    "Service updated (already connected): {}",
                                    service_info.instance_name
                                );
                            } else {
                                info!(
                                    "Service updated (not yet connected, attempting): {}",
                                    service_info.instance_name
                                );
                                Self::try_attach_service(
                                    &service_info,
                                    &clients,
                                    &shutdown,
                                    recorder_id,
                                    &recorder_name,
                                    cut_tx.as_ref(),
                                )
                                .await;
                            }
                        }
                        ServiceEvent::ServiceRemoved(instance_name) => {
                            info!("Service removed: {}", instance_name);
                            known_services.remove(&instance_name);

                            if let Some(mut client_info) = clients.lock().await.remove(&instance_name) {
                                client_info.client.disconnect().await;
                                info!("Disconnected from service: {}", instance_name);
                            }
                        }
                    }
                }

                _ = reconcile_tick.tick() => {
                    // Detect services we know about (per mDNS) that lost their
                    // client mid-flight — typically because the backend was
                    // restarted within the daemon's TTL window, so no mDNS
                    // event re-fires, but our previous client failed a send
                    // and got dropped from the clients map.
                    let connected: HashSet<String> = clients
                        .lock()
                        .await
                        .keys()
                        .cloned()
                        .collect();

                    for (name, info) in &known_services {
                        if connected.contains(name) {
                            continue;
                        }
                        info!(
                            "Reconciler: {} has no live client, attempting to (re-)attach",
                            name
                        );
                        Self::try_attach_service(
                            info,
                            &clients,
                            &shutdown,
                            recorder_id,
                            &recorder_name,
                            cut_tx.as_ref(),
                        )
                        .await;
                    }
                }
            }
        }

        info!("Service discovery task stopped");
    }

    /// Attempt to register a ChunkSinkClient + command listener for a service.
    /// No-op if the service has no IPv4 address yet or if we're already
    /// connected to that instance name.
    async fn try_attach_service(
        service_info: &chunk_source::mdns::service_tracker::ServiceInfo,
        clients: &Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: &Arc<AtomicBool>,
        recorder_id: Uuid,
        recorder_name: &str,
        cut_tx: Option<&tokio::sync::mpsc::Sender<CutTrigger>>,
    ) {
        let Some(url) = service_info.connection_url() else {
            debug!(
                "Service {} has no IPv4 address yet, skipping",
                service_info.instance_name
            );
            return;
        };

        // Re-check under lock to avoid racing duplicate attaches.
        if clients
            .lock()
            .await
            .contains_key(&service_info.instance_name)
        {
            return;
        }

        let config = ChunkSinkConfig {
            server_address: url.clone(),
            recorder_id: recorder_id.to_string(),
            recorder_name: recorder_name.to_string(),
            connect_timeout: Duration::from_secs(10),
            request_timeout: Duration::from_secs(5),
        };

        let mut client = ChunkSinkClientService::new(config);

        match client.connect().await {
            Ok(_) => {
                info!(
                    "Connected to service: {} (recorder_id={})",
                    service_info.instance_name, recorder_id
                );

                clients
                    .lock()
                    .await
                    .insert(service_info.instance_name.clone(), ClientInfo::new(client));

                if let Some(cut_tx) = cut_tx {
                    let listener_url = url.clone();
                    let listener_recorder = recorder_id.to_string();
                    let listener_shutdown = shutdown.clone();
                    let listener_name = service_info.instance_name.clone();
                    let cut_tx = cut_tx.clone();
                    info!(
                        "Spawning command listener for {} → {}",
                        listener_name, listener_url
                    );
                    tokio::spawn(async move {
                        command_listener_task(
                            listener_name,
                            listener_url,
                            listener_recorder,
                            cut_tx,
                            listener_shutdown,
                        )
                        .await;
                    });
                } else {
                    warn!(
                        "No cut channel available — command listener for {} not spawned",
                        service_info.instance_name
                    );
                }
            }
            Err(e) => {
                error!("Failed to connect to {}: {}", service_info.instance_name, e);
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn audio_processing_task(
        mut samples_rx: tokio::sync::mpsc::Receiver<Vec<f32>>,
        clients: Arc<tokio::sync::Mutex<HashMap<String, ClientInfo>>>,
        shutdown: Arc<AtomicBool>,
        status: Arc<tokio::sync::Mutex<RecorderStatusInfo>>,
        detector: DetectorConfig,
        mut cut_rx: tokio::sync::mpsc::Receiver<CutTrigger>,
        led_rec_tx: Option<tokio::sync::mpsc::Sender<LedRecEvent>>,
        outbox: Arc<tokio::sync::Mutex<Outbox>>,
    ) {
        info!(
            "Audio processing task started (threshold={} dB, window={}s, attack={}s, release={}s)",
            detector.threshold_db,
            detector.window_time_s,
            detector.attack_time_s,
            detector.release_time_s
        );

        let window_samples = ((detector.sample_rate as f64 * detector.window_time_s) as usize)
            * detector.num_channels as usize;
        let attack_windows =
            ((detector.attack_time_s / detector.window_time_s).ceil() as u32).max(1);
        let release_windows =
            ((detector.release_time_s / detector.window_time_s).ceil() as u32).max(1);

        let mut window_buf: Vec<f32> = Vec::with_capacity(window_samples * 2);

        let mut above_count: u32 = 0;
        let mut below_count: u32 = 0;
        let mut recording = false;
        let mut session_id: Option<String> = None;
        let mut chunk_counter = 0u32;

        while !shutdown.load(Ordering::Relaxed) {
            tokio::select! {
                maybe_chunk = samples_rx.recv() => {
                    let Some(chunk) = maybe_chunk else { break; };
                    window_buf.extend_from_slice(&chunk);
                }
                cut = cut_rx.recv() => {
                    let Some(trigger) = cut else { break; };
                    if recording {
                        let new_id = Uuid::new_v4().to_string();
                        info!("Cut ({:?}): ending current session, starting {}", trigger, new_id);
                        session_id = Some(new_id);
                        chunk_counter = 0;
                        if let Some(tx) = &led_rec_tx {
                            let _ = tx.try_send(LedRecEvent::NewSession);
                            debug!("LED rec-state: NewSession (cut)");
                        }
                    } else {
                        info!("Cut ({:?}) ignored: not currently recording", trigger);
                    }
                    continue;
                }
            }

            // Process every full window currently buffered
            while window_buf.len() >= window_samples {
                let window: Vec<f32> = window_buf.drain(..window_samples).collect();

                let mean_square: f32 =
                    window.iter().map(|&x| x * x).sum::<f32>() / window.len() as f32;
                let rms = mean_square.sqrt();
                let rms_db = if rms < 1e-9 {
                    RMS_FLOOR_DB
                } else {
                    20.0 * (rms as f64).log10()
                };
                let clipping = window.iter().any(|&x| x.abs() >= CLIPPING_THRESHOLD);

                let signal_above = rms_db >= detector.threshold_db;

                debug!(
                    "window: rms={:.2} dB, signal_above={}, above={}/{}, below={}/{}, recording={}, clipping={}",
                    rms_db,
                    signal_above,
                    above_count,
                    attack_windows,
                    below_count,
                    release_windows,
                    recording,
                    clipping
                );

                if signal_above {
                    above_count = above_count.saturating_add(1);
                    below_count = 0;
                    if !recording && above_count >= attack_windows {
                        recording = true;
                        let new_id = Uuid::new_v4().to_string();
                        info!(
                            "Recording started (RMS {:.1} dB), session: {}",
                            rms_db, new_id
                        );
                        session_id = Some(new_id);
                        chunk_counter = 0;
                        if let Some(tx) = &led_rec_tx {
                            let _ = tx.try_send(LedRecEvent::State(true));
                            let _ = tx.try_send(LedRecEvent::NewSession);
                            debug!("LED rec-state: State(true) + NewSession");
                        }
                    }
                } else {
                    below_count = below_count.saturating_add(1);
                    above_count = 0;
                    if recording && below_count >= release_windows {
                        recording = false;
                        info!("Recording stopped (RMS {:.1} dB)", rms_db);
                        session_id = None;
                        if let Some(tx) = &led_rec_tx {
                            let _ = tx.try_send(LedRecEvent::State(false));
                            debug!("LED rec-state: State(false)");
                        }
                    }
                }

                // The signal_status in the wire message must reflect the smoothed
                // (recording) state, not the raw per-window comparison — otherwise
                // a quiet window mid-recording reports NoSignal to the server, which
                // closes the session while we keep streaming chunks for it.
                let status_info = RecorderStatusInfo {
                    signal_status: if recording {
                        SignalStatus::Signal
                    } else {
                        SignalStatus::NoSignal
                    },
                    rms_percent: (rms as f64) * 100.0,
                    clipping,
                };
                {
                    *status.lock().await = status_info.clone();
                }

                // Fan out RecorderStatus to every connected client this window
                {
                    let mut g = clients.lock().await;
                    let n_clients = g.len();
                    let mut failed = Vec::new();
                    for (name, ci) in g.iter_mut() {
                        if let Err(e) = ci.client.set_recorder_status(status_info.clone()).await {
                            warn!("Failed to send status to {}: {}", name, e);
                            failed.push(name.clone());
                        }
                    }
                    for n in failed {
                        if let Some(mut ci) = g.remove(&n) {
                            ci.client.disconnect().await;
                            warn!("Removed failed client during status update: {}", n);
                        }
                    }
                    debug!(
                        "status sent to {} client(s): signal={:?} rms={:.2}% clip={}",
                        n_clients,
                        status_info.signal_status,
                        status_info.rms_percent,
                        status_info.clipping
                    );
                }

                // If recording, enqueue the window's samples for the sender task.
                if recording {
                    let Some(current_session) = session_id.as_ref() else {
                        continue;
                    };

                    let data: Vec<u32> = window
                        .iter()
                        .map(|&s| {
                            let i = (s.clamp(-1.0, 1.0) * 32767.0) as i16;
                            i as i32 as u32
                        })
                        .collect();

                    let chunk = AudioChunk {
                        session_id: current_session.clone(),
                        chunk_count: chunk_counter,
                        data,
                        timestamp: SystemTime::now(),
                    };

                    let queue_len = {
                        let mut g = outbox.lock().await;
                        g.push(chunk);
                        g.len()
                    };
                    debug!(
                        "chunk #{} enqueued (outbox depth={}, session={})",
                        chunk_counter, queue_len, current_session
                    );
                    chunk_counter = chunk_counter.wrapping_add(1);
                }
            }
        }

        info!("Audio processing task stopped");
    }

    fn is_running(&self) -> bool {
        !self.shutdown_signal.load(Ordering::Relaxed)
    }

    async fn get_status(&self) -> RecorderStatusInfo {
        self.recorder_status.lock().await.clone()
    }

    async fn get_client_count(&self) -> usize {
        self.grpc_clients.lock().await.len()
    }
}

/// Subscribe to the ChunkSink server's command stream and forward `CmdCutSession`
/// to the cut channel. Exits when the stream ends or shutdown is signalled.
async fn command_listener_task(
    service_name: String,
    url: String,
    recorder_id: String,
    cut_tx: tokio::sync::mpsc::Sender<CutTrigger>,
    shutdown: Arc<AtomicBool>,
) {
    info!(
        "Command listener: connecting to {} at {}",
        service_name, url
    );
    let endpoint = match Channel::from_shared(url.clone()) {
        Ok(b) => b.connect_timeout(Duration::from_secs(5)),
        Err(e) => {
            warn!("Command listener: invalid URL {}: {}", url, e);
            return;
        }
    };
    let channel = match endpoint.connect().await {
        Ok(c) => c,
        Err(e) => {
            warn!("Command listener: cannot connect to {}: {}", url, e);
            return;
        }
    };

    let mut client = ChunkSinkClient::new(channel);
    info!(
        "Command listener: opening GetCommands stream for recorder {}",
        recorder_id
    );
    let request = Request::new(GetCommandRequest {
        recorder_id: recorder_id.clone(),
    });
    let mut stream = match client.get_commands(request).await {
        Ok(r) => r.into_inner(),
        Err(e) => {
            warn!(
                "Command listener: GetCommands failed for {} (recorder {}): {}",
                service_name, recorder_id, e
            );
            return;
        }
    };

    info!(
        "Command listener attached to {} as recorder {}",
        service_name, recorder_id
    );

    while !shutdown.load(Ordering::Relaxed) {
        match tokio::time::timeout(Duration::from_secs(1), stream.message()).await {
            Ok(Ok(Some(cmd))) => match cmd.command {
                Some(Command::CmdCutSession(_)) => {
                    info!("Received CutSession from {}", service_name);
                    let _ = cut_tx.try_send(CutTrigger::Remote);
                }
                Some(Command::Reboot(_)) => {
                    info!("Received Reboot from {} (no-op)", service_name);
                }
                None => {
                    warn!("Empty command from {}", service_name);
                }
            },
            Ok(Ok(None)) => {
                info!("Command stream from {} closed by server", service_name);
                break;
            }
            Ok(Err(e)) => {
                warn!("Command stream from {} errored: {}", service_name, e);
                break;
            }
            Err(_) => {
                // 1s timeout — loop back and check shutdown
            }
        }
    }

    info!("Command listener for {} stopped", service_name);
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Default: info everywhere, but silence the chatty mdns_sd crate.
    // Override with e.g. RUST_LOG=chunk_source=debug,mdns_sd=warn
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info,mdns_sd=off"))
        .init();

    let args = Args::parse();

    info!("Starting Session Recorder");

    let mut recorder = SessionRecorder::new(args);
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
