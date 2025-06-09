# Protocol Generation Dockerfile
FROM node:18-alpine AS protocols

# Install protoc
RUN apk add --no-cache protobuf protobuf-dev

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy proto files
COPY proto/ proto/
COPY Makefile go.mod ./

# Generate protocol files
RUN make all

# This is a build-only container
# Generated files will be copied to other services during their builds