# Development

## Server setup:

To be able to store chunks and create sessions, you need a minio server:

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

Currently `go run cmd/chunk_sink/main.go` runs a chunk sink server and a session source server, so we should probably
rename it soon. This works fine for the chunk sink client (=the recoder) and the go implementation of the session source
client, but to run it from a web client, we need to run a proxy:

```
grpcwebproxy
    --server_tls_cert_file=${GOPATH}/src/github.com/improbable-eng/grpc-web/misc/localhost.crt \ 
    --server_tls_key_file=/${GOPATH}/src/github.com/improbable-eng/grpc-web/misc/localhost.key \
    --backend_addr=<your-ip>:8780 \
    --backend_tls_noverify \
    --allow_all_origins \
    --server_bind_address=<your-ip>
```
