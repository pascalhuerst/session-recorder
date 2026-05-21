# session_recorder_server

The Session Recorder backend. A single binary that hosts the **ChunkSink** gRPC
service (receives audio chunks from recorders), the **SessionSource** gRPC service
(serves sessions/recorders to clients), and an embedded gRPC-Web HTTP listener for
the web UI (no separate proxy container needed).

## Development setup

To store chunks and create sessions you need a MinIO server (or use the
`--storage-fs-root` flag for a local-filesystem backend — see `../../README.md`):

```
docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   --user $(id -u):$(id -g) \
   --name minio1 \
   -e "MINIO_ROOT_USER=paso" \
   -e "MINIO_ROOT_PASSWORD=hnw4main" \
   -v ${HOME}/minio/data:/data \
   quay.io/minio/minio server /data --console-address ":9090"
```

## Run

```
go run ./cmd/session_recorder_server
```

Run with `-h` for the full flag list. The web UI talks to the embedded gRPC-Web
listener directly; the recorder and the Go SessionSource test client talk to the
gRPC services.
