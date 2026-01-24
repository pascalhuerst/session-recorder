use chunk_source::SessionRecorder;
use env_logger::Env;
use log::{Level, info};
use std::env;
use std::io::Write;
use std::process;
use std::sync::Arc;
use std::sync::atomic::Ordering;
use std::time::Duration;
use tokio::time::sleep;
use uuid::Uuid;

fn init_logger() {
    env_logger::Builder::from_env(Env::default().default_filter_or("info"))
        .format(|buf, record| {
            let level_colored = match record.level() {
                Level::Trace => "\x1b[35mTRACE\x1b[0m",
                Level::Debug => "\x1b[34mDEBUG\x1b[0m",
                Level::Info => "\x1b[32mINFO\x1b[0m",
                Level::Warn => "\x1b[33mWARN\x1b[0m",
                Level::Error => "\x1b[31mERROR\x1b[0m",
            };
            let timestamp = buf.timestamp_millis();
            let file = record.file().unwrap_or("unknown");
            let line = record.line().unwrap_or(0);
            writeln!(
                buf,
                "{timestamp} [{level}] {file}:{line} - {message}",
                level = level_colored,
                message = record.args()
            )
        })
        .init();
}

fn usage(program: &str) {
    eprintln!(
        "Usage: {program} --recorder-id <UUID> --recorder-name <NAME>",
        program = program
    );
}

fn parse_cli_args() -> Result<(Uuid, String), String> {
    let mut args = env::args();
    args.next();

    let mut recorder_id: Option<Uuid> = None;
    let mut recorder_name: Option<String> = None;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--recorder-id" => {
                let value = args
                    .next()
                    .ok_or_else(|| format!("Missing value for --recorder-id\n"))?;
                let parsed = Uuid::parse_str(&value)
                    .map_err(|_| format!("Invalid UUID passed to --recorder-id\n"))?;
                recorder_id = Some(parsed);
            }
            "--recorder-name" => {
                let value = args
                    .next()
                    .ok_or_else(|| format!("Missing value for --recorder-name\n"))?;
                recorder_name = Some(value);
            }
            other => {
                return Err(format!("Unknown argument: {other}\n"));
            }
        }
    }

    let id = recorder_id.ok_or_else(|| format!("--recorder-id <UUID> is required\n"))?;
    let name = recorder_name.ok_or_else(|| format!("--recorder-name <NAME> is required\n"))?;

    Ok((id, name))
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    init_logger();
    info!("Starting Session Recorder");

    let (recorder_id, recorder_name) = match parse_cli_args() {
        Ok(values) => values,
        Err(message) => {
            let program = env::args()
                .next()
                .unwrap_or_else(|| "chunk-source".to_string());
            eprint!("{message}");
            usage(&program);
            process::exit(1);
        }
    };

    let mut recorder = SessionRecorder::new(recorder_id, recorder_name.clone());
    recorder.start().await?;

    let shutdown_flag = recorder.shutdown_signal();
    ctrlc::set_handler({
        let shutdown = Arc::clone(&shutdown_flag);
        move || {
            shutdown.store(true, Ordering::Relaxed);
            info!("Received shutdown signal");
        }
    })?;

    info!("Session Recorder running. Press Ctrl+C to stop.");

    while recorder.is_running() {
        sleep(Duration::from_secs(1)).await;

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
