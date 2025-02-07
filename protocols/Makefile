PROTOC ?= protoc
PROTOC_INCLUDES ?= /usr/include/google/protobuf
GRPC_CPP_PLUGIN ?= $(shell which grpc_cpp_plugin)
GRPC_TS_PLUGIN ?= ./node_modules/.bin/protoc-gen-ts_proto
MD ?= mkdir
CP ?= cp

PROTOS = \
	common \
	chunksink \
	sessionsource

all: create_dirs ${PROTOS} gomod

gomod:
	cp go.mod ./go

create_dirs:
	$(MD) -p cpp
	$(MD) -p ts
	$(MD) -p go
	$(MD) -p rust

chunksink:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto

	## Rust
	$(MD) -p rust/$@

	$(PROTOC) \
	    --proto_path=${PROTOC_INCLUDES}:./proto \
		--rs_out=./rust/$@ \
        --rust-grpc_out=./rust/$@ \
        proto/$@.proto

sessionsource:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--plugin=protoc-gen-ts_proto=$(GRPC_TS_PLUGIN) \
		--ts_proto_out=./ts \
		--ts_proto_opt=outputServices=nice-grpc,outputServices=generic-definitions,useExactTypes=false,esModuleInterop=true \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto

	## Rust
	$(MD) -p rust/$@

	$(PROTOC) \
	    --proto_path=${PROTOC_INCLUDES}:./proto \
		--rs_out=./rust/$@ \
        --rust-grpc_out=./rust/$@ \
        proto/$@.proto

common:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--plugin=protoc-gen-ts_proto=$(GRPC_TS_PLUGIN) \
		--ts_proto_out=./ts \
		--ts_proto_opt=outputServices=nice-grpc,outputServices=generic-definitions,useExactTypes=false,esModuleInterop=true \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto

	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		$@.proto

	## Rust
	$(MD) -p rust/$@

	$(PROTOC) \
	    --proto_path=${PROTOC_INCLUDES}:./proto \
        --rs_out=./rust/$@ \
        --rust-grpc_out=./rust/$@ \
        proto/$@.proto


clean:
	rm -rf cpp
	rm -rf ts
	rm -rf go
	rm -rf rust
