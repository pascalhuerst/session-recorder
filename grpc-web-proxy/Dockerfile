# gRPC-Web Proxy Dockerfile
FROM envoyproxy/envoy:v1.28-latest

# Copy Envoy configuration
COPY envoy.yaml /etc/envoy/envoy.yaml

# Expose port
EXPOSE 8080

# Use the default Envoy entrypoint