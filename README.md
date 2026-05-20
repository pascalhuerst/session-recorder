# Session Recorder

A distributed audio recording system with C++ audio capture clients, Go backend, and Vue.js web interface.

## I want to... develop locally

**Prerequisites:** [pnpm](https://pnpm.io/), [Go](https://golang.org/) 1.19+, [Docker](https://www.docker.com/)

```bash
cd web
pnpm install
pnpm run dev
```

This starts everything with hot reload:
- Docker services (MinIO)
- Go backend with [air](https://github.com/air-verse/air) (auto-restarts on `.go` changes; serves gRPC-Web in-process on 8081)
- Vite dev server (HMR)

**Stop:** `Ctrl+C`, then `pnpm run dev:stop`

**Individual services:**
```bash
pnpm run dev:docker    # Docker only
pnpm run dev:backend   # Go backend only
pnpm run dev:web       # Vite only
```

| Service | URL |
|---------|-----|
| Web Interface | http://localhost:4200 |
| MinIO Console | http://localhost:9091 (admin/password123) |
| SessionSource gRPC-Web | http://localhost:8081 |

---

## I want to... deploy with Docker

```bash
./docker-build.sh up --build
```

| Command | Action |
|---------|--------|
| `./docker-build.sh up --build` | Start all services |
| `./docker-build.sh ps` | View status |
| `./docker-build.sh logs` | Follow logs |
| `./docker-build.sh down` | Stop services |
| `./docker-build.sh clean` | Remove everything |

| Service | URL |
|---------|-----|
| Web Interface | http://localhost:3000 |
| MinIO Console | http://localhost:9090 (admin/password123) |
| SessionSource gRPC-Web | http://localhost:8081 |

---

## I want to... connect a recording device

### 1. Install dependencies (Fedora)

```bash
dnf install cmake make gcc-c++ alsa-lib-devel avahi-devel \
    grpc-data grpc grpc-cpp grpc-plugins grpc-devel \
    protobuf-devel boost-devel
```

### 2. Generate protocols

```bash
cd protocols
pnpm install
make all
```

### 3. Build C++ client

```bash
cd cpp/chunk-sink-client
cmake --build .
```

### 4. Run

```bash
./cpp/chunk-sink-client/chunk-sink-client \
  --recorder-id $(uuidgen) \
  --recorder-name "Living Room" \
  --device default
```

---

## I want to... enable email sharing

Set these environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `SMTP_HOST` | Yes | SMTP server hostname |
| `SMTP_PORT` | No | Port (default: 587) |
| `SMTP_USERNAME` | Yes | Auth username |
| `SMTP_PASSWORD` | Yes | Auth password |
| `SMTP_FROM` | No | Sender email |
| `SMTP_FROM_NAME` | No | Sender name |

---

## I want to... share files via Dropbox

Set `FILE_SHARE_METHOD=dropbox` and configure:

| Variable | Required |
|----------|----------|
| `FILE_SHARE_DROPBOX_ACCESS_TOKEN` | Yes |
| `FILE_SHARE_DROPBOX_FOLDER` | No (default: `/SessionRecorder`) |

**Setup:**
1. Create app at [Dropbox App Console](https://www.dropbox.com/developers/apps)
2. Enable permissions: `files.content.write`, `sharing.write`
3. Generate access token in Settings > OAuth 2

---

## I want to... share files via S3

Set `FILE_SHARE_METHOD=s3_copy` and configure:

| Variable | Required | Default |
|----------|----------|---------|
| `FILE_SHARE_S3_ENDPOINT` | Yes | - |
| `FILE_SHARE_S3_PUBLIC_ENDPOINT` | No | Same as endpoint |
| `FILE_SHARE_S3_ACCESS_KEY` | Yes | - |
| `FILE_SHARE_S3_SECRET_KEY` | Yes | - |
| `FILE_SHARE_S3_BUCKET` | No | `shared-files` |
| `FILE_SHARE_S3_USE_SSL` | No | `true` |

---

## Reference

### Backend CLI flags

| Flag | Env Var | Default |
|------|---------|---------|
| `-chunk-sink-port` | `CHUNK_SINK_PORT` | 8779 |
| `-session-source-port` | `SESSION_SOURCE_PORT` | 8780 |
| `-grpcweb-port` | `GRPCWEB_PORT` | 8081 |
| `-s3-endpoint` | `S3_ENDPOINT` | localhost:9000 |
| `-s3-local-endpoint` | `S3_LOCAL_ENDPOINT` | (s3-endpoint) |
| `-s3-public-endpoint` | `S3_PUBLIC_ENDPOINT` | (s3-local-endpoint) |
| `-s3-access-key` | `S3_ACCESS_KEY` | (required) |
| `-s3-secret-key` | `S3_SECRET_KEY` | (required) |

### Architecture

```
┌─────────────────┐     gRPC       ┌──────────────────────────┐
│  C++ Client     │ ─────────────► │  Go Backend              │
│  (ALSA capture) │                │  ChunkSink (gRPC :8779)  │
└─────────────────┘                │  SessionSource (:8780)   │
                                   │  SessionSource gRPC-Web  │
┌─────────────────┐    gRPC-Web    │  (HTTP :8081, embedded)  │
│  Vue.js Web UI  │ ◄────────────► │                          │
└─────────────────┘                └────────────┬─────────────┘
                                                │
                                                ▼
                                       ┌─────────────────┐
                                       │  MinIO (S3)     │
                                       └─────────────────┘
```
