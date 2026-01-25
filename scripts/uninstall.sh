#!/bin/bash
set -e

# WoL-NUT Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/uninstall.sh | sudo bash

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
    echo "-> Removing installation directory..."
    rm -rf "$INSTALL_DIR"
fi

# Config
if [ -d "$CONFIG_DIR" ]; then
    echo ""
    read -p "Remove configuration ($CONFIG_DIR)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        echo "-> Configuration removed"
    else
        echo "-> Configuration kept at $CONFIG_DIR"
    fi
fi

# Data
if [ -d "$DATA_DIR" ]; then
    echo ""
    read -p "Remove data ($DATA_DIR)? This includes all devices and UPS connections. [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
        echo "-> Data removed"
    else
        echo "-> Data kept at $DATA_DIR"
    fi
fi

# User
if id "$USER" &>/dev/null; then
    echo ""
    read -p "Remove system user ($USER)? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel "$USER" 2>/dev/null || true
        echo "-> User removed"
    else
        echo "-> User kept"
    fi
fi

echo ""
echo "======================================"
echo "   WoL-NUT uninstalled                "
echo "======================================"
echo ""
echo "To reinstall, run:"
echo "  curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/install.sh | sudo bash"
echo ""
