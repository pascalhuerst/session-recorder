//! In-process mDNS discovery backend (pure-Rust `mdns-sd`).
//!
//! Note: this runs its own mDNS responder/browser in-process. On a host that
//! already runs `avahi-daemon`, prefer the [`super::avahi`] backend to avoid two
//! mDNS stacks competing for UDP port 5353.

use log::{debug, info, warn};
use mdns_sd::{ServiceDaemon, ServiceEvent as MdnsServiceEvent};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};
use tokio::sync::mpsc;

use super::{DiscoveryConfig, ServiceDiscovery, ServiceEvent, ServiceInfo};

/// mDNS service discovery backend.
pub struct MdnsDiscovery {
    config: DiscoveryConfig,
    daemon: Option<ServiceDaemon>,
    services: Arc<Mutex<HashMap<String, ServiceInfo>>>,
    worker_handle: Option<thread::JoinHandle<()>>,
    is_running: Arc<Mutex<bool>>,
}

impl MdnsDiscovery {
    pub fn new(config: DiscoveryConfig) -> anyhow::Result<Self> {
        let daemon = ServiceDaemon::new()?;
        Ok(Self {
            config,
            daemon: Some(daemon),
            services: Arc::new(Mutex::new(HashMap::new())),
            worker_handle: None,
            is_running: Arc::new(Mutex::new(false)),
        })
    }

    /// Handle mDNS events and convert them to [`ServiceEvent`]s.
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
                    services_lock.insert(service_info.instance_name.clone(), service_info.clone());
                    ServiceEvent::ServiceUpdated(service_info)
                } else {
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
            MdnsServiceEvent::SearchStarted(_) => debug!("mDNS search started"),
            MdnsServiceEvent::SearchStopped(_) => debug!("mDNS search stopped"),
            MdnsServiceEvent::ServiceFound(type_name, instance_name) => {
                debug!("Service found: {} of type {}", instance_name, type_name);
                // ServiceFound is followed by ServiceResolved; nothing to do here.
            }
        }
    }
}

#[async_trait::async_trait]
impl ServiceDiscovery for MdnsDiscovery {
    async fn start(&mut self) -> anyhow::Result<mpsc::UnboundedReceiver<ServiceEvent>> {
        if *self.is_running.lock().unwrap() {
            anyhow::bail!("mDNS discovery is already running");
        }

        let (event_sender, event_receiver) = mpsc::unbounded_channel();

        let daemon = self
            .daemon
            .take()
            .ok_or_else(|| anyhow::anyhow!("mDNS daemon not available"))?;
        let service_type = self.config.service_type.clone();
        let services = Arc::clone(&self.services);
        let is_running = Arc::clone(&self.is_running);

        *self.is_running.lock().unwrap() = true;

        let worker_handle = thread::spawn(move || {
            info!("Starting mDNS service discovery for: {}", service_type);

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

            // Small delay to allow the service to initialize.
            thread::sleep(Duration::from_millis(50));

            // recv_timeout returns Err on both Timeout and Disconnected —
            // distinguish via the receiver's own state.
            while *is_running.lock().unwrap() {
                match receiver.recv_timeout(Duration::from_secs(1)) {
                    Ok(event) => Self::handle_mdns_event(event, &services, &event_sender),
                    Err(_) => {
                        if receiver.is_disconnected() {
                            warn!("mDNS browse channel disconnected, exiting worker");
                            break;
                        }
                    }
                }
            }

            info!("mDNS service discovery worker stopped");
        });

        self.worker_handle = Some(worker_handle);
        info!("mDNS discovery started successfully");
        Ok(event_receiver)
    }

    async fn stop(&mut self) {
        debug!("Stopping mDNS discovery...");
        *self.is_running.lock().unwrap() = false;
        thread::sleep(Duration::from_millis(50));
        if let Some(handle) = self.worker_handle.take() {
            let _ = handle.join();
        }
        debug!("mDNS discovery stopped");
    }
}

impl Drop for MdnsDiscovery {
    fn drop(&mut self) {
        if *self.is_running.lock().unwrap() {
            *self.is_running.lock().unwrap() = false;
            thread::sleep(Duration::from_millis(50));
            if let Some(handle) = self.worker_handle.take() {
                let _ = handle.join();
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn create_mdns_discovery() {
        let tracker = MdnsDiscovery::new(DiscoveryConfig::default());
        assert!(tracker.is_ok(), "Should be able to create mDNS discovery");
    }
}
