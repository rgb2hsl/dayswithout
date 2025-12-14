#!/usr/bin/env bash
set -e

APP_NAME="dayswithout"
BUILD_DIR="build"
BIN_PATH="/usr/local/bin/$APP_NAME"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"

echo "[INFO] Running build.sh..."
./build.sh

# Stop service if it exists and is running
if systemctl list-units --full -all | grep -Fq "$APP_NAME.service"; then
  if systemctl is-active --quiet "$APP_NAME"; then
    echo "[INFO] Stopping $APP_NAME service..."
    sudo systemctl stop "$APP_NAME"

    echo "[INFO] Waiting for service to fully stop..."
    while systemctl is-active --quiet "$APP_NAME"; do
      sleep 0.2
    done
  fi
fi

echo "[INFO] Installing binary to $BIN_PATH"
sudo cp "$BUILD_DIR/$APP_NAME" "$BIN_PATH"
sudo chmod +x "$BIN_PATH"

# Create systemd unit if not exists
if [ ! -f "$SERVICE_FILE" ]; then
  echo "[INFO] Creating systemd service..."

  sudo tee "$SERVICE_FILE" > /dev/null <<EOF
[Unit]
Description=$APP_NAME Telegram bot
After=network.target

[Service]
Type=simple
ExecStart=$BIN_PATH
WorkingDirectory=$(pwd)
Restart=always
RestartSec=2
User=$USER
Environment=PATH=/usr/local/bin:/usr/bin:/bin

# Hard kill if shutdown hangs
TimeoutStopSec=10
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

  echo "[INFO] Enabling service..."
  sudo systemctl daemon-reload
  sudo systemctl enable "$APP_NAME"
fi

echo "[INFO] Starting service..."
sudo systemctl daemon-reload
sudo systemctl start "$APP_NAME"

echo "[INFO] Deployment complete."
echo "[INFO] Check logs with:"
echo "  journalctl -u $APP_NAME -f"
