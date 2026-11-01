package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantTx executes a function within a database transaction scoped strictly to an organization.
// It sets PostgreSQL session setting 'app.current_organization_id' using 'is_local = true'.
// This ensures that:
// 1. PostgreSQL Row-Level Security (RLS) policies enforce organization boundary.
// 2. The setting automatically expires upon transaction commit or rollback, preventing context leakage between pooled connections.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, orgID string, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Set local transaction-scoped tenant ID and enforce non-bypass RLS role
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE clientpulse_app"); err != nil {
		return fmt.Errorf("failed to set local role: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_organization_id', $1, true)", orgID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tenant transaction: %w", err)
	}

	return nil
}

// WithSystemTx executes a function within a transaction where RLS bypass is enabled locally.
// Used only for system-level maintenance, user authentication lookup across tenants, or migrations.
func WithSystemTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin system transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "SELECT set_config('app.bypass_rls', 'on', true)")
	if err != nil {
		return fmt.Errorf("failed to set system rls bypass: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit system transaction: %w", err)
	}

	return nil
}
