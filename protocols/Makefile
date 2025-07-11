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

# Default target builds everything
all: cpp ts go gomod

# Language-specific targets
cpp: create_cpp_dir $(PROTOS:%=%-cpp)
ts: create_ts_dir $(PROTOS:%=%-ts)
go: create_go_dir $(PROTOS:%=%-go)

# Create directories
create_dirs: create_cpp_dir create_ts_dir create_go_dir

create_cpp_dir:
	$(MD) -p cpp

create_ts_dir:
	$(MD) -p ts

create_go_dir:
	$(MD) -p go

# C++ targets
common-cpp:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		common.proto

chunksink-cpp:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		chunksink.proto

sessionsource-cpp:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		sessionsource.proto

# TypeScript targets
common-ts:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--ts_out=./ts \
		--ts_opt=generate_dependencies \
		--ts_opt=long_type_string \
		--plugin=protoc-gen-ts=$(PROTOBUF_TS_PLUGIN) \
		--proto_path=proto \
		common.proto

chunksink-ts:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--ts_out=./ts \
		--ts_opt=generate_dependencies \
		--ts_opt=long_type_string \
		--plugin=protoc-gen-ts=$(PROTOBUF_TS_PLUGIN) \
		--proto_path=proto \
		chunksink.proto

sessionsource-ts:
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--ts_out=./ts \
		--ts_opt=generate_dependencies \
		--ts_opt=long_type_string \
		--ts_opt=client_generic \
		--plugin=protoc-gen-ts=$(PROTOBUF_TS_PLUGIN) \
		--proto_path=proto \
		sessionsource.proto

# Go targets
common-go:
	$(MD) -p go/common
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/common common.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/common common.proto

chunksink-go:
	$(MD) -p go/chunksink
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/chunksink chunksink.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/chunksink chunksink.proto

sessionsource-go:
	$(MD) -p go/sessionsource
	$(PROTOC) \
		--proto_path=${PROTOC_INCLUDES} \
		--go_out=paths=source_relative:./go/sessionsource sessionsource.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/sessionsource sessionsource.proto

# Copy go.mod for Go modules
gomod:
	cp go.mod ./go

# Legacy targets for backward compatibility
common: common-cpp common-ts common-go
chunksink: chunksink-cpp chunksink-go
sessionsource: sessionsource-ts sessionsource-go

clean:
	rm -rf cpp
	rm -rf ts
	rm -rf go

.PHONY: all cpp ts go create_dirs create_cpp_dir create_ts_dir create_go_dir gomod clean common chunksink sessionsource
