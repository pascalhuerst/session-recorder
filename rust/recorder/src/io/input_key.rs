//! Input key handling for Linux using evdev
//!
//! This module provides a high-level interface for handling input events from
//! Linux input devices (keyboards, buttons, etc.) using the evdev subsystem.

use evdev::{Device, EventType, InputEvent, KeyCode};
use log::{error, info, warn};
use std::collections::HashMap;
use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};

/// Callback function type for key press events
pub type PressCallback = Box<dyn Fn() + Send + Sync>;

/// Callback function type for key release events
/// The parameter is the duration the key was held down
pub type ReleaseCallback = Box<dyn Fn(Duration) + Send + Sync>;

/// Event handler for a specific key
#[derive(Clone)]
pub struct KeyEvent {
    pub press_callback: Arc<PressCallback>,
    pub release_callback: Arc<ReleaseCallback>,
    pub press_timestamp: Option<Instant>,
}

/// Input key handler that monitors Linux input events
pub struct InputKey {
    device: Arc<Mutex<Device>>,
    event_map: Arc<Mutex<HashMap<KeyCode, KeyEvent>>>,
    is_running: Arc<AtomicBool>,
    terminate_request: Arc<AtomicBool>,
    worker_handle: Option<thread::JoinHandle<()>>,
}

impl InputKey {
    /// Create a new InputKey instance for the specified input device number
    ///
    /// # Arguments
    /// * `input_nr` - The input device number (corresponds to /dev/input/eventX)
    ///
    /// # Returns
    /// * `Result<Self, Box<dyn std::error::Error>>` - The InputKey instance or an error
    pub fn new(input_nr: u32) -> Result<Self, Box<dyn std::error::Error>> {
        let device_path = format!("/dev/input/event{}", input_nr);
        Self::from_path(&device_path)
    }

    /// Create a new InputKey instance from a device path
    ///
    /// # Arguments
    /// * `device_path` - Path to the input device (e.g., "/dev/input/event0")
    ///
    /// # Returns
    /// * `Result<Self, Box<dyn std::error::Error>>` - The InputKey instance or an error
    pub fn from_path<P: AsRef<Path>>(device_path: P) -> Result<Self, Box<dyn std::error::Error>> {
        let device = Device::open(device_path)?;

        info!(
            "Opened input device: {}",
            device.name().unwrap_or("unknown")
        );

        Ok(Self {
            device: Arc::new(Mutex::new(device)),
            event_map: Arc::new(Mutex::new(HashMap::new())),
            is_running: Arc::new(AtomicBool::new(false)),
            terminate_request: Arc::new(AtomicBool::new(false)),
            worker_handle: None,
        })
    }

    /// Register a key event handler
    ///
    /// # Arguments
    /// * `key` - The key code to monitor
    /// * `press_callback` - Callback function called when the key is pressed
    /// * `release_callback` - Callback function called when the key is released
    pub fn register_key<P, R>(&mut self, key: KeyCode, press_callback: P, release_callback: R)
    where
        P: Fn() + Send + Sync + 'static,
        R: Fn(Duration) + Send + Sync + 'static,
    {
        let event = KeyEvent {
            press_callback: Arc::new(Box::new(press_callback)),
            release_callback: Arc::new(Box::new(release_callback)),
            press_timestamp: None,
        };

        let mut event_map = self.event_map.lock().unwrap();
        event_map.insert(key, event);

        info!("Registered key handler for key: {:?}", key);
    }

    /// Start monitoring input events
    ///
    /// This method starts a background thread that monitors input events
    /// and calls the registered callbacks when keys are pressed or released.
    pub fn start(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        if self.is_running.load(Ordering::Relaxed) {
            warn!("InputKey is already running");
            return Ok(());
        }

        self.is_running.store(true, Ordering::Relaxed);
        self.terminate_request.store(false, Ordering::Relaxed);

        let device = Arc::clone(&self.device);
        let event_map = Arc::clone(&self.event_map);
        let is_running = Arc::clone(&self.is_running);
        let terminate_request = Arc::clone(&self.terminate_request);

        let worker_handle = thread::spawn(move || {
            Self::worker_thread(device, event_map, is_running, terminate_request);
        });

        self.worker_handle = Some(worker_handle);
        info!("Started input key monitoring");

        Ok(())
    }

    /// Stop monitoring input events
    ///
    /// This method stops the background thread and waits for it to complete.
    pub fn stop(&mut self) {
        if !self.is_running.load(Ordering::Relaxed) {
            return;
        }

        info!("Stopping input key monitoring...");
        self.terminate_request.store(true, Ordering::Relaxed);

        if let Some(handle) = self.worker_handle.take()
            && let Err(e) = handle.join()
        {
            error!("Error joining worker thread: {:?}", e);
        }

        self.is_running.store(false, Ordering::Relaxed);
        info!("Input key monitoring stopped");
    }

    /// Worker thread function that handles input events
    fn worker_thread(
        device: Arc<Mutex<Device>>,
        event_map: Arc<Mutex<HashMap<KeyCode, KeyEvent>>>,
        is_running: Arc<AtomicBool>,
        terminate_request: Arc<AtomicBool>,
    ) {
        const POLL_TIMEOUT: Duration = Duration::from_millis(100);

        while !terminate_request.load(Ordering::Relaxed) {
            // Check for events without blocking
            let has_events = {
                let mut device_guard = match device.lock() {
                    Ok(guard) => guard,
                    Err(e) => {
                        error!("Failed to lock device: {}", e);
                        break;
                    }
                };

                // Use a non-blocking approach to check for events
                match device_guard.fetch_events() {
                    Ok(events) => {
                        let collected_events: Vec<_> = events.collect();
                        if collected_events.is_empty() {
                            false
                        } else {
                            // Process events immediately while we have the lock
                            for event in collected_events {
                                if event.event_type() == EventType::KEY {
                                    let key = KeyCode::new(event.code());
                                    Self::handle_key_event(key, event, &event_map);
                                }
                            }
                            true
                        }
                    }
                    Err(e) => {
                        if e.kind() == std::io::ErrorKind::WouldBlock {
                            // No events available
                            false
                        } else {
                            error!("Error fetching events: {}", e);
                            break;
                        }
                    }
                }
            };

            // If no events were processed, sleep briefly to prevent busy waiting
            if !has_events {
                thread::sleep(POLL_TIMEOUT);
            } else {
                // Small sleep to prevent busy waiting even when processing events
                thread::sleep(Duration::from_millis(1));
            }
        }

        is_running.store(false, Ordering::Relaxed);
        info!("Input key worker thread exiting");
    }

    /// Handle a key event
    fn handle_key_event(
        key: KeyCode,
        event: InputEvent,
        event_map: &Arc<Mutex<HashMap<KeyCode, KeyEvent>>>,
    ) {
        let mut event_map_guard = match event_map.lock() {
            Ok(guard) => guard,
            Err(e) => {
                error!("Failed to lock event map: {}", e);
                return;
            }
        };

        if let Some(key_event) = event_map_guard.get_mut(&key) {
            match event.value() {
                1 => {
                    // Key press
                    key_event.press_timestamp = Some(Instant::now());
                    (key_event.press_callback)();
                }
                0 => {
                    // Key release
                    if let Some(press_time) = key_event.press_timestamp.take() {
                        let hold_duration = press_time.elapsed();
                        (key_event.release_callback)(hold_duration);
                    } else {
                        // Release without corresponding press - still call the callback
                        (key_event.release_callback)(Duration::from_secs(0));
                    }
                }
                2 => {
                    // Key repeat (auto-repeat while held) - we ignore this for now
                }
                _ => {
                    // Unknown value
                    warn!(
                        "Unknown key event value: {} for key: {:?}",
                        event.value(),
                        key
                    );
                }
            }
        }
    }

    /// Check if the input key handler is currently running
    pub fn is_running(&self) -> bool {
        self.is_running.load(Ordering::Relaxed)
    }

    /// Get the name of the input device
    pub fn device_name(&self) -> Option<String> {
        self.device
            .lock()
            .ok()
            .and_then(|device| device.name().map(|s| s.to_string()))
    }

    /// List all available input devices
    pub fn list_devices() -> Vec<(String, String)> {
        let mut devices = Vec::new();

        for i in 0..32 {
            let device_path = format!("/dev/input/event{}", i);
            if let Ok(device) = Device::open(&device_path) {
                let name = device.name().unwrap_or("unknown").to_string();
                devices.push((device_path, name));
            }
        }

        devices
    }
}

impl Drop for InputKey {
    fn drop(&mut self) {
        self.stop();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use std::sync::atomic::{AtomicU32, Ordering};

    #[test]
    fn test_input_key_creation() {
        // This test will only pass if there's an actual input device available
        // In a real test environment, you might want to mock the device
        match InputKey::new(0) {
            Ok(_) => {
                // Test passed if device exists
            }
            Err(_) => {
                // Expected if no device exists
            }
        }
    }

    #[test]
    fn test_key_registration() {
        if let Ok(mut input_key) = InputKey::new(0) {
            let press_count = Arc::new(AtomicU32::new(0));
            let release_count = Arc::new(AtomicU32::new(0));

            let press_count_clone = Arc::clone(&press_count);
            let release_count_clone = Arc::clone(&release_count);

            input_key.register_key(
                KeyCode::KEY_SPACE,
                move || {
                    press_count_clone.fetch_add(1, Ordering::Relaxed);
                },
                move |_duration| {
                    release_count_clone.fetch_add(1, Ordering::Relaxed);
                },
            );

            // The callbacks should be registered without panicking
            assert_eq!(press_count.load(Ordering::Relaxed), 0);
            assert_eq!(release_count.load(Ordering::Relaxed), 0);
        }
    }
}
