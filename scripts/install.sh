#!/bin/bash
set -e

# WoL-NUT Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/install.sh | sudo bash

REPO="aloks98/wolnut"
INSTALL_DIR="/opt/wol-nut"
CONFIG_DIR="/etc/wol-nut"
DATA_DIR="/var/lib/wol-nut"
SERVICE_NAME="wol-nut"
USER="wol-nut"

echo "======================================"
echo "       WoL-NUT Installer              "
echo "======================================"
echo ""
echo "  Wake-on-LAN + NUT UPS Dashboard"
echo "  https://github.com/${REPO}"
echo ""

# Check root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Check for required commands
for cmd in curl systemctl; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: Required command '$cmd' not found"
        exit 1
    fi
done

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="armv7" ;;
    armv6l)  ARCH="armv7" ;;  # Fallback for older Pis
    *)       echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac
echo "-> Architecture: $ARCH"

# Get latest release
echo "-> Fetching latest release..."
API_RESPONSE=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest")
if echo "$API_RESPONSE" | grep -q "Not Found"; then
    echo "Error: Could not fetch releases. Is the repository public?"
    echo "API response: $API_RESPONSE"
    exit 1
fi
if echo "$API_RESPONSE" | grep -q "API rate limit"; then
    echo "Error: GitHub API rate limit exceeded. Try again later."
    exit 1
fi
LATEST=$(echo "$API_RESPONSE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Error: Could not determine latest release"
    echo "API response: $API_RESPONSE"
    exit 1
fi
echo "-> Latest version: $LATEST"

# Create user
if ! id "$USER" &>/dev/null; then
    echo "-> Creating system user..."
    useradd --system --no-create-home --shell /usr/sbin/nologin "$USER"
fi

# Create directories
echo "-> Creating directories..."
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR"

# Download binary
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/wol-nut-linux-${ARCH}"
echo "-> Downloading binary from:"
echo "   $DOWNLOAD_URL"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/wol-nut"; then
    echo "Error: Failed to download binary"
    echo "Check if release exists: https://github.com/${REPO}/releases/tag/${LATEST}"
    exit 1
fi
chmod +x "$INSTALL_DIR/wol-nut"

# Verify binary
if ! "$INSTALL_DIR/wol-nut" --version &>/dev/null; then
    echo "Warning: Could not verify binary version"
fi

# Create default config if not exists
if [ ! -f "$CONFIG_DIR/config.yml" ]; then
    echo "-> Creating default config..."
    cat > "$CONFIG_DIR/config.yml" << 'EOF'
# WoL-NUT Configuration
# Documentation: https://github.com/aloks98/wolnut

server:
  port: 8080
  host: "0.0.0.0"

data:
  path: "/var/lib/wol-nut"

log:
  level: "info"
EOF
fi

# Set permissions
chown -R "$USER:$USER" "$DATA_DIR"
chown -R root:root "$CONFIG_DIR"
chmod 644 "$CONFIG_DIR/config.yml"

# Create systemd service
echo "-> Creating systemd service..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=WoL-NUT Dashboard
Documentation=https://github.com/${REPO}
After=network.target

[Service]
Type=simple
User=$USER
Group=$USER
ExecStart=$INSTALL_DIR/wol-nut --config $CONFIG_DIR/config.yml
Restart=on-failure
RestartSec=5
WorkingDirectory=$DATA_DIR

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR
PrivateTmp=true

# Network capabilities for WoL broadcast
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
EOF

# Start service
echo "-> Starting service..."
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

# Wait and check
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    IP=$(hostname -I | awk '{print $1}')
    echo ""
    echo "======================================"
    echo "   Installation complete!             "
    echo "======================================"
    echo ""
    echo "  Dashboard:  http://${IP}:8080"
    echo "  Config:     $CONFIG_DIR/config.yml"
    echo "  Data:       $DATA_DIR/"
    echo ""
    echo "  Commands:"
    echo "    Status:   sudo systemctl status $SERVICE_NAME"
    echo "    Logs:     sudo journalctl -u $SERVICE_NAME -f"
    echo "    Restart:  sudo systemctl restart $SERVICE_NAME"
    echo "    Update:   curl -fsSL https://raw.githubusercontent.com/${REPO}/master/scripts/update.sh | sudo bash"
    echo ""
else
    echo "Error: Service failed to start"
    echo "Check logs: journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi
