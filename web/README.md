# Session Recorder Web

Vue 3 web interface for session playback, waveform visualization, and recorder management.

## Development

```bash
pnpm install
pnpm run dev       # Starts Docker (MinIO + Envoy) + Go backend + Vite
```

Or run components individually:

```bash
pnpm run dev:docker    # MinIO + Envoy only
pnpm run dev:backend   # Go backend with air (hot reload)
pnpm run dev:web       # Vite dev server only
pnpm run dev:stop      # Stop Docker services
```

## Build & Test

```bash
npm run build          # Production build
npm run test           # Vitest
npx nx lint            # ESLint
npx vue-tsc --noEmit   # Type check
npx nx storybook       # Storybook (session-waveform library)
```

## Environment

Create `.env` from `.env.example`:

```bash
VITE_GRPC_SERVER_URL=http://localhost:8080
```

## Conventions

- Vue 3 Composition API with `<script setup lang="ts">`
- Pinia stores with function syntax (`defineStore('name', () => { ... })`)
- gRPC streaming with cleanup functions (stop/restart pattern)
- Nanostores for cross-component state in `libs/session-waveform/`
- CSS uses pollen-css variables (`--scale-*`, `--size-*`, `--color-grey-*`, `--radius-*`, `--weight-*`)
- No `--color-white` — use `white` directly; for opacity use `color-mix()`
