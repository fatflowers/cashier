# syntax=docker/dockerfile:1

# --- Builder stage -----------------------------------------------------------
FROM golang:1.25.0-bookworm AS builder

WORKDIR /src

# Ensure reproducible, static-ish binary (no CGO; Linux target)
ENV CGO_ENABLED=0 \
    GOOS=linux

# Pre-cache modules
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source
COPY . .

# Build the API binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api


# --- Runtime stage -----------------------------------------------------------
FROM debian:bookworm-slim AS runtime

ENV TZ=UTC \
    GIN_MODE=release

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary only
COPY --from=builder /out/api /usr/local/bin/api

# Optionally supply config at runtime via env or volume
# - APP_CONFIG_FILE=/app/config/config.yaml (mount your file)
# - or use APP_* env overrides as supported by pkg/config

# Match sample config default port (update if you change config)
# Expose app port and optional metrics port (if used via APP_METRICS_ADDR)
EXPOSE 8888 90

# Run as non-root (nobody)
USER 65532:65532

ENTRYPOINT ["api"]
