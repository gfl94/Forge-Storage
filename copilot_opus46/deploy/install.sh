#!/bin/bash
# install.sh – Sets up the Storage API service on an Ubuntu host.
# Run as root or with sudo.

set -euo pipefail

APP_USER="storage-api"
APP_DIR="/opt/storage"
BIN_DIR="${APP_DIR}/bin"
CONF_DIR="/etc/storage-api"
DATA_DIR="/var/lib/storage-api"
CACHE_DIR="/var/cache/storage-api"
FRONTEND_DIR="/var/www/storage-ui"

echo "==> Creating service user..."
id "${APP_USER}" &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin "${APP_USER}"

echo "==> Creating directories..."
mkdir -p "${BIN_DIR}" "${CONF_DIR}" "${DATA_DIR}" "${CACHE_DIR}" "${FRONTEND_DIR}"

echo "==> Copying binary..."
if [ -f "storage-api" ]; then
    cp storage-api "${BIN_DIR}/storage-api"
    chmod 755 "${BIN_DIR}/storage-api"
else
    echo "   WARNING: storage-api binary not found in current directory."
    echo "   Build it first with: cd copilot_opus46 && go build -o storage-api ./cmd/server"
fi

echo "==> Copying frontend assets..."
if [ -d "frontend" ]; then
    cp -r frontend/* "${FRONTEND_DIR}/"
else
    echo "   WARNING: frontend directory not found."
fi

echo "==> Installing configuration..."
if [ ! -f "${CONF_DIR}/storage-api.env" ]; then
    cp deploy/storage-api.env.example "${CONF_DIR}/storage-api.env"
    chmod 600 "${CONF_DIR}/storage-api.env"
    echo "   Created ${CONF_DIR}/storage-api.env – edit it with your Azure credentials."
else
    echo "   ${CONF_DIR}/storage-api.env already exists, skipping."
fi

echo "==> Installing systemd service..."
cp deploy/storage-api.service /etc/systemd/system/storage-api.service

echo "==> Setting permissions..."
chown -R "${APP_USER}:${APP_USER}" "${DATA_DIR}" "${CACHE_DIR}"
chown -R root:root "${BIN_DIR}" "${FRONTEND_DIR}"

echo "==> Reloading systemd..."
systemctl daemon-reload

echo "==> Installing Nginx config..."
if [ -d "/etc/nginx/sites-available" ]; then
    cp deploy/nginx-storage.conf /etc/nginx/sites-available/storage
    ln -sf /etc/nginx/sites-available/storage /etc/nginx/sites-enabled/storage
    echo "   Nginx config installed. Edit server_name and TLS settings, then reload:"
    echo "   sudo nginx -t && sudo systemctl reload nginx"
else
    echo "   /etc/nginx/sites-available not found – install Nginx config manually."
fi

echo ""
echo "==> Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Edit /etc/storage-api/storage-api.env with your Azure credentials"
echo "  2. Edit /etc/nginx/sites-available/storage with your domain and TLS certs"
echo "  3. sudo nginx -t && sudo systemctl reload nginx"
echo "  4. sudo systemctl enable --now storage-api"
echo "  5. sudo journalctl -u storage-api -f"
echo ""
