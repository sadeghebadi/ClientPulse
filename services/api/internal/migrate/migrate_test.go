package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"clientpulse/services/api/internal/database"
)

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

// TestMigrationAtomicityOnFailure verifies that a migration containing a valid statement
// followed by an invalid statement rolls back completely and leaves NO partial state.
func TestMigrationAtomicityOnFailure(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{DatabaseURL: dbURL})
	if err != nil {
		t.Skip("skipping integration test: database not reachable at", dbURL)
		return
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Skip("skipping integration test: database ping failed:", err)
		return
	}

	tmpDir, err := os.MkdirTemp("", "migration_atomic_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a migration that creates a table, but then has a fatal syntax error
	// High version number (999999) to avoid colliding with real migrations
	brokenUp := `
		CREATE TABLE IF NOT EXISTS atomic_test_table (id INT PRIMARY KEY);
		THIS IS INTENTIONAL INVALID SQL THAT WILL FAIL;
	`
	brokenDown := `DROP TABLE IF EXISTS atomic_test_table;`

	_ = os.WriteFile(filepath.Join(tmpDir, "999999_atomic_fail.up.sql"), []byte(brokenUp), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "999999_atomic_fail.down.sql"), []byte(brokenDown), 0644)

	engine := New(db.Pool, tmpDir)

	// Ensure any leftover test table is dropped before test
	_, _ = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS atomic_test_table")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = 999999")

	// Run migration Up - expect failure
	_, err = engine.Up(ctx)
	if err == nil {
		t.Fatal("expected migration Up to fail due to syntax error, but it succeeded")
	}

	// VERIFY ATOMICITY: Table 'atomic_test_table' must NOT exist in the database!
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
		t.Errorf("ATOMICITY VIOLATION: table 'atomic_test_table' was created despite migration transaction failure!")
	}

	// VERIFY CONSISTENCY: schema_migrations must NOT contain version 999999
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
		t.Errorf("CONSISTENCY VIOLATION: failed migration 999999 was recorded in schema_migrations!")
	}
}
