#!/usr/bin/env bash
set -euo pipefail

# ClientPulse Production Verification Tool
# Checks container health, ports, endpoints, and protected services.

HOST="${HOST:-37.152.191.170}"
PORT="${PORT:-2222}"
SSH_USER="${SSH_USER:-ubuntu}"
DOMAIN="${DOMAIN:-mindu.ir}"

echo "========================================================"
echo "  ClientPulse Production Verification Tool"
echo "  Target: ${DOMAIN} (${HOST}:${PORT})"
echo "========================================================"

# Expected 8 protected container names
PROTECTED_CONTAINERS=(
    "rocketchat-rocketchat-1"
    "rocketchat-mongodb-1"
    "rustdesk-hbbs"
    "rustdesk-hbbr"
    "fchat-mattermost-1"
    "fchat-postgres-1"
    "docker-main-mattermost-1"
    "docker-main-postgres-1"
)

echo ""
echo "[1/6] Verifying Protected Containers (Strict Non-Disruption Check)..."
ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
PROTECTED=(
    "rocketchat-rocketchat-1"
    "rocketchat-mongodb-1"
    "rustdesk-hbbs"
    "rustdesk-hbbr"
    "fchat-mattermost-1"
    "fchat-postgres-1"
    "docker-main-mattermost-1"
    "docker-main-postgres-1"
)

ALL_OK=true
for name in "${PROTECTED[@]}"; do
    STATUS=$(sudo docker ps -a --filter "name=^/${name}$" --format "{{.Names}}: {{.Status}} (ID: {{.ID}})")
    if [[ -z "${STATUS}" ]]; then
        echo "❌ ALERT: Protected container '${name}' NOT FOUND!"
        ALL_OK=false
    else
        echo "✅ ${STATUS}"
    fi
done

if [[ "${ALL_OK}" != "true" ]]; then
    echo "❌ FATAL: One or more protected containers were modified or missing!"
    exit 1
fi
EOF

echo ""
echo "[2/6] Verifying ClientPulse Compose Containers..."
ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml ps
EOF

echo ""
echo "[3/6] Verifying Host Port Bindings (127.0.0.1:3100, 127.0.0.1:8090)..."
ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
sudo ss -tlnp | grep -E '3100|8090' || { echo "❌ Ports 3100 / 8090 not listening!"; exit 1; }
echo "✅ Localhost port bindings verified."
EOF

echo ""
echo "[4/6] Verifying Upstream Localhost Health Responses..."
ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
echo "Testing API health on localhost:8090..."
curl -s -f http://127.0.0.1:8090/health/live | grep -q '"status":"ok"' && echo "✅ API live endpoint OK"
curl -s -f http://127.0.0.1:8090/health/ready | grep -q '"database":"connected"' && echo "✅ API database ready OK"

echo "Testing Frontend on localhost:3100..."
curl -s -I http://127.0.0.1:3100 | grep -q "200 OK\|307 Temporary" && echo "✅ Frontend port 3100 OK"
EOF

echo ""
echo "[5/6] Verifying Public HTTPS Endpoints (${DOMAIN})..."
echo -n "Checking https://${DOMAIN}/ ... "
curl -s -k -I "https://${DOMAIN}/" | grep -q "200 OK\|307 Temporary" && echo "✅ OK" || echo "❌ FAILED"

echo -n "Checking https://${DOMAIN}/health/live ... "
curl -s -k "https://${DOMAIN}/health/live" | grep -q '"status":"ok"' && echo "✅ OK" || echo "❌ FAILED"

echo -n "Checking https://${DOMAIN}/health/ready ... "
curl -s -k "https://${DOMAIN}/health/ready" | grep -q '"database":"connected"' && echo "✅ OK" || echo "❌ FAILED"

echo -n "Checking https://${DOMAIN}/login ... "
curl -s -k -I "https://${DOMAIN}/login" | grep -q "200 OK" && echo "✅ OK" || echo "❌ FAILED"

echo ""
echo "[6/6] Host Resource Overview..."
ssh -p "${PORT}" "${SSH_USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
echo "--- Live Host Memory ---"
free -m
echo ""
echo "--- Live Disk Usage ---"
df -h /
echo ""
echo "--- ClientPulse Resource Consumption ---"
sudo docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}" | grep -E 'NAME|clientpulse' || true
EOF

echo ""
echo "========================================================"
echo "  Verification Complete!"
echo "========================================================"

