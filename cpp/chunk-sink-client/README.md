# Chunk Sink Client, a.k.a Chunk Source

This is part of sesion-recorder.

## What does it do?

This opens an alsa device and detects whether there is something to record or not. If signal is detected, chunk-source starts pushing chunks of audio to chunk-sinks. Chunk-sinks are found using mdns.

## Building

### Raspberry

Build on RPi:

```
sudo apt-get install git cmake git g++ libboost-program-options-dev libavahi-client-dev libavahi-core-dev libgrpc++-dev libprotobuf-dev libasound2-dev
