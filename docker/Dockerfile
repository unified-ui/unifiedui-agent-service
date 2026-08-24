# Production Dockerfile for unified-ui Agent Service
# Multi-stage build for minimal image size

# ============================================
# Stage 1: Build stage
# ============================================
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/agent-service \
    ./cmd/server

# ============================================
# Stage 2: Production stage
# ============================================
FROM scratch

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/agent-service .

# Expose the API port
EXPOSE 8085

# Run as non-root user
USER nobody:nogroup

# Health check (note: scratch images don't have curl, health checks done via orchestrator)
# HEALTHCHECK is handled by Kubernetes/Docker Compose

# Run the application
ENTRYPOINT ["/app/agent-service"]
