#!/bin/bash
set -e

# WoL-NUT Updater
# Usage: curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/update.sh | sudo bash

REPO="aloks98/wolnut"
INSTALL_DIR="/opt/wol-nut"
SERVICE_NAME="wol-nut"

echo "======================================"
echo "       WoL-NUT Updater                "
echo "======================================"

# Check root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Check if installed
if [ ! -f "$INSTALL_DIR/wol-nut" ]; then
    echo "Error: WoL-NUT is not installed at $INSTALL_DIR"
    echo "Run the installer first:"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/master/scripts/install.sh | sudo bash"
    exit 1
fi

# Get current version
CURRENT=$($INSTALL_DIR/wol-nut --version 2>/dev/null | awk '{print $NF}' || echo "unknown")
echo "-> Current version: $CURRENT"

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="armv7" ;;
    armv6l)  ARCH="armv7" ;;
    *)       echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
echo "-> Checking for updates..."
API_RESPONSE=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest")
if echo "$API_RESPONSE" | grep -q "Not Found"; then
    echo "Error: Could not fetch releases"
    exit 1
fi
if echo "$API_RESPONSE" | grep -q "API rate limit"; then
    echo "Error: GitHub API rate limit exceeded. Try again later."
    exit 1
fi
LATEST=$(echo "$API_RESPONSE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Error: Could not determine latest release"
    exit 1
fi
echo "-> Latest version:  $LATEST"

# Compare versions
if [ "$CURRENT" = "$LATEST" ]; then
    echo ""
    echo "Already running the latest version!"
    exit 0
fi

echo ""
echo "-> Updating from $CURRENT to $LATEST..."

# Download new binary
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/wol-nut-linux-${ARCH}"
echo "-> Downloading..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/wol-nut.new"; then
    echo "Error: Failed to download binary"
    exit 1
fi
chmod +x "$INSTALL_DIR/wol-nut.new"

# Verify new binary
if ! "$INSTALL_DIR/wol-nut.new" --version &>/dev/null; then
    echo "Error: Downloaded binary appears to be invalid"
    rm -f "$INSTALL_DIR/wol-nut.new"
    exit 1
fi

# Stop service
echo "-> Stopping service..."
systemctl stop "$SERVICE_NAME" || true

# Backup old binary
if [ -f "$INSTALL_DIR/wol-nut" ]; then
    mv "$INSTALL_DIR/wol-nut" "$INSTALL_DIR/wol-nut.bak"
fi

# Replace binary
mv "$INSTALL_DIR/wol-nut.new" "$INSTALL_DIR/wol-nut"

# Start service
echo "-> Starting service..."
systemctl start "$SERVICE_NAME"

# Verify service started
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    # Remove backup
    rm -f "$INSTALL_DIR/wol-nut.bak"

    echo ""
    echo "======================================"
    echo "   Update complete!                   "
    echo "======================================"
    echo ""
    echo "  Updated from $CURRENT to $LATEST"
    echo ""
else
    echo "Error: Service failed to start after update"
    echo "Rolling back..."
    if [ -f "$INSTALL_DIR/wol-nut.bak" ]; then
        mv "$INSTALL_DIR/wol-nut.bak" "$INSTALL_DIR/wol-nut"
        systemctl start "$SERVICE_NAME"
        echo "Rolled back to previous version"
    fi
    echo "Check logs: journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi
