# --- Build stage -------------------------------------------------
FROM golang:1.24-alpine AS builder

# Set environment variables for Go
ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /build

# install build tools
RUN apk add --no-cache git protoc protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy the entire source code
COPY . .

# Download and verify dependencies
RUN go mod download && \
  go mod verify && \
  go mod tidy

# Generate protobuf files
RUN find api/protos -name "*.proto" -exec \
  protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative {} +

# Build the application
RUN go build -o server ./cmd/server

# --- Runtime stage ----------------------------------------------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/server .

EXPOSE 50051 9090
CMD ["./server"]
