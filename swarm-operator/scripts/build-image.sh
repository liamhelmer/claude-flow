#!/usr/bin/env bash
# Build production Docker image for Claude Flow Swarm Operator

set -euo pipefail

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
ORGANIZATION="${ORGANIZATION:-claude-flow}"
IMAGE_NAME="${IMAGE_NAME:-swarm-operator}"
VERSION="${VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --org)
            ORGANIZATION="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --platform)
            PLATFORM="--platform $2"
            shift 2
            ;;
        --push)
            PUSH_IMAGE="true"
            shift
            ;;
        --scan)
            SCAN_IMAGE="true"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Usage: $0 [--registry REGISTRY] [--org ORG] [--version VERSION] [--no-cache] [--platform PLATFORM] [--push] [--scan]"
            exit 1
            ;;
    esac
done

# Construct full image name
FULL_IMAGE="${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:${VERSION}"

log_info "Building Docker image: ${FULL_IMAGE}"

# Ensure we're in the right directory
cd "$(dirname "$0")/.."

# Build the image
log_info "Starting Docker build..."
docker build \
    ${NO_CACHE:-} \
    ${PLATFORM:-} \
    -t "${FULL_IMAGE}" \
    -f Dockerfile \
    .

if [ $? -eq 0 ]; then
    log_info "Docker build completed successfully"
else
    log_error "Docker build failed"
    exit 1
fi

# Scan the image if requested
if [ "${SCAN_IMAGE}" == "true" ]; then
    log_info "Scanning image for vulnerabilities..."
    docker run --rm \
        -v /var/run/docker.sock:/var/run/docker.sock \
        aquasec/trivy:latest image \
        --severity HIGH,CRITICAL \
        "${FULL_IMAGE}"
fi

# Tag additional versions
if [ "${VERSION}" != "latest" ]; then
    # Also tag as latest
    docker tag "${FULL_IMAGE}" "${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:latest"
    
    # Tag with short SHA if in git repo
    if git rev-parse --short HEAD &>/dev/null; then
        SHORT_SHA=$(git rev-parse --short HEAD)
        docker tag "${FULL_IMAGE}" "${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:${SHORT_SHA}"
    fi
fi

# Push the image if requested
if [ "${PUSH_IMAGE}" == "true" ]; then
    log_info "Pushing image to registry..."
    docker push "${FULL_IMAGE}"
    
    # Push additional tags
    if [ "${VERSION}" != "latest" ]; then
        docker push "${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:latest"
        if [ -n "${SHORT_SHA:-}" ]; then
            docker push "${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:${SHORT_SHA}"
        fi
    fi
fi

log_info "Image build complete: ${FULL_IMAGE}"

# Display image size
docker images "${FULL_IMAGE}" --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}"