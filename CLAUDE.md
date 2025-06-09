# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Reference

### Fast Commands
```bash
# Docker deployment (recommended)
./docker-build.sh up --build

# Local build all components
./build.sh

# Protocol generation
cd protocols/ && make all
```

### Documentation
Detailed guides are available in the `.claude/` directory:
- **Development**: `.claude/development.md` - Local setup and workflow
- **Architecture**: `.claude/architecture.md` - System design and components
- **Docker**: `.claude/docker.md` - Container deployment guide