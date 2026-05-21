//! mDNS service tracker for chunk-sink servers
//!
//! This module provides functionality to discover and track chunk-sink servers
//! on the local network using mDNS (Multicast DNS) service discovery.

use log::{debug, info, warn};
use mdns_sd::{ServiceDaemon, ServiceEvent as MdnsServiceEvent};
use std::collections::HashMap;
use std::net::Ipv4Addr;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};
use tokio::sync::mpsc;

/// Information about a discovered chunk-sink server
#[derive(Debug, Clone)]
pub struct ServiceInfo {
    /// Service instance name
    pub instance_name: String,
    /// Hostname of the server
    pub hostname: String,
    /// IP addresses of the server
    pub addresses: Vec<Ipv4Addr>,
    /// Port number the server is listening on
    pub port: u16,
    /// TXT record properties
    pub properties: HashMap<String, String>,
    /// Time when the service was first discovered
    pub discovered_at: Instant,
    /// Time when the service was last seen
    pub last_seen: Instant,
}

impl ServiceInfo {
    /// Get the primary address for connecting to this service
    pub fn primary_address(&self) -> Option<Ipv4Addr> {
        self.addresses.first().copied()
    }

    /// Get the full connection URL for this service
    pub fn connection_url(&self) -> Option<String> {
        self.primary_address()
            .map(|addr| format!("http://{}:{}", addr, self.port))
    }
}

/// Events that can occur during service discovery. Removal is driven solely
/// by mdns-sd's own ServiceRemoved event — there is no local timeout.
#[derive(Debug, Clone)]
pub enum ServiceEvent {
    ServiceDiscovered(ServiceInfo),
    ServiceUpdated(ServiceInfo),
    ServiceRemoved(String), // instance_name
}

/// Configuration for the service tracker
#[derive(Debug, Clone)]
pub struct ServiceTrackerConfig {
    /// Service type to search for
    pub service_type: String,
    /// Maximum number of services to track
    pub max_services: usize,
}

impl Default for ServiceTrackerConfig {
    fn default() -> Self {
        Self {
            service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
            max_services: 100,
        }
    }
}

/// Service tracker that discovers and monitors chunk-sink servers
pub struct ServiceTracker {
    config: ServiceTrackerConfig,
    daemon: Option<ServiceDaemon>,
    services: Arc<Mutex<HashMap<String, ServiceInfo>>>,
    event_sender: Option<mpsc::UnboundedSender<ServiceEvent>>,
    worker_handle: Option<thread::JoinHandle<()>>,
    is_running: Arc<Mutex<bool>>,
}

impl ServiceTracker {
    /// Create a new service tracker
    pub fn new(config: ServiceTrackerConfig) -> Result<Self, Box<dyn std::error::Error>> {
        let daemon = ServiceDaemon::new()?;

        Ok(Self {
            config,
            daemon: Some(daemon),
            services: Arc::new(Mutex::new(HashMap::new())),
            event_sender: None,
            worker_handle: None,
            is_running: Arc::new(Mutex::new(false)),
        })
    }

    /// Start discovering services. Returns an unbounded receiver of service
    /// events; the sender lives in the mdns worker thread and uses
    /// `UnboundedSender::send` (synchronous, non-blocking) since it runs on a
    /// plain `std::thread`, not a tokio task.
    pub fn start(
        &mut self,
    ) -> Result<mpsc::UnboundedReceiver<ServiceEvent>, Box<dyn std::error::Error>> {
        if *self.is_running.lock().unwrap() {
            return Err("Service tracker is already running".into());
        }

        let (event_sender, event_receiver) = mpsc::unbounded_channel();
        self.event_sender = Some(event_sender.clone());

        let daemon = self.daemon.take().ok_or("Daemon not available")?;
        let service_type = self.config.service_type.clone();
        let services = Arc::clone(&self.services);
        let is_running = Arc::clone(&self.is_running);
        let worker_event_sender = event_sender.clone();

        *self.is_running.lock().unwrap() = true;

        // Start the main discovery worker
        let worker_handle = thread::spawn(move || {
            info!("Starting mDNS service discovery for: {}", service_type);

            // Browse for services
            let receiver = match daemon.browse(&service_type) {
                Ok(receiver) => {
                    info!("mDNS discovery started for: {}", service_type);
                    receiver
                }
                Err(e) => {
                    warn!("Could not start mDNS browsing: {}", e);
                    *is_running.lock().unwrap() = false;
                    return;
                }
            };

            // Small delay to allow the service to initialize
            thread::sleep(Duration::from_millis(50));

            // Process mDNS events. recv_timeout returns Err on both Timeout and
            // Disconnected — distinguish via the receiver's own state instead of
            // string-matching the error message.
            while *is_running.lock().unwrap() {
                match receiver.recv_timeout(Duration::from_secs(1)) {
                    Ok(event) => {
                        Self::handle_mdns_event(event, &services, &worker_event_sender);
                    }
                    Err(_) => {
                        if receiver.is_disconnected() {
                            warn!("mDNS browse channel disconnected, exiting worker");
                            break;
                        }
                        // Plain timeout: keep polling
                    }
                }
            }

            info!("mDNS service discovery worker stopped");
        });

        self.worker_handle = Some(worker_handle);
        drop(event_sender);

        info!("Service tracker started successfully");
        Ok(event_receiver)
    }

    /// Stop the service tracker
    pub fn stop(&mut self) {
        debug!("Stopping service tracker...");

        // Signal threads to stop
        *self.is_running.lock().unwrap() = false;

        // Give threads a moment to stop gracefully
        thread::sleep(Duration::from_millis(50));

        if let Some(handle) = self.worker_handle.take() {
            let _ = handle.join();
        }

        self.event_sender = None;
        debug!("Service tracker stopped");
    }

    /// Get a list of currently discovered services
    pub fn get_services(&self) -> Vec<ServiceInfo> {
        let services = self.services.lock().unwrap();
        services.values().cloned().collect()
    }

    /// Get a specific service by instance name
    pub fn get_service(&self, instance_name: &str) -> Option<ServiceInfo> {
        let services = self.services.lock().unwrap();
        services.get(instance_name).cloned()
    }

    /// Get the number of currently tracked services
    pub fn service_count(&self) -> usize {
        let services = self.services.lock().unwrap();
        services.len()
    }

    /// Check if the tracker is running
    pub fn is_running(&self) -> bool {
        *self.is_running.lock().unwrap()
    }

    /// Handle mDNS events and convert them to service events
    fn handle_mdns_event(
        event: MdnsServiceEvent,
        services: &Arc<Mutex<HashMap<String, ServiceInfo>>>,
        event_sender: &mpsc::UnboundedSender<ServiceEvent>,
    ) {
        match event {
            MdnsServiceEvent::ServiceResolved(info) => {
                debug!("Service resolved: {}", info.get_fullname());

                let service_info = ServiceInfo {
                    instance_name: info.get_fullname().to_string(),
                    hostname: info.get_hostname().to_string(),
                    addresses: info
                        .get_addresses()
                        .iter()
                        .filter_map(|addr| {
                            if let std::net::IpAddr::V4(ipv4) = addr {
                                Some(*ipv4)
                            } else {
                                None
                            }
                        })
                        .collect(),
                    port: info.get_port(),
                    properties: info
                        .get_properties()
                        .iter()
                        .map(|prop| (prop.key().to_string(), prop.val_str().to_string()))
                        .collect(),
                    discovered_at: Instant::now(),
                    last_seen: Instant::now(),
                };

                let mut services_lock = services.lock().unwrap();
                let event_to_send = if services_lock.contains_key(&service_info.instance_name) {
                    // Update existing service
                    services_lock.insert(service_info.instance_name.clone(), service_info.clone());
                    ServiceEvent::ServiceUpdated(service_info)
                } else {
                    // New service discovered
                    services_lock.insert(service_info.instance_name.clone(), service_info.clone());
                    ServiceEvent::ServiceDiscovered(service_info)
                };
                drop(services_lock);

                if let Err(e) = event_sender.send(event_to_send) {
                    debug!("Service event send failed: {}", e);
                }
            }
            MdnsServiceEvent::ServiceRemoved(type_name, instance_name) => {
                debug!("Service removed: {} of type {}", instance_name, type_name);

                let mut services_lock = services.lock().unwrap();
                if services_lock.remove(&instance_name).is_some() {
                    drop(services_lock);

                    if let Err(e) = event_sender.send(ServiceEvent::ServiceRemoved(instance_name)) {
                        debug!("Service removed event send failed: {}", e);
                    }
                }
            }
            MdnsServiceEvent::SearchStarted(_) => {
                debug!("mDNS search started");
            }
            MdnsServiceEvent::SearchStopped(_) => {
                debug!("mDNS search stopped");
            }
            MdnsServiceEvent::ServiceFound(type_name, instance_name) => {
                debug!("Service found: {} of type {}", instance_name, type_name);
                // ServiceFound is followed by ServiceResolved, so we don't need to act here
            }
        }
    }

}

impl Drop for ServiceTracker {
    fn drop(&mut self) {
        if self.is_running() {
            self.stop();
        }
    }
}

/// Convenience function to create a service tracker with default configuration
pub fn create_default_tracker() -> Result<ServiceTracker, Box<dyn std::error::Error>> {
    let config = ServiceTrackerConfig::default();
    ServiceTracker::new(config)
}

/// Convenience function to create a service tracker for a specific service type
pub fn create_tracker_for_service(
    service_type: &str,
) -> Result<ServiceTracker, Box<dyn std::error::Error>> {
    let config = ServiceTrackerConfig {
        service_type: service_type.to_string(),
        ..Default::default()
    };
    ServiceTracker::new(config)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_service_tracker_creation() {
        let config = ServiceTrackerConfig::default();
        let tracker = ServiceTracker::new(config);
        assert!(tracker.is_ok(), "Should be able to create service tracker");
    }

    #[test]
    fn test_service_info_connection_url() {
        let service = ServiceInfo {
            instance_name: "test-service".to_string(),
            hostname: "test-host".to_string(),
            addresses: vec![Ipv4Addr::new(192, 168, 1, 100)],
            port: 8080,
            properties: HashMap::new(),
            discovered_at: Instant::now(),
            last_seen: Instant::now(),
        };

        let url = service.connection_url();
        assert_eq!(url, Some("http://192.168.1.100:8080".to_string()));
    }

    #[test]
    fn test_default_config() {
        let config = ServiceTrackerConfig::default();
        assert_eq!(
            config.service_type,
            "_session-recorder-chunksink._tcp.local."
        );
        assert_eq!(config.max_services, 100);
    }

    #[test]
    fn test_convenience_functions() {
        let tracker1 = create_default_tracker();
        assert!(
            tracker1.is_ok(),
            "Default tracker should be created successfully"
        );

        let tracker2 = create_tracker_for_service("_custom-service._tcp.local.");
        assert!(
            tracker2.is_ok(),
            "Custom service tracker should be created successfully"
        );
    }
}
