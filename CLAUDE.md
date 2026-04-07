# OpenWolf

@.wolf/OPENWOLF.md

This project uses OpenWolf for context management. Read and follow .wolf/OPENWOLF.md every session. Check .wolf/cerebrum.md before generating code. Check .wolf/anatomy.md before reading files.


# Session Recorder

Distributed audio recording system — captures audio from ALSA devices via gRPC, manages sessions with S3 storage, and provides a web interface for playback.

See `README.md` for setup, commands, and architecture.
See `docs/state-lifecycle.md` for state machines and data flow.
See `go doc ./storage/` and `go doc ./broadcast/` for backend internals.

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


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
