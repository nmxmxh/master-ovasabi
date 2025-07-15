# Dockerfile for main app (master-ovasabi)
# This Dockerfile builds and packages the main app using the common go-builder.
# syntax=docker/dockerfile:1.4
# --- Build Stage ---
# Use the common builder stage which contains all source code and dependencies.
FROM ovasabi-go-builder:latest AS builder

# The working directory is /app, inherited from the builder.
# All source code is present, and protoc has been run.

# Build the master-ovasabi binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/master-ovasabi ./cmd/server

# ----------- Final Stage -----------
FROM alpine:latest

# Add OCI labels (optional, helpful for tracking in registries)
LABEL org.opencontainers.image.title="master-ovasabi"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/nmxmxh/master-ovasabi"
LABEL org.opencontainers.image.authors="hello@ovasabi.studio"

# Install basic tools
RUN apk add --no-cache curl

# Copy built binary and config file
COPY --from=builder /app/master-ovasabi /
COPY --from=builder /app/config/config.yaml /config/
COPY --from=builder /app/config/service_registration.json /config/

# Copy the pre-downloaded health probe tool
COPY --from=builder /app/tools/grpc_health_probe /usr/local/bin/grpc_health_probe
RUN chmod +x /usr/local/bin/grpc_health_probe

# Expose application port
EXPOSE 50051 9090

# Set environment variables
ENV GIN_MODE=release

# Start the application
CMD ["/master-ovasabi"]

