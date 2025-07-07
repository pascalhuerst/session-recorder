# Web Interface Dockerfile  
FROM node:18-alpine AS builder

# Set working directory
WORKDIR /build

# Copy protocols with generated TypeScript files
COPY protocols/ ./protocols/

# Install protocols dependencies
WORKDIR /build/protocols
RUN npm ci

# Switch to web directory
WORKDIR /build/web

# Copy web package files and workspace config
COPY web/package*.json ./
COPY web/nx.json ./
COPY web/project.json ./
COPY web/tsconfig*.json ./
COPY web/vite.config.ts ./

# Copy session-waveform library configuration for workspace dependencies
COPY web/libs/session-waveform/package.json ./libs/session-waveform/
COPY web/libs/session-waveform/project.json ./libs/session-waveform/
COPY web/libs/session-waveform/tsconfig*.json ./libs/session-waveform/
COPY web/libs/session-waveform/vite.config.ts ./libs/session-waveform/

# Install web dependencies (including workspace dependencies)
RUN npm ci

# Copy web source code
COPY web/src/ ./src/
COPY web/libs/ ./libs/
COPY web/public/ ./public/
COPY web/assets/ ./assets/
COPY web/index.html ./

# Set build-time environment variables with defaults for containerized deployment
ARG VITE_GRPC_SERVER_URL=http://grpc-proxy:8080

# Create environment file for build
RUN echo "VITE_GRPC_SERVER_URL=${VITE_GRPC_SERVER_URL}" > .env.production

# Build the application with NX verbose output for debugging
RUN npx nx build web --verbose

# Production stage
FROM nginx:alpine

# Copy built assets from builder stage
COPY --from=builder /build/web/dist/web /usr/share/nginx/html

# Copy nginx configuration
COPY web/nginx.conf /etc/nginx/nginx.conf

# Expose port
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]