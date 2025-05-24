# Dockerfile for main app (master-ovasabi)
# This Dockerfile builds and packages the main app using a multi-stage build and Alpine runtime.
# For the Nexus service, see Dockerfile.nexus.

# ----------- Build Stage -----------
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev curl

# Set working directory
WORKDIR /app

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
  export PATH="$PATH:$(go env GOPATH)/bin"

# Copy go mod files first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Generate protobuf code and build
RUN export PATH="$PATH:$(go env GOPATH)/bin" && \
  make proto && \
  CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/master-ovasabi ./cmd/server

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
COPY --from=builder /app/service_registration.json /

# Add grpc_health_probe for gRPC health checks
ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.19/grpc_health_probe-linux-amd64 /usr/local/bin/grpc_health_probe
RUN chmod +x /usr/local/bin/grpc_health_probe

# Expose application port
EXPOSE 50051 9090

# Set environment variables
ENV GIN_MODE=release

# Start the application
CMD ["/master-ovasabi"]

