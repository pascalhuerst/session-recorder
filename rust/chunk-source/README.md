# Chunk Source - Session Recorder Audio Client

A comprehensive Rust implementation of an audio processing client with mDNS service discovery, gRPC communication, and hardware I/O support for the Session Recorder system.

## Features

### 🎵 Audio Processing
- **Real-time Audio I/O**: Low-latency ALSA-based audio capture and playback
- **Lock-free Communication**: Ring buffer-based audio pipeline
- **Multi-threaded Architecture**: Dedicated threads for audio processing
- **CPU Affinity**: Optimized for real-time performance

### 🔍 mDNS Service Discovery
- **Automatic Discovery**: Finds chunk-sink servers on the local network
- **Real-time Monitoring**: Tracks service availability and health
- **Event-driven Architecture**: Service discovered/updated/removed events
- **Configurable Timeouts**: Customizable service detection parameters

### 🌐 gRPC Client
- **Full ChunkSink Support**: Implements all ChunkSink service methods
- **Audio Channel Integration**: Seamless audio data transmission
- **Command Processing**: Receives and processes server commands
- **Automatic Reconnection**: Built-in retry logic and error handling

### 🎛️ Hardware I/O
- **Input Event Handling**: Keyboard, button, and device input processing
- **LED Control**: Linux sysfs-based LED management
- **Type-safe APIs**: Modern Rust interfaces for hardware control

## Quick Start

### Basic Usage

```bash
# Build the project
cargo build --release

# Run with default settings
./target/release/chunk-source

# Run with mDNS discovery enabled
ENABLE_MDNS_TEST=1 ./target/release/chunk-source

# Run with verbose logging
RUST_LOG=info ./target/release/chunk-source
```

### mDNS Service Discovery

```rust
use chunk_source::mdns::service_tracker::{ServiceTracker, ServiceTrackerConfig};

// Create service tracker
let config = ServiceTrackerConfig::default();
let mut tracker = ServiceTracker::new(config)?;

// Start discovery
let event_receiver = tracker.start()?;

// Handle discovered services
while let Ok(event) = event_receiver.recv() {
    match event {
        ServiceEvent::ServiceDiscovered(info) => {
            println!("Found server: {}", info.connection_url().unwrap());
            // Connect with gRPC client
        }
        ServiceEvent::ServiceRemoved(name) => {
            println!("Lost server: {}", name);
        }
        _ => {}
    }
}
```

### gRPC Client Integration

```rust
use chunk_source::grpc::chunk_sink_client::{ChunkSinkClientService, ChunkSinkConfig};

// Create client
let config = ChunkSinkConfig {
    server_address: "http://192.168.1.100:50051".to_string(),
    recorder_id: "my-recorder".to_string(),
    ..Default::default()
};

let mut client = ChunkSinkClientService::new(config);
client.initialize_channels();

// Connect and send data
client.connect().await?;
client.send_audio_data(&audio_samples)?;
```

## Configuration

### Audio Settings

```rust
let audio_settings = AudioSettings {
    input_device: "default".to_string(),
    output_device: "default".to_string(),
    num_channels: 2,
    period_size: 512,
    buffer_size: 2048,
    sample_rate: 44100,
};
```

### mDNS Configuration

```rust
let tracker_config = ServiceTrackerConfig {
    service_type: "_session-recorder-chunksink._tcp.local.".to_string(),
    service_timeout: Duration::from_secs(60),
    cleanup_interval: Duration::from_secs(10),
    max_services: 100,
};
```

### gRPC Configuration

```rust
let grpc_config = ChunkSinkConfig {
    server_address: "http://localhost:50051".to_string(),
    recorder_id: "audio-recorder-001".to_string(),
    recorder_name: "Audio Recorder".to_string(),
    connect_timeout: Duration::from_secs(10),
    request_timeout: Duration::from_secs(5),
    retry_interval: Duration::from_secs(3),
    max_retries: 5,
    audio_buffer_size: 8192,
    parameter_buffer_size: 64,
};
```

## Examples

### Complete Integration Example

```bash
# Run the full audio pipeline with mDNS discovery
cargo run --example grpc_audio_pipeline

# Run mDNS service discovery demo
cargo run --example mdns_service_discovery
```

### Hardware I/O Examples

```rust
// LED Control
use chunk_source::io::led::Led;

let led = Led::new("power")?;
led.on()?;
led.set_brightness_percent(0.5)?;
led.set_timer(1000, 1000)?; // 1 second on, 1 second off

// Input Key Handling
use chunk_source::io::input_key::InputKey;

let mut input_key = InputKey::new(3)?; // Keyboard device
input_key.register_key(
    KeyCode::KEY_SPACE,
    || println!("Space pressed!"),
    |duration| println!("Space held for {:?}", duration)
);
input_key.start()?;
```

## Architecture

### System Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Audio Input   │───▶│  Audio Pipeline │───▶│   gRPC Client   │
│   (Microphone)  │    │  (Ring Buffers) │    │   (Network)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │ Audio Processing│    │  mDNS Discovery │
                       │   (Main Thread) │    │   (Auto-connect)│
                       └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  Audio Output   │    │ Server Commands │
                       │   (Speakers)    │    │  (Cut/Shutdown) │
                       └─────────────────┘    └─────────────────┘
```

### Thread Architecture

- **Audio Callback Thread**: Real-time audio I/O (pinned to dedicated CPU core)
- **Main Processing Thread**: Audio effects and processing
- **mDNS Discovery Thread**: Service discovery and monitoring
- **gRPC Client Thread**: Network communication and command handling

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RUST_LOG` | Logging level (error, warn, info, debug, trace) | `error` |
| `ENABLE_MDNS_TEST` | Enable mDNS testing in main application | `disabled` |

## Dependencies

### Core Dependencies
- `tokio` - Async runtime
- `tonic` - gRPC client
- `alsa` - Audio I/O
- `mdns-sd` - mDNS service discovery
- `evdev` - Input event handling
- `ringbuf` - Lock-free ring buffers

### System Requirements
- Linux (ALSA support required)
- Rust 1.70+ (2021 edition)
- Audio hardware (microphone/speakers)
- Network interface (for mDNS discovery)

## Troubleshooting

### Common Issues

#### Audio Permissions
```bash
# Add user to audio group
sudo usermod -a -G audio $USER

# Or run with elevated privileges
sudo ./target/release/chunk-source
```

#### mDNS Not Working
```bash
# Check if avahi daemon is running
systemctl status avahi-daemon

# Install avahi if needed
sudo apt-get install avahi-daemon
```

#### Normal mDNS Error Messages
You may see this error message when starting the application:
```
ERROR mdns_sd::service_daemon] Failed to send SearchStarted(_session-recorder-chunksink._tcp.local.)(repeating:true): sending on a closed channel
```

**This is completely normal behavior** and not an actual error. It occurs when:
- No chunk-sink servers are currently available on the network
- The mDNS service discovery is working correctly but has nothing to report yet

The service discovery will continue running and will automatically connect when servers become available, even hours later.

#### Network Discovery Issues
```bash
# Check firewall settings
sudo ufw status

# Allow mDNS traffic
sudo ufw allow 5353/udp
```

### Debug Mode

```bash
# Enable detailed logging
RUST_LOG=debug ./target/release/chunk-source

# Enable mDNS debugging
RUST_LOG=debug,mdns_sd=trace ./target/release/chunk-source
```

## Development

### Building

```bash
# Debug build
cargo build

# Release build (optimized)
cargo build --release

# Build with all features
cargo build --release --all-features
```

### Testing

```bash
# Run unit tests
cargo test

# Run with mDNS tests
ENABLE_MDNS_TEST=1 cargo test

# Run integration tests
cargo test --test integration
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## How mDNS Service Discovery Works

The mDNS service discovery in this project is designed to:

1. **Wait Patiently**: Continuously monitor the network for chunk-sink servers
2. **Auto-Connect**: Automatically connect to servers when they appear
3. **Handle Disconnections**: Gracefully handle server disappearances
4. **Resume Operations**: Reconnect when servers come back online

### Expected Behavior

- **No Servers**: Discovery runs quietly in the background
- **Server Appears**: Automatic connection and data transmission begins
- **Server Disappears**: Clean disconnection and return to waiting state
- **Server Returns**: Automatic reconnection without user intervention

This means the system can run for hours or days, automatically adapting to the network environment as chunk-sink servers come and go.

## Support

For issues and questions:
- Check the troubleshooting section above
- Review the examples in the `examples/` directory
- Open an issue on the project repository