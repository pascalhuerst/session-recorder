//! Simple mDNS service discovery demonstration
//!
//! This example shows the basic mDNS service discovery behavior:
//! - Starts discovery and waits for chunk-sink servers
//! - Shows that the "closed channel" error is normal behavior
//! - Demonstrates continuous monitoring until servers appear
//!
//! Run this to see how the discovery works in practice.

use chunk_source::mdns::service_tracker::{ServiceEvent, ServiceTracker, ServiceTrackerConfig};
use log::info;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logger
    env_logger::init();

    println!("🚀 Simple mDNS Service Discovery Demo");
    println!("=====================================");
    println!();
    println!("This demo shows how mDNS service discovery works:");
    println!("• Starts looking for '_session-recorder-chunksink._tcp.local.' services");
    println!("• You may see an ERROR message about 'closed channel' - this is NORMAL");
    println!("• The discovery continues running in the background");
    println!("• When a chunk-sink server appears, it will be discovered automatically");
    println!("• Press Ctrl+C to stop");
    println!();

    // Create service tracker configuration
    let tracker_config = ServiceTrackerConfig {
        service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
        service_timeout: Duration::from_secs(60),
        cleanup_interval: Duration::from_secs(10),
        max_services: 10,
    };

    // Create and start the service tracker
    let mut tracker = ServiceTracker::new(tracker_config)?;
    let event_receiver = tracker.start()?;

    // Shutdown signal
    let shutdown_signal = Arc::new(AtomicBool::new(false));
    let shutdown_clone = shutdown_signal.clone();

    // Set up graceful shutdown
    ctrlc::set_handler(move || {
        println!("\n🛑 Received shutdown signal, stopping discovery...");
        shutdown_clone.store(true, Ordering::Relaxed);
    })
    .expect("Error setting Ctrl+C handler");

    info!("🔍 mDNS service discovery started");
    info!("Waiting for chunk-sink servers to appear on the network...");

    let mut last_status_time = std::time::Instant::now();
    let mut discovery_count = 0;

    // Main discovery loop
    while !shutdown_signal.load(Ordering::Relaxed) {
        // Check for service events
        match event_receiver.recv_timeout(Duration::from_millis(1000)) {
            Ok(event) => match event {
                ServiceEvent::ServiceDiscovered(service_info) => {
                    discovery_count += 1;
                    println!(
                        "\n🎉 DISCOVERED #{}: {}",
                        discovery_count, service_info.instance_name
                    );
                    println!("   📍 Host: {}", service_info.hostname);
                    println!("   🌐 Port: {}", service_info.port);
                    println!("   📡 Addresses: {:?}", service_info.addresses);

                    if let Some(url) = service_info.connection_url() {
                        println!("   🔗 Connection URL: {}", url);
                        println!("   ✅ Ready to connect with gRPC client!");
                    }

                    if !service_info.properties.is_empty() {
                        println!("   🏷️  Properties: {:?}", service_info.properties);
                    }
                }
                ServiceEvent::ServiceUpdated(service_info) => {
                    println!("\n🔄 UPDATED: {}", service_info.instance_name);
                    println!("   📍 Host: {}", service_info.hostname);
                    println!("   🌐 Port: {}", service_info.port);
                }
                ServiceEvent::ServiceRemoved(instance_name) => {
                    println!("\n🗑️  REMOVED: {}", instance_name);
                    println!("   📤 Server went offline");
                }
            },
            Err(_) => {
                // Timeout - no events received (this is normal)

                // Show periodic status updates
                if last_status_time.elapsed() >= Duration::from_secs(30) {
                    let services = tracker.get_services();
                    if services.is_empty() {
                        info!("⏳ Still waiting for chunk-sink servers... (discovery continues)");
                    } else {
                        info!("📊 Currently tracking {} service(s)", services.len());
                    }
                    last_status_time = std::time::Instant::now();
                }
            }
        }
    }

    println!("\n🔄 Stopping service discovery...");
    tracker.stop();

    let final_services = tracker.get_services();
    println!("📋 Final summary:");
    println!(
        "   • Discovered {} service(s) during this session",
        discovery_count
    );
    println!("   • {} service(s) currently tracked", final_services.len());

    if !final_services.is_empty() {
        println!("   • Active services:");
        for service in final_services {
            println!(
                "     - {} ({})",
                service.instance_name,
                service.connection_url().unwrap_or("unknown".to_string())
            );
        }
    }

    println!("\n✅ Discovery demo completed!");
    println!("\nKey Points:");
    println!("• The mDNS discovery works continuously in the background");
    println!("• ERROR messages about 'closed channel' are normal when no servers are present");
    println!(
        "• Servers can appear at any time (even hours later) and will be discovered automatically"
    );
    println!(
        "• In a real application, gRPC clients would automatically connect to discovered servers"
    );

    Ok(())
}
