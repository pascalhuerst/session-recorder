use std::error::Error;
use std::sync::{Arc, Mutex};
use zeroconf::prelude::*;
use zeroconf::{MdnsService, ServiceRegistration, ServiceType, TxtRecord};

pub struct AvahiService {
    _service: MdnsService,
}

#[derive(Default, Debug)]
struct ServiceContext {
    _port: u16,
}

impl AvahiService {
    /// Create a new mDNS service and announce the chunk-sink service using Avahi
    pub fn new(port: u16) -> Result<Self, Box<dyn Error>> {
        let service_type = ServiceType::new("session-recorder-chunksink", "tcp")?;
        let mut service = MdnsService::new(service_type, port);
        let mut txt_record = TxtRecord::new();
        let context = Arc::new(Mutex::new(ServiceContext { _port: port }));

        // Add some TXT record properties
        txt_record.insert("version", "1.0")?;
        txt_record.insert("service", "chunk-sink")?;

        // Get hostname for the service name
        let hostname = gethostname::gethostname()
            .into_string()
            .unwrap_or_else(|_| "recorder-display".to_string());

        service.set_name(&hostname);
        service.set_registered_callback(Box::new(on_service_registered));
        service.set_context(Box::new(context));
        service.set_txt_record(txt_record);

        // Register the service - this will use Avahi on Linux
        let _event_loop = service.register()?;

        println!(
            "Announced mDNS service: _session-recorder-chunksink._tcp on port {}",
            port
        );

        Ok(AvahiService { _service: service })
    }
}

fn on_service_registered(
    result: zeroconf::Result<ServiceRegistration>,
    _context: Option<Arc<dyn std::any::Any>>,
) {
    match result {
        Ok(service) => {
            println!("mDNS service registered successfully: {}", service.name());
        }
        Err(e) => {
            eprintln!("Failed to register mDNS service: {}", e);
        }
    }
}

impl Drop for AvahiService {
    fn drop(&mut self) {
        println!("mDNS service announcement stopped");
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_avahi_service_creation() {
        // Test that we can create an Avahi service without panicking
        // Note: This will actually announce the service during tests
        let result = AvahiService::new(0); // Use port 0 to let the system choose
        // We don't assert success here because Avahi might not be available in test environment
        match result {
            Ok(_) => println!("Avahi service created successfully"),
            Err(e) => println!(
                "Could not create Avahi service (expected in test env): {}",
                e
            ),
        }
    }
}
