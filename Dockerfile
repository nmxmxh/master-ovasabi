# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git protoc protobuf-dev

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy go mod files
COPY go.mod go.sum ./

# Initialize and download dependencies
RUN go mod download && \
  go get google.golang.org/grpc@latest && \
  go get google.golang.org/protobuf@latest && \
  go get github.com/golang-jwt/jwt/v5@latest && \
  go get go.uber.org/zap@latest && \
  go get golang.org/x/crypto/bcrypt@latest && \
  go get github.com/google/uuid@latest && \
  go get google.golang.org/grpc/health@latest && \
  go get google.golang.org/grpc/health/grpc_health_v1@latest && \
  go get google.golang.org/grpc/reflection@latest && \
  go mod tidy

# Generate protobuf files
COPY api/protos/ api/protos/
RUN find api/protos -name "*.proto" -exec \
  protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  {} +

# Copy source code
COPY . .

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