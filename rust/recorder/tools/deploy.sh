#!/usr/bin/env bash
# Builds the recorder for a Raspberry Pi (aarch64) on this machine and
# installs it on the device over ssh - instead of compiling on the Pi, where a
# release build takes the better part of an hour.
#
# The build runs in a Debian container of the same release as the device (so
# the same glibc and ALSA), with the arm64 cross toolchain. Then, on the
# device: the new binary replaces the old one (which is kept next to it), the
# tempo model is copied if this machine has one, and the recorder's systemd
# --user unit is restarted and checked.
#
# Usage: tools/deploy.sh [options] [ssh host]
#
#   --build-only          build, do not touch the device
#   --no-restart          install, leave the service running the old binary
#   --yes                 do not ask before restarting
#   --fetch-tempo-model   get the tempo model first if this machine has none
#                         (tools/fetch-tempo-model.sh)
#   -h, --help            show this text
#
# Settings, from the environment or from tools/deploy.env (not in git - put
# your device there once):
#
#   DEPLOY_HOST      ssh host of the device                  (or the argument)
#   DEPLOY_BINARY    where the binary lives there, relative to the login
#                    user's home or absolute   [.local/bin/session-recorder-alsa-reader]
#   DEPLOY_SERVICE   the recorder's systemd --user unit      [session-recorder-alsa-reader.service]
#   DEBIAN_RELEASE   the device's Debian release             [trixie]
#   RUST_VERSION     toolchain to build with                 [1.85.1]
#   TEMPO_MODEL      local tempo model to copy  [$XDG_DATA_HOME/session-recorder/deeptemp-k16-3.onnx]

set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../../.." && pwd)"
[[ -f "$here/deploy.env" ]] && source "$here/deploy.env"

DEPLOY_HOST="${DEPLOY_HOST:-}"
DEPLOY_BINARY="${DEPLOY_BINARY:-.local/bin/session-recorder-alsa-reader}"
DEPLOY_SERVICE="${DEPLOY_SERVICE:-session-recorder-alsa-reader.service}"
DEBIAN_RELEASE="${DEBIAN_RELEASE:-trixie}"
RUST_VERSION="${RUST_VERSION:-1.85.1}"
TEMPO_MODEL="${TEMPO_MODEL:-${XDG_DATA_HOME:-$HOME/.local/share}/session-recorder/deeptemp-k16-3.onnx}"
TARGET=aarch64-unknown-linux-gnu
# Bump the revision when the Dockerfile below changes.
IMAGE="session-recorder-cross:${DEBIAN_RELEASE}-rust${RUST_VERSION}-r2"
# Sources are copied here first (Docker cannot always read the checkout, e.g.
# on NFS); the build directory and cargo's registry stay between runs.
CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/session-recorder-cross"

build_only=false restart=true assume_yes=false fetch_model=false
usage() { sed -n '2,/^$/s/^# \{0,1\}//p' "${BASH_SOURCE[0]}"; }
for arg in "$@"; do
    case "$arg" in
        --build-only) build_only=true ;;
        --no-restart) restart=false ;;
        --yes) assume_yes=true ;;
        --fetch-tempo-model) fetch_model=true ;;
        -h | --help) usage; exit 0 ;;
        -*) echo "error: unknown option $arg" >&2; exit 2 ;;
        *) DEPLOY_HOST="$arg" ;;
    esac
done
if ! $build_only && [[ -z "$DEPLOY_HOST" ]]; then
    echo "error: no device - give an ssh host, or set DEPLOY_HOST (tools/deploy.env)" >&2
    exit 2
fi
command -v docker > /dev/null || { echo "error: docker is needed for the cross build" >&2; exit 1; }

cd "$repo"
echo "==> Recorder from $(git branch --show-current 2> /dev/null || echo '?') @ $(git rev-parse --short HEAD 2> /dev/null || echo '?') for $TARGET (Debian $DEBIAN_RELEASE, rustc $RUST_VERSION)"
if [[ -n "$(git status --porcelain --untracked-files=no -- rust protocols 2> /dev/null)" ]]; then
    echo "    note: rust/ or protocols/ have uncommitted changes - they are part of this build."
fi

if ! docker image inspect "$IMAGE" > /dev/null 2>&1; then
    echo "==> Building the cross toolchain image (once)"
    docker build -t "$IMAGE" --build-arg DEBIAN_RELEASE="$DEBIAN_RELEASE" --build-arg RUST_VERSION="$RUST_VERSION" - << 'DOCKERFILE'
ARG DEBIAN_RELEASE
FROM debian:${DEBIAN_RELEASE}
ARG RUST_VERSION
RUN dpkg --add-architecture arm64 \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl build-essential pkg-config \
        protobuf-compiler libprotobuf-dev \
        gcc-aarch64-linux-gnu libc6-dev-arm64-cross libasound2-dev:arm64 \
    && rm -rf /var/lib/apt/lists/*
ENV RUSTUP_HOME=/usr/local/rustup CARGO_HOME=/usr/local/cargo PATH=/usr/local/cargo/bin:$PATH
RUN curl -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal --default-toolchain "$RUST_VERSION" \
        --target aarch64-unknown-linux-gnu \
    && chmod -R a+rX /usr/local/rustup /usr/local/cargo
ENV CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc \
    PKG_CONFIG_ALLOW_CROSS=1 \
    PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig
DOCKERFILE
fi

mkdir -p "$CACHE/src" "$CACHE/target" "$CACHE/cargo-home"
find "$CACHE/src" -mindepth 1 -delete
tar --exclude='target' --exclude='node_modules' -c rust/recorder protocols/proto | tar -x -C "$CACHE/src"

echo "==> Compiling"
docker run --rm --user "$(id -u):$(id -g)" \
    -v "$CACHE/src":/src:ro -v "$CACHE/target":/target -v "$CACHE/cargo-home":/cargo-home \
    -e CARGO_HOME=/cargo-home -e CARGO_TARGET_DIR=/target -e HOME=/tmp \
    -w /src/rust/recorder "$IMAGE" \
    cargo build --release --locked --target "$TARGET"

binary="$CACHE/target/$TARGET/release/recorder"
file "$binary" | grep -q 'ARM aarch64' || { echo "error: $binary is not an aarch64 binary" >&2; exit 1; }
echo "    $binary ($(($(wc -c < "$binary") / 1000000)) MB)"

if $build_only; then
    echo "==> Built (--build-only: device not touched)"
    exit 0
fi

if [[ ! -f "$TEMPO_MODEL" ]] && $fetch_model; then
    "$here/fetch-tempo-model.sh" "$TEMPO_MODEL"
fi

stamp="$(date +%Y%m%d-%H%M%S)"
echo "==> Installing on $DEPLOY_HOST as $DEPLOY_BINARY"
scp -q "$binary" "$DEPLOY_HOST:recorder.new-$stamp"
# Renamed into place, not overwritten: the running process keeps its old file
# and the service never sees a half-written binary. The previous binary stays
# as <binary>.previous (one generation).
ssh "$DEPLOY_HOST" "set -e
    chmod +x recorder.new-$stamp
    ./recorder.new-$stamp --help > /dev/null   # must at least start on this machine
    mkdir -p \"\$(dirname '$DEPLOY_BINARY')\"
    if [ -f '$DEPLOY_BINARY' ]; then cp -p '$DEPLOY_BINARY' '$DEPLOY_BINARY.previous'; fi
    mv recorder.new-$stamp '$DEPLOY_BINARY'"

if [[ -f "$TEMPO_MODEL" ]]; then
    echo "==> Tempo model -> $DEPLOY_HOST:.local/share/session-recorder/"
    ssh "$DEPLOY_HOST" 'mkdir -p .local/share/session-recorder'
    scp -q "$TEMPO_MODEL" "$DEPLOY_HOST:.local/share/session-recorder/deeptemp-k16-3.onnx"
else
    echo "    no tempo model at $TEMPO_MODEL - the recorder will run without live tempo"
    echo "    (tools/fetch-tempo-model.sh gets it, or pass --fetch-tempo-model)"
fi

if ! $restart; then
    echo "==> Installed (--no-restart: the service keeps running the old binary until restarted)"
    exit 0
fi

journal="journalctl _SYSTEMD_USER_UNIT=$DEPLOY_SERVICE --no-pager"
if ! $assume_yes; then
    if ssh "$DEPLOY_HOST" "$journal -n 3 | grep -q 'recording=yes'"; then
        echo "==> The recorder is recording right now; a restart cuts that recording."
    fi
    read -r -p "==> Restart $DEPLOY_SERVICE on $DEPLOY_HOST now? [y/N] " answer
    [[ "$answer" == [yYjJ] ]] || { echo "    Not restarted. Later: ssh $DEPLOY_HOST systemctl --user restart $DEPLOY_SERVICE"; exit 0; }
fi

# The recorder finds its backend over mDNS at start, and now and then misses
# it; a second start has always found it.
for try in 1 2 3; do
    ssh "$DEPLOY_HOST" "systemctl --user restart $DEPLOY_SERVICE && sleep 9"
    if ssh "$DEPLOY_HOST" "$journal --since '-12s' | grep -q 'Connected to service'"; then
        echo "==> Running and connected to its backend:"
        ssh "$DEPLOY_HOST" "$journal --since '-14s' | grep -E 'Live (fingerprint|tempo)|Connected to service' | sed 's/^.*recorder\] /    /'"
        exit 0
    fi
    echo "    start $try: no backend found yet"
done
echo "error: the recorder runs but has not connected to a backend - check: ssh $DEPLOY_HOST $journal -n 30" >&2
exit 1
