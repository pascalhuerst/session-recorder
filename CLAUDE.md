# Session Recorder

Distributed audio recording system — captures audio from ALSA devices via gRPC, manages sessions with S3 storage, and provides a web interface for playback.

## Tech Stack

| Component | Directory | Stack |
|-----------|-----------|-------|
| Web Interface | `web/` | Vue 3.4, TypeScript 5.8, Vite 5, Nx 17, npm |
| Go Backend | `go/` | Go 1.24, gRPC, MinIO (S3), zerolog, avahi |
| Recorder (device client) | `rust/recorder/` | Rust, ALSA, tonic (gRPC), mdns-sd |
| Protocols | `protocols/` | Protocol Buffers → Go, TypeScript stubs |

## Project Structure

```
session-recorder/
├── web/                       # Vue.js web interface
│   ├── src/
│   │   ├── views/             # Page components
│   │   ├── layout/            # Layout components
│   │   ├── store/             # Pinia/Nanostores
│   │   ├── grpc/              # gRPC client setup
│   │   ├── services/          # Business logic (Toaster, etc.)
│   │   ├── types.ts           # Type definitions
│   │   ├── env.ts             # Environment config
│   │   └── main.ts            # Entry point
│   └── libs/
│       └── session-waveform/  # Waveform visualization library
├── go/                        # Go backend services
│   ├── cmd/
│   │   ├── session_recorder_server/  # Backend (hosts ChunkSink + SessionSource gRPC services)
│   │   ├── chunk_sink_client/      # Test client for ChunkSink
│   │   └── session_source_client/  # Test client for SessionSource
│   ├── grpc/                  # gRPC server implementations
│   ├── storage/               # MinIO/S3 storage layer
│   └── render/                # Audio rendering (waveforms)
├── rust/
│   ├── recorder/              # Device client: ALSA capture → chunksink protocol client
│   └── recorder-display/      # On-device status display (SessionSource client)
├── protocols/                 # Protocol Buffer definitions
│   └── proto/                 # .proto files (generates go/, ts/)
├── docker-compose.yml         # Production setup
└── docker-compose.dev.yml     # Development setup (MinIO only)
```

Protocol naming is written from the backend's perspective: the backend hosts the
**ChunkSink** service (the sink that receives audio chunks); the recorder is the
client of that protocol. The terms "chunk source" / "chunk sink client" are
equivalent to the recorder but are used **only** in the protocol definitions and
generated stubs — everywhere else the device client is called the **recorder**.

## Commands

### Docker (from project root)
```bash
./docker-build.sh up --build   # Start all services
./docker-build.sh down         # Stop services
./docker-build.sh logs         # View logs
./docker-build.sh clean        # Full cleanup
```

### Development Mode (from project root)
```bash
docker compose -f docker-compose.dev.yml up    # MinIO only
docker compose -f docker-compose.dev.yml down
```
Then run the Go backend natively (`cd go && source sourceme.sh && go run ./cmd/session_recorder_server`) and the web with `cd web && pnpm start`.

### Web (from `web/`)
```bash
npm start                      # Dev server (via Nx)
npm run build                  # Production build
npm run test                   # Run tests (Vitest)
npx nx lint                    # Lint
npx vue-tsc --noEmit           # Type check
npx nx storybook               # Storybook
```

### Protocols (from `protocols/`)
```bash
make all                       # Generate all (Go, TS)
make ts                        # TypeScript only
make go                        # Go only
make clean                     # Clean generated files
```

### Go (from `go/`)
```bash
go run ./cmd/session_recorder_server  # Run backend (ChunkSink + SessionSource gRPC)
go run ./cmd/chunk_sink_client      # Run ChunkSink test client
go run ./cmd/session_source_client  # Run SessionSource test client
go test ./...                       # Run all tests
go build ./...                      # Build all
go vet ./...                        # Vet
```

## Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| Web Interface | 3000 | HTTP |
| SessionSource gRPC-Web (session_recorder_server HTTP listener) | 8081 | HTTP |
| ChunkSink gRPC | 8779 | gRPC |
| SessionSource gRPC | 8780 | gRPC |
| MinIO API | 9000 | HTTP |
| MinIO Console | 9090 | HTTP |

## Environment Variables

### Go Backend
```bash
S3_ENDPOINT=localhost:9000
S3_PUBLIC_ENDPOINT=localhost:9000
S3_ACCESS_KEY=admin
S3_SECRET_KEY=password123
S3_USE_SSL=false
```

### Web (Vite)
```bash
VITE_GRPC_SERVER_URL=http://localhost:8081
```

## Architecture

- **gRPC + Protocol Buffers** for type-safe streaming between components
- **MinIO** for S3-compatible self-hosted audio storage
- **In-process gRPC-Web** wrapper inside `session_recorder_server` (no envoy/proxy container)
- **Monorepo** with optimal language per component

## Patterns & Conventions

### Vue / TypeScript
- Vue 3 Composition API with `<script setup lang="ts">`
- Pinia stores with function syntax (`defineStore('name', () => { ... })`)
- gRPC streaming with cleanup functions (stop/restart pattern)
- Symbol-based provide/inject for context
- Zod for runtime validation
- Nanostores for cross-component state in the waveform library
- CSS uses pollen-css variables (`--scale-*`, `--size-*`, `--color-grey-*`, `--radius-*`, `--weight-*`)
- No `--color-white` — use `white` directly; for opacity use `color-mix()`

### Common Imports
```typescript
// gRPC clients
import { SessionSourceClient } from '@session-recorder/protocols/sessionsource.client';
import type { Recorder, Session, Segment } from '@session-recorder/protocols/sessionsource';

// Nanostores (waveform library)
import { useAtom } from '@nanostores/vue';
import { atom, computed as nanoComputed } from 'nanostores';

// Toast notifications
import { toastService } from '@/services/Toaster';
```

### Git
- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`
- Branch naming: `issy/<topic>` or `feature/<description>`
