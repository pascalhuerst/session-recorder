PROTOC ?= protoc
PROTOC_INCLUDES ?= /usr/include/google/protobuf
GRPC_CPP_PLUGIN ?= $(shell which grpc_cpp_plugin)
PROTOBUF_TS_PLUGIN ?= ./node_modules/.bin/protoc-gen-ts
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

chunksink:
	${PROTOC} \
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


sessionsource:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--ts_out=./ts \
		--ts_opt=generate_dependencies \
		--ts_opt=long_type_string \
		--ts_opt=client_generic \
		--plugin=protoc-gen-ts=$(PROTOBUF_TS_PLUGIN) \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto

common:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--ts_out=./ts \
		--ts_opt=generate_dependencies \
		--ts_opt=long_type_string \
		--plugin=protoc-gen-ts=$(PROTOBUF_TS_PLUGIN) \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto

	${PROTOC} \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		$@.proto

clean:
	rm -rf cpp
	rm -rf ts
	rm -rf go
