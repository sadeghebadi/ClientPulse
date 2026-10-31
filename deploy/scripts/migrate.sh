#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

CMD="${1:-up}"

if command -v docker >/dev/null 2>&1 && docker compose ps --services 2>/dev/null | grep -q api; then
    echo "Executing migrations via Docker Compose..."
    docker compose run --rm --entrypoint /app/migrate api "${CMD}"
else
    echo "Executing migrations via local Go binary..."
    cd "${REPO_ROOT}/services/api"
    go run ./cmd/migrate "${CMD}"
fi
