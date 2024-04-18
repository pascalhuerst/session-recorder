# Chunk Sink Client, a.k.a Chunk Source

This is part of sesion-recorder.

## What does it do?

This opens an alsa device and detects whether there is something to record or not. If signal is detected, chunk-source starts pushing chunks of audio to chunk-sinks. Chunk-sinks are found using mdns.

## Building

### Fedora

Dependencies:

```
dnf install alsa-lib-devel avahi-devel grpc-data grpc grpc-cpp grpc-plugins grpc-devel
```
