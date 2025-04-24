# --- Build stage -------------------------------------------------
FROM golang:1.24-alpine AS builder

# Set environment variables for Go
ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOPATH=/go \
  GOPRIVATE=github.com/nmxmxh/*

WORKDIR /go/src/github.com/nmxmxh/master-ovasabi

# install build tools
RUN apk add --no-cache git protoc protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 \
  && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# copy module files and download deps that DON'T reference local code
COPY go.mod go.sum ./
RUN go mod download

# copy the rest of the source
COPY . .

# generate protobuf
RUN find api/protos -name "*.proto" -exec \
  protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative {} +

# configure local module
RUN go mod edit -replace github.com/nmxmxh/master-ovasabi=./ && \
  go mod tidy

# build binary
RUN go build -o server ./cmd/server

# --- Runtime stage ----------------------------------------------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /go/src/github.com/nmxmxh/master-ovasabi/server .

EXPOSE 50051 9090
CMD ["./server"]
