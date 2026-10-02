# Multi-stage Dockerfile for Go ai-callcenter-orchestrator binary
FROM golang:alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

# Install git and build essentials
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build statically linked orchestrator binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/orchestrator ./cmd/server

# Final minimal runtime image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

# Copy compiled binary from builder
COPY --from=builder /app/orchestrator /app/orchestrator

# Expose orchestrator REST and WebSocket port
EXPOSE 8080

ENV PORT=8080
ENV ENVIRONMENT=production

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/orchestrator"]
