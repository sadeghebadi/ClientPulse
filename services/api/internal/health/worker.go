package health

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkerHeartbeat encapsulates heartbeat metadata for a worker process.
type WorkerHeartbeat struct {
	WorkerID        string    `json:"worker_id"`
	Hostname        string    `json:"hostname"`
	PID             int       `json:"pid"`
	StartedAt       time.Time `json:"started_at"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
	Status          string    `json:"status"`
}

// WorkerHeartbeatChecker verifies the health of worker instances.
type WorkerHeartbeatChecker interface {
	GetLatestHeartbeat(ctx context.Context) (*WorkerHeartbeat, error)
}

// WorkerHeartbeatRecorder manages a worker's heartbeat lifecycle.
type WorkerHeartbeatRecorder interface {
	Register(ctx context.Context, workerID, hostname string, pid int) error
	Heartbeat(ctx context.Context, workerID string) error
	Stop(ctx context.Context, workerID string) error
}

// PGWorkerHeartbeat implements both WorkerHeartbeatChecker and WorkerHeartbeatRecorder using PostgreSQL.
type PGWorkerHeartbeat struct {
	pool *pgxpool.Pool
}

// NewPGWorkerHeartbeat constructs a new PostgreSQL worker heartbeat manager.
func NewPGWorkerHeartbeat(pool *pgxpool.Pool) *PGWorkerHeartbeat {
	return &PGWorkerHeartbeat{pool: pool}
}

// Register registers or reactivates a worker instance.
func (p *PGWorkerHeartbeat) Register(ctx context.Context, workerID, hostname string, pid int) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO worker_heartbeats (worker_id, hostname, pid, started_at, last_heartbeat_at, status)
		VALUES ($1, $2, $3, NOW(), NOW(), 'active')
		ON CONFLICT (worker_id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			pid = EXCLUDED.pid,
			started_at = NOW(),
			last_heartbeat_at = NOW(),
			status = 'active';
	`
	_, err := p.pool.Exec(ctx, query, workerID, hostname, pid)
	if err != nil {
		return fmt.Errorf("failed to register worker heartbeat: %w", err)
	}
	return nil
}

// Heartbeat refreshes the last_heartbeat_at timestamp.
func (p *PGWorkerHeartbeat) Heartbeat(ctx context.Context, workerID string) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE worker_heartbeats
		SET last_heartbeat_at = NOW(), status = 'active'
		WHERE worker_id = $1;
	`
	tag, err := p.pool.Exec(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return p.Register(ctx, workerID, "unknown", 0)
	}
	return nil
}

// Stop marks the worker instance as stopped upon graceful termination.
func (p *PGWorkerHeartbeat) Stop(ctx context.Context, workerID string) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE worker_heartbeats
		SET status = 'stopped', last_heartbeat_at = NOW()
		WHERE worker_id = $1;
	`
	_, err := p.pool.Exec(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("failed to mark worker stopped: %w", err)
	}
	return nil
}

// GetLatestHeartbeat fetches the most recent active worker heartbeat.
func (p *PGWorkerHeartbeat) GetLatestHeartbeat(ctx context.Context) (*WorkerHeartbeat, error) {
	if p == nil || p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT worker_id, hostname, pid, started_at, last_heartbeat_at, status
		FROM worker_heartbeats
		WHERE status = 'active'
		ORDER BY last_heartbeat_at DESC
		LIMIT 1;
	`
	row := p.pool.QueryRow(ctx, query)
	var hb WorkerHeartbeat
	err := row.Scan(&hb.WorkerID, &hb.Hostname, &hb.PID, &hb.StartedAt, &hb.LastHeartbeatAt, &hb.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No active worker found
		}
		return nil, fmt.Errorf("failed querying worker heartbeats: %w", err)
	}
	return &hb, nil
}

