package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"clientpulse/services/api/internal/database"
)

// TestMigration0013_Reproducibility tests both:
// Scenario 1: Fresh Database - applying the full migration chain (000001 through 000013).
// Scenario 2: Existing Database with Emergency Policy - reconciling the emergency policy created during TASK-INCIDENT-001.
func TestMigration0013_Reproducibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adminDBURL := os.Getenv("TEST_DATABASE_URL")
	if adminDBURL == "" {
		adminDBURL = "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"
	}

	adminDB, err := database.New(ctx, database.Config{DatabaseURL: adminDBURL})
	if err != nil {
		t.Skip("skipping reproducibility test: database not reachable at", adminDBURL)
		return
	}
	defer adminDB.Close()

	// Locate migrations directory relative to test execution
	// Usually tests run with working directory set to services/api/internal/migrate
	// Try finding the directory ../../../../migrations
	migDirCandidates := []string{
		"../../../../migrations",
		"../../../migrations",
		"../../migrations",
		"./migrations",
	}
	var migDir string
	for _, candidate := range migDirCandidates {
		absPath, err := filepath.Abs(candidate)
		if err == nil {
			if _, err := os.Stat(filepath.Join(absPath, "000013_assets_scheduler_rls_hardening.up.sql")); err == nil {
				migDir = absPath
				break
			}
		}
	}
	if migDir == "" {
		t.Fatalf("could not locate migrations directory containing 000013_assets_scheduler_rls_hardening.up.sql")
	}

	// Helper to create and drop a fresh database
	createDB := func(dbName string) (*database.DB, func()) {
		_, _ = adminDB.Pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		_, err := adminDB.Pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			t.Fatalf("failed to create database %s: %v", dbName, err)
		}

		freshDBURL := fmt.Sprintf("postgres://clientpulse:clientpulse_dev_password@localhost:5433/%s?sslmode=disable", dbName)
		freshDB, err := database.New(ctx, database.Config{DatabaseURL: freshDBURL})
		if err != nil {
			t.Fatalf("failed to connect to fresh db %s: %v", dbName, err)
		}

		cleanup := func() {
			freshDB.Close()
			_, _ = adminDB.Pool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		}
		return freshDB, cleanup
	}

	// SCENARIO 1: Fresh Database Migration Chain (000001 -> 000013)
	t.Run("Scenario1_Fresh_Database_Migration_Chain", func(t *testing.T) {
		dbName := fmt.Sprintf("test_fresh_mig_%d", time.Now().UnixNano()%100000)
		db, cleanup := createDB(dbName)
		defer cleanup()

		engine := New(db.Pool, migDir)
		appliedCount, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("engine.Up failed on fresh database: %v", err)
		}

		t.Logf("Fresh DB migrations applied: %d migrations", appliedCount)

		// Verify 000013 was applied
		var version13Applied bool
		err = db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 13)").Scan(&version13Applied)
		if err != nil || !version13Applied {
			t.Fatalf("expected version 13 to be applied in schema_migrations, err: %v", err)
		}

		// Verify assets_definer_policy exists and is SELECT only
		var policyName, cmd string
		var roles []string
		err = db.Pool.QueryRow(ctx, `
			SELECT policyname, cmd, roles
			FROM pg_policies
			WHERE schemaname = 'public' AND tablename = 'assets' AND policyname = 'assets_definer_policy'
		`).Scan(&policyName, &cmd, &roles)
		if err != nil {
			t.Fatalf("assets_definer_policy was not found on fresh db: %v", err)
		}

		if cmd != "SELECT" {
			t.Fatalf("expected assets_definer_policy cmd='SELECT', got '%s'", cmd)
		}
		if len(roles) == 0 || roles[0] != "clientpulse_definer" {
			t.Fatalf("expected role clientpulse_definer, got %v", roles)
		}

		// Verify normal tenant isolation policy remains active
		var hasTenantPolicy bool
		err = db.Pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_policies
				WHERE schemaname = 'public' AND tablename = 'assets' AND policyname = 'assets_tenant_isolation'
			)
		`).Scan(&hasTenantPolicy)
		if err != nil || !hasTenantPolicy {
			t.Fatalf("expected assets_tenant_isolation policy to be present on fresh db")
		}

		// Test Down migration for top migration down to 12
		for {
			rolledBackVersion, err := engine.Down(ctx)
			if err != nil {
				t.Fatalf("failed to rollback migration: %v", err)
			}
			if rolledBackVersion == 13 {
				break
			}
			if rolledBackVersion == 0 {
				t.Fatalf("unexpected end of migrations before rolling back 13")
			}
		}

		// Verify policy dropped after down migration
		var policyExistsAfterRollback bool
		_ = db.Pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_policies
				WHERE schemaname = 'public' AND tablename = 'assets' AND policyname = 'assets_definer_policy'
			)
		`).Scan(&policyExistsAfterRollback)
		if policyExistsAfterRollback {
			t.Fatalf("assets_definer_policy should have been removed after rollback")
		}

		// Re-apply migrations back up
		reAppliedCount, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("failed to re-apply migration: %v", err)
		}
		if reAppliedCount < 1 {
			t.Fatalf("expected at least 1 migration re-applied, got count: %d", reAppliedCount)
		}
	})

	// SCENARIO 2: Existing Database with Emergency Policy (Reconciliation)
	t.Run("Scenario2_Existing_Database_With_Emergency_Policy", func(t *testing.T) {
		dbName := fmt.Sprintf("test_reconcile_mig_%d", time.Now().UnixNano()%100000)
		db, cleanup := createDB(dbName)
		defer cleanup()

		// 1. Apply migrations up to version 12
		engine := New(db.Pool, migDir)
		allMigrations, err := engine.LoadMigrations()
		if err != nil {
			t.Fatalf("failed to load migrations: %v", err)
		}

		if err := engine.EnsureTable(ctx); err != nil {
			t.Fatalf("failed to ensure schema_migrations: %v", err)
		}

		// Apply version 1 to 12 manually
		for _, m := range allMigrations {
			if m.Version > 12 {
				continue
			}
			_, err := db.Pool.Exec(ctx, m.UpSQL)
			if err != nil {
				t.Fatalf("failed to apply migration %d (%s): %v", m.Version, m.Name, err)
			}
			_, err = db.Pool.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name)
			if err != nil {
				t.Fatalf("failed to record migration %d: %v", m.Version, err)
			}
		}

		// 2. Pre-create the exact emergency policy used in production hotfix:
		// FOR ALL TO clientpulse_definer USING (true) WITH CHECK (true)
		_, err = db.Pool.Exec(ctx, `
			CREATE POLICY assets_definer_policy
			ON public.assets
			FOR ALL
			TO clientpulse_definer
			USING (true)
			WITH CHECK (true);
		`)
		if err != nil {
			t.Fatalf("failed to pre-create emergency policy: %v", err)
		}

		// Verify emergency policy is currently cmd='ALL'
		var preCmd string
		err = db.Pool.QueryRow(ctx, `
			SELECT cmd FROM pg_policies WHERE tablename = 'assets' AND policyname = 'assets_definer_policy'
		`).Scan(&preCmd)
		if err != nil || preCmd != "ALL" {
			t.Fatalf("expected pre-existing emergency policy to have cmd='ALL', got %s, err: %v", preCmd, err)
		}

		// 3. Apply migration 13 through the engine (Up)
		appliedCount, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("engine.Up failed when reconciling existing emergency policy: %v", err)
		}

		if appliedCount < 1 {
			t.Fatalf("expected at least migration 13 to be applied, got count: %d", appliedCount)
		}

		// 4. Verify resulting policy is reconciled to least-privilege definition (cmd='SELECT')
		var postCmd string
		var countPolicies int
		err = db.Pool.QueryRow(ctx, `
			SELECT cmd FROM pg_policies WHERE tablename = 'assets' AND policyname = 'assets_definer_policy'
		`).Scan(&postCmd)
		if err != nil {
			t.Fatalf("assets_definer_policy not found after reconciliation: %v", err)
		}
		if postCmd != "SELECT" {
			t.Fatalf("expected reconciled policy to be cmd='SELECT', got '%s'", postCmd)
		}

		err = db.Pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_policies WHERE tablename = 'assets' AND policyname = 'assets_definer_policy'
		`).Scan(&countPolicies)
		if err != nil || countPolicies != 1 {
			t.Fatalf("expected exactly 1 assets_definer_policy, got %d", countPolicies)
		}

		// 5. Verify unrelated policies (e.g. assets_tenant_isolation) are preserved
		var tenantPolicyCount int
		err = db.Pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_policies WHERE tablename = 'assets' AND policyname = 'assets_tenant_isolation'
		`).Scan(&tenantPolicyCount)
		if err != nil || tenantPolicyCount != 1 {
			t.Fatalf("expected assets_tenant_isolation to remain intact, got count=%d", tenantPolicyCount)
		}
	})
}
