# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git protoc protobuf-dev

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy entire source code first
COPY . .

# Generate protobuf files
RUN find api/protos -name "*.proto" -exec \
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  {} +

# Initialize and download dependencies
RUN go mod download && \
  go get -u ./... && \
  go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Expose ports
EXPOSE 50051
EXPOSE 9090

# Run the application
CMD ["./server"] 