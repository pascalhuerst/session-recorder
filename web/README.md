# Session Recorder Web Interface

Vue.js web application for managing audio recording sessions with real-time waveform visualization.

## Features

- **Session Management**: View, start, stop, and manage recording sessions
- **Real-time Visualization**: Audio waveform display using Peaks.js
- **Multi-source Support**: Monitor multiple connected audio clients
- **Export Functionality**: Export sessions in various formats
- **Responsive UI**: Modern Vue.js interface with NX workspace

## Quick Start

### Docker (Recommended)
```bash
# From project root
./docker-build.sh up --build
# Access: http://localhost:3000
```

### Local Development
```bash
npm install
npm start
```

## Development

### Prerequisites
- Node.js & npm
- Backend services running (see main README.md)

### Environment Setup
Create `.env` file with:
```bash
VITE_GRPC_SERVER_URL=http://localhost:8080
VITE_FILE_SERVER_URL=http://localhost:9000
```

### Libraries

#### Session Waveform Component
Reusable waveform visualization library:
```bash
npx nx storybook --project session-waveform
```

### Build
```bash
npm run build    # Production build
npm test         # Run tests
```

## Architecture

- **NX Workspace**: Monorepo with shared libraries
- **Vue.js 3**: Composition API with TypeScript
- **gRPC-Web**: Communication with Go backend
- **Peaks.js**: Audio waveform visualization
- **Vite**: Build tool and dev server

See `.claude/architecture.md` for complete system design.
