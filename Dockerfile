# Build stage
FROM golang:1.24-alpine AS builder

# Set GOPRIVATE to skip GitHub authentication
ENV GOPRIVATE=github.com/ovasabi/*

WORKDIR /go/src/github.com/ovasabi/master-ovasabi

# Install build dependencies
RUN apk add --no-cache git protoc protobuf-dev

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy go mod files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate protobuf files
RUN find api/protos -name "*.proto" -exec \
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  {} +

# Update dependencies after protobuf generation
RUN go mod tidy && \
  go get google.golang.org/grpc/health && \
  go get google.golang.org/grpc/health/grpc_health_v1 && \
  go get google.golang.org/grpc/reflection && \
  go get google.golang.org/grpc/codes && \
  go get google.golang.org/grpc/status && \
  go get google.golang.org/protobuf/reflect/protoreflect && \
  go get google.golang.org/protobuf/runtime/protoimpl && \
  go get github.com/golang-jwt/jwt/v5 && \
  go get go.uber.org/zap && \
  go get golang.org/x/crypto/bcrypt && \
  go get github.com/google/uuid && \
  go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /go/src/github.com/ovasabi/master-ovasabi/server .

# Expose ports
EXPOSE 50051
EXPOSE 9090

# Run the application
CMD ["./server"] 