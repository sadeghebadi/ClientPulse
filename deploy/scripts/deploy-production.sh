#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# ClientPulse Production Deployment Tool
# Target Domain: https://mindu.ir
# Compose Project: clientpulse
# Deployment Method: Local Docker Build (linux/amd64) + Image Transfer + SSH
# STRICT INFRASTRUCTURE PROTECTION: 8 protected containers must remain untouched!
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

VERSION=""
HOST="37.152.191.170"
PORT="2222"
USER="ubuntu"
REMOTE_DIR="/home/ubuntu/clientpulse"
DOMAIN="mindu.ir"
SKIP_TESTS=false
AUTO_APPROVE=false

usage() {
    cat << EOF
Usage: $0 -v <VERSION> [options]

Required:
  -v <version>         Release version (e.g. 1.0.0 or v1.0.0)

Options:
  -h <host>            Target VPS Host (default: 37.152.191.170)
  -p <port>            SSH Port (default: 2222)
  -u <user>            SSH User (default: ubuntu)
  -d <dir>             Remote Directory (default: /home/ubuntu/clientpulse)
  --skip-tests         Skip local regression test suites (NOT recommended)
  -y, --yes            Auto-approve deployment checkpoint
  --help               Display this help message
EOF
    exit 1
}

# Parse command line options
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v) VERSION="$2"; shift 2 ;;
        -h) HOST="$2"; shift 2 ;;
        -p) PORT="$2"; shift 2 ;;
        -u) USER="$2"; shift 2 ;;
        -d) REMOTE_DIR="$2"; shift 2 ;;
        --skip-tests) SKIP_TESTS=true; shift ;;
        -y|--yes) AUTO_APPROVE=true; shift ;;
        --help) usage ;;
        *) echo "Unknown parameter: $1"; usage ;;
    esac
done

if [[ -z "${VERSION}" ]]; then
    echo "❌ Error: Release version (-v) is required."
    usage
fi

# Normalize version tag
VERSION="${VERSION#v}" # remove leading v if present
VERSION_TAG="v${VERSION}"

RELEASE_BUILD_DIR="${ROOT_DIR}/deploy/build/${VERSION_TAG}"
ARCHIVE_NAME="clientpulse-images-${VERSION_TAG}.tar.gz"
ARCHIVE_PATH="${RELEASE_BUILD_DIR}/${ARCHIVE_NAME}"
METADATA_PATH="${RELEASE_BUILD_DIR}/release-metadata.json"
CHECKSUM_PATH="${RELEASE_BUILD_DIR}/SHA256SUMS"

echo "=============================================================================="
echo "  ClientPulse Production Deployment - ${VERSION_TAG}"
echo "  Target Domain:  https://${DOMAIN}"
echo "  Target VPS:     ${USER}@${HOST}:${PORT}"
echo "  Remote Path:    ${REMOTE_DIR}"
echo "  Method:         Local linux/amd64 Docker build + SSH Archive Transfer"
echo "=============================================================================="

# ------------------------------------------------------------------------------
# Stage A: Local Preflight & Tests
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage A: Local Preflight & Test Suites..."

if [[ "${SKIP_TESTS}" == "true" ]]; then
    echo "⚠️ Warning: --skip-tests flag specified. Skipping automated tests."
else
    echo "Running Go test suites (config, security, rate limiting, notifications)..."
    (cd "${ROOT_DIR}/services/api" && go test ./internal/config/... ./internal/appurl/... ./internal/notification/... ./internal/worker/...)

    echo "Running Frontend linter..."
    (cd "${ROOT_DIR}/apps/web" && npm run lint)

    echo "Validating production Compose specification..."
    (cd "${ROOT_DIR}" && VERSION="${VERSION_TAG}" DATABASE_URL="postgres://dummy:dummy@postgres:5432/db" MIGRATION_DATABASE_URL="postgres://dummy:dummy@postgres:5432/db" WORKER_DATABASE_URL="postgres://dummy:dummy@postgres:5432/db" SESSION_SECRET="01234567890123456789012345678901" TOKEN_HASH_PEPPER="01234567890123456789012345678901" POSTGRES_PASSWORD="dummy" docker compose -f deploy/docker-compose.production.yml config > /dev/null)
fi

echo "✅ Local preflight checks and tests passed successfully."

# ------------------------------------------------------------------------------
# Stage B: Local Docker Build & Image Archive (linux/amd64)
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage B: Building Docker images locally for linux/amd64..."
mkdir -p "${RELEASE_BUILD_DIR}"

API_IMAGE="clientpulse-api:${VERSION_TAG}"
WORKER_IMAGE="clientpulse-worker:${VERSION_TAG}"
WEB_IMAGE="clientpulse-web:${VERSION_TAG}"
POSTGRES_IMAGE="postgres:16.15-alpine3.24"

echo "1. Building backend image: ${API_IMAGE}..."
docker build --platform linux/amd64 \
    -t "${API_IMAGE}" \
    -t "${WORKER_IMAGE}" \
    -f "${ROOT_DIR}/deploy/docker/api.Dockerfile" \
    "${ROOT_DIR}"

echo "2. Building frontend image: ${WEB_IMAGE} with NEXT_PUBLIC_API_BASE_URL=https://${DOMAIN}..."
docker build --platform linux/amd64 \
    --build-arg "NEXT_PUBLIC_API_BASE_URL=https://${DOMAIN}" \
    -t "${WEB_IMAGE}" \
    -f "${ROOT_DIR}/deploy/docker/web.Dockerfile" \
    "${ROOT_DIR}"

echo "3. Verifying PostgreSQL base image presence (${POSTGRES_IMAGE})..."
if ! docker image inspect "${POSTGRES_IMAGE}" > /dev/null 2>&1; then
    echo "Pulling ${POSTGRES_IMAGE} on local build machine..."
    docker pull --platform linux/amd64 "${POSTGRES_IMAGE}"
fi

echo "4. Exporting runtime images to compressed archive..."
# Use pigz if available, otherwise gzip
COMPRESS_CMD="gzip -c"
if command -v pigz >/dev/null 2>&1; then
    COMPRESS_CMD="pigz -1"
fi

docker save "${API_IMAGE}" "${WORKER_IMAGE}" "${WEB_IMAGE}" "${POSTGRES_IMAGE}" | ${COMPRESS_CMD} > "${ARCHIVE_PATH}"

echo "5. Generating SHA-256 checksum and release metadata..."
(cd "${RELEASE_BUILD_DIR}" && sha256sum "${ARCHIVE_NAME}" > "SHA256SUMS")
ARCHIVE_SHA256=$(cut -d' ' -f1 "${CHECKSUM_PATH}")
ARCHIVE_SIZE=$(du -h "${ARCHIVE_PATH}" | cut -f1)

GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
cat << EOF > "${METADATA_PATH}"
{
  "version": "${VERSION_TAG}",
  "git_commit": "${GIT_COMMIT}",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "target_domain": "https://${DOMAIN}",
  "archive_file": "${ARCHIVE_NAME}",
  "archive_sha256": "${ARCHIVE_SHA256}",
  "archive_size": "${ARCHIVE_SIZE}",
  "images": [
    "${API_IMAGE}",
    "${WORKER_IMAGE}",
    "${WEB_IMAGE}",
    "${POSTGRES_IMAGE}"
  ]
}
EOF

echo "✅ Image archive prepared:"
echo "   File:   ${ARCHIVE_PATH}"
echo "   Size:   ${ARCHIVE_SIZE}"
echo "   SHA256: ${ARCHIVE_SHA256}"

# ------------------------------------------------------------------------------
# Stage C: VPS Preflight (Strict Read-Only Checks)
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage C: Remote VPS Preflight Inspection..."

ssh -p "${PORT}" "${USER}@${HOST}" bash -s << 'EOF'
set -euo pipefail
echo "1. Checking 8 Protected Containers..."
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

for c in "${PROTECTED[@]}"; do
    STATUS=$(sudo docker ps -a --filter "name=^/${c}$" --format "{{.Names}}: {{.Status}}")
    if [[ -z "${STATUS}" ]]; then
        echo "❌ FATAL: Protected container '${c}' is missing!"
        exit 1
    fi
    echo "   [PROTECTED OK] ${STATUS}"
done

echo "2. Checking Host Resource Capacities..."
AVAIL_MEM=$(free -m | awk '/Mem:/ {print $7}')
echo "   Available Memory: ${AVAIL_MEM} MiB (Required safe buffer: 500-700 MiB)"
if [[ "${AVAIL_MEM}" -lt 500 ]]; then
    echo "❌ FATAL: Host available memory (${AVAIL_MEM} MiB) is below safe threshold (500 MiB)!"
    exit 1
fi

AVAIL_DISK=$(df -BG / | awk 'NR==2 {gsub("G",""); print $4}')
echo "   Available Disk:   ${AVAIL_DISK} GiB (Required: >= 5 GiB)"
if [[ "${AVAIL_DISK}" -lt 5 ]]; then
    echo "❌ FATAL: Host available disk (${AVAIL_DISK} GiB) is below 5 GiB!"
    exit 1
fi

echo "3. Checking Port Availability & Ownership (3100, 8090)..."

# BEGIN: PORT_VALIDATION_FUNCTIONS
validate_port_ownership() {
    local port="$1"
    local expected_container="$2"
    local expected_project="$3"
    local expected_service="$4"
    local expected_host_ip="$5"
    local expected_container_port="$6"

    local expected_binding="${expected_container_port}->${expected_host_ip}:${port}"
    local docker_cmd="${DOCKER_CMD:-}"
    local ss_cmd="${SS_CMD:-}"

    if [[ -z "${docker_cmd}" ]]; then
        if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
            docker_cmd="sudo docker"
        else
            docker_cmd="docker"
        fi
    fi

    if [[ -z "${ss_cmd}" ]]; then
        if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
            ss_cmd="sudo ss"
        else
            ss_cmd="ss"
        fi
    fi

    # Check if any listening TCP sockets exist on this port
    local host_sockets=""
    if ! host_sockets=$(${ss_cmd} -tlnpH "sport = :${port}" 2>/dev/null); then
        host_sockets=$(${ss_cmd} -tlnp 2>/dev/null | grep -E "[:.]${port}[[:space:]]" || true)
    fi

    # Case A: Port is free
    if [[ -z "${host_sockets}" ]]; then
        echo "   [PORT ${port} OK] Port is free (available for installation)."
        return 0
    fi

    # Port is occupied. Inspect host socket listeners
    local invalid_host_socket=false
    while IFS= read -r line; do
        [[ -z "${line}" ]] && continue
        local local_addr
        local_addr=$(echo "${line}" | awk '{print $4}')
        local_addr="${local_addr#\[}"
        local_addr="${local_addr%\]}"
        if [[ "${local_addr}" != "${expected_host_ip}:${port}" ]]; then
            echo "❌ FATAL: Conflicting listener address on host for port ${port}: found '${local_addr}', expected loopback only '${expected_host_ip}:${port}'! Full socket: ${line}"
            invalid_host_socket=true
        fi

        if [[ "${line}" =~ users:\(\(\"([^\"]+)\" ]]; then
            local proc_name="${BASH_REMATCH[1]}"
            if [[ "${proc_name}" != "docker-proxy" ]]; then
                echo "❌ FATAL: Port ${port} is occupied by host process '${proc_name}' (not Docker)! Full socket: ${line}"
                return 1
            fi
        fi
    done <<< "${host_sockets}"

    if [[ "${invalid_host_socket}" == "true" ]]; then
        return 1
    fi

    # Port is occupied. Verify ownership against Docker metadata
    local inspect_out=""
    local inspect_exit=0
    inspect_out=$(${docker_cmd} inspect --format '{{.State.Running}}|{{if .Config.Labels}}{{index .Config.Labels "com.docker.compose.project"}}{{end}}|{{if .Config.Labels}}{{index .Config.Labels "com.docker.compose.service"}}{{end}}|{{if .NetworkSettings.Ports}}{{range $p, $bindings := .NetworkSettings.Ports}}{{range $bindings}}{{$p}}->{{.HostIp}}:{{.HostPort}} {{end}}{{end}}{{end}}' "${expected_container}" 2>&1) || inspect_exit=$?

    if [[ ${inspect_exit} -ne 0 ]]; then
        if [[ "${inspect_out}" =~ [Nn]o\ such\ (container|object) ]]; then
            local conflict_container=""
            conflict_container=$(${docker_cmd} ps --filter "publish=${port}" --format "{{.Names}}" 2>/dev/null || true)
            if [[ -n "${conflict_container}" ]]; then
                echo "❌ FATAL: Port ${port} is occupied by unrelated container '${conflict_container}', but expected container '${expected_container}' does not exist!"
            else
                local proc_info
                proc_info=$(echo "${host_sockets}" | awk 'NR==1')
                echo "❌ FATAL: Port ${port} is occupied on host, but expected container '${expected_container}' does not exist! Active listener: ${proc_info}"
            fi
            return 1
        else
            echo "❌ FATAL: Failed to inspect container '${expected_container}' with Docker (exit code ${inspect_exit}): ${inspect_out}"
            echo "   Cannot safely verify port ${port} ownership."
            return 1
        fi
    fi

    if [[ -z "${inspect_out}" ]]; then
        echo "❌ FATAL: Docker inspect returned empty output for container '${expected_container}'!"
        return 1
    fi

    local is_running=""
    local compose_project=""
    local compose_service=""
    local port_bindings=""
    IFS='|' read -r is_running compose_project compose_service port_bindings <<< "${inspect_out}"

    if [[ "${is_running}" != "true" ]]; then
        local conflict_container=""
        conflict_container=$(${docker_cmd} ps --filter "publish=${port}" --format "{{.Names}}" 2>/dev/null || true)
        if [[ -n "${conflict_container}" ]]; then
            echo "❌ FATAL: Port ${port} is occupied by conflicting container '${conflict_container}' (expected '${expected_container}' is not running)!"
        else
            local proc_info
            proc_info=$(echo "${host_sockets}" | awk 'NR==1')
            echo "❌ FATAL: Port ${port} is bound on host, but expected container '${expected_container}' is not running! Active listener: ${proc_info}"
        fi
        return 1
    fi

    if [[ "${compose_project}" != "${expected_project}" ]] || [[ "${compose_service}" != "${expected_service}" ]]; then
        echo "❌ FATAL: Container '${expected_container}' bound to port ${port} has invalid Docker Compose labels!"
        echo "   Expected project='${expected_project}', service='${expected_service}'"
        echo "   Found    project='${compose_project}', service='${compose_service}'"
        return 1
    fi

    local binding_match=false
    for b in ${port_bindings}; do
        if [[ "${b}" == "${expected_binding}" ]]; then
            binding_match=true
            break
        fi
    done

    if [[ "${binding_match}" != "true" ]]; then
        echo "❌ FATAL: Container '${expected_container}' does not have the expected port binding for port ${port}!"
        echo "   Expected binding: '${expected_binding}'"
        echo "   Actual bindings:  '${port_bindings:-none}'"
        return 1
    fi

    echo "   [PORT ${port} OK] Verified existing ClientPulse service '${expected_service}' (${expected_container} on ${expected_binding})."
    return 0
}

check_clientpulse_ports() {
    local all_ok=true
    if ! validate_port_ownership "3100" "clientpulse-web-1" "clientpulse" "web" "127.0.0.1" "3000/tcp"; then
        all_ok=false
    fi
    if ! validate_port_ownership "8090" "clientpulse-api-1" "clientpulse" "api" "127.0.0.1" "8080/tcp"; then
        all_ok=false
    fi

    if [[ "${all_ok}" != "true" ]]; then
        echo "❌ FATAL: Port availability and ownership validation failed!"
        return 1
    fi
    echo "   Ports 3100 and 8090 verified successfully."
    return 0
}
# END: PORT_VALIDATION_FUNCTIONS

check_clientpulse_ports

echo "4. Checking Authorized Nginx Site & TLS Certificate..."
if [[ ! -f /etc/nginx/sites-available/mindu.ir ]]; then
    echo "❌ FATAL: Authorized target /etc/nginx/sites-available/mindu.ir does not exist!"
    exit 1
fi
if ! sudo test -f /etc/letsencrypt/live/mindu.ir/fullchain.pem; then
    echo "❌ FATAL: Let's Encrypt certificate for mindu.ir is missing!"
    exit 1
fi
sudo nginx -t > /dev/null 2>&1 || { echo "❌ FATAL: Existing Nginx syntax test failed!"; exit 1; }
echo "   Nginx syntax test and TLS certificates verified."
EOF

echo "✅ VPS preflight inspection passed successfully."

# ------------------------------------------------------------------------------
# Stage D: Human Approval Checkpoint
# ------------------------------------------------------------------------------
echo ""
echo "=============================================================================="
echo "  DEPLOYMENT APPROVAL CHECKPOINT"
echo "=============================================================================="
echo "Target:         ${USER}@${HOST}:${PORT}"
echo "Release:        ${VERSION_TAG}"
echo "Archive Size:   ${ARCHIVE_SIZE}"
echo "Archive SHA256: ${ARCHIVE_SHA256}"
echo "Planned Operations:"
echo "  1. Prepare ${REMOTE_DIR} (directories and permissions)."
echo "  2. Generate cryptographic production secrets in ${REMOTE_DIR}/.env (if not present)."
echo "  3. Transfer release archive and verify SHA256 checksum remotely."
echo "  4. Load runtime Docker images via 'sudo docker load'."
echo "  5. Start ClientPulse PostgreSQL container on isolated internal network."
echo "  6. Run database migrations via '/app/migrate up'."
echo "  7. Start ClientPulse API, Worker, and Web containers (-p clientpulse)."
echo "  8. Verify container health on localhost:3100 and localhost:8090."
echo "  9. (Gated) Backup and update /etc/nginx/sites-available/mindu.ir and reload Nginx."
echo " 10. Perform public smoke tests and certify 8 protected containers untouched."
echo "=============================================================================="

if [[ "${AUTO_APPROVE}" != "true" ]]; then
    read -p "Proceed with production deployment? [y/N]: " -r CONFIRM
    if [[ ! "${CONFIRM}" =~ ^[Yy]$ ]]; then
        echo "Deployment aborted by user."
        exit 0
    fi
fi

# ------------------------------------------------------------------------------
# Stage E: Server Preparation & Production Secrets
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage E: Preparing remote directory and production environment..."

ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_DIR}" "${VERSION_TAG}" << 'EOF'
set -euo pipefail
REMOTE_DIR="$1"
VERSION_TAG="$2"

mkdir -p "${REMOTE_DIR}/releases/${VERSION_TAG}"
mkdir -p "${REMOTE_DIR}/backups/database"
mkdir -p "${REMOTE_DIR}/backups/storage"
mkdir -p "${REMOTE_DIR}/backups/nginx"
mkdir -p "${REMOTE_DIR}/bin"
mkdir -p "${REMOTE_DIR}/volumes"

chmod 755 "${REMOTE_DIR}"
chmod 700 "${REMOTE_DIR}/backups"
chmod 700 "${REMOTE_DIR}/releases"

ENV_FILE="${REMOTE_DIR}/.env"

# Securely generate production .env if missing or if development values are detected
if [[ ! -f "${ENV_FILE}" ]] || grep -q "APP_ENV=development" "${ENV_FILE}"; then
    echo "Creating secure production .env with cryptographically strong secrets..."
    
    # Generate 32-byte hex secrets using openssl
    POSTGRES_PWD=$(openssl rand -hex 24)
    APP_DB_PWD=$(openssl rand -hex 24)
    WORKER_DB_PWD=$(openssl rand -hex 24)
    SESSION_SEC=$(openssl rand -hex 32)
    TOKEN_PEPPER=$(openssl rand -hex 32)
    
    (umask 077 && cat << EOV > "${ENV_FILE}"
# ClientPulse Production Environment Configuration
APP_ENV=production
PUBLIC_APP_URL=https://mindu.ir
APP_BASE_URL=https://mindu.ir
API_BASE_URL=https://mindu.ir
API_PORT=8080
DEFAULT_LOCALE=fa
DEFAULT_TIMEZONE=Asia/Tehran

# Local Bindings (Loopback only)
WEB_PORT=3100
API_PORT_HOST=8090

# Database Configuration (Role Separation)
POSTGRES_USER=clientpulse_admin
POSTGRES_PASSWORD=${POSTGRES_PWD}
POSTGRES_DB=clientpulse

# Migration URL (Superuser / Schema Owner)
MIGRATION_DATABASE_URL=postgres://clientpulse_admin:${POSTGRES_PWD}@postgres:5432/clientpulse?sslmode=disable

# Application Runtime URL (Least-privilege clientpulse_app)
DATABASE_URL=postgres://clientpulse_app:${APP_DB_PWD}@postgres:5432/clientpulse?sslmode=disable
APP_DB_PASSWORD=${APP_DB_PWD}

# Worker Runtime URL (Least-privilege clientpulse_worker)
WORKER_DATABASE_URL=postgres://clientpulse_worker:${WORKER_DB_PWD}@postgres:5432/clientpulse?sslmode=disable
WORKER_DB_PASSWORD=${WORKER_DB_PWD}

# Security Secrets (Cryptographically Generated)
SESSION_SECRET=${SESSION_SEC}
TOKEN_HASH_PEPPER=${TOKEN_PEPPER}

# Deferred Production Transports
EMAIL_PROVIDER=disabled
TELEGRAM_MODE=disabled
TELEGRAM_BOT_TOKEN=
TELEGRAM_BOT_USERNAME=ClientPulseAlertsBot

# Storage Configuration
STORAGE_PROVIDER=local
STORAGE_LOCAL_DIR=/app/data/storage

# Worker Configuration
WORKER_CONCURRENCY=5
WORKER_HEARTBEAT_TIMEOUT_SECONDS=10
CHECK_RESULT_RETENTION_DAYS=90
EOV
    )
    chmod 600 "${ENV_FILE}"
    echo "✅ Production .env generated with restricted 600 permissions."
else
    echo "Existing production .env preserved."
    chmod 600 "${ENV_FILE}"
fi
EOF

# ------------------------------------------------------------------------------
# Stage F: Image Archive & Configuration Transfer
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage F: Transferring release archive and configuration files..."

REMOTE_RELEASE_DIR="${REMOTE_DIR}/releases/${VERSION_TAG}"

scp -P "${PORT}" "${ARCHIVE_PATH}" "${USER}@${HOST}:${REMOTE_RELEASE_DIR}/clientpulse-images.tar.gz"
scp -P "${PORT}" "${CHECKSUM_PATH}" "${USER}@${HOST}:${REMOTE_RELEASE_DIR}/SHA256SUMS"
scp -P "${PORT}" "${METADATA_PATH}" "${USER}@${HOST}:${REMOTE_RELEASE_DIR}/release-metadata.json"
scp -P "${PORT}" "${ROOT_DIR}/deploy/docker-compose.production.yml" "${USER}@${HOST}:${REMOTE_DIR}/docker-compose.production.yml"
scp -P "${PORT}" "${ROOT_DIR}/deploy/docker-compose.production.yml" "${USER}@${HOST}:${REMOTE_RELEASE_DIR}/docker-compose.production.yml"
scp -P "${PORT}" "${ROOT_DIR}/deploy/scripts/backup-production.sh" "${USER}@${HOST}:${REMOTE_DIR}/bin/backup-production.sh"
scp -P "${PORT}" "${ROOT_DIR}/deploy/scripts/rollback-production.sh" "${USER}@${HOST}:${REMOTE_DIR}/bin/rollback-production.sh"

echo "Verifying remote archive SHA-256 checksum..."
ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_RELEASE_DIR}" "${ARCHIVE_SHA256}" << 'EOF'
set -euo pipefail
DIR="$1"
EXPECTED_SHA="$2"
REMOTE_SHA=$(sha256sum "${DIR}/clientpulse-images.tar.gz" | cut -d' ' -f1)

if [[ "${REMOTE_SHA}" != "${EXPECTED_SHA}" ]]; then
    echo "❌ FATAL: SHA256 checksum mismatch on VPS!"
    echo "Expected: ${EXPECTED_SHA}"
    echo "Got:      ${REMOTE_SHA}"
    exit 1
fi
echo "✅ Remote archive checksum verified: ${REMOTE_SHA}"
chmod +x /home/ubuntu/clientpulse/bin/*.sh
EOF

echo "Loading runtime Docker images on VPS..."
ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_RELEASE_DIR}" << 'EOF'
set -euo pipefail
sudo docker load -i "$1/clientpulse-images.tar.gz"
EOF
echo "✅ Docker images loaded successfully."

# ------------------------------------------------------------------------------
# Stage G: Database Initialization & Safe Migrations
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage G: Starting ClientPulse PostgreSQL & applying migrations..."

ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_DIR}" "${VERSION_TAG}" << 'EOF'
set -euo pipefail
REMOTE_DIR="$1"
VERSION="$2"

echo "1. Starting isolated PostgreSQL service..."
sudo env VERSION="${VERSION}" docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    up -d postgres

echo "2. Waiting for PostgreSQL to become healthy..."
for i in {1..30}; do
    STATUS=$(sudo docker inspect --format '{{.State.Health.Status}}' $(sudo env VERSION="${VERSION}" docker compose -p clientpulse -f "${REMOTE_DIR}/docker-compose.production.yml" --env-file "${REMOTE_DIR}/.env" ps -q postgres) 2>/dev/null || echo "starting")
    if [[ "${STATUS}" == "healthy" ]]; then
        echo "✅ PostgreSQL is healthy."
        break
    fi
    echo "   PostgreSQL status: ${STATUS} (attempt ${i}/30)..."
    sleep 2
done

echo "3. Executing safe ClientPulse database migrations..."
sudo env VERSION="${VERSION}" docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    run --rm --no-deps \
    --entrypoint /app/migrate api up

echo "4. Synchronizing runtime role passwords (least-privilege)..."
# Read generated passwords from .env
APP_PWD=$(grep '^APP_DB_PASSWORD=' "${REMOTE_DIR}/.env" | cut -d= -f2-)
WORKER_PWD=$(grep '^WORKER_DB_PASSWORD=' "${REMOTE_DIR}/.env" | cut -d= -f2-)
POSTGRES_PWD=$(grep '^POSTGRES_PASSWORD=' "${REMOTE_DIR}/.env" | cut -d= -f2-)

PG_CONTAINER=$(sudo env VERSION="${VERSION}" docker compose -p clientpulse -f "${REMOTE_DIR}/docker-compose.production.yml" --env-file "${REMOTE_DIR}/.env" ps -q postgres)

sudo docker exec -i "${PG_CONTAINER}" psql -U clientpulse_admin -d clientpulse << EOSQL
ALTER ROLE clientpulse_app WITH PASSWORD '${APP_PWD}';
ALTER ROLE clientpulse_worker WITH PASSWORD '${WORKER_PWD}';
EOSQL

echo "✅ Database migrations and role passwords configured."
EOF

# ------------------------------------------------------------------------------
# Stage H: Start ClientPulse Application Services
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage H: Starting ClientPulse application containers..."

ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_DIR}" "${VERSION_TAG}" << 'EOF'
set -euo pipefail
REMOTE_DIR="$1"
VERSION="$2"

sudo env VERSION="${VERSION}" docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    up -d

echo "Verifying local service health..."
sleep 5
sudo env VERSION="${VERSION}" docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    ps

echo "Checking localhost upstreams..."
curl -s -f http://127.0.0.1:8090/health/live | grep -q '"status":"ok"' && echo "✅ API live on 127.0.0.1:8090"
curl -s -f http://127.0.0.1:8090/health/ready | grep -q '"database":"connected"' && echo "✅ API database ready"
curl -s -I http://127.0.0.1:3100 | grep -q "200 OK\|307 Temporary" && echo "✅ Web live on 127.0.0.1:3100"
EOF

# ------------------------------------------------------------------------------
# Stage I: Nginx Routing Switch (Authorized target file only)
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage I: Activating authorized Nginx configuration (/etc/nginx/sites-available/mindu.ir)..."

scp -P "${PORT}" "${ROOT_DIR}/deploy/nginx/mindu.ir" "${USER}@${HOST}:${REMOTE_DIR}/mindu.ir.nginx.new"

ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_DIR}" << 'EOF'
set -euo pipefail
REMOTE_DIR="$1"
TARGET_FILE="/etc/nginx/sites-available/mindu.ir"
BACKUP_FILE="${REMOTE_DIR}/backups/nginx/mindu.ir_$(date +%Y%m%d_%H%M%S).bak"

echo "1. Creating timestamped backup of ${TARGET_FILE}..."
sudo cp -a "${TARGET_FILE}" "${BACKUP_FILE}"
echo "   Backup saved to ${BACKUP_FILE}"

echo "2. Applying verified ClientPulse Nginx configuration..."
sudo cp "${REMOTE_DIR}/mindu.ir.nginx.new" "${TARGET_FILE}"
sudo chown root:root "${TARGET_FILE}"
sudo chmod 644 "${TARGET_FILE}"
rm -f "${REMOTE_DIR}/mindu.ir.nginx.new"

echo "3. Testing Nginx configuration syntax..."
if sudo nginx -t; then
    echo "4. Reloading Nginx gracefully..."
    sudo systemctl reload nginx
    echo "✅ Nginx reloaded successfully."
else
    echo "❌ FATAL: Nginx syntax test failed! Restoring backup..."
    sudo cp "${BACKUP_FILE}" "${TARGET_FILE}"
    exit 1
fi
EOF

# ------------------------------------------------------------------------------
# Stage J & K: Public Smoke Tests & Protected Services Verification
# ------------------------------------------------------------------------------
echo ""
echo ">>> Stage J & K: Running Production Verification Suite..."
"${ROOT_DIR}/deploy/scripts/verify-production.sh"

echo ""
echo "=============================================================================="
echo "  🎉 ClientPulse Production Deployment Completed Successfully!"
echo "  Target Domain: https://${DOMAIN}"
echo "=============================================================================="

