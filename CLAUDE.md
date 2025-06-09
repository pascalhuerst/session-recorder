# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Reference

### Fast Commands
```bash
# Local build protocols and C++ client only
./build-audio-client.sh

# Docker deployment (recommended)
./docker-build.sh up --build
```

### Build System
The build script now focuses on essential components:
- **Protocol generation**: Creates gRPC interfaces for all languages
- **C++ client build**: Compiles the audio capture client
- **Docker deployment**: Handles Go backend and web interface builds automatically