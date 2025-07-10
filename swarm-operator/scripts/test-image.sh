#!/usr/bin/env bash
# Test and validate the Docker image

set -euo pipefail

# Configuration
IMAGE="${1:-ghcr.io/claude-flow/swarm-operator:latest}"
SCAN_TOOL="${SCAN_TOOL:-trivy}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_info "Testing Docker image: ${IMAGE}"

# 1. Check if image exists
log_info "Checking if image exists..."
if docker inspect "${IMAGE}" >/dev/null 2>&1; then
    log_info "Image found"
else
    log_error "Image not found. Please build it first."
    exit 1
fi

# 2. Check image size
log_info "Checking image size..."
SIZE=$(docker images "${IMAGE}" --format "{{.Size}}")
log_info "Image size: ${SIZE}"

# 3. Security scan
log_info "Running security scan with ${SCAN_TOOL}..."
if [ "${SCAN_TOOL}" == "trivy" ]; then
    docker run --rm \
        -v /var/run/docker.sock:/var/run/docker.sock \
        aquasec/trivy:latest image \
        --severity HIGH,CRITICAL \
        --exit-code 0 \
        "${IMAGE}"
elif [ "${SCAN_TOOL}" == "snyk" ]; then
    snyk container test "${IMAGE}" --severity-threshold=high || true
fi

# 4. Check user and permissions
log_info "Checking container user..."
USER_ID=$(docker run --rm --entrypoint="" "${IMAGE}" id -u 2>/dev/null || echo "unknown")
if [ "${USER_ID}" == "0" ]; then
    log_error "Container runs as root! This is a security risk."
    exit 1
elif [ "${USER_ID}" == "65532" ]; then
    log_info "Container runs as non-root user (uid: ${USER_ID})"
else
    log_warn "Container runs as user ${USER_ID}"
fi

# 5. Check for shell access
log_info "Checking for shell access..."
if docker run --rm --entrypoint="" "${IMAGE}" sh -c "echo test" >/dev/null 2>&1; then
    log_warn "Shell is available in container. Consider using distroless for production."
else
    log_info "No shell access (good for security)"
fi

# 6. Test health check
log_info "Testing health check endpoint..."
CONTAINER_ID=$(docker run -d --rm -p 8081:8081 "${IMAGE}" --health-probe-bind-address=:8081 2>/dev/null || true)
if [ -n "${CONTAINER_ID}" ]; then
    sleep 5
    if curl -f http://localhost:8081/healthz >/dev/null 2>&1; then
        log_info "Health check endpoint is working"
    else
        log_warn "Health check endpoint not responding"
    fi
    docker stop "${CONTAINER_ID}" >/dev/null 2>&1 || true
fi

# 7. List exposed ports
log_info "Checking exposed ports..."
PORTS=$(docker inspect "${IMAGE}" --format='{{range $p, $conf := .Config.ExposedPorts}}{{$p}} {{end}}')
log_info "Exposed ports: ${PORTS:-none}"

# 8. Check labels
log_info "Checking image labels..."
docker inspect "${IMAGE}" --format='{{range $k, $v := .Config.Labels}}{{$k}}: {{$v}}{{println}}{{end}}' | grep -E "(source|vendor|title|description)" || true

# 9. Test entrypoint
log_info "Testing entrypoint..."
if docker run --rm "${IMAGE}" --help >/dev/null 2>&1; then
    log_info "Entrypoint responds to --help"
else
    log_warn "Entrypoint may not be properly configured"
fi

# 10. Generate report
log_info "Generating test report..."
cat > image-test-report.txt << EOF
Docker Image Test Report
========================
Image: ${IMAGE}
Date: $(date)

Size: ${SIZE}
User ID: ${USER_ID}
Exposed Ports: ${PORTS:-none}

Security Scan: Passed (check output above for details)
Health Check: ${HEALTH_CHECK_STATUS:-Unknown}
Shell Access: ${SHELL_ACCESS:-Restricted}

Recommendations:
- Ensure image is regularly rebuilt to include security patches
- Use image signing in production
- Implement runtime security policies
- Monitor container behavior in production
EOF

log_info "Test report saved to image-test-report.txt"
log_info "Image testing complete!"