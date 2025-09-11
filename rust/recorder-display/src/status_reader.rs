use std::error::Error;
use tonic::{Request, Response, Status, transport::Server};

use crate::mdns_service::AvahiService;

// Generated protobuf code
pub mod chunksink {
    tonic::include_proto!("chunksink");
}

pub mod common {
    tonic::include_proto!("common");
}

use chunksink::chunk_sink_server::{ChunkSink, ChunkSinkServer};
use chunksink::{Chunks, Command, GetCommandRequest};
use common::{RecorderStatus, Respone, SignalStatus};

#[derive(Debug, Default)]
pub struct StatusReaderService {}

#[tonic::async_trait]
impl ChunkSink for StatusReaderService {
    async fn set_recorder_status(
        &self,
        request: Request<RecorderStatus>,
    ) -> Result<Response<Respone>, Status> {
        let status = request.into_inner();

        // Log the received status
        self.log_recorder_status(&status);

        // Return success response
        let response = Respone {
            success: true,
            error_message: String::new(),
        };

        Ok(Response::new(response))
    }

    async fn set_chunks(&self, request: Request<Chunks>) -> Result<Response<Respone>, Status> {
        let chunks = request.into_inner();

        println!(
            "Received {} chunks from recorder: {}",
            chunks.chunk_count, chunks.recorder_id
        );

        // Return success response
        let response = Respone {
            success: true,
            error_message: String::new(),
        };

        Ok(Response::new(response))
    }

    type GetCommandsStream = tokio_stream::wrappers::ReceiverStream<Result<Command, Status>>;

    async fn get_commands(
        &self,
        request: Request<GetCommandRequest>,
    ) -> Result<Response<Self::GetCommandsStream>, Status> {
        let req = request.into_inner();
        println!("Recorder {} requesting commands", req.recorder_id);

        // Create a channel for streaming commands
        let (tx, rx) = tokio::sync::mpsc::channel(4);

        // For now, we don't send any commands - just keep the stream alive
        // In a real implementation, you could spawn a task to send commands when needed
        tokio::spawn(async move {
            // Keep the stream alive but don't send any commands for now
            // You could implement command sending logic here
            drop(tx); // This will close the stream
        });

        let output_stream = tokio_stream::wrappers::ReceiverStream::new(rx);
        Ok(Response::new(output_stream))
    }
}

impl StatusReaderService {
    /// Log recorder status information
    fn log_recorder_status(&self, status: &RecorderStatus) {
        println!("=== Recorder Status Update ===");
        println!("Recorder ID: {}", status.recorder_id);
        println!("Recorder Name: {}", status.recorder_name);
        println!(
            "Signal Status: {}",
            self.signal_status_to_string(status.signal_status)
        );
        println!("RMS Percent: {:.2}%", status.rms_percent);
        println!("Clipping: {}", if status.clipping { "Yes" } else { "No" });
        println!("==============================");
    }

    /// Convert SignalStatus enum to human-readable string
    fn signal_status_to_string(&self, status: i32) -> &'static str {
        match SignalStatus::try_from(status) {
            Ok(SignalStatus::Unknown) => "Unknown",
            Ok(SignalStatus::NoSignal) => "No Signal",
            Ok(SignalStatus::Signal) => "Signal",
            Err(_) => "Invalid Status",
        }
    }
}

/// Convenience function to create and start a status reader server
pub async fn run_status_reader_server(addr: &str) -> Result<(), Box<dyn Error>> {
    let addr: std::net::SocketAddr = addr.parse()?;

    // Extract port from address for mDNS announcement
    let port = addr.port();

    // Start mDNS service announcement
    let _mdns_service = AvahiService::new(port)?;

    println!("Starting ChunkSink gRPC server on {}", addr);

    Server::builder()
        .add_service(ChunkSinkServer::new(StatusReaderService::default()))
        .serve(addr)
        .await?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_signal_status_conversion() {
        let service = StatusReaderService::default();

        assert_eq!(
            service.signal_status_to_string(SignalStatus::Unknown.into()),
            "Unknown"
        );
        assert_eq!(
            service.signal_status_to_string(SignalStatus::NoSignal.into()),
            "No Signal"
        );
        assert_eq!(
            service.signal_status_to_string(SignalStatus::Signal.into()),
            "Signal"
        );
    }

    #[tokio::test]
    async fn test_set_recorder_status() {
        let service = StatusReaderService::default();

        let status = RecorderStatus {
            recorder_id: "test-recorder".to_string(),
            recorder_name: "Test Recorder".to_string(),
            signal_status: SignalStatus::Signal.into(),
            rms_percent: 50.0,
            clipping: false,
        };

        let request = Request::new(status);
        let response = service.set_recorder_status(request).await;

        assert!(response.is_ok());
        let response = response.unwrap().into_inner();
        assert!(response.success);
        assert!(response.error_message.is_empty());
    }
}
