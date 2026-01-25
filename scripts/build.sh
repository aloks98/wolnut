#!/bin/bash
set -e

# WoL-NUT Build Script
# Builds the application from source

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "======================================"
echo "       WoL-NUT Build Script           "
echo "======================================"

cd "$PROJECT_DIR"

# Check for required tools
echo "-> Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    echo "Install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "   Go version: $GO_VERSION"

if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed"
    echo "Install Node.js 22+ from https://nodejs.org/"
    exit 1
fi

NODE_VERSION=$(node --version)
echo "   Node.js version: $NODE_VERSION"

if ! command -v pnpm &> /dev/null; then
    echo "-> Installing pnpm..."
    npm install -g pnpm
fi

PNPM_VERSION=$(pnpm --version)
echo "   pnpm version: $PNPM_VERSION"

# Build frontend
echo ""
echo "-> Building frontend..."
cd "$PROJECT_DIR/web"
pnpm install --frozen-lockfile 2>/dev/null || pnpm install
pnpm build

# Build backend
echo ""
echo "-> Building backend..."
cd "$PROJECT_DIR"

VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$COMMIT"

# Build for current platform
OUTPUT="wol-nut"
if [ "$(uname)" = "Darwin" ]; then
    OUTPUT="wol-nut-darwin"
fi

go build -ldflags "$LDFLAGS" -o "$OUTPUT"

echo ""
echo "======================================"
echo "   Build complete!                    "
echo "======================================"
echo ""
echo "  Binary: $PROJECT_DIR/$OUTPUT"
echo "  Version: $VERSION"
echo "  Commit: $COMMIT"
echo ""
echo "  Run with:"
echo "    ./$OUTPUT"
echo ""

# Optional: Build for all platforms
if [ "$1" = "--all" ]; then
    echo "-> Building for all platforms..."

    GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "wol-nut-linux-amd64"
    echo "   Built: wol-nut-linux-amd64"

    GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "wol-nut-linux-arm64"
    echo "   Built: wol-nut-linux-arm64"

    GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "$LDFLAGS" -o "wol-nut-linux-armv7"
    echo "   Built: wol-nut-linux-armv7"

    echo ""
    echo "All binaries built successfully!"
fi
