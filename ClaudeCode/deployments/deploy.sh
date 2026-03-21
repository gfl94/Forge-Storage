#!/bin/bash

# Deployment script for Storage Gateway
# This script should be run as root or with sudo

set -e

echo "=== Storage Gateway Deployment Script ==="
echo ""

# Variables
APP_NAME="storage-api"
APP_USER="storage-api"
APP_GROUP="storage-api"
INSTALL_DIR="/opt/storage"
BIN_DIR="$INSTALL_DIR/bin"
CONFIG_DIR="/etc/$APP_NAME"
DATA_DIR="/var/lib/$APP_NAME"
CACHE_DIR="/var/cache/$APP_NAME"
FRONTEND_DIR="/var/www/storage-ui"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"

# Step 1: Create user and group
echo "Step 1: Creating user and group..."
if ! id -u $APP_USER > /dev/null 2>&1; then
    useradd -r -s /bin/false -d $INSTALL_DIR $APP_USER
    echo "User $APP_USER created"
else
    echo "User $APP_USER already exists"
fi

# Step 2: Create directories
echo "Step 2: Creating directories..."
mkdir -p $BIN_DIR
mkdir -p $CONFIG_DIR
mkdir -p $DATA_DIR
mkdir -p $CACHE_DIR
mkdir -p $FRONTEND_DIR

# Step 3: Build the Go application
echo "Step 3: Building Go application..."
cd "$(dirname "$0")/.."
go build -o $BIN_DIR/$APP_NAME ./cmd/server

# Step 4: Copy frontend files
echo "Step 4: Copying frontend files..."
cp -r frontend/* $FRONTEND_DIR/

# Step 5: Copy configuration files
echo "Step 5: Setting up configuration..."
if [ ! -f $CONFIG_DIR/$APP_NAME.env ]; then
    cp deployments/$APP_NAME.env.example $CONFIG_DIR/$APP_NAME.env
    echo "Configuration file created at $CONFIG_DIR/$APP_NAME.env"
    echo "IMPORTANT: Edit this file with your Azure credentials!"
else
    echo "Configuration file already exists, skipping"
fi

# Step 6: Set permissions
echo "Step 6: Setting permissions..."
chown -R $APP_USER:$APP_GROUP $INSTALL_DIR
chown -R $APP_USER:$APP_GROUP $DATA_DIR
chown -R $APP_USER:$APP_GROUP $CACHE_DIR
chmod 600 $CONFIG_DIR/$APP_NAME.env
chmod 755 $BIN_DIR/$APP_NAME

# Step 7: Install systemd service
echo "Step 7: Installing systemd service..."
cp deployments/$APP_NAME.service $SERVICE_FILE
systemctl daemon-reload

# Step 8: Display next steps
echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps:"
echo "1. Edit configuration: nano $CONFIG_DIR/$APP_NAME.env"
echo "2. Add your Azure Storage credentials to the config file"
echo "3. Start the service: systemctl start $APP_NAME"
echo "4. Enable on boot: systemctl enable $APP_NAME"
echo "5. Check status: systemctl status $APP_NAME"
echo "6. View logs: journalctl -u $APP_NAME -f"
echo ""
echo "7. Configure Nginx:"
echo "   - Copy deployments/nginx.conf.example to /etc/nginx/sites-available/$APP_NAME"
echo "   - Edit the file with your domain and SSL certificates"
echo "   - Enable: ln -s /etc/nginx/sites-available/$APP_NAME /etc/nginx/sites-enabled/"
echo "   - Test: nginx -t"
echo "   - Reload: systemctl reload nginx"
echo ""
echo "Default login credentials:"
echo "  Username: admin"
echo "  Password: admin123"
echo "  CHANGE THESE IMMEDIATELY IN PRODUCTION!"
