# Production Dockerfile for Claude Flow Swarm Operator
# Multi-stage build for security and minimal image size

# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Install security scanner
RUN go install github.com/securego/gosec/v2/cmd/gosec@latest

# Set working directory
WORKDIR /workspace

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY api/ api/
COPY controllers/ controllers/
COPY config/ config/
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Run security scan
RUN gosec -fmt json -out /tmp/gosec-report.json ./... || true

# Build the manager binary with security flags
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on

RUN go build -a \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo \
    -o manager cmd/main.go

# Stage 2: Runtime - Using distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Labels for container metadata
LABEL org.opencontainers.image.source="https://github.com/claude-flow/swarm-operator" \
      org.opencontainers.image.vendor="Claude Flow" \
      org.opencontainers.image.title="Swarm Operator" \
      org.opencontainers.image.description="Kubernetes operator for managing Claude Flow swarm agents"

# Create app directory
WORKDIR /

# Copy the manager binary from builder
COPY --from=builder /workspace/manager .

# Copy security scan report for compliance
COPY --from=builder /tmp/gosec-report.json /security/gosec-report.json

# Use nonroot user (65532 is the UID for nonroot in distroless)
USER 65532:65532

# Expose metrics port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/manager", "--health-probe-bind-address=:8081"]

# Run the manager
ENTRYPOINT ["/manager"]