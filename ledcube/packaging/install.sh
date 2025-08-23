#!/usr/bin/env bash
set -euo pipefail
DEST=/opt/ledcube
sudo mkdir -p "$DEST"
sudo cp -r bin ledcube web README.md config.yaml.sample docs "$DEST" 2>/dev/null || true
# Copy server binary if present
if [ -f bin/ledcube ]; then sudo cp bin/ledcube "$DEST/"; fi
# Install systemd unit
sudo cp packaging/ledcube.service /etc/systemd/system/ledcube.service
sudo systemctl daemon-reload
echo "To enable at boot: sudo systemctl enable --now ledcube"
