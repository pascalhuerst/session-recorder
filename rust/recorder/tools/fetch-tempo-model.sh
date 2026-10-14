#!/usr/bin/env bash
# Gets the TempoCNN model the recorder estimates the live tempo with, in the
# form it needs: downloads Essentia's TensorFlow graph and converts it to an
# ONNX file with a fixed input size (see "Tempo model" in the README for why).
#
# The model is not part of this repository and has its own license:
#   - TempoCNN: Hendrik Schreiber and Meinard Müller (ISMIR 2018, SMC 2019),
#     https://github.com/hendriks73/tempo-cnn, AGPL-3.0
#   - this graph: prepared for Essentia by Pablo Alonso, Music Technology
#     Group (UPF), https://essentia.upf.edu/models/tempo/tempocnn/ (AGPL-3.0
#     LICENSE there; https://essentia.upf.edu/models/LICENSE: CC BY-NC-SA 4.0
#     for the MTG's model files, derived from Schreiber's AGPL-3.0 work)
# Fetching and running it yourself is fine. Passing the file on, converted or
# not, is distribution under those licenses - see "Source and license of the
# model" in the README. Thanks to the authors for publishing it.
#
# Needs curl and uv (https://docs.astral.sh/uv/); the conversion downloads
# TensorFlow into uv's cache the first time (several hundred MB). Run it on
# any machine - the result is one small file, good for every architecture.
#
# Usage: fetch-tempo-model.sh [output file]
#   default output: $XDG_DATA_HOME/session-recorder/deeptemp-k16-3.onnx
#   (~/.local/share/...), where the recorder looks for it by itself.

set -euo pipefail

NAME=deeptemp-k16-3
URL="https://essentia.upf.edu/models/tempo/tempocnn/$NAME.pb"
OUT="${1:-${XDG_DATA_HOME:-$HOME/.local/share}/session-recorder/$NAME.onnx}"

for tool in curl uv; do
    command -v "$tool" > /dev/null || { echo "error: $tool is needed" >&2; exit 1; }
done

work="$(mktemp -d)"
cleanup() { rm -f "$work/$NAME.pb" "$work/fixed.onnx" "$work/$NAME.onnx"; rmdir "$work" 2> /dev/null || true; }
trap cleanup EXIT

echo "==> Downloading $URL"
curl -fsSL -o "$work/$NAME.pb" "$URL"

echo "==> Converting to ONNX (input fixed to [1, 40 mel bands, 256 frames, 1])"
uv run --quiet --no-project --python 3.10 \
    --with "tensorflow-cpu==2.13.1" --with "tf2onnx==1.16.1" --with "numpy<2" --with "protobuf<4" --with "onnx<1.17" \
    python -m tf2onnx.convert --graphdef "$work/$NAME.pb" \
    --inputs "input:0[1,40,256,1]" --outputs output:0 --opset 13 --output "$work/fixed.onnx" > "$work/convert.log" 2>&1 \
    || { tail -20 "$work/convert.log" >&2; rm -f "$work/convert.log"; exit 1; }
rm -f "$work/convert.log"

echo "==> Simplifying (folds the shape arithmetic the recorder's engine cannot run)"
uv run --quiet --no-project --python 3.10 \
    --with "numpy<2" --with "onnx<1.17" --with onnxsim --with "onnxruntime<1.20" \
    python -m onnxsim "$work/fixed.onnx" "$work/$NAME.onnx" > /dev/null

mkdir -p "$(dirname "$OUT")"
mv "$work/$NAME.onnx" "$OUT"
echo "==> $OUT ($(wc -c < "$OUT") bytes)"
echo "    The recorder finds it there by itself; elsewhere, pass --tempo-model <file>."
