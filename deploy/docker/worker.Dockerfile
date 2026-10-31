# Multi-stage Dockerfile for ClientPulse Worker service
FROM golang:1.23-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY services/api/go.mod services/api/go.sum ./
RUN go mod download

COPY services/api/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/worker ./cmd/worker

# Production Runner
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S clientpulse && adduser -S clientpulse -G clientpulse

COPY --from=builder /bin/worker /app/worker

USER clientpulse

ENTRYPOINT ["/app/worker"]
