#!/usr/bin/env bash
set -e

APP_NAME="dayswithout"
BUILD_DIR="build"
BIN_PATH="/usr/local/bin/$APP_NAME"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"

echo "[INFO] Running build.sh..."
./build.sh

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
ExecStart=$BIN_PATH
WorkingDirectory=$(pwd)
Restart=always
User=$USER
Environment=PATH=/usr/local/bin:/usr/bin:/bin
# Optionally set env vars if needed:
# Environment=BOT_TOKEN=...

[Install]
WantedBy=multi-user.target
EOF

  echo "[INFO] Enabling service..."
  sudo systemctl daemon-reload
  sudo systemctl enable $APP_NAME
fi

echo "[INFO] Restarting service..."
sudo systemctl daemon-reload
sudo systemctl restart $APP_NAME

echo "[INFO] Deployment complete. Check logs with:"
echo "  journalctl -u $APP_NAME -f"
