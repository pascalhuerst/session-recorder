# Go Backend Services Dockerfile
FROM golang:1.24-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o session_recorder_server ./cmd/session_recorder_server

# Final stage
FROM alpine:latest

# Install runtime dependencies. sox encodes data.flac / data.ogg; waveform.dat
# is generated in-process (pure Go), so audiowaveform is no longer needed.
RUN apk --no-cache add ca-certificates dbus sox libsndfile

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/session_recorder_server .

# Expose ports
EXPOSE 8779 8780

# Command to run
CMD ["./session_recorder_server"]
