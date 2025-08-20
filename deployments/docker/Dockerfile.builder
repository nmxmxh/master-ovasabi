# This is a common builder stage for all Go services in the project.
# It copies all necessary source code and dependencies into one place.
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev curl

# Set working directory
WORKDIR /app

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Copy go mod files first to leverage dependency caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy assets and tooling next, as they change less frequently
COPY Makefile ./Makefile
COPY tools/ ./tools/
COPY scripts/ ./scripts/


# --- Godot Builder Context ---
COPY godot/project /godot/project/
COPY config/ /godot/config/
# .env is not copied; environment variables are injected via docker-compose

# Copy API definitions and generate protobuf code.
# This step is only re-run if the .proto files change.
COPY api/ ./api/
RUN export PATH="$PATH:$(go env GOPATH)/bin" && make proto

# Copy the rest of the source code last, as it changes most frequently.
# A change here will not cause re-running of the steps above.

COPY pkg/ ./pkg/
COPY config/ ./config/
COPY database/ ./database/
COPY amadeus/ ./amadeus/
COPY internal/ ./internal/
COPY cmd/ ./cmd/
COPY start/ ./start/

# The final stage of this file is the builder image itself.
# We can give it a more explicit name, though it's aliased as go-builder.
FROM go-builder AS ovasabi-go-builder
