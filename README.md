# Session Recorder

## System Overview

Session Recorder is a distributed audio recording system with three main components:

- **C++ Chunk Source Client**: Captures audio from ALSA devices and streams chunks via gRPC
- **Go Backend Server**: Receives audio chunks, manages sessions, provides API
- **Vue.js Web Interface**: User interface for managing recordings and sessions

## Running Backend Server and Web Interface

### With Docker

```bash
# Start all backend services
./docker-build.sh up --build

# View service status
./docker-build.sh ps

# Follow logs
./docker-build.sh logs

# Stop services
./docker-build.sh down

# Complete cleanup
./docker-build.sh clean
```

This starts:

- **Web Interface**: http://localhost:3000
- **MinIO Console**: http://localhost:9090 (admin/password123)
- **Go Backend Services**:
    - ChunkSink gRPC: localhost:8779
    - SessionSource gRPC: localhost:8780
- **gRPC-Web Proxy**: http://localhost:8080
- **MinIO API**: http://localhost:9000


## Connecting Audio Sources

### Build the chunk sink (recording device)

Before building the components, ensure you have the required dependencies installed:

```bash
dnf install cmake make gcc-c++ alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel protobuf-devel boost-devel
```

#### 1. Generate Protocol Buffers

```bash
cd protocols
npm install
make clean
make all
cd ..
```

#### 2. Build C++ Client

```bash
cd ./cpp/chunk-sink-client
cmake --build .
chmod +x ./chunk-sink-client
cd ../..
```

### Execute

```bash
# Run from project root directory
./cpp/chunk-sink-client/chunk-sink-client --recorder-id <uuid> --recorder-name <name>
# ./cpp/chunk-sink-client/chunk-sink-client --recorder-id 10b26ce0-75ff-4548-84e1-c91d955b1151 --recorder-name "Living Room"
```

## Configuration

### Email Sharing (SMTP)

Email sharing allows sending session and segment download links via email. Configure with these environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `SMTP_HOST` | Yes | SMTP server hostname |
| `SMTP_PORT` | No | SMTP port (default: 587) |
| `SMTP_USERNAME` | Yes | SMTP authentication username |
| `SMTP_PASSWORD` | Yes | SMTP authentication password |
| `SMTP_FROM` | No | Sender email address |
| `SMTP_FROM_NAME` | No | Sender display name |

If `SMTP_HOST` is not set, email sharing is disabled.

### File Sharing

File sharing generates download links for sessions and segments. Three methods are available:

| Method | `FILE_SHARE_METHOD` | Description |
|--------|---------------------|-------------|
| Direct | `direct` (default) | Presigned URLs from the main S3 storage |
| S3 Copy | `s3_copy` | Copies files to a separate S3 bucket for sharing |
| Dropbox | `dropbox` | Uploads to Dropbox with shareable link |

**Dropbox configuration:**

| Variable | Required | Default |
|----------|----------|---------|
| `FILE_SHARE_DROPBOX_ACCESS_TOKEN` | Yes | - |
| `FILE_SHARE_DROPBOX_FOLDER` | No | `/SessionRecorder` |

To use Dropbox:

1. **Create a Dropbox App**:
   - Go to [Dropbox App Console](https://www.dropbox.com/developers/apps)
   - Click "Create app"
   - Select "Scoped access"
   - Choose "Full Dropbox" (access to all files) or "App folder" (isolated folder)
   - Enter an app name (e.g., "Session Recorder")
   - Click "Create app"

2. **Configure Permissions**:
   - In your app settings, go to the "Permissions" tab
   - Enable these permissions:
     - `files.content.write` - to upload files
     - `sharing.write` - to create shared links
   - Click "Submit" to save permissions

3. **Generate Access Token**:
   - Go to the "Settings" tab
   - Scroll to "OAuth 2" section
   - Under "Generated access token", click "Generate"
   - Copy the token (it won't be shown again)

   **Note**: This generates a long-lived access token for development. For production, consider implementing OAuth 2.0 refresh token flow.

4. **Set the environment variable**:
   ```bash
   export FILE_SHARE_DROPBOX_ACCESS_TOKEN="your_access_token_here"
   ```

5. **(Optional) Custom folder path**:
   ```bash
   export FILE_SHARE_DROPBOX_FOLDER="/MyRecordings"
   ```

**S3 Copy configuration:**

| Variable | Required | Default |
|----------|----------|---------|
| `FILE_SHARE_S3_ENDPOINT` | Yes | - |
| `FILE_SHARE_S3_PUBLIC_ENDPOINT` | No | Same as endpoint |
| `FILE_SHARE_S3_ACCESS_KEY` | Yes | - |
| `FILE_SHARE_S3_SECRET_KEY` | Yes | - |
| `FILE_SHARE_S3_BUCKET` | No | `shared-files` |
| `FILE_SHARE_S3_USE_SSL` | No | `true` |

## Dev Environment

```bash
# Start envoy & minio
./start-dev.sh

# Stop envoy & minio
./stop-dev.sh
```

```bash
# Start go Backend
cd ./go/cmd/chunk_sink
S3_ENDPOINT=localhost:9000 S3_PUBLIC_ENDPOINT=localhost:9000 S3_ACCESS_KEY=admin S3_SECRET_KEY=password123 \
SMTP_HOST=<smtp_host> SMTP_PORT=<smtp_port> SMTP_USERNAME=<smtp_username> SMTP_PASSWORD=<smtp_password> SMTP_FROM=<from_email> SMTP_FROM_NAME=<from_name> \
FILE_SHARE_METHOD=dropbox FILE_SHARE_DROPBOX_ACCESS_TOKEN=<your_token> \
go run .
```

```bash
# Start web interface
cd ./web
npm start
```
