package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationLockID is the unique PostgreSQL advisory lock ID for ClientPulse migrations.
const MigrationLockID = 982347102

// Migration represents a versioned migration pair.
type Migration struct {
	Version      int64
	Name         string
	UpFilePath   string
	DownFilePath string
	UpSQL        string
	DownSQL      string
}

// MigrationStatus reports the status of a specific migration version.
type MigrationStatus struct {
	Version   int64      `json:"version"`
	Name      string     `json:"name"`
	Applied   bool       `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// Engine executes versioned SQL migrations using PostgreSQL connection-scoped advisory locking.
type Engine struct {
	pool           *pgxpool.Pool
	migrationsPath string
}

// New creates a new migration engine.
func New(pool *pgxpool.Pool, migrationsPath string) *Engine {
	return &Engine{
		pool:           pool,
		migrationsPath: migrationsPath,
	}
}

// EnsureTable creates the schema_migrations table using the pool if it doesn't already exist.
func (e *Engine) EnsureTable(ctx context.Context) error {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for EnsureTable: %w", err)
	}
	defer conn.Release()

	return e.ensureTableOnConn(ctx, conn)
}

func (e *Engine) ensureTableOnConn(ctx context.Context, conn *pgxpool.Conn) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	_, err := conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// LoadMigrations reads and parses migration files from the configured directory.
func (e *Engine) LoadMigrations() ([]Migration, error) {
	files, err := os.ReadDir(e.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory %s: %w", e.migrationsPath, err)
	}

	re := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)
	migrationsMap := make(map[int64]*Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := re.FindStringSubmatch(file.Name())
		if len(matches) != 4 {
			continue
		}

		ver, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}

		name := matches[2]
		direction := matches[3]
		fullPath := filepath.Join(e.migrationsPath, file.Name())

		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		m, exists := migrationsMap[ver]
		if !exists {
			m = &Migration{
				Version: ver,
				Name:    name,
			}
			migrationsMap[ver] = m
		}

		if direction == "up" {
			m.UpFilePath = fullPath
			m.UpSQL = string(content)
		} else if direction == "down" {
			m.DownFilePath = fullPath
			m.DownSQL = string(content)
		}
	}

	var list []Migration
	for _, m := range migrationsMap {
		list = append(list, *m)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})

	return list, nil
}

// GetApplied returns a map of applied versions and timestamps using a dedicated connection.
func (e *Engine) GetApplied(ctx context.Context) (map[int64]time.Time, error) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection for GetApplied: %w", err)
	}
	defer conn.Release()

	return e.getAppliedOnConn(ctx, conn)
}

func (e *Engine) getAppliedOnConn(ctx context.Context, conn *pgxpool.Conn) (map[int64]time.Time, error) {
	if err := e.ensureTableOnConn(ctx, conn); err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, "SELECT version, applied_at FROM schema_migrations ORDER BY version ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]time.Time)
	for rows.Next() {
		var ver int64
		var appliedAt time.Time
		if err := rows.Scan(&ver, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[ver] = appliedAt
	}

	return applied, nil
}

// Up applies all pending migrations in sequential order.
// Crucially, it acquires a single dedicated database connection from the pool,
// acquires a session-level advisory lock on that exact connection, executes all
// migrations on that same connection, and unlocks/releases in a guaranteed defer block.
func (e *Engine) Up(ctx context.Context) (int, error) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire connection for migration Up: %w", err)
	}
	defer conn.Release()

	// Acquire session-level advisory lock on this specific connection
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", MigrationLockID); err != nil {
		return 0, fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	defer func() {
		// Use detached context with short timeout to ensure unlock completes even if ctx was cancelled
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", MigrationLockID)
	}()

	if err := e.ensureTableOnConn(ctx, conn); err != nil {
		return 0, err
	}

	migrations, err := e.LoadMigrations()
	if err != nil {
		return 0, err
	}

	applied, err := e.getAppliedOnConn(ctx, conn)
	if err != nil {
		return 0, err
	}

	appliedCount := 0
	for _, m := range migrations {
		if _, alreadyApplied := applied[m.Version]; alreadyApplied {
			continue
		}

		if strings.TrimSpace(m.UpSQL) == "" {
			return appliedCount, fmt.Errorf("migration %06d_%s has empty up SQL", m.Version, m.Name)
		}

		// Execute in transaction ON THIS SAME CONNECTION
		tx, err := conn.Begin(ctx)
		if err != nil {
			return appliedCount, fmt.Errorf("failed to begin transaction for migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			_ = tx.Rollback(ctx)
			return appliedCount, fmt.Errorf("failed executing migration %06d_%s: %w", m.Version, m.Name, err)
		}

		insertQuery := "INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, NOW())"
		if _, err := tx.Exec(ctx, insertQuery, m.Version, m.Name); err != nil {
			_ = tx.Rollback(ctx)
			return appliedCount, fmt.Errorf("failed recording migration %06d_%s: %w", m.Version, m.Name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return appliedCount, fmt.Errorf("failed to commit transaction for migration %06d_%s: %w", m.Version, m.Name, err)
		}

		appliedCount++
	}

	return appliedCount, nil
}

// Down rolls back the most recent applied migration using the same acquired connection session.
func (e *Engine) Down(ctx context.Context) (int64, error) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire connection for migration Down: %w", err)
	}
	defer conn.Release()

	// Acquire session-level advisory lock on this specific connection
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", MigrationLockID); err != nil {
		return 0, fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", MigrationLockID)
	}()

	if err := e.ensureTableOnConn(ctx, conn); err != nil {
		return 0, err
	}

	migrations, err := e.LoadMigrations()
	if err != nil {
		return 0, err
	}

	applied, err := e.getAppliedOnConn(ctx, conn)
	if err != nil {
		return 0, err
	}

	if len(applied) == 0 {
		return 0, nil // Nothing to roll back
	}

	// Find the highest applied version
	var latestVersion int64 = -1
	for v := range applied {
		if v > latestVersion {
			latestVersion = v
		}
	}

	var targetMigration *Migration
	for _, m := range migrations {
		if m.Version == latestVersion {
			targetMigration = &m
			break
		}
	}

	if targetMigration == nil {
		return 0, fmt.Errorf("migration file for applied version %d not found on disk", latestVersion)
	}

	if strings.TrimSpace(targetMigration.DownSQL) == "" {
		return 0, fmt.Errorf("migration %06d_%s has empty down SQL", targetMigration.Version, targetMigration.Name)
	}

	// Execute rollback in transaction ON THIS SAME CONNECTION
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, targetMigration.DownSQL); err != nil {
		_ = tx.Rollback(ctx)
		return 0, fmt.Errorf("failed rolling back migration %06d_%s: %w", targetMigration.Version, targetMigration.Name, err)
	}

	deleteQuery := "DELETE FROM schema_migrations WHERE version = $1"
	if _, err := tx.Exec(ctx, deleteQuery, targetMigration.Version); err != nil {
		_ = tx.Rollback(ctx)
		return 0, fmt.Errorf("failed deleting migration record %06d_%s: %w", targetMigration.Version, targetMigration.Name, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	return latestVersion, nil
}

// Status returns the list of all known migrations and whether they have been applied.
func (e *Engine) Status(ctx context.Context) ([]MigrationStatus, error) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection for Status: %w", err)
	}
	defer conn.Release()

	migrations, err := e.LoadMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := e.getAppliedOnConn(ctx, conn)
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, m := range migrations {
		appliedAt, isApplied := applied[m.Version]
		status := MigrationStatus{
			Version: m.Version,
			Name:    m.Name,
			Applied: isApplied,
		}
		if isApplied {
			status.AppliedAt = &appliedAt
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}
