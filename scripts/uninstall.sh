#!/bin/bash
set -e

SERVICE_NAME="wol-nut"
INSTALL_DIR="/opt/wol-nut"
CONFIG_DIR="/etc/wol-nut"
DATA_DIR="/var/lib/wol-nut"
USER="wol-nut"

echo "======================================"
echo "       WoL-NUT Uninstaller            "
echo "======================================"

if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Stop service
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo "-> Stopping service..."
    systemctl stop "$SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo "-> Disabling service..."
    systemctl disable "$SERVICE_NAME"
fi

# Remove service file
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo "-> Removing service file..."
    rm "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload
fi

# Remove binary
if [ -d "$INSTALL_DIR" ]; then
    echo "-> Removing installation..."
    rm -rf "$INSTALL_DIR"
fi

# Config
if [ -d "$CONFIG_DIR" ]; then
    read -p "-> Remove config ($CONFIG_DIR)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
    fi
fi

# Data
if [ -d "$DATA_DIR" ]; then
    read -p "-> Remove data ($DATA_DIR)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
    fi
fi

# User
if id "$USER" &>/dev/null; then
    read -p "-> Remove system user ($USER)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel "$USER"
    fi
fi

echo ""
echo "WoL-NUT uninstalled"
