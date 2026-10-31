# ClientPulse API Specification & Guidelines

## 1. Conventions

- **Base URL:** `/api/v1` for business APIs; `/health/*` for infrastructure probes.
- **Protocol:** HTTP/1.1 or HTTP/2 over TLS in production.
- **Request / Response Format:** `application/json; charset=utf-8`.
- **Date/Time:** ISO 8601 UTC format (e.g. `2026-10-31T14:00:00Z`).
- **Correlation ID:** Every request receives or generates an `X-Request-ID` header, returned in responses and attached to log entries.

## 2. Standard Error Response

All API errors return a standardized JSON structure:

```json
{
  "error": {
    "code": "ERROR_CODE_CONSTANT",
    "message": "Human readable error description",
    "request_id": "c1f7a022-8b4e-4f51-b84e-289c029143ef",
    "details": {}
  }
}
```

Common error codes:
- `VALIDATION_FAILED` (400)
- `UNAUTHORIZED` (401)
- `FORBIDDEN` (403)
- `NOT_FOUND` (404)
- `RATE_LIMIT_EXCEEDED` (429)
- `INTERNAL_ERROR` (500)
- `SERVICE_UNAVAILABLE` (503)

## 3. Foundation Endpoints (Phase 0)

### Liveness Probe
`GET /health/live`
- Returns 200 OK when the HTTP server process is running and accepting connections.
- Response:
```json
{
  "status": "ok",
  "service": "clientpulse-api",
  "version": "1.0.0",
  "timestamp": "2026-10-31T14:00:00Z"
}
```

### Readiness Probe
`GET /health/ready`
- Checks connectivity to downstream dependencies (PostgreSQL database pool).
- Returns 200 OK when healthy, or 503 Service Unavailable if downstream dependencies fail.
- Response (200):
```json
{
  "status": "ok",
  "service": "clientpulse-api",
  "database": "connected",
  "timestamp": "2026-10-31T14:00:00Z"
}
```

### Worker Health Probe
`GET /health/worker`
- Queries persistent worker heartbeats stored in PostgreSQL (`worker_heartbeats`).
- Returns HTTP 200 OK when an active worker heartbeat was recorded within `WORKER_HEARTBEAT_TIMEOUT_SECONDS` (default: 10s).
- Returns HTTP 503 Service Unavailable when the worker heartbeat is stale, missing, or stopped.

Healthy Response (200 OK):
```json
{
  "status": "ok",
  "service": "clientpulse-worker",
  "worker_id": "worker-3e7deaa90cac-1",
  "hostname": "3e7deaa90cac",
  "last_heartbeat_at": "2026-10-31T15:43:46Z",
  "age_seconds": 0.16,
  "timeout_seconds": 10,
  "timestamp": "2026-10-31T15:43:46Z"
}
```

Unhealthy Response (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "service": "clientpulse-worker",
  "worker_id": "worker-3e7deaa90cac-1",
  "hostname": "3e7deaa90cac",
  "last_heartbeat_at": "2026-10-31T15:27:56Z",
  "age_seconds": 30.03,
  "timeout_seconds": 10,
  "timestamp": "2026-10-31T15:28:26Z",
  "error": "worker heartbeat is stale"
}
```

