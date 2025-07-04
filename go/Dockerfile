# Go Backend Services Dockerfile
FROM golang:1.23-alpine AS builder

# Install system dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go/go.mod go/go.sum ./

# Copy protocols directory for local dependency (needed before go mod download)
COPY protocols ../protocols

# Download dependencies
RUN go mod download

# Copy source code
COPY go/ .

# Update dependencies and download
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chunk_sink ./cmd/chunk_sink

# Audiowaveform builder stage
FROM alpine:latest AS audiowaveform-builder

# Install build dependencies for audiowaveform
RUN apk add --no-cache \
    git make cmake gcc g++ \
    libmad-dev libid3tag-dev libsndfile-dev \
    gd-dev boost-dev libgd libpng-dev zlib-dev \
    autoconf automake libtool gettext

# Build FLAC from source (required for audiowaveform)
WORKDIR /tmp
RUN wget https://github.com/xiph/flac/archive/1.3.3.tar.gz && \
    tar xzf 1.3.3.tar.gz && \
    cd flac-1.3.3 && \
    ./autogen.sh && \
    ./configure --enable-shared=no && \
    make && \
    make install

# Build audiowaveform from source
RUN git clone https://github.com/bbc/audiowaveform.git && \
    cd audiowaveform && \
    mkdir build && \
    cd build && \
    cmake -DENABLE_TESTS=0 .. && \
    make && \
    make install

# Final stage
FROM alpine:latest

# Install runtime dependencies for audio processing
RUN apk --no-cache add ca-certificates dbus sox \
    libmad libid3tag libsndfile gd libgd libpng zlib boost1.84-filesystem boost1.84-program_options

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/chunk_sink .

# Copy audiowaveform binary from audiowaveform-builder stage
COPY --from=audiowaveform-builder /usr/local/bin/audiowaveform /usr/local/bin/audiowaveform

# Expose ports
EXPOSE 8779 8780

# Command to run
CMD ["./chunk_sink"]