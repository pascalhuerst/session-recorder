# Go Backend Services Dockerfile
FROM golang:1.23-alpine AS builder

# Install system dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go/go.mod go/go.sum ./

# Copy protocols directory for local dependency
COPY protocols ./protocols

# Copy source code
COPY go/ .

# Update dependencies and download
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chunk_sink ./cmd/chunk_sink

# Final stage
FROM alpine:latest

# Install runtime dependencies for audio processing
RUN apk --no-cache add ca-certificates dbus sox \
    libmad libid3tag libsndfile gd libgd libpng zlib boost1.84-filesystem boost1.84-program_options

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/chunk_sink .

# Expose ports
EXPOSE 8779 8780

# Command to run
CMD ["./chunk_sink"]