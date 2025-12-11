#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${REPO_ROOT}/build"
BIN_NAME="cf-ip-guard"
OUT_BIN="${BUILD_DIR}/${BIN_NAME}"

mkdir -p "${BUILD_DIR}"

echo "[build] go build -o ${OUT_BIN} ."
GO111MODULE=on go build -o "${OUT_BIN}" .

echo "[install] sudo install -m 0755 ${OUT_BIN} /usr/local/bin/${BIN_NAME}"
sudo install -m 0755 "${OUT_BIN}" "/usr/local/bin/${BIN_NAME}"

echo "[install] sudo install -m 0644 deploy/${BIN_NAME}.service /etc/systemd/system/${BIN_NAME}.service"
sudo install -m 0644 "${REPO_ROOT}/deploy/${BIN_NAME}.service" "/etc/systemd/system/${BIN_NAME}.service"

if [[ ! -f /etc/cf-ip-guard.env ]]; then
  echo "[init] create /etc/cf-ip-guard.env (editable flags)"
  sudo tee /etc/cf-ip-guard.env >/dev/null <<'EOF'
# Additional CLI flags for cf-ip-guard daemon.
# Example: CF_IP_GUARD_OPTS="--ipset4 cloudflare4 --ipset6 cloudflare6 --interval 30m --log-level info"
CF_IP_GUARD_OPTS=""
EOF
fi

echo "[systemd] daemon-reload"
sudo systemctl daemon-reload

echo "[systemd] enable cf-ip-guard (will start on boot)"
sudo systemctl enable cf-ip-guard

echo "[systemd] start/restart cf-ip-guard now"
sudo systemctl restart cf-ip-guard

echo "done. adjust /etc/cf-ip-guard.env and restart with: sudo systemctl restart cf-ip-guard"


