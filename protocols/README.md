# Dependencies

To build the ptotobuf stuff, you need the following:

## For typescript:

```
npm install nice-grpc
npm install protobufjs long
npm install --save-dev grpc-tools ts-proto
```

## For go and cpp

```
dnf install grpc-plugins
```

## To use a specific version of `protoc`, go to the github page, download a release and set the variables accordingly:
```
export PROTOC_INCLUDES=~/Downloads/protoc-23.4-linux-x86_64/include/google/protobuf
export PROTOC=~/Downloads/protoc-23.4-linux-x86_64/bin/protoc
```