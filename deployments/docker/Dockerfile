# ----------- Build Stage -----------
FROM golang:1.22.2-alpine3.19 AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Generate protobuf code
RUN make proto

# Build stripped binary for minimal size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/master-ovasabi ./cmd/server

# ----------- Final Stage -----------
FROM cgr.dev/chainguard/static:latest

# Add OCI labels (optional, helpful for tracking in registries)
LABEL org.opencontainers.image.title="master-ovasabi"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/your-org/your-repo"
LABEL org.opencontainers.image.authors="you@yourdomain.com"

# Copy built binary and config file
COPY --from=builder /app/master-ovasabi /
COPY --from=builder /app/config/config.yaml /config/

# Expose application port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release

# Start the application
CMD ["/master-ovasabi"]

