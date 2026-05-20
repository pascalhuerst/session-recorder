# Session Recorder Web

## Development

### Libraries

#### Session Waveform (Player App)

The easiest way to work on the Player app is by launching Storybook:

```
npx nx storybook --project session-waveform
```

### Application

#### Dev Mode

To run application in dev mode, go to `/web` directory:

##### Run `pnpm install`

##### Make sure the Go backend is running

The chunk_sink binary serves gRPC-Web in-process (default port 8081) — no separate
proxy container needed. From the repo root:

```
cd go && source sourceme.sh && go run ./cmd/chunk_sink
```

##### Setup .env

`.env.development` is already wired to `http://localhost:8081`. Use `.env.example`
as a reference if you need to override.

##### Start the app

```
pnpm start
```
