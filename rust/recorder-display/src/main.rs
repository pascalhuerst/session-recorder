use clap::{Arg, Command};
use std::collections::HashMap;
use std::error::Error;
use std::sync::Arc;
use std::time::{Duration, Instant};

mod mdns_service;
mod status_reader;
mod ui;

use status_reader::{
    common::{RecorderStatus, SignalStatus},
    run_status_reader_server,
};
use ui::{RecorderStatusMap, RmsLevel, run_gui};

fn main() -> Result<(), Box<dyn Error>> {
    let matches = Command::new("recorder-display")
        .about("A GUI application that displays recorder status updates")
        .arg(
            Arg::new("address")
                .short('a')
                .long("address")
                .value_name("ADDRESS")
                .help("Address to bind the gRPC server to")
                .default_value("0.0.0.0:50051"),
        )
        .arg(
            Arg::new("test-mode")
                .short('t')
                .long("test-mode")
                .action(clap::ArgAction::SetTrue)
                .help("Run in test mode with simulated recorder data"),
        )
        .arg(
            Arg::new("disable-mdns")
                .short('d')
                .long("disable-mdns")
                .action(clap::ArgAction::SetTrue)
                .help("Disable mDNS-SD service announcement"),
        )
        .get_matches();

    let address = matches.get_one::<String>("address").unwrap().to_string();
    let test_mode = matches.get_flag("test-mode");
    let disable_mdns = matches.get_flag("disable-mdns");

    // Create shared state for recorder statuses
    let recorder_statuses: RecorderStatusMap = Arc::new(std::sync::Mutex::new(HashMap::new()));
    let recorder_statuses_clone = recorder_statuses.clone();

    if test_mode {
        println!("Starting in TEST MODE with simulated data");

        // Start test data generator in background thread
        let test_recorder_statuses = recorder_statuses.clone();
        let _test_handle = std::thread::spawn(move || {
            generate_test_data(test_recorder_statuses);
        });
    } else {
        println!("Starting recorder display server on: {}", address);

        // Start gRPC server in background thread
        let _server_handle = std::thread::spawn(move || {
            let rt = tokio::runtime::Runtime::new().unwrap();
            rt.block_on(async {
                if let Err(e) =
                    run_status_reader_server(&address, recorder_statuses_clone, disable_mdns).await
                {
                    eprintln!("Error running status reader server: {}", e);
                }
            })
        });
    }

    // Start GUI on main thread (this blocks until GUI is closed)
    let gui_result = run_gui(recorder_statuses);

    // Signal server thread to shutdown (we can't gracefully abort it, but process exit will handle it)
    match gui_result {
        Ok(()) => Ok(()),
        Err(e) => {
            eprintln!("GUI error: {}", e);
            std::process::exit(1);
        }
    }
}

/// Generate simulated test data for demonstration purposes
fn generate_test_data(recorder_statuses: RecorderStatusMap) {
    let test_recorders = vec![
        ("studio-1", "Studio 1 Main", 0.0),
        ("studio-2", "Studio 2 Backup", 2.5),
        ("studio-3", "Control Room", 5.0),
        ("mobile-1", "Mobile Unit Alpha", 7.5),
    ];

    let mut time_offset = 0.0f64;

    loop {
        time_offset += 0.1;

        for (_i, (id, name, phase_offset)) in test_recorders.iter().enumerate() {
            let t = time_offset + phase_offset;

            // Generate varying signal patterns
            let signal_present = (t * 0.3).sin() > -0.5;
            let base_rms = if signal_present {
                ((t * 0.8).sin() * 0.5 + 0.5) * 80.0 + 10.0
            } else {
                ((t * 2.0).sin() * 0.5 + 0.5) * 5.0
            };

            // Add some randomness
            let rms_percent = (base_rms + (t * 13.7).sin() * 5.0).max(0.0).min(100.0);

            // Clipping occurs when RMS is very high
            let clipping = rms_percent > 85.0 && (t * 3.0).sin() > 0.7;

            // Occasional signal dropouts
            let signal_status = if signal_present && (t * 0.1).sin() > -0.9 {
                SignalStatus::Signal as i32
            } else if (t * 0.05).sin() > -0.8 {
                SignalStatus::NoSignal as i32
            } else {
                SignalStatus::Unknown as i32
            };

            let status = RecorderStatus {
                recorder_id: id.to_string(),
                recorder_name: name.to_string(),
                signal_status,
                rms_percent,
                clipping,
            };

            // Update the shared state
            if let Ok(mut statuses) = recorder_statuses.lock() {
                statuses.insert(
                    id.to_string(),
                    (status, Instant::now(), None, RmsLevel::new()),
                );
            }
        }

        // Update every 100ms for smooth animation
        std::thread::sleep(Duration::from_millis(100));
    }
}
