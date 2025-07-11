//! ChunkSink gRPC client implementation
//!
//! This module provides a client interface for the ChunkSink gRPC service,
//! allowing communication with the session recorder backend.

use crate::audio::channels::{AudioChannels, ParameterChannels, Parameters};
use futures_util::StreamExt;
use log::{error, info, warn};
use prost_types::Timestamp;
use ringbuf::traits::{Consumer, Producer};
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::time::sleep;
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
use chunksink::{Chunks, GetCommandRequest};
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
    pub retry_interval: Duration,
    pub max_retries: u32,
    pub audio_buffer_size: usize,
    pub parameter_buffer_size: usize,
}

impl Default for ChunkSinkConfig {
    fn default() -> Self {
        Self {
            server_address: "http://localhost:50051".to_string(),
            recorder_id: "default-recorder".to_string(),
            recorder_name: "Default Recorder".to_string(),
            connect_timeout: Duration::from_secs(10),
            request_timeout: Duration::from_secs(5),
            retry_interval: Duration::from_secs(5),
            max_retries: 3,
            audio_buffer_size: 8192,
            parameter_buffer_size: 64,
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

/// Commands that can be received from the server
#[derive(Debug, Clone)]
pub enum ServerCommand {
    CutSession,
    Reboot,
}

/// ChunkSink gRPC client service
pub struct ChunkSinkClientService {
    config: ChunkSinkConfig,
    client: Option<ChunkSinkClient<Channel>>,
    audio_channels: Option<AudioChannels>,
    parameter_channels: Option<ParameterChannels>,
}

impl ChunkSinkClientService {
    /// Create a new ChunkSink client service
    pub fn new(config: ChunkSinkConfig) -> Self {
        Self {
            config,
            client: None,
            audio_channels: None,
            parameter_channels: None,
        }
    }

    /// Initialize channels for audio and parameter communication
    pub fn initialize_channels(&mut self) {
        use crate::audio::channels::{AudioChannelPair, ParameterChannelPair};

        // Create audio channels for chunk data
        let audio_pair = AudioChannelPair::new(self.config.audio_buffer_size);
        self.audio_channels = Some(audio_pair.main_channels);

        // Create parameter channels for commands
        let param_pair = ParameterChannelPair::new(self.config.parameter_buffer_size);
        self.parameter_channels = Some(param_pair.main_channels);
    }

    /// Connect to the ChunkSink server
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

    /// Disconnect from the server
    pub async fn disconnect(&mut self) {
        self.client = None;
        info!("Disconnected from ChunkSink server");
    }

    /// Check if the client is connected
    pub fn is_connected(&self) -> bool {
        self.client.is_some()
    }

    /// Send recorder status to the server
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

    /// Send audio chunks to the server using audio channels
    pub async fn process_audio_chunks(&mut self) -> Result<(), ChunkSinkError> {
        let mut buffer = vec![0.0f32; 1024]; // Chunk size buffer
        let mut chunk_count = 0u32;

        loop {
            // Try to read audio data from the channel
            let samples_read = {
                let audio_channels =
                    self.audio_channels
                        .as_mut()
                        .ok_or(ChunkSinkError::InvalidData(
                            "Audio channels not initialized".to_string(),
                        ))?;
                audio_channels.output_consumer.pop_slice(&mut buffer)
            };

            if samples_read > 0 {
                // Convert f32 to u32 for protobuf transmission
                // Scale float samples to 32-bit integer range
                let data: Vec<u32> = buffer[..samples_read]
                    .iter()
                    .map(|&sample| {
                        // Scale from [-1.0, 1.0] to [0, u32::MAX]
                        let scaled = (sample + 1.0) / 2.0;
                        (scaled.clamp(0.0, 1.0) * u32::MAX as f32) as u32
                    })
                    .collect();

                let chunk = AudioChunk {
                    session_id: format!("session_{}", chunk_count / 100), // Example session ID
                    chunk_count,
                    data,
                    timestamp: SystemTime::now(),
                };

                if let Err(e) = self.set_chunks(chunk).await {
                    error!("Failed to send audio chunk: {}", e);
                    return Err(e);
                }

                chunk_count += 1;
            } else {
                // No data available, yield control
                tokio::task::yield_now().await;
            }

            // Check for shutdown via parameter channels
            if let Some(ref mut param_channels) = self.parameter_channels {
                let mut param_buffer = [Parameters::Cut(); 1];
                if param_channels.input_consumer.pop_slice(&mut param_buffer) > 0 {
                    match param_buffer[0] {
                        Parameters::Shutdown() => {
                            info!("Shutdown signal received, stopping audio processing");
                            break;
                        }
                        Parameters::Cut() => {
                            info!("Cut signal received");
                            // Handle cut command if needed
                        }
                    }
                }
            }
        }

        Ok(())
    }

    /// Send a single audio chunk to the server
    async fn set_chunks(&mut self, chunk: AudioChunk) -> Result<bool, ChunkSinkError> {
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

    /// Start listening for server commands and forward them to parameter channels
    pub async fn start_command_listener(&mut self) -> Result<(), ChunkSinkError> {
        let client = self
            .client
            .as_mut()
            .ok_or(ChunkSinkError::ClientNotConnected)?;

        let request = Request::new(GetCommandRequest {
            recorder_id: self.config.recorder_id.clone(),
        });

        let response = client.get_commands(request).await?;
        let mut stream = response.into_inner();

        // Store commands in a simple way that we can check later
        info!("Starting command listener for server commands");

        // Spawn a task to handle the command stream
        tokio::spawn(async move {
            while let Some(result) = stream.next().await {
                match result {
                    Ok(command) => {
                        match command.command {
                            Some(chunksink::command::Command::CmdCutSession(_)) => {
                                info!("Received CutSession command from server");
                                // In a real implementation, you would send this to a channel
                                // or set a flag that can be checked elsewhere
                            }
                            Some(chunksink::command::Command::Reboot(_)) => {
                                info!("Received Reboot command from server");
                                // In a real implementation, you would send this to a channel
                                // or set a flag that can be checked elsewhere
                            }
                            None => {
                                warn!("Received empty command from server");
                                continue;
                            }
                        };
                    }
                    Err(e) => {
                        error!("Error receiving command from server: {}", e);
                        break;
                    }
                }
            }
            info!("Command listener stream ended");
        });

        Ok(())
    }

    /// Connect with retry logic
    pub async fn connect_with_retry(&mut self) -> Result<(), ChunkSinkError> {
        let mut attempts = 0;
        let max_attempts = self.config.max_retries;

        while attempts < max_attempts {
            match self.connect().await {
                Ok(_) => return Ok(()),
                Err(e) => {
                    attempts += 1;
                    if attempts >= max_attempts {
                        error!("Failed to connect after {} attempts: {}", max_attempts, e);
                        return Err(e);
                    }
                    warn!(
                        "Connection attempt {} failed: {}. Retrying in {:?}...",
                        attempts, e, self.config.retry_interval
                    );
                    sleep(self.config.retry_interval).await;
                }
            }
        }

        Err(ChunkSinkError::ClientNotConnected)
    }

    /// Get access to audio channels for external audio processing
    pub fn get_audio_channels(&mut self) -> Option<&mut AudioChannels> {
        self.audio_channels.as_mut()
    }

    /// Get access to parameter channels for external command handling
    pub fn get_parameter_channels(&mut self) -> Option<&mut ParameterChannels> {
        self.parameter_channels.as_mut()
    }

    /// Send audio data to the gRPC transmission pipeline
    pub fn send_audio_data(&mut self, data: &[f32]) -> Result<usize, ChunkSinkError> {
        let audio_channels = self
            .audio_channels
            .as_mut()
            .ok_or(ChunkSinkError::InvalidData(
                "Audio channels not initialized".to_string(),
            ))?;

        Ok(audio_channels.input_producer.push_slice(data))
    }

    /// Helper method to convert audio data before sending
    pub fn convert_and_send_audio(&mut self, data: &[f32]) -> Result<usize, ChunkSinkError> {
        self.send_audio_data(data)
    }

    /// Check for incoming parameters/commands
    pub fn receive_parameters(
        &mut self,
        buffer: &mut [Parameters],
    ) -> Result<usize, ChunkSinkError> {
        let parameter_channels =
            self.parameter_channels
                .as_mut()
                .ok_or(ChunkSinkError::InvalidData(
                    "Parameter channels not initialized".to_string(),
                ))?;

        Ok(parameter_channels.input_consumer.pop_slice(buffer))
    }

    /// Send a parameter/command to the parameter channel
    pub fn send_parameter(&mut self, parameter: Parameters) -> Result<usize, ChunkSinkError> {
        let parameter_channels =
            self.parameter_channels
                .as_mut()
                .ok_or(ChunkSinkError::InvalidData(
                    "Parameter channels not initialized".to_string(),
                ))?;

        let param_slice = [parameter];
        Ok(parameter_channels.output_producer.push_slice(&param_slice))
    }

    /// Send recorder status with retry logic
    pub async fn set_recorder_status_with_retry(
        &mut self,
        status: RecorderStatusInfo,
    ) -> Result<bool, ChunkSinkError> {
        let mut attempts = 0;
        let max_attempts = self.config.max_retries;

        while attempts < max_attempts {
            match self.set_recorder_status(status.clone()).await {
                Ok(success) => return Ok(success),
                Err(e) => {
                    attempts += 1;
                    if attempts >= max_attempts {
                        error!(
                            "Failed to set recorder status after {} attempts: {}",
                            max_attempts, e
                        );
                        return Err(e);
                    }
                    warn!(
                        "Status update attempt {} failed: {}. Retrying in {:?}...",
                        attempts, e, self.config.retry_interval
                    );
                    sleep(self.config.retry_interval).await;
                }
            }
        }

        Err(ChunkSinkError::ClientNotConnected)
    }

    /// Send chunks with retry logic
    pub async fn set_chunks_with_retry(
        &mut self,
        chunk: AudioChunk,
    ) -> Result<bool, ChunkSinkError> {
        let mut attempts = 0;
        let max_attempts = self.config.max_retries;

        while attempts < max_attempts {
            match self.set_chunks(chunk.clone()).await {
                Ok(success) => return Ok(success),
                Err(e) => {
                    attempts += 1;
                    if attempts >= max_attempts {
                        error!(
                            "Failed to set chunks after {} attempts: {}",
                            max_attempts, e
                        );
                        return Err(e);
                    }
                    warn!(
                        "Chunk upload attempt {} failed: {}. Retrying in {:?}...",
                        attempts, e, self.config.retry_interval
                    );
                    sleep(self.config.retry_interval).await;
                }
            }
        }

        Err(ChunkSinkError::ClientNotConnected)
    }

    /// Get the current configuration
    pub fn config(&self) -> &ChunkSinkConfig {
        &self.config
    }

    /// Update the configuration
    pub fn update_config(&mut self, config: ChunkSinkConfig) {
        self.config = config;
    }
}

/// Helper function to create a timestamp from SystemTime
pub fn system_time_to_timestamp(time: SystemTime) -> Result<Timestamp, ChunkSinkError> {
    let duration = time
        .duration_since(UNIX_EPOCH)
        .map_err(|e| ChunkSinkError::InvalidData(format!("Invalid timestamp: {}", e)))?;

    Ok(Timestamp {
        seconds: duration.as_secs() as i64,
        nanos: duration.subsec_nanos() as i32,
    })
}

/// Helper function to create a timestamp for the current time
pub fn current_timestamp() -> Result<Timestamp, ChunkSinkError> {
    system_time_to_timestamp(SystemTime::now())
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

    #[test]
    fn test_timestamp_conversion() {
        let now = SystemTime::now();
        let timestamp = system_time_to_timestamp(now).unwrap();
        assert!(timestamp.seconds > 0);
        assert!(timestamp.nanos >= 0);
    }

    #[test]
    fn test_current_timestamp() {
        let timestamp = current_timestamp().unwrap();
        assert!(timestamp.seconds > 0);
        assert!(timestamp.nanos >= 0);
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
