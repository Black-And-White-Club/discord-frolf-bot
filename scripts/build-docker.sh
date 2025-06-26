#!/bin/bash
set -e

# Build script for Discord Frolf Bot using Docker BuildKit features
# This script demonstrates 2025 best practices for Docker builds

echo "🏗️  Building Discord Frolf Bot with optimizations..."

# Enable Docker BuildKit for advanced features
export DOCKER_BUILDKIT=1

# Build with cache mount, security, and multi-platform support
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --cache-from type=local,src=/tmp/.buildx-cache \
  --cache-to type=local,dest=/tmp/.buildx-cache-new,mode=max \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  --pull \
  --tag discord-frolf-bot:latest \
  --tag discord-frolf-bot:$(git rev-parse --short HEAD) \
  --progress=plain \
  .

# Replace old cache
rm -rf /tmp/.buildx-cache
mv /tmp/.buildx-cache-new /tmp/.buildx-cache

echo "✅ Build complete!"
echo "📦 Images tagged:"
echo "   - discord-frolf-bot:latest"
echo "   - discord-frolf-bot:$(git rev-parse --short HEAD)"
