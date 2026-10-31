# Multi-stage Dockerfile for ClientPulse API service
FROM golang:1.23-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY services/api/go.mod services/api/go.sum ./
RUN go mod download

COPY services/api/ ./

# Build statically linked binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate

# Production Runner
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S clientpulse && adduser -S clientpulse -G clientpulse

COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate
COPY migrations /app/migrations

USER clientpulse

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health/live || exit 1

ENTRYPOINT ["/app/api"]

