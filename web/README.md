# Session Recorder Web

## Development

### Libraries

#### Session Waveform (Player App)

The easiest way to work on the Player app is by launching Storybook:

```
npx nx storybook --project session-waveform
```

### Application

#### Dev Mode

To run application in dev mode, go to `/web` directory:

##### Run `npm install`

##### Start grpcwebproxy!

Replace `--server-bind-address` with your machine's IP (run `ip a` to find it)
Replace `--backend-addr` with the IP of the server running the chunk_sink

If running the chunk sink server and the web locally, use `127.0.0.1:8780` as backend address and `127.0.0.1` as bind
address.

```shell
grpcwebproxy --server_tls_cert_file=${GOPATH}/src/github.com/improbable-eng/grpc-web/misc/localhost.crt --server_tls_key_file=/${GOPATH}/src/github.com/improbable-eng/grpc-web/misc/localhost.key --backend_addr=127.0.0.1:8780 --backend_tls_noverify --allow_all_origins --server_bind_address=127.0.0.1 --server_http_max_read_timeout=0 --server_http_max_write_timeout=0
```

##### Setup .env

- create `.env` using `.env.example` as a reference

##### Start the app

- run npm `npm start`
