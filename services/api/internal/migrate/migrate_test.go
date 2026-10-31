package migrate

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"clientpulse/services/api/internal/database"
)

func getTestDB(t *testing.T) *database.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{DatabaseURL: dbURL})
	if err != nil {
		t.Skip("skipping integration test: database not reachable at", dbURL)
		return nil
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skip("skipping integration test: database ping failed:", err)
		return nil
	}

	return db
}

func TestLoadMigrations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migrations_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"000001_foundation.up.sql":   "CREATE EXTENSION IF NOT EXISTS pgcrypto;",
		"000001_foundation.down.sql": "DROP EXTENSION IF EXISTS pgcrypto;",
		"000002_sample.up.sql":       "CREATE TABLE test (id INT);",
		"000002_sample.down.sql":     "DROP TABLE test;",
		"ignored_file.txt":           "not a migration",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test file %s: %v", name, err)
		}
	}

	engine := New(nil, tmpDir)
	migrations, err := engine.LoadMigrations()
	if err != nil {
		t.Fatalf("failed to load migrations: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}

	if migrations[0].Version != 1 || migrations[0].Name != "foundation" {
		t.Errorf("expected migration 1 'foundation', got %d '%s'", migrations[0].Version, migrations[0].Name)
	}

	if migrations[1].Version != 2 || migrations[1].Name != "sample" {
		t.Errorf("expected migration 2 'sample', got %d '%s'", migrations[1].Version, migrations[1].Name)
	}
}

// TestMigrationAtomicityOnFailure verifies transactional rollback when a migration statement fails.
func TestMigrationAtomicityOnFailure(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "migration_atomic_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	brokenUp := `
		CREATE TABLE IF NOT EXISTS atomic_test_table (id INT PRIMARY KEY);
		THIS IS INTENTIONAL INVALID SQL THAT WILL FAIL;
	`
	brokenDown := `DROP TABLE IF EXISTS atomic_test_table;`

	_ = os.WriteFile(filepath.Join(tmpDir, "999999_atomic_fail.up.sql"), []byte(brokenUp), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "999999_atomic_fail.down.sql"), []byte(brokenDown), 0644)

	engine := New(db.Pool, tmpDir)

	_, _ = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS atomic_test_table")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = 999999")

	_, err = engine.Up(ctx)
	if err == nil {
		t.Fatal("expected migration Up to fail due to syntax error, but it succeeded")
	}

	var exists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'atomic_test_table'
		);
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if exists {
		t.Errorf("ATOMICITY VIOLATION: table 'atomic_test_table' exists despite transaction rollback!")
	}

	var recorded bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = 999999
		);
	`).Scan(&recorded)
	if err != nil {
		t.Fatalf("failed to check schema_migrations: %v", err)
	}

	if recorded {
		t.Errorf("CONSISTENCY VIOLATION: failed migration was recorded in schema_migrations!")
	}
}

// TestConcurrentMigrationLocking launches two concurrent migration engines
// operating on separate sessions to verify that PostgreSQL advisory locking serializes execution,
// prevents race conditions, and avoids duplicate execution.
func TestConcurrentMigrationLocking(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "migration_concurrent_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test migration
	testUp := `
		CREATE TABLE IF NOT EXISTS concurrent_lock_test (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		-- Slight sleep to ensure concurrent operation encounters the lock
		SELECT pg_sleep(0.5);
	`
	testDown := `DROP TABLE IF EXISTS concurrent_lock_test;`

	_ = os.WriteFile(filepath.Join(tmpDir, "888888_concurrent_lock.up.sql"), []byte(testUp), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "888888_concurrent_lock.down.sql"), []byte(testDown), 0644)

	// Clean up any leftovers
	_, _ = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS concurrent_lock_test")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = 888888")

	engine1 := New(db.Pool, tmpDir)
	engine2 := New(db.Pool, tmpDir)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	appliedCounts := make(chan int, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		count, err := engine1.Up(ctx)
		if err != nil {
			errs <- err
			return
		}
		appliedCounts <- count
	}()

	go func() {
		defer wg.Done()
		count, err := engine2.Up(ctx)
		if err != nil {
			errs <- err
			return
		}
		appliedCounts <- count
	}()

	wg.Wait()
	close(errs)
	close(appliedCounts)

	for err := range errs {
		t.Fatalf("concurrent migration failed with error: %v", err)
	}

	totalApplied := 0
	for count := range appliedCounts {
		totalApplied += count
	}

	// Exactly ONE of the two operations should have applied the migration (count=1),
	// and the other should have found it already applied (count=0), resulting in totalApplied == 1.
	if totalApplied != 1 {
		t.Errorf("expected exactly 1 applied migration across both concurrent sessions, got %d", totalApplied)
	}

	// Clean up
	_, _ = engine1.Down(ctx)
}

// TestAdvisoryLockReleaseAfterFailure verifies that an advisory lock is released
// even after a migration operation fails, allowing subsequent operations to proceed immediately.
func TestAdvisoryLockReleaseAfterFailure(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "migration_lock_release_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	badUp := `THIS_SQL_IS_COMPLETELY_INVALID;`
	_ = os.WriteFile(filepath.Join(tmpDir, "777777_fail_lock.up.sql"), []byte(badUp), 0644)

	engine := New(db.Pool, tmpDir)

	// Run migration that fails
	_, err = engine.Up(ctx)
	if err == nil {
		t.Fatal("expected migration to fail, but it succeeded")
	}

	// Verify that the advisory lock is released and can immediately be acquired by a separate connection
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection to verify lock: %v", err)
	}
	defer conn.Release()

	var lockAcquired bool
	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", MigrationLockID).Scan(&lockAcquired)
	if err != nil {
		t.Fatalf("failed to probe advisory lock: %v", err)
	}

	if !lockAcquired {
		t.Errorf("LOCK LEAK: pg_advisory_lock was still held after failed migration!")
	} else {
		// Cleanly unlock the probed lock
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", MigrationLockID)
	}

	// Clean up record
	_, _ = db.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = 777777")
}
