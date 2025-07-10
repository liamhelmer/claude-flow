#!/usr/bin/env bash
# Build multi-architecture Docker images for production

set -euo pipefail

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
ORGANIZATION="${ORGANIZATION:-claude-flow}"
IMAGE_NAME="${IMAGE_NAME:-swarm-operator}"
VERSION="${VERSION:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if buildx is available
if ! docker buildx version >/dev/null 2>&1; then
    log_warn "Docker buildx not found. Installing..."
    docker buildx create --use --name multiarch-builder
fi

# Ensure buildx builder supports multi-platform
BUILDER_NAME="multiarch-builder"
if ! docker buildx inspect "${BUILDER_NAME}" >/dev/null 2>&1; then
    log_info "Creating buildx builder: ${BUILDER_NAME}"
    docker buildx create --name "${BUILDER_NAME}" --driver docker-container --use
    docker buildx inspect --bootstrap
fi

# Use the multi-arch builder
docker buildx use "${BUILDER_NAME}"

# Build for all platforms
FULL_IMAGE="${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:${VERSION}"
log_info "Building multi-architecture image: ${FULL_IMAGE}"
log_info "Platforms: ${PLATFORMS}"

# Build and push in one step for efficiency
docker buildx build \
    --platform "${PLATFORMS}" \
    --tag "${FULL_IMAGE}" \
    --tag "${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:latest" \
    --push \
    --cache-from "type=registry,ref=${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:buildcache" \
    --cache-to "type=registry,ref=${REGISTRY}/${ORGANIZATION}/${IMAGE_NAME}:buildcache,mode=max" \
    .

# Create and push manifest list
log_info "Creating manifest list..."
docker buildx imagetools create \
    --tag "${FULL_IMAGE}-manifest" \
    "${FULL_IMAGE}"

# Inspect the manifest
log_info "Manifest details:"
docker buildx imagetools inspect "${FULL_IMAGE}"

log_info "Multi-architecture build complete!"