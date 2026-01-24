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
        "Usage: {program} --recorder-id <UUID> --recorder-name <NAME> [--detector-threshold <PERCENT>] [--detector-succession <COUNT>]",
        program = program
    );
}

struct CliArgs {
    recorder_id: Uuid,
    recorder_name: String,
    detector_threshold: f64,
    detector_succession: u32,
}

fn parse_cli_args() -> Result<CliArgs, String> {
    let mut args = env::args();
    args.next();

    let mut recorder_id: Option<Uuid> = None;
    let mut recorder_name: Option<String> = None;
    let mut detector_threshold: f64 = 10.0;
    let mut detector_succession: u32 = 5;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--recorder-id" => {
                let value = args
                    .next()
                    .ok_or_else(|| String::from("Missing value for --recorder-id\n"))?;
                let parsed = Uuid::parse_str(&value)
                    .map_err(|_| String::from("Invalid UUID passed to --recorder-id\n"))?;
                recorder_id = Some(parsed);
            }
            "--recorder-name" => {
                let value = args
                    .next()
                    .ok_or_else(|| String::from("Missing value for --recorder-name\n"))?;
                recorder_name = Some(value);
            }
            "--detector-threshold" => {
                let value = args
                    .next()
                    .ok_or_else(|| String::from("Missing value for --detector-threshold\n"))?;
                detector_threshold = value
                    .parse::<f64>()
                    .map_err(|_| String::from("Invalid value for --detector-threshold\n"))?;
            }
            "--detector-succession" => {
                let value = args
                    .next()
                    .ok_or_else(|| String::from("Missing value for --detector-succession\n"))?;
                detector_succession = value
                    .parse::<u32>()
                    .map_err(|_| String::from("Invalid value for --detector-succession\n"))?;
            }
            other => {
                return Err(format!("Unknown argument: {other}\n"));
            }
        }
    }

    let recorder_id =
        recorder_id.ok_or_else(|| String::from("--recorder-id <UUID> is required\n"))?;
    let recorder_name =
        recorder_name.ok_or_else(|| String::from("--recorder-name <NAME> is required\n"))?;

    Ok(CliArgs {
        recorder_id,
        recorder_name,
        detector_threshold,
        detector_succession,
    })
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    init_logger();
    info!("Starting Session Recorder");

    let args = match parse_cli_args() {
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

    let CliArgs {
        recorder_id,
        recorder_name,
        detector_threshold,
        detector_succession,
    } = args;

    let mut recorder = SessionRecorder::new(
        recorder_id,
        recorder_name,
        detector_threshold,
        detector_succession,
    );
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
