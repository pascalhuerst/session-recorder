# Web Interface

Vue 3 + Vite + Nx + TypeScript. Talks to the Go backend over gRPC-Web
(`@protobuf-ts/grpcweb-transport`) against `VITE_GRPC_SERVER_URL`. The
backend serves gRPC-Web in-process — there is no separate proxy container.

## Layout

```
web/
├── src/                              # Application
│   ├── views/                        # Page components
│   ├── layout/                       # Layout components
│   ├── store/                        # Pinia stores
│   ├── grpc/                         # gRPC client wiring
│   ├── services/                     # Cross-cutting services (e.g. Toaster)
│   ├── types.ts
│   ├── env.ts                        # Zod-validated env config
│   └── main.ts
├── libs/
│   └── session-waveform/             # Reusable waveform/editor library (also has stories)
├── .env.development                  # VITE_GRPC_SERVER_URL for local dev (defaults to :8081)
├── .env.production                   # VITE_GRPC_SERVER_URL for builds
└── vite.config.ts
```

## Dev mode

1. Make sure the Go backend is running. From the repo root:

   ```bash
   cd go && source sourceme.sh && go run ./cmd/chunk_sink
   ```

   The backend serves gRPC-Web on port `8081` by default, which is what
   `.env.development` already points to.

2. Install deps and start vite:

   ```bash
   pnpm install
   pnpm start          # vite dev server on http://localhost:4200
   ```

`pnpm run dev` (note: `dev`, not `start`) instead spins up docker + air +
vite together in one terminal — convenient when you want the whole stack
running with one command.

### Pointing at a different backend

Override `VITE_GRPC_SERVER_URL` either in `.env.development.local` (not
committed) or as a localStorage value at runtime — see `env.ts` for the
override mechanism.

## Storybook

The waveform library is easiest to work on through Storybook:

```bash
npx nx storybook --project session-waveform
```

## Build / lint / test

```bash
pnpm run build              # production build (uses .env.production)
pnpm run test               # vitest
npx nx lint                 # eslint
npx vue-tsc --noEmit        # type check
```

## Notes

- The waveform display uses [Peaks.js](https://github.com/bbc/peaks.js)
  with waveform data pre-rendered server-side (see `go/render/waveform.go`).
- Filesystem-storage backend mode on the server side disables UI playback:
  the UI fetches audio through presigned URLs that only MinIO can produce.
  Use MinIO if you need in-browser playback.
