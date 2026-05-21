# Session Recorder

A distributed audio recording system with a native audio-capture client (the
**recorder**, in Rust), a Go backend, and a Vue.js web interface.

## I want to... develop locally

**Prerequisites:** [pnpm](https://pnpm.io/), [Go](https://golang.org/) 1.24+, [Docker](https://www.docker.com/)

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

If you'd rather run MinIO natively too, the backend can also use a local
directory instead of MinIO — see the `--storage-fs-root` flag in
[`go/README.md`](go/README.md).

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

The **recorder** is the device-side client. It reads audio from a soundcard,
detects when a signal is present, and streams the captured audio to the backend.
It implements a client for the chunksink protocol — see
[`rust/recorder/README.md`](rust/recorder/README.md) for details.

#### 1. Build

```bash
cd rust/recorder
cargo build --release
```

#### 2. Run

```bash
./target/release/recorder \
  --recorder-id $(uuidgen) \
  --recorder-name "Living Room" \
  --device default
```

Useful optional flags (see `recorder --help` for the full list):

| Flag | Purpose |
|---|---|
| `--detector-threshold-db -45` | dB-FS RMS above which signal is considered "present" |
| `--attack-time 0.5` / `--release-time 5.0` | seconds for silence→signal / signal→silence transitions |
| `--led-rec-state <sysfs-name>` | LED on while recording, blinks 10× on new session |
| `--led-upload <sysfs-name>` | LED pulses on each successful chunk upload |
| `--input-event 3 --input-keycode 28 --input-hold-ms 800` | Local "cut session" button on `/dev/input/eventNN` |

## I want to... enable email sharing

Set these environment variables on the backend:

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

## Architecture

```
┌─────────────────┐     gRPC       ┌──────────────────────────┐
│  recorder       │ ─────────────► │  Go Backend              │
│  (Rust, ALSA)   │                │  ChunkSink (gRPC :8779)  │
└─────────────────┘                │  SessionSource (:8780)   │
                                   │  SessionSource gRPC-Web  │
┌─────────────────┐    gRPC-Web    │  (HTTP :8081, embedded)  │
│  Vue.js Web UI  │ ◄────────────► │                          │
└─────────────────┘                └────────────┬─────────────┘
                                                │
                                                ▼
                                       ┌─────────────────┐
                                       │ MinIO (S3) OR   │
                                       │ local directory │
                                       └─────────────────┘
```

Protocol names are written from the backend's perspective: the backend hosts the
**ChunkSink** service (it is the sink that receives chunks); the recorder is the
client of that protocol. mDNS-driven discovery: the recorder finds the backend;
the backend finds nothing (it advertises and waits).

---

## Per-component documentation

- [`go/README.md`](go/README.md) — backend binary, CLI flags, storage backends, test clients
- [`web/README.md`](web/README.md) — dev mode, environment, storybook
- [`protocols/README.md`](protocols/README.md) — proto regeneration, codegen deps
- [`rust/recorder/README.md`](rust/recorder/README.md) — the recorder (Rust chunksink-protocol client)
- [`readme_deployment.md`](readme_deployment.md) — production deployment notes
