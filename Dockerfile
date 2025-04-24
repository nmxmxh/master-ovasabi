# -------------------------------------------------------------
# Build stage
# -------------------------------------------------------------
FROM golang:1.24-alpine AS builder

# Build-time settings
ENV CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /build

# ----- tools ---------------------------------------------------
RUN apk add --no-cache git protoc protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# ----- module cache layer --------------------------------------
COPY go.mod go.sum ./
RUN go mod download

# ----- copy source ---------------------------------------------
COPY . .

# ----- generate protobuf code BEFORE tidy ----------------------
# This produces api/protos/*.pb.go so the package exists locally
RUN find api/protos -name '*.proto' -exec \
  protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative {} +

# ----- tidy now that generated code is present -----------------
RUN go mod tidy

# ----- build binary -------------------------------------------
RUN go build -o server ./cmd/server

# -------------------------------------------------------------
# Runtime stage
# -------------------------------------------------------------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/server .

EXPOSE 50051 9090
CMD ["./server"]