PROTOC ?= protoc
GRPC_CPP_PLUGIN ?= $(shell which grpc_cpp_plugin)
GRPC_TS_PLUGIN ?= ./node_modules/.bin/protoc-gen-ts_proto
MD ?= mkdir

PROTOS = \
	chunksource \
	sessionsource

all: create_dirs ${PROTOS}

create_dirs:
	$(MD) -p cpp
	$(MD) -p ts
	$(MD) -p go

chunksource:
	${PROTOC} \
		--grpc_out=generate_mock_code=false:./cpp \
		--cpp_out=./cpp \
		--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto


sessionsource:
	$(PROTOC) \
		--plugin=protoc-gen-ts_proto=$(GRPC_TS_PLUGIN) \
		--ts_proto_out=./ts \
		--ts_proto_opt=outputServices=nice-grpc,outputServices=generic-definitions,useExactTypes=false \
		--proto_path=proto \
		$@.proto

	$(MD) -p go/$@
	$(PROTOC) \
		--go_out=paths=source_relative:./go/$@ $@.proto \
		--go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
		--proto_path=proto \
		--go-grpc_out=./go/$@ $@.proto
	
clean:
	rm -rf cpp
	rm -rf ts
	rm -rf go