//! ChunkSink gRPC client implementation
//!
//! This module provides a client interface for the ChunkSink gRPC service,
//! allowing communication with the session recorder backend.

use log::{error, info, warn};
use prost_types::Timestamp;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tonic::transport::Channel;
use tonic::{Request, Status};

// Include the generated protobuf code
pub mod chunksink {
    tonic::include_proto!("chunksink");
}

pub mod common {
    tonic::include_proto!("common");
}

use chunksink::chunk_sink_client::ChunkSinkClient;
use chunksink::Chunks;
use common::{RecorderStatus, SignalStatus};

/// Error types for ChunkSink client operations
#[derive(Debug)]
pub enum ChunkSinkError {
    ConnectionError(tonic::transport::Error),
    GrpcError(Status),
    InvalidData(String),
    ClientNotConnected,
    InvalidUri(String),
}

impl std::fmt::Display for ChunkSinkError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ChunkSinkError::ConnectionError(e) => write!(f, "Connection error: {}", e),
            ChunkSinkError::GrpcError(e) => write!(f, "gRPC error: {}", e),
            ChunkSinkError::InvalidData(msg) => write!(f, "Invalid data: {}", msg),
            ChunkSinkError::ClientNotConnected => write!(f, "Client not connected"),
            ChunkSinkError::InvalidUri(msg) => write!(f, "Invalid URI: {}", msg),
        }
    }
}

impl std::error::Error for ChunkSinkError {}

impl From<tonic::transport::Error> for ChunkSinkError {
    fn from(error: tonic::transport::Error) -> Self {
        ChunkSinkError::ConnectionError(error)
    }
}

impl From<Status> for ChunkSinkError {
    fn from(error: Status) -> Self {
        ChunkSinkError::GrpcError(error)
    }
}

impl From<tonic::codegen::http::uri::InvalidUri> for ChunkSinkError {
    fn from(error: tonic::codegen::http::uri::InvalidUri) -> Self {
        ChunkSinkError::InvalidUri(error.to_string())
    }
}

/// Configuration for the ChunkSink client
#[derive(Debug, Clone)]
pub struct ChunkSinkConfig {
    pub server_address: String,
    pub recorder_id: String,
    pub recorder_name: String,
    pub connect_timeout: Duration,
    pub request_timeout: Duration,
}

impl Default for ChunkSinkConfig {
    fn default() -> Self {
        Self {
            server_address: "http://localhost:50051".to_string(),
            recorder_id: "default-recorder".to_string(),
            recorder_name: "Default Recorder".to_string(),
            connect_timeout: Duration::from_secs(10),
            request_timeout: Duration::from_secs(5),
        }
    }
}

/// Audio chunk data for transmission
#[derive(Debug, Clone)]
pub struct AudioChunk {
    pub session_id: String,
    pub chunk_count: u32,
    pub data: Vec<u32>,
    pub timestamp: SystemTime,
}

/// Recorder status information
#[derive(Debug, Clone)]
pub struct RecorderStatusInfo {
    pub signal_status: SignalStatus,
    pub rms_percent: f64,
    pub clipping: bool,
}

/// ChunkSink gRPC client service
pub struct ChunkSinkClientService {
    config: ChunkSinkConfig,
    client: Option<ChunkSinkClient<Channel>>,
}

impl ChunkSinkClientService {
    pub fn new(config: ChunkSinkConfig) -> Self {
        Self {
            config,
            client: None,
        }
    }

    pub async fn connect(&mut self) -> Result<(), ChunkSinkError> {
        info!(
            "Connecting to ChunkSink server at {}",
            self.config.server_address
        );

        let channel = Channel::from_shared(self.config.server_address.clone())?
            .connect_timeout(self.config.connect_timeout)
            .timeout(self.config.request_timeout)
            .connect()
            .await?;

        self.client = Some(ChunkSinkClient::new(channel));
        info!("Successfully connected to ChunkSink server");

        Ok(())
    }

    pub async fn disconnect(&mut self) {
        self.client = None;
        info!("Disconnected from ChunkSink server");
    }

    pub fn is_connected(&self) -> bool {
        self.client.is_some()
    }

    pub async fn set_recorder_status(
        &mut self,
        status: RecorderStatusInfo,
    ) -> Result<bool, ChunkSinkError> {
        let client = self
            .client
            .as_mut()
            .ok_or(ChunkSinkError::ClientNotConnected)?;

        let request = Request::new(RecorderStatus {
            recorder_id: self.config.recorder_id.clone(),
            recorder_name: self.config.recorder_name.clone(),
            signal_status: status.signal_status.into(),
            rms_percent: status.rms_percent,
            clipping: status.clipping,
        });

        match client.set_recorder_status(request).await {
            Ok(response) => {
                let resp = response.into_inner();
                if !resp.success {
                    warn!("Server reported error: {}", resp.error_message);
                }
                Ok(resp.success)
            }
            Err(e) => {
                error!("Failed to set recorder status: {}", e);
                Err(ChunkSinkError::GrpcError(e))
            }
        }
    }

    pub async fn set_chunks(&mut self, chunk: AudioChunk) -> Result<bool, ChunkSinkError> {
        let client = self
            .client
            .as_mut()
            .ok_or(ChunkSinkError::ClientNotConnected)?;

        let timestamp = chunk
            .timestamp
            .duration_since(UNIX_EPOCH)
            .map_err(|e| ChunkSinkError::InvalidData(format!("Invalid timestamp: {}", e)))?;

        let request = Request::new(Chunks {
            recorder_id: self.config.recorder_id.clone(),
            session_id: chunk.session_id,
            chunk_count: chunk.chunk_count,
            time_created: Some(Timestamp {
                seconds: timestamp.as_secs() as i64,
                nanos: timestamp.subsec_nanos() as i32,
            }),
            data: chunk.data,
        });

        match client.set_chunks(request).await {
            Ok(response) => {
                let resp = response.into_inner();
                if !resp.success {
                    warn!("Server reported error: {}", resp.error_message);
                }
                Ok(resp.success)
            }
            Err(e) => {
                error!("Failed to set chunks: {}", e);
                Err(ChunkSinkError::GrpcError(e))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_default() {
        let config = ChunkSinkConfig::default();
        assert_eq!(config.server_address, "http://localhost:50051");
        assert_eq!(config.recorder_id, "default-recorder");
        assert_eq!(config.recorder_name, "Default Recorder");
    }

    #[test]
    fn test_chunk_sink_client_creation() {
        let config = ChunkSinkConfig::default();
        let client = ChunkSinkClientService::new(config);
        assert!(!client.is_connected());
    }

    #[tokio::test]
    async fn test_client_not_connected_error() {
        let config = ChunkSinkConfig::default();
        let mut client = ChunkSinkClientService::new(config);

        let status = RecorderStatusInfo {
            signal_status: SignalStatus::Signal,
            rms_percent: 0.5,
            clipping: false,
        };

        let result = client.set_recorder_status(status).await;
        assert!(matches!(result, Err(ChunkSinkError::ClientNotConnected)));
    }
}
