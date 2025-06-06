# Dependencies

To build the ptotobuf stuff, you need the following:

## For typescript:

```
npm install @protobuf-ts/runtime @protobuf-ts/runtime-rpc @protobuf-ts/grpcweb-transport
npm install --save-dev @protobuf-ts/plugin grpc-tools
npm install @protobuf-ts/grpcweb-transport
```

## For rust

```
cargo install grpc-compiler
cargo install protobuf-codegen
```

## For go and cpp

```
dnf install grpc-plugins golang-google-protobuf golang-google-grpc
```

## To use a specific version of `protoc`, go to the github page, download a release and set the variables accordingly:
```
export PROTOC_INCLUDES=~/Downloads/protoc-23.4-linux-x86_64/include/google/protobuf
export PROTOC=~/Downloads/protoc-23.4-linux-x86_64/bin/protoc
```
