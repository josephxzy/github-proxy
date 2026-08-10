#!/usr/bin/env bash
# GitHub Proxy Build Script
# Usage: ./build.sh [version]
# Example: ./build.sh v1.0.0
#
# 流程：构建前端（web/ → cmd/github-proxy/public，供 embed 打包）→
#       交叉编译 Go 二进制（./cmd/github-proxy）→ 复制配置到 build/。
set -euo pipefail

VERSION=${1:-"dev"}
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$ROOT_DIR/build"

echo "============================================"
echo " GitHub Proxy Build Script"
echo " Version: $VERSION"
echo "============================================"
echo ""

# 1. Create build directory
mkdir -p "$BUILD_DIR"
rm -rf "$BUILD_DIR"/*

# 2. Build frontend (输出到 cmd/github-proxy/public，由 //go:embed 打包)
echo "[1/4] Building frontend..."
cd "$ROOT_DIR/web"
npm install --silent
npm run build --silent
echo "  Frontend built -> cmd/github-proxy/public/"
echo ""

# 3. Build Go backends (cross-compile)
cd "$ROOT_DIR"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
export CGO_ENABLED=0

echo "[2/4] Building Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/github-proxy-linux-amd64" ./cmd/github-proxy
echo "  Done -> build/github-proxy-linux-amd64"

echo "[3/4] Building Windows amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/github-proxy-windows-amd64.exe" ./cmd/github-proxy
echo "  Done -> build/github-proxy-windows-amd64.exe"

echo ""

# 4. Copy config
cp "$ROOT_DIR/config.toml" "$BUILD_DIR/"
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
