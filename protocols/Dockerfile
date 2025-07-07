# Protocol Generation Dockerfile
FROM node:18-alpine AS protocols

# Install protoc, make, go and grpc plugins
RUN apk add --no-cache protobuf protobuf-dev make go grpc grpc-dev

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Install Go protobuf plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add Go bin to PATH
ENV PATH="/root/go/bin:${PATH}"

# Copy proto files
COPY proto/ proto/
COPY Makefile go.mod ./

# Generate protocol files
RUN make all

# This is a build-only container
# Generated files will be copied to other services during their builds