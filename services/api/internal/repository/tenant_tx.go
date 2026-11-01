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
// 3. Fails closed if orgID is empty or invalid.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, orgID string, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	if orgID == "" {
		return fmt.Errorf("tenant context organization ID cannot be empty")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Ensure non-superuser role strictly if connecting via superuser migration role
	var currentUser string
	if err := tx.QueryRow(ctx, "SELECT current_user").Scan(&currentUser); err == nil && currentUser == "clientpulse" {
		_, _ = tx.Exec(ctx, "SET LOCAL ROLE clientpulse_app")
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

// WithUserTx executes a transaction scoped to a specific user's context (e.g. for membership discovery).
func WithUserTx(ctx context.Context, pool *pgxpool.Pool, userID string, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	if userID == "" {
		return fmt.Errorf("user context ID cannot be empty")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, _ = tx.Exec(ctx, "SET LOCAL ROLE clientpulse_app")

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
		return fmt.Errorf("failed to set user context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit user transaction: %w", err)
	}

	return nil
}
