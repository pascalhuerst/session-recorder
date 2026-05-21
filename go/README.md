# Go Backend

Hosts the `session_recorder_server` binary (the backend) and two CLI test clients.

```
go/
├── cmd/
│   ├── session_recorder_server/ # Backend (ChunkSink + SessionSource gRPC + embedded gRPC-Web)
│   ├── chunk_sink_client/      # Test client that pushes test audio chunks
│   └── session_source_client/  # Test client to list/watch recorders, list sessions, etc.
├── broadcast/                  # In-memory fan-out for streaming RPCs
├── email/                      # SMTP sender (optional — used for session sharing)
├── fileshare/                  # External file-sharing backends (Dropbox, S3-copy, direct)
├── grpc/                       # gRPC server impls + gRPC-Web HTTP wrapper
├── logger/                     # zerolog setup
├── mdns/                       # Avahi-backed mDNS service publisher
├── render/                     # Audio rendering: waveform, flac, ogg, segment clipping
├── storage/                    # Storage backends: MinIO (S3) and local filesystem
└── utils/                      # Small env helpers
```

## Build & run

```bash
go build ./...
go test ./...
go vet ./...

# Run the backend (needs MinIO if you don't use --storage-fs-root)
source sourceme.sh           # or export S3_ACCESS_KEY / S3_SECRET_KEY yourself
go run ./cmd/session_recorder_server
```

Hot reload via [air](https://github.com/air-verse/air): `air` (config in `.air.toml`).

## CLI flags

Run `go run ./cmd/session_recorder_server -h` for the authoritative help (incl. the
multi-line explanation of `--storage-fs-root`).

| Flag | Env Var | Default | Purpose |
|---|---|---|---|
| `--chunk-sink-port` | `CHUNK_SINK_PORT` | `8779` | ChunkSink gRPC port |
| `--session-source-port` | `SESSION_SOURCE_PORT` | `8780` | SessionSource gRPC port |
| `--grpcweb-port` | `GRPCWEB_PORT` | `8081` | HTTP port for the embedded gRPC-Web wrapper of SessionSource (replaces the old envoy proxy) |
| `--generate-waveform` | `GENERATE_WAVEFORM` | `false` | Generate `waveform.dat` + `overview.png` on close (needs the `audiowaveform` binary). `data.flac`/`data.ogg` are always rendered. |
| `--storage-fs-root` | `STORAGE_FS_ROOT` | _(unset)_ | If set: store everything under this directory and ignore all `--s3-*` flags. If unset: use MinIO. |
| `--s3-endpoint` | `S3_ENDPOINT` | `localhost:9000` | Internal MinIO endpoint |
| `--s3-access-key` | `S3_ACCESS_KEY` | _(required for MinIO)_ | |
| `--s3-secret-key` | `S3_SECRET_KEY` | _(required for MinIO)_ | |
| `--s3-local-endpoint` | `S3_LOCAL_ENDPOINT` | _(derived)_ | Endpoint baked into UI URLs |
| `--s3-local-host` / `--s3-local-port` | `S3_LOCAL_HOST` / `S3_LOCAL_PORT` | _(derived)_ | Split form of `--s3-local-endpoint` |
| `--s3-public-endpoint` | `S3_PUBLIC_ENDPOINT` | _(derived)_ | Endpoint baked into email-sharing URLs |
| `--s3-public-host` / `--s3-public-port` | `S3_PUBLIC_HOST` / `S3_PUBLIC_PORT` | _(derived)_ | Split form of `--s3-public-endpoint` |

## Storage backends

`session_recorder_server` ships with two storage backends behind the `storage.Storage`
interface. The on-disk layout is identical so a snapshot of one can be loaded
by the other by copying files.

### MinIO (default)

S3-compatible. Required for the web UI to play audio back, because the UI
fetches files via presigned URLs that only MinIO can generate. Used unless
`--storage-fs-root` is set.

### Filesystem (`--storage-fs-root <path>`)

No container, no credentials. Audio is appended directly to
`<root>/<recorder-id>/sessions/<session-id>/data.raw` and rendered in place
to `data.flac`, `data.ogg`, `waveform.dat`, `overview.png`.

Differences vs MinIO worth knowing:

- ✅ No 5 MB part-size floor → short sessions render with no trailing
  zero-padding (the bug we fixed for MinIO is structurally impossible here).
- ✅ External file sharing (Dropbox / s3_copy / direct) still works — it
  reads through `Storage.GetSessionFileReader` which the fs backend implements.
- ❌ Web UI playback **does not work** — `GetPresignedURL` returns
  `ErrNotSupportedByFsBackend`. The `session_source_client` CLI works fine.

## Test clients

### `chunk_sink_client`

Streams a baked-in 1 kHz sine into the ChunkSink for the given host. Useful
for proving the wire format end-to-end without setting up the Rust/C++
client.

```bash
go run ./cmd/chunk_sink_client --host localhost
```

### `session_source_client`

CLI against the SessionSource service.

```bash
# One-shot snapshot of currently-known recorders (with status if available)
go run ./cmd/session_source_client list-recorders

# One-shot list of sessions for a recorder
go run ./cmd/session_source_client list-sessions <recorder-id>

# Long-lived: watch status updates as they stream in (Ctrl-C to stop)
go run ./cmd/session_source_client watch-recorders
go run ./cmd/session_source_client watch-recorders <recorder-id>   # filtered

# Delete a session
go run ./cmd/session_source_client delete-session <recorder-id> <session-id>
```

All commands connect to `localhost:8780` (SessionSource gRPC).

## Logging

Uses [zerolog](https://github.com/rs/zerolog). `logger.Setup()` is the
single entrypoint. Output is plain text by default.

## Tests

Standard Go tests:

```bash
go test ./...
go test -v -race ./storage   # one specific package
```

The `render` package tests need `audiowaveform` on `PATH` to run the
waveform-generation test (skipped otherwise).
