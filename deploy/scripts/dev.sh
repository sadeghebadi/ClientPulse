#!/usr/bin/env bash
set -euo pipefail

echo "=== ClientPulse Development Helper ==="
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

case "${1:-up}" in
    up)
        echo "Starting Docker Compose services..."
        docker compose up -d --build
        echo "Checking health endpoints..."
        sleep 3
        curl -s http://localhost:8080/health/live || true
        echo ""
        ;;
    down)
        echo "Stopping Docker Compose services..."
        docker compose down
        ;;
    logs)
        docker compose logs -f "${2:-}"
        ;;
    migrate)
        echo "Running database migrations..."
        docker compose run --rm api /app/migrate "${2:-up}"
        ;;
    test)
        echo "Running automated tests..."
        (cd services/api && go test -v ./...)
        (cd apps/web && npm run lint)
        ;;
    *)
        echo "Usage: $0 {up|down|logs|migrate|test}"
        exit 1
        ;;
esac

