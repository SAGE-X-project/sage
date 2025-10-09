#!/bin/bash
# SAGE Docker Build Script
# Builds Docker images with proper versioning and multi-arch support

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
IMAGE_NAME="${IMAGE_NAME:-sage-backend}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

echo -e "${GREEN}SAGE Docker Build${NC}"
echo "================================"
echo "Image: $IMAGE_NAME"
echo "Version: $VERSION"
echo "Build Date: $BUILD_DATE"
echo "Platforms: $PLATFORMS"
echo ""

# Check if Docker is available
if ! command -v docker >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker not found${NC}"
    exit 1
fi

# Check if buildx is available for multi-arch
if [ "$PLATFORMS" != "linux/amd64" ] && [ "$PLATFORMS" != "linux/arm64" ]; then
    if ! docker buildx version >/dev/null 2>&1; then
        echo -e "${YELLOW}WARNING: Docker buildx not available, building single platform${NC}"
        PLATFORMS="linux/amd64"
    fi
fi

# Build arguments
BUILD_ARGS=(
    --build-arg "VERSION=$VERSION"
    --build-arg "BUILD_DATE=$BUILD_DATE"
    --tag "$IMAGE_NAME:$VERSION"
    --tag "$IMAGE_NAME:latest"
)

# Multi-arch build or single arch
if echo "$PLATFORMS" | grep -q ","; then
    echo -e "${BLUE}Building multi-arch image...${NC}"

    # Create buildx builder if it doesn't exist
    if ! docker buildx ls | grep -q sage-builder; then
        echo "Creating buildx builder..."
        docker buildx create --name sage-builder --use
    else
        docker buildx use sage-builder
    fi

    # Build and push (requires registry)
    if [ -n "$DOCKER_REGISTRY" ]; then
        echo "Building for platforms: $PLATFORMS"
        docker buildx build \
            "${BUILD_ARGS[@]}" \
            --platform "$PLATFORMS" \
            --push \
            --tag "$DOCKER_REGISTRY/$IMAGE_NAME:$VERSION" \
            --tag "$DOCKER_REGISTRY/$IMAGE_NAME:latest" \
            .
        echo -e "${GREEN}Multi-arch images pushed to $DOCKER_REGISTRY${NC}"
    else
        echo "Building for platforms: $PLATFORMS (load to local)"
        docker buildx build \
            "${BUILD_ARGS[@]}" \
            --platform "$PLATFORMS" \
            --load \
            .
        echo -e "${GREEN}Multi-arch images loaded locally${NC}"
    fi
else
    echo -e "${BLUE}Building single-arch image for $PLATFORMS...${NC}"
    docker build \
        "${BUILD_ARGS[@]}" \
        --platform "$PLATFORMS" \
        .
    echo -e "${GREEN}Image built successfully${NC}"
fi

# Display image info
echo ""
echo -e "${GREEN}Build complete!${NC}"
echo "================================"
docker images | grep "$IMAGE_NAME" | head -5

echo ""
echo -e "${YELLOW}Usage examples:${NC}"
echo "  docker run --rm $IMAGE_NAME:$VERSION sage-crypto --version"
echo "  docker-compose up -d"
echo "  docker run -it $IMAGE_NAME:$VERSION /bin/sh"
