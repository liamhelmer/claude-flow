# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download || true

# Copy source code
COPY cmd/ cmd/

# Tidy and download any missing dependencies
RUN go mod tidy

# Build the operator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o manager cmd/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the binary
COPY --from=builder /app/manager .

# Run as non-root user
USER 65532:65532

ENTRYPOINT ["/manager"]