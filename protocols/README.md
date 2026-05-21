# Protocols

Protocol Buffer definitions for the services exposed by the Go backend, plus
the generated Go, TypeScript, and C++ stubs.

```
protocols/
├── proto/                     # source .proto files
│   ├── common.proto           # shared types (SignalStatus, RecorderStatus, Response)
│   ├── chunksink.proto        # ChunkSink service (client → backend)
│   └── sessionsource.proto    # SessionSource service (web UI ↔ backend)
├── go/                        # generated Go stubs (committed)
├── ts/                        # generated TypeScript stubs (committed)
├── Makefile                   # regen targets
└── Dockerfile                 # used by docker-compose to regenerate stubs in CI
```

## Regenerating stubs

```bash
cd protocols
make all                       # ts + go (+ gomod)
make ts                        # just TypeScript
make go                        # just Go
make clean                     # remove all generated dirs
```

The default `make all` writes to `ts/` and `go/` next to the
`Makefile`. The generated Go subpackage gets its own `go.mod` copied in
(via the `gomod` target) so it can be consumed as a Go module by the
backend and the test clients.

## Toolchain

### TypeScript

```
pnpm install                   # picks up protoc-gen-ts from devDependencies
```

(Or the npm equivalent: `npm install @protobuf-ts/runtime @protobuf-ts/runtime-rpc @protobuf-ts/grpcweb-transport && npm install --save-dev @protobuf-ts/plugin grpc-tools`.)

### Go and C++

On Fedora:

```bash
dnf install grpc-plugins golang-google-protobuf golang-google-grpc
```

### Pinning `protoc`

If the distro's `protoc` is too old, grab a release from the
[protobuf GitHub releases](https://github.com/protocolbuffers/protobuf/releases)
and point the Makefile at it:

```bash
export PROTOC=~/Downloads/protoc-23.4-linux-x86_64/bin/protoc
export PROTOC_INCLUDES=~/Downloads/protoc-23.4-linux-x86_64/include/google/protobuf
make all
```

## Rust stubs

There is no Rust target in the Makefile (yet). The Rust recorder client in
`rust/chunk-source/` generates its own stubs at build time via
`tonic-build` in its `build.rs`. The check-in copies under
`rust/grpc_test/src/` are stale demo files (referenced by `TODO.md`).

## Known issues

Two message types in `proto/` have typos that have leaked into every
generated stub. They are *not* fixed yet because renaming touches all four
languages at once. See `../TODO.md` for the full call-site inventory and
suggested process:

- `common.proto:23` — `Respone` → `Response`
- `sessionsource.proto:13` — `RecordereRemoved` → `RecorderRemoved`
