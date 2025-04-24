# --- Build stage ---------------------------------------------
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /build

RUN apk add --no-cache git protoc protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# copy modules first to leverage layer cache
COPY go.mod go.sum ./
RUN go mod download

# copy the rest of the source
COPY . .

# ---- generate protobuf BEFORE tidy --------------------------
RUN find api/protos -name '*.proto' -exec \
  protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative {} +

# now that generated Go files exist, tidy works
RUN go mod tidy

# build the binary
RUN go build -o server ./cmd/server

# --- Runtime stage -------------------------------------------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/server .

EXPOSE 50051 9090
CMD ["./server"]