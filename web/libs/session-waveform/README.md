# Session Waveform Library

Reusable Vue.js component library for audio waveform visualization and session management.

## Features

- **Waveform Visualization**: Real-time audio waveform display using Peaks.js
- **Audio Playback Controls**: Play, pause, seek, zoom functionality
- **Segment Management**: Create, edit, and manage audio segments
- **Vue 3 Components**: Composition API with TypeScript
- **Storybook Integration**: Component development environment

## Development

### Storybook
Launch component development environment:
```bash
npx nx storybook --project session-waveform
```

### Testing
Run unit tests:
```bash
nx test session-waveform
```

### Build
Build the library:
```bash
nx build session-waveform
```

## Components

- **WaveformEditor**: Main waveform visualization component
- **Audio Controls**: Playback control components
- **Segment Tools**: Segment creation and management
- **Zoom Controls**: Waveform zoom and navigation

## Usage

This library is used by the main Session Recorder web interface for audio visualization and session management.

## Architecture

- **NX Library**: Part of the Session Recorder NX workspace
- **Peaks.js**: Audio waveform visualization engine
- **Vue 3**: Component framework with Composition API
- **Vite**: Build tool and bundler

See main project documentation for complete system architecture.
