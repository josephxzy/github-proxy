#!/usr/bin/env bash
# GitHub Proxy Build Script
# Usage: ./build.sh [version]
# Example: ./build.sh v1.0.0

set -euo pipefail

VERSION=${1:-"dev"}
BUILD_DIR="$(cd "$(dirname "$0")" && pwd)/build"
SRC_DIR="$(cd "$(dirname "$0")" && pwd)/src"

echo "============================================"
echo " GitHub Proxy Build Script"
echo " Version: $VERSION"
echo "============================================"
echo ""

# 1. Create build directory
mkdir -p "$BUILD_DIR"

# 2. Build frontend
echo "[1/4] Building frontend..."
cd "$SRC_DIR/frontend"
npm install --silent
npm run build --silent
echo "  Frontend built -> src/public/"
echo ""

# 3. Build Go backends (cross-compile)
cd "$SRC_DIR"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
export CGO_ENABLED=0

echo "[2/4] Building Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/github-proxy-linux-amd64" .
echo "  Done -> build/github-proxy-linux-amd64"

echo "[3/4] Building Windows amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/github-proxy-windows-amd64.exe" .
echo "  Done -> build/github-proxy-windows-amd64.exe"

echo ""

# 4. Copy config
cp "$SRC_DIR/config.toml" "$BUILD_DIR/"
echo "[4/4] Config copied to build/"
echo ""

echo "============================================"
echo " Build complete!"
echo " Output: $BUILD_DIR/"
echo ""
echo "Files:"
ls -lh "$BUILD_DIR"/
echo ""
echo "Docker build: docker build -t github-proxy:$VERSION ."
echo "============================================"
