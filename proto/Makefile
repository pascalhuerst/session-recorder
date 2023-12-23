PROTOC ?= protoc
GRPC_CPP_PLUGIN ?= $(shell which grpc_cpp_plugin)
GRPC_TS_PLUGIN ?= $(shell which protoc-gen-ts)
MD ?= mkdir

PROTOS = \
	audiosource

all: ${PROTOS}

chunksource:
	${PROTOC} --go-grpc_out=paths=source_relative:./go $@.proto
	${PROTOC} --grpc_out=generate_mock_code=false:./cpp \
			  --cpp_out=./cpp \
			  --plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
			  --proto_path=. \
			  $@.proto
	$(PROTOC) --plugin=protoc-gen-ts=${GRPC_TS_PLUGIN} \
			  --proto_path=. \
			  --ts_out="service=grpc-web:./ts" \
  			  --js_out="import_style=commonjs,binary:./ts" \
			  $@.proto

audiosource:
	${PROTOC} --go_out=paths=source_relative:../$@ $@.proto \
			  --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
			  --go-grpc_out=../$@ $@.proto
	${MD} -p ./cpp/$@
	${PROTOC} --grpc_out=generate_mock_code=false:./cpp/$@ \
			  --cpp_out=./cpp/$@ \
			  --plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN} \
			  --proto_path=. \
			  $@.proto	

sessionsource:
	${PROTOC} --go_out=paths=source_relative:../$@ $@.proto \
			  --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
			  --go-grpc_out=../$@ $@.proto
	$(PROTOC) --plugin=protoc-gen-ts=${GRPC_TS_PLUGIN} \
			  --proto_path=. \
			  --ts_out="service=grpc-web:./ts" \
			  --js_out="import_style=commonjs,binary:./ts" \
			  $@.proto

clean:
	rm -f cpp/*
	rm -f ts/*
