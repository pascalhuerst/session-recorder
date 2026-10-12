# recorder

The **recorder** is the device-side component of the Session Recorder system. It
reads audio from a soundcard (ALSA), detects when a signal is present, and streams
the captured audio to the backend.

## What it is

- Captures stereo audio from an ALSA device (48 kHz, 16-bit).
- Runs a level detector (RMS in dB with attack/release smoothing) and only records
  while a signal is present, opening a new session on each silence→signal transition.
- Buffers ready-to-send chunks in a bounded, time-based outbox so a backend outage
  (restart, network blip) doesn't lose audio.
- Discovers backends on the local network via mDNS and streams to every one it finds.
- Optional hardware I/O: status/upload LEDs (Linux sysfs) and a physical "cut session"
  button (`/dev/input/eventNN`).
- Listens for server commands (e.g. remote "cut session") over a streaming RPC.

## Protocol relationship

The recorder **implements a client for the chunksink protocol**.

Protocol names are always written from the **backend's** perspective:

- The backend hosts the **ChunkSink** service — it is the *sink* that receives audio
  chunks. See [`protocols/proto/chunksink.proto`](../../protocols/proto/chunksink.proto).
- The recorder is the *source* of those chunks, i.e. a **ChunkSink client**.

The terms **"chunk source"** and **"chunk sink client"** are equivalent and are used
**only** inside the protocol definitions and the generated stubs (e.g. tonic's
`ChunkSinkClient`). Everywhere else — the binary, the crate, the docs — this component
is called the **recorder**.

## Build

```bash
cd rust/recorder
cargo build --release
```

The protobuf stubs are generated at build time from `../../protocols/proto`
(`chunksink.proto`, `common.proto`) via `build.rs`.

## Run

```bash
./target/release/recorder \
  --recorder-id $(uuidgen) \
  --recorder-name "Living Room" \
  --device default
```

The recorder discovers backends advertising `_session-recorder-chunksink._tcp` over
mDNS and connects automatically. No backend address is configured by hand.

### Options

| Flag | Purpose | Default |
|---|---|---|
| `--recorder-id <uuid>` | Stable unique id of this recorder (required) | — |
| `--recorder-name <name>` | Human-readable name (required) | — |
| `--device <alsa>` | ALSA capture device | `default` |
| `--period-size <frames>` | ALSA period size | `512` |
| `--buffer-size <frames>` | ALSA buffer size | `2048` |
| `--detector-threshold-db <db>` | RMS dB-FS above which a signal is "present" | `-45.0` |
| `--window-time <s>` | RMS analysis window length | `0.25` |
| `--attack-time <s>` | silence → signal transition time | `5.0` |
| `--release-time <s>` | signal → silence transition time | `30.0` |
| `--send-buffer-secs <s>` | seconds of chunks to buffer when no backend is reachable (drops oldest when full) | `120.0` |
| `--led-rec-state <sysfs>` | LED on while recording, blinks on new session | — |
| `--led-upload <sysfs>` | LED pulses on each successful chunk upload | — |
| `--input-event <N>` `--input-keycode <K>` `--input-hold-ms <ms>` | local "cut session" button on `/dev/input/eventN` | — |
| `--fingerprint <true\|false>` | master switch for the live analysis: every 12 s the last 25 s of the running recording are analysed and the results published in the recorder status. `false` turns all of it off; recording is the same either way. Each part below has its own switch | `true` |
| `--chromaprint <true\|false>` | the Chromaprint fingerprint of each window (at the speed it was captured). `false` leaves it out, and the speed variants with it | `true` |
| `--fingerprint-speed-range <percent>` | also fingerprint each window for assumed playback speeds from -N to +N % (e.g. `8` for a turntable's pitch fader); `0` = off. One after another on one thread (~0.23 s each on a Pi 3), closest speeds first, each published as soon as it is done | `0` |
| `--fingerprint-speed-step <percent>` | distance between two assumed speeds | `1` |
| `--peak-constellation <true\|false>` | also publish each window's spectral peaks, a speed-invariant fingerprint (a client matches a pitched record with it and learns the speed); about as cheap as one plain fingerprint | `false` |
| `--fingerprint-spread <true\|false>` | pace a window's fingerprint computations evenly until the next window instead of one burst: level load, but the last speeds arrive up to ~11 s after the window instead of ~4 s | `false` |
| `--tempo <true\|false>` | estimate each window's tempo (BPM) with TempoCNN and publish it in the recorder status, if the model file is found (see "Tempo model" below); without the file the recorder runs without tempo. About 0.3 s per window on a Pi 3 | `true` |
| `--tempo-model <file.onnx>` | the model file, if it is not in one of the places looked in: `$XDG_DATA_HOME/session-recorder/deeptemp-k16-3.onnx` (`~/.local/share/...`), `/usr/local/share/session-recorder/...`, `/usr/share/session-recorder/...`. Given explicitly, it must load or the recorder does not start | — |

### Live analysis is optional

Chromaprint, its speed variants, the peak constellation and the tempo are four independent extras on top
of recording, each with its own switch (above). Whatever is off is simply not computed and not sent; the
status tells clients which parts are on (`chromaprintEnabled`, `chromaprintSpeedCount`,
`peakConstellationEnabled`, `tempoEnabled`). Neither the backend nor recording depends on any of them:
the backend relays the status as it comes, and a recorder with everything off behaves exactly like one
from before these features.

### Tempo model

The live tempo needs TempoCNN (Schreiber & Müller 2018) as an ONNX file with a fixed input of
`[1, 40, 256, 1]`. The model is not part of this repository - it is published by the Music Technology
Group (UPF) under <https://essentia.upf.edu/models/tempo/tempocnn/>; mind its license before passing the
file on. One command gets and converts it:

```bash
tools/fetch-tempo-model.sh            # -> ~/.local/share/session-recorder/deeptemp-k16-3.onnx
```

It needs `curl` and [`uv`](https://docs.astral.sh/uv/) and downloads TensorFlow into uv's cache the first
time (the conversion uses `tf2onnx` and `onnxsim`). Run it on any machine: the result is one 1.3 MB file,
the same for every architecture. The recorder picks it up from the places listed under `--tempo-model`;
`tools/deploy.sh` copies it to the device.

The fixed input size and the simplification matter: without them the graph keeps shape computations
that `tract`, the pure-Rust engine the recorder runs the model with, cannot execute. The audio side
(mel bands, patches, normalization) is in `src/tempo.rs` and follows Essentia's
`TensorflowPredictTempoCNN`; on 250 excerpts of 25 s the recorder's path gave the same BPM as Essentia
in every one. `cargo run --release --example tempo_dump -- <model> <rate> <channels> <raw s16le files>`
runs it on files; `TEMPO_MODEL_TEST=<model> cargo test` includes the test that needs the model.

### Deploying to a Raspberry Pi

`tools/deploy.sh` builds the recorder for aarch64 on this machine (in a Debian container of the device's
release, with the cross toolchain - a release build on a Pi 3 takes the better part of an hour) and
installs it on the device over ssh:

```bash
tools/fetch-tempo-model.sh        # once, if the live tempo is wanted
tools/deploy.sh mypi.local        # build, copy binary and model, restart the unit, check it connected
```

The device, the binary's place there and the name of its systemd `--user` unit come from the
environment or from `tools/deploy.env` (not in git): `DEPLOY_HOST`, `DEPLOY_BINARY`
(default `.local/bin/session-recorder-alsa-reader`), `DEPLOY_SERVICE`
(default `session-recorder-alsa-reader.service`); `DEBIAN_RELEASE` (default `trixie`) and `RUST_VERSION`
(default `1.85.1`, what trixie ships - the code and `Cargo.lock` are kept buildable with it) pick the
build container. `--build-only`, `--no-restart`, `--yes` and `--fetch-tempo-model` do what they say;
`tools/deploy.sh --help` has the details. The previous binary is kept next to the new one as
`<binary>.previous`. The script only replaces the binary and the model: the unit file and its options
(recorder id and name, `--fingerprint-speed-range`, ...) are set up once by hand or by the `.deb`
(`packaging/`). It needs Docker and ssh access to the device.

### Logging

```bash
# info everywhere, silence the chatty mdns_sd crate (the default)
RUST_LOG=info ./target/release/recorder ...

# verbose recorder logs
RUST_LOG=recorder=debug,mdns_sd=warn ./target/release/recorder ...
```

## Architecture

```
 ALSA capture ──► capture ring ──► drain task ──► detector / windowing
 (callback thread)                                      │
                                                        ▼ (while recording)
                                                   send-buffer outbox
                                                        │
                                                        ▼
 mDNS discovery ──► gRPC clients ◄──── chunk sender task (drains when ≥1 backend)
                         ▲
                         └── command listener (GetCommands stream: remote cut)
```

- **Audio callback thread** (RT-pinned) pushes samples into a lock-free ring.
- **Drain task** moves samples off the ring so a slow network send can never stall
  the soundcard.
- **Audio processing task** runs the detector and enqueues finished chunks.
- **Chunk sender task** drains the outbox to every connected backend.
- **Discovery task** keeps the set of connected backends in sync with mDNS.

## Requirements

- Linux with ALSA.
- avahi-daemon running (for mDNS discovery).
- Membership in the `audio` group (or equivalent) to open the capture device.
