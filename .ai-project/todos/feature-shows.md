---
id: feature-shows
title: Shows — Pre-planned Session Recording with Post-Show Distribution
priority: high
category: feature
status: open
created: 2026-02-13
updated: 2026-02-13
labels: [shows, distribution, email, scheduling]
---

# Shows — Pre-planned Session Recording with Post-Show Distribution

## Description

Add a Shows feature that lets organizers pre-plan a recording schedule (e.g., a conference day with multiple talks), start a recording session tied to the show, and after the show bulk-render individual segments and distribute them to speakers via email.

## Data Model

### Show

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Unique identifier |
| name | string | e.g., "Day 1 — Main Stage" |
| date | date | Scheduled date |
| state | enum | `DRAFT`, `LIVE`, `ENDED`, `ARCHIVED` |
| recorderID | UUID | Which recorder to use |
| sessionID | UUID? | Linked recording session (set when show starts) |
| acts | Act[] | Ordered list of acts |

### Act

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Unique identifier |
| name | string | e.g., "Opening Keynote" |
| plannedStart | DateTime | Absolute planned start time |
| plannedEnd | DateTime | Absolute planned end time |
| emails | string[] | Recipient addresses for distribution |
| segmentID | UUID? | Linked segment (set after show ends) |
| actualStart | DateTime? | Operator-adjusted start (post-show) |
| actualEnd | DateTime? | Operator-adjusted end (post-show) |

## Lifecycle

1. **DRAFT** — Operator creates show, adds acts with names, times, emails
2. **DRAFT → LIVE** — "Start Show" triggers a new recording session on selected recorder
3. **LIVE** — Timeline view with current act, manual "Next Act" advance captures actual timestamps
4. **LIVE → ENDED** — Session stops (existing silence detection), auto-generates segments from act boundaries
5. **ENDED** — Operator adjusts times, bulk renders, bulk distributes via email
6. **ENDED → ARCHIVED** — Read-only summary with re-distribute option

## UI Structure

- **New top-level "Shows" section** alongside Recorders
- **Shows list**: Upcoming (DRAFT) and Past (ENDED/ARCHIVED) groups
- **Show detail (DRAFT)**: name, date, recorder selector, acts editor
- **Show detail (LIVE)**: timeline bar, active act, "Next Act" button
- **Show detail (ENDED)**: adjustable times, email editing, "Render All" + "Distribute All"
- **Show detail (ARCHIVED)**: read-only summary

## Backend

- **New `show.proto`** with Show/Act messages, ShowState enum, ShowService RPCs
- **Go handler** persists shows as JSON in MinIO
- **Reuses existing infrastructure**: session start/stop, segment model, sox rendering pipeline, share + email

## Phases

### Phase 1 — Core (MVP)

- [x] Show/Act proto definitions (added to `sessionsource.proto`)
- [x] Go handler: CRUD, persistence in MinIO
- [x] `StartShow` RPC — creates recording session, links to show
- [x] On session stop — transition show to ENDED, auto-create segments from act times
- [x] `RenderAll` RPC — queue all acts for segment rendering
- [x] `DistributeAll` RPC — share rendered segments to act emails
- [x] Shows list page (Upcoming / Past)
- [x] Show detail page — DRAFT (create/edit acts with times and emails)
- [x] Show detail page — ENDED (adjust times, render, distribute)
- [x] Show detail page — ARCHIVED (read-only summary)

### Phase 2 — Live Mode

- [x] Live timeline view during recording
- [x] Manual "Next Act" advance with actual timestamp capture
- [x] Auto-advance based on planned times (falls back to planned times for untouched acts)
- [x] Real-time progress indicator
- [x] `AdvanceAct` RPC

### Deferred

- [ ] Backend intelligence for automatic segmentation
- [ ] Import schedules from external sources (CSV, Pretalx, etc.)
- [ ] Waveform overlay with draggable boundary markers in post-show editing

## Affected Files

| Area | Changes Needed |
|------|----------------|
| `protocols/proto/show.proto` | New proto file with Show/Act messages and ShowService |
| `protocols/Makefile` | Add show.proto to generation targets |
| `go/cmd/chunk_sink/` | New show handler, wire into gRPC server |
| `go/storage/` | Show persistence in MinIO |
| `web/src/views/shows/` | New Shows list and detail pages |
| `web/src/grpc/` | Show gRPC client setup |
| `web/src/router.ts` | Add shows routes |
| `web/src/layout/` | Add Shows nav item |
| `grpc-web-proxy/` | Update Envoy config if new gRPC service port needed |
