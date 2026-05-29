//! Service discovery for chunk-sink servers.
//!
//! Two interchangeable backends discover chunk-sink servers on the local
//! network and report the same [`ServiceEvent`]s:
//!
//! * [`mdns`] — in-process mDNS via the pure-Rust `mdns-sd` crate.
//! * [`avahi`] — talks to the system `avahi-daemon` over D-Bus (zbus).
//!
//! The backend is chosen at startup via [`DiscoveryMethod`]; everything
//! downstream consumes the shared [`ServiceEvent`] stream and is unaware of
//! which backend produced it.

pub mod avahi;
pub mod mdns;

use std::collections::HashMap;
use std::net::Ipv4Addr;
use std::time::Instant;

use tokio::sync::mpsc;

/// Information about a discovered chunk-sink server.
#[derive(Debug, Clone)]
pub struct ServiceInfo {
    /// Service instance name (used as the stable key for a service).
    pub instance_name: String,
    /// Hostname of the server.
    pub hostname: String,
    /// IPv4 addresses of the server.
    pub addresses: Vec<Ipv4Addr>,
    /// Port number the server is listening on.
    pub port: u16,
    /// TXT record properties.
    pub properties: HashMap<String, String>,
    /// Time when the service was first discovered.
    pub discovered_at: Instant,
    /// Time when the service was last seen.
    pub last_seen: Instant,
}

impl ServiceInfo {
    /// Get the primary address for connecting to this service.
    pub fn primary_address(&self) -> Option<Ipv4Addr> {
        self.addresses.first().copied()
    }

    /// Get the full connection URL for this service.
    pub fn connection_url(&self) -> Option<String> {
        self.primary_address()
            .map(|addr| format!("http://{}:{}", addr, self.port))
    }
}

/// Events reported by a discovery backend.
#[derive(Debug, Clone)]
pub enum ServiceEvent {
    ServiceDiscovered(ServiceInfo),
    ServiceUpdated(ServiceInfo),
    ServiceRemoved(String), // instance_name
}

/// Configuration shared by all discovery backends.
#[derive(Debug, Clone)]
pub struct DiscoveryConfig {
    /// Service type to search for, in mDNS form (e.g.
    /// `_session-recorder-chunksink._tcp.local.`). The avahi backend strips the
    /// trailing domain for the Avahi API.
    pub service_type: String,
    /// Maximum number of services to track.
    pub max_services: usize,
}

impl Default for DiscoveryConfig {
    fn default() -> Self {
        Self {
            service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
            max_services: 100,
        }
    }
}

/// Which discovery backend to use.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, clap::ValueEnum)]
pub enum DiscoveryMethod {
    /// Talk to the system avahi-daemon over D-Bus (default).
    #[default]
    Avahi,
    /// In-process mDNS via the `mdns-sd` crate.
    Mdns,
}

/// A pluggable service-discovery backend. Both backends report the same
/// [`ServiceEvent`] stream so callers don't care which one is active.
#[async_trait::async_trait]
pub trait ServiceDiscovery: Send {
    /// Start discovering services, returning a receiver of discovery events.
    async fn start(&mut self) -> anyhow::Result<mpsc::UnboundedReceiver<ServiceEvent>>;

    /// Stop discovery and release resources.
    async fn stop(&mut self);
}

/// Build the configured discovery backend.
pub fn create(
    method: DiscoveryMethod,
    config: DiscoveryConfig,
) -> anyhow::Result<Box<dyn ServiceDiscovery>> {
    match method {
        DiscoveryMethod::Avahi => Ok(Box::new(avahi::AvahiDiscovery::new(config))),
        DiscoveryMethod::Mdns => Ok(Box::new(mdns::MdnsDiscovery::new(config)?)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn connection_url_formats_primary_address() {
        let service = ServiceInfo {
            instance_name: "test-service".to_string(),
            hostname: "test-host".to_string(),
            addresses: vec![Ipv4Addr::new(192, 168, 1, 100)],
            port: 8080,
            properties: HashMap::new(),
            discovered_at: Instant::now(),
            last_seen: Instant::now(),
        };
        assert_eq!(
            service.connection_url(),
            Some("http://192.168.1.100:8080".to_string())
        );
    }

    #[test]
    fn connection_url_none_without_address() {
        let service = ServiceInfo {
            instance_name: "x".to_string(),
            hostname: "h".to_string(),
            addresses: vec![],
            port: 8080,
            properties: HashMap::new(),
            discovered_at: Instant::now(),
            last_seen: Instant::now(),
        };
        assert_eq!(service.connection_url(), None);
    }

    #[test]
    fn default_method_is_avahi() {
        assert_eq!(DiscoveryMethod::default(), DiscoveryMethod::Avahi);
    }

    #[test]
    fn default_config_service_type() {
        let config = DiscoveryConfig::default();
        assert_eq!(
            config.service_type,
            "_session-recorder-chunksink._tcp.local."
        );
    }
}
