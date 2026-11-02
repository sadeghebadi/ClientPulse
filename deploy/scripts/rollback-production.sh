#!/usr/bin/env bash
set -euo pipefail

# ClientPulse Production Rollback Tool
# Rolls back ClientPulse to a previous release archive.
# Strictly scoped to Compose project 'clientpulse'.
# Preserves production .env and database volumes.
# NEVER drops schemas or runs down migrations automatically.

HOST="${HOST:-37.152.191.170}"
PORT="${PORT:-2222}"
SSH_USER="${SSH_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/clientpulse}"
TARGET_VERSION=""

usage() {
    echo "Usage: $0 -v <TARGET_VERSION> [options]"
    echo "Options:"
    echo "  -v <version>       Release version to rollback to (required)"
    echo "  -h <host>          VPS Host (default: 37.152.191.170)"
    echo "  -p <port>          SSH Port (default: 2222)"
    echo "  -u <user>          SSH User (default: ubuntu)"
    echo "  --restore-nginx    Also restore latest Nginx backup"
    exit 1
}

RESTORE_NGINX=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v) TARGET_VERSION="$2"; shift 2 ;;
        -h) HOST="$2"; shift 2 ;;
        -p) PORT="$2"; shift 2 ;;
        -u) SSH_USER="$2"; shift 2 ;;
        --restore-nginx) RESTORE_NGINX=true; shift ;;
        *) usage ;;
    esac
done

if [[ -z "${TARGET_VERSION}" ]]; then
    echo "❌ Error: Target release version (-v) is required."
    usage
fi

echo "========================================================"
echo "  ClientPulse Production Rollback"
echo "  Target Release: ${TARGET_VERSION}"
echo "  Target VPS:     ${SSH_USER}@${HOST}:${PORT}"
echo "========================================================"

ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s -- "${TARGET_VERSION}" "${RESTORE_NGINX}" "${REMOTE_DIR}" << 'EOF'
set -euo pipefail
VERSION="$1"
RESTORE_NGINX="$2"
REMOTE_DIR="$3"
RELEASE_DIR="${REMOTE_DIR}/releases/${VERSION}"

echo "[1/5] Checking release directory for version ${VERSION}..."
if [[ ! -d "${RELEASE_DIR}" ]]; then
    echo "❌ ERROR: Release directory ${RELEASE_DIR} does not exist!"
    echo "Available releases in ${REMOTE_DIR}/releases/:"
    ls -la "${REMOTE_DIR}/releases" || true
    exit 1
fi

echo "[2/5] Loading Docker images for release ${VERSION}..."
if [[ -f "${RELEASE_DIR}/clientpulse-images.tar.gz" ]]; then
    sudo docker load -i "${RELEASE_DIR}/clientpulse-images.tar.gz"
elif [[ -f "${RELEASE_DIR}/clientpulse-images-${VERSION}.tar.gz" ]]; then
    sudo docker load -i "${RELEASE_DIR}/clientpulse-images-${VERSION}.tar.gz"
else
    echo "⚠️ Warning: Archive file not found in ${RELEASE_DIR}. Checking if images are already loaded..."
fi

echo "[3/5] Restoring Compose configuration for ${VERSION}..."
if [[ -f "${RELEASE_DIR}/docker-compose.production.yml" ]]; then
    cp "${RELEASE_DIR}/docker-compose.production.yml" "${REMOTE_DIR}/docker-compose.production.yml"
fi

echo "[4/5] Recreating ClientPulse containers with target release..."
sudo docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    up -d --force-recreate

echo "[5/5] Checking container status..."
sudo docker compose -p clientpulse -f "${REMOTE_DIR}/docker-compose.production.yml" ps

if [[ "${RESTORE_NGINX}" == "true" ]]; then
    echo "Restoring latest Nginx configuration backup..."
    LATEST_NGINX_BACKUP=$(ls -t "${REMOTE_DIR}/backups/nginx/"*mindu.ir* 2>/dev/null | head -n1 || true)
    if [[ -n "${LATEST_NGINX_BACKUP}" && -f "${LATEST_NGINX_BACKUP}" ]]; then
        echo "Restoring ${LATEST_NGINX_BACKUP} -> /etc/nginx/sites-available/mindu.ir"
        sudo cp "${LATEST_NGINX_BACKUP}" /etc/nginx/sites-available/mindu.ir
        sudo nginx -t && sudo systemctl reload nginx
        echo "✅ Nginx restored and reloaded."
    else
        echo "⚠️ No Nginx backup found in ${REMOTE_DIR}/backups/nginx/."
    fi
fi

echo "✅ Rollback to ${VERSION} completed successfully!"
EOF

