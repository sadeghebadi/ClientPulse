package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMigrations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migrations_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test migration files
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

