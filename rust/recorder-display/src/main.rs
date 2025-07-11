use clap::{Arg, Command};
use std::error::Error;

mod mdns_service;
mod status_reader;

use status_reader::run_status_reader_server;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let matches = Command::new("recorder-display")
        .about("A gRPC server that receives recorder status updates")
        .arg(
            Arg::new("address")
                .short('a')
                .long("address")
                .value_name("ADDRESS")
                .help("Address to bind the server to")
                .default_value("0.0.0.0:50051"),
        )
        .get_matches();

    let address = matches.get_one::<String>("address").unwrap();

    println!("Starting recorder display server on: {}", address);

    if let Err(e) = run_status_reader_server(address).await {
        eprintln!("Error running status reader server: {}", e);
        std::process::exit(1);
    }

    Ok(())
}
