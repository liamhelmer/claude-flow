#!/bin/bash
set -e

# Build and Push Multi-Platform Images to DockerHub
# Repository: liamhelmer

echo "🐋 Multi-Platform Docker Build Script"
echo "===================================="
echo ""

# Check if logged in to DockerHub
if ! docker info 2>/dev/null | grep -q "Username"; then
    echo "❌ Not logged in to DockerHub"
    echo "Please run: docker login"
    exit 1
fi

# DockerHub repository
DOCKER_REPO="liamhelmer"

# Image versions
VERSION="2.0.0"
LATEST="latest"

# Platforms to build for
PLATFORMS="linux/amd64,linux/arm64"

echo "📦 Building for platforms: $PLATFORMS"
echo "🏷️  Version: $VERSION"
echo "🔗 Repository: $DOCKER_REPO"
echo ""

# Function to build and push multi-platform image
build_and_push() {
    local name=$1
    local dockerfile=$2
    local context=$3
    
    echo "🔨 Building $name..."
    
    # Build and push with version tag
    docker buildx build \
        --platform=$PLATFORMS \
        --tag=$DOCKER_REPO/$name:$VERSION \
        --tag=$DOCKER_REPO/$name:$LATEST \
        --file=$dockerfile \
        --push \
        $context
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully built and pushed $name"
    else
        echo "❌ Failed to build $name"
        exit 1
    fi
}

# Ensure buildx is available and create builder
echo "🛠️  Setting up Docker buildx..."
if ! docker buildx version &>/dev/null; then
    echo "❌ Docker buildx not available"
    echo "Please ensure Docker Desktop or buildx is installed"
    exit 1
fi

# Create and use a new builder instance
BUILDER_NAME="claude-flow-builder"
if ! docker buildx ls | grep -q $BUILDER_NAME; then
    echo "📐 Creating new buildx builder..."
    docker buildx create --name $BUILDER_NAME --use
    docker buildx inspect --bootstrap
else
    echo "📐 Using existing buildx builder..."
    docker buildx use $BUILDER_NAME
fi

# Build MCP Server
echo ""
echo "🐝 Building MCP Server..."
build_and_push \
    "claude-flow-mcp" \
    "./build/Dockerfile.mcp-server" \
    "../"

# Build Swarm Executor
echo ""
echo "🚀 Building Swarm Executor..."
build_and_push \
    "swarm-executor" \
    "./swarm-operator/build/Dockerfile.swarm-executor" \
    "./swarm-operator"

# Build Swarm Operator
echo ""
echo "🎛️  Building Swarm Operator..."
# First ensure we have the latest dependencies
echo "📦 Updating Go modules..."
cd swarm-operator
go mod download
cd ..

build_and_push \
    "swarm-operator" \
    "./swarm-operator/Dockerfile" \
    "./swarm-operator"

echo ""
echo "🎉 All images built and pushed successfully!"
echo ""
echo "📋 Images available at:"
echo "  - $DOCKER_REPO/claude-flow-mcp:$VERSION"
echo "  - $DOCKER_REPO/swarm-executor:$VERSION"
echo "  - $DOCKER_REPO/swarm-operator:$VERSION"
echo ""
echo "🏷️  All images also tagged as :latest"
echo ""
echo "📝 Update your Kubernetes manifests to use these images:"
echo "  image: $DOCKER_REPO/claude-flow-mcp:$VERSION"
echo "  image: $DOCKER_REPO/swarm-executor:$VERSION"
echo "  image: $DOCKER_REPO/swarm-operator:$VERSION"