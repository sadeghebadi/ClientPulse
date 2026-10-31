# ClientPulse Operations and Runbook

## 1. Local Development Quickstart

Ensure Docker and Docker Compose are installed.

```bash
# 1. Clone repository and copy sample environment variables
cp .env.example .env

# 2. Start services using Docker Compose
docker compose up -d --build

# 3. Verify health status
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## 2. Port Allocation

To prevent port conflicts with host services (e.g. existing local PostgreSQL instances), ports can be configured via `.env`:

| Service | Internal Port | Default Host Port | Config Variable |
|---|---|---|---|
| Web (Next.js) | 3000 | 3000 | `WEB_PORT` |
| API Server | 8080 | 8080 | `API_PORT` |
| PostgreSQL | 5432 | 5433 | `POSTGRES_PORT` |
| Mailpit (SMTP) | 1025 | 1025 | `MAILPIT_SMTP_PORT` |
| Mailpit (UI) | 8025 | 8025 | `MAILPIT_UI_PORT` |

## 3. Database Migrations

Migrations are stored in `migrations/` as versioned SQL files:
`{version}_{description}.up.sql` and `{version}_{description}.down.sql`.

### Running migrations via Go CLI:
```bash
# Apply all pending migrations
go run ./services/api/cmd/migrate up

# Rollback last migration
go run ./services/api/cmd/migrate down
```

### Running migrations via Docker Compose:
```bash
docker compose run --rm api /app/migrate up
```

## 4. Backup & Restore Operations

### Daily Backup Runbook:
```bash
docker compose exec -T postgres pg_dump -U clientpulse clientpulse | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz
```

### Restore Runbook:
```bash
gunzip < backup_file.sql.gz | docker compose exec -T postgres psql -U clientpulse clientpulse
```

