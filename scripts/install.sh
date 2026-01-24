#!/bin/bash
set -e

# WoL-NUT Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/aloks98/wol-nut/main/scripts/install.sh | sudo bash

REPO="aloks98/wol-nut"
INSTALL_DIR="/opt/wol-nut"
CONFIG_DIR="/etc/wol-nut"
DATA_DIR="/var/lib/wol-nut"
SERVICE_NAME="wol-nut"
USER="wol-nut"

echo "======================================"
echo "       WoL-NUT Installer              "
echo "======================================"

# Check root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="armv7" ;;
    *)       echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac
echo "-> Architecture: $ARCH"

# Get latest release
echo "-> Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Error: Could not determine latest release"
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
echo "-> Downloading binary..."
curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/wol-nut"
chmod +x "$INSTALL_DIR/wol-nut"

# Create default config if not exists
if [ ! -f "$CONFIG_DIR/config.yml" ]; then
    echo "-> Creating default config..."
    cat > "$CONFIG_DIR/config.yml" << 'EOF'
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
    echo ""
else
    echo "Error: Service failed to start"
    echo "Check logs: journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi
