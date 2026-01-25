#!/bin/bash
set -e

REPO="aloks98/wolnut"
INSTALL_DIR="/opt/wol-nut"
SERVICE_NAME="wol-nut"

echo "-> Checking for updates..."

# Get current version
CURRENT=$($INSTALL_DIR/wol-nut --version 2>/dev/null | awk '{print $2}' || echo "unknown")
echo "  Current: $CURRENT"

# Get latest version
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
echo "  Latest:  $LATEST"

if [ "$CURRENT" = "$LATEST" ]; then
    echo "Already up to date"
    exit 0
fi

echo "-> Updating to $LATEST..."

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="armv7" ;;
esac

# Download
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/wol-nut-linux-${ARCH}"
curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/wol-nut.new"
chmod +x "$INSTALL_DIR/wol-nut.new"

# Replace and restart
sudo systemctl stop "$SERVICE_NAME"
mv "$INSTALL_DIR/wol-nut.new" "$INSTALL_DIR/wol-nut"
sudo systemctl start "$SERVICE_NAME"

echo "Updated to $LATEST"
