#!/usr/bin/env bash
set -euo pipefail

# ClientPulse Production Backup Tool
# Creates timestamped database and local storage backups.
# Keeps 7 daily backups and logs status.
# Strictly operates within /home/ubuntu/clientpulse.

CLIENTPULSE_DIR="${CLIENTPULSE_DIR:-/home/ubuntu/clientpulse}"
BACKUP_DIR="${CLIENTPULSE_DIR}/backups"
DB_BACKUP_DIR="${BACKUP_DIR}/database"
STORAGE_BACKUP_DIR="${BACKUP_DIR}/storage"
LOG_FILE="${BACKUP_DIR}/backup.log"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
RETENTION_DAYS=7

mkdir -p "${DB_BACKUP_DIR}" "${STORAGE_BACKUP_DIR}"
chmod 700 "${BACKUP_DIR}" "${DB_BACKUP_DIR}" "${STORAGE_BACKUP_DIR}"

log() {
    local msg="[$(date -Iseconds)] $1"
    echo "$msg"
    echo "$msg" >> "${LOG_FILE}"
}

log "Starting ClientPulse production backup (${TIMESTAMP})..."

# 1. Database backup via pg_dump in clientpulse postgres container
DB_CONTAINER=$(sudo docker ps -q -f "label=com.docker.compose.project=clientpulse" -f "label=com.docker.compose.service=postgres")
if [[ -z "${DB_CONTAINER}" ]]; then
    # Fallback to name search scoped to clientpulse
    DB_CONTAINER=$(sudo docker ps -q -f "name=clientpulse.*postgres")
fi

if [[ -z "${DB_CONTAINER}" ]]; then
    log "ERROR: ClientPulse PostgreSQL container not found or not running. Aborting database backup."
    exit 1
fi

DB_BACKUP_FILE="${DB_BACKUP_DIR}/clientpulse_db_${TIMESTAMP}.sql.gz"

# Safely extract POSTGRES_USER and POSTGRES_DB from environment or container
DB_USER=$(sudo docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${DB_CONTAINER}" | grep '^POSTGRES_USER=' | cut -d= -f2- || true)
DB_NAME=$(sudo docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${DB_CONTAINER}" | grep '^POSTGRES_DB=' | cut -d= -f2- || true)
DB_USER="${DB_USER:-clientpulse_admin}"
DB_NAME="${DB_NAME:-clientpulse}"

log "Dumping database '${DB_NAME}' from container '${DB_CONTAINER}'..."
sudo docker exec "${DB_CONTAINER}" pg_dump -U "${DB_USER}" -d "${DB_NAME}" --clean --if-exists | gzip -9 > "${DB_BACKUP_FILE}"
chmod 600 "${DB_BACKUP_FILE}"

DB_SIZE=$(du -h "${DB_BACKUP_FILE}" | cut -f1)
log "Database backup successful: ${DB_BACKUP_FILE} (${DB_SIZE})"

# 2. Storage backup (PDFs, logos)
STORAGE_BACKUP_FILE="${STORAGE_BACKUP_DIR}/clientpulse_storage_${TIMESTAMP}.tar.gz"

# Back up named Docker volume clientpulse_storage
log "Backing up clientpulse_storage volume..."
sudo docker run --rm \
    -v clientpulse_storage:/data:ro \
    alpine:3.24.3 \
    tar -czf - -C /data . > "${STORAGE_BACKUP_FILE}"

chmod 600 "${STORAGE_BACKUP_FILE}"
STORAGE_SIZE=$(du -h "${STORAGE_BACKUP_FILE}" | cut -f1)
log "Storage backup successful: ${STORAGE_BACKUP_FILE} (${STORAGE_SIZE})"

# 3. Retention policy: prune backups older than RETENTION_DAYS
log "Pruning ClientPulse backups older than ${RETENTION_DAYS} days..."
find "${DB_BACKUP_DIR}" -name "clientpulse_db_*.sql.gz" -type f -mtime +"${RETENTION_DAYS}" -delete
find "${STORAGE_BACKUP_DIR}" -name "clientpulse_storage_*.tar.gz" -type f -mtime +"${RETENTION_DAYS}" -delete

log "Backup completed successfully at $(date -Iseconds)."

