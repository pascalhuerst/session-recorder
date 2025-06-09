# C++ Client Build Dockerfile
FROM fedora:39

# Install build dependencies
RUN dnf update -y && dnf install -y \
    cmake \
    gcc-c++ \
    make \
    pkg-config \
    alsa-lib-devel \
    avahi-devel \
    grpc-devel \
    grpc-cpp \
    grpc-plugins \
    protobuf-devel \
    openssl-devel \
    boost-devel \
    c-ares-devel \
    re2-devel \
    zlib-devel \
    && dnf clean all

WORKDIR /app

# Copy proto source files instead of generated ones
COPY protocols/proto/ proto/

# Copy C++ source code
COPY cpp/chunk-sink-client/ chunk-sink-client/

# Generate protocol files using system protoc to ensure compatibility
RUN protoc --proto_path=proto --cpp_out=chunk-sink-client --grpc_out=chunk-sink-client --plugin=protoc-gen-grpc=/usr/bin/grpc_cpp_plugin common.proto
RUN protoc --proto_path=proto --cpp_out=chunk-sink-client --grpc_out=chunk-sink-client --plugin=protoc-gen-grpc=/usr/bin/grpc_cpp_plugin chunksink.proto
RUN protoc --proto_path=proto --cpp_out=chunk-sink-client --grpc_out=chunk-sink-client --plugin=protoc-gen-grpc=/usr/bin/grpc_cpp_plugin sessionsource.proto

# Fix CMakeLists.txt to use local generated files instead of the protocols/cpp path
RUN sed -i 's|file(GLOB GRPC_SOURCES.*|file(GLOB GRPC_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/*.cc")|' chunk-sink-client/CMakeLists.txt
RUN sed -i 's|include_directories.*protocols/cpp.*|include_directories("${CMAKE_CURRENT_SOURCE_DIR}")|' chunk-sink-client/CMakeLists.txt

# Build the application
WORKDIR /app/chunk-sink-client
RUN cmake . -Wno-dev && cmake --build .
RUN chmod +x chunk-sink-client

# This is a build-only container
