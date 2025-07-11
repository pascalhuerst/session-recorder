use mdns_sd::{ServiceDaemon, ServiceInfo};
use std::collections::HashMap;
use std::error::Error;
use std::net::Ipv4Addr;

pub struct MdnsService {
    _daemon: ServiceDaemon,
}

impl MdnsService {
    /// Create a new mDNS service and announce the chunk-sink service
    pub fn new(port: u16) -> Result<Self, Box<dyn Error>> {
        // Create the mDNS daemon
        let daemon = ServiceDaemon::new()?;

        // Create service info for chunk-sink
        let service_type = "_session-recorder-chunksink._tcp.local.";
        let instance_name = ""; //"session-recorder-chunksink";
        let host_name = gethostname::gethostname()
            .into_string()
            .unwrap_or_else(|_| "recorder-display".to_string());

        // Get the local IP address (we'll use all available addresses)
        let addresses = Ipv4Addr::new(127, 0, 0, 1);

        // Create properties (TXT records) - empty for now but can be extended
        let properties = HashMap::new();

        let service_info = ServiceInfo::new(
            service_type,
            instance_name,
            &host_name,
            addresses,
            port,
            Some(properties),
        )?;

        // Register the service
        daemon.register(service_info)?;

        println!("Announced mDNS service: {} on port {}", service_type, port);

        Ok(MdnsService { _daemon: daemon })
    }
}

impl Drop for MdnsService {
    fn drop(&mut self) {
        println!("mDNS service announcement stopped");
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mdns_service_creation() {
        // Test that we can create an mDNS service without panicking
        // Note: This will actually announce the service during tests
        let result = MdnsService::new(0); // Use port 0 to let the system choose
        assert!(result.is_ok(), "Should be able to create mDNS service");
    }
}
