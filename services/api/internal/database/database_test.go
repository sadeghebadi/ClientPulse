package database

import (
	"context"
	"testing"
	"time"
)

func TestDatabaseNilPing(t *testing.T) {
	var db *DB
	ctx := context.Background()
	if err := db.Ping(ctx); err == nil {
		t.Error("expected error when pinging nil database, got nil")
	}
}

func TestDatabaseInvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := New(ctx, Config{
		DatabaseURL: "invalid-url",
	})
	if err == nil {
		t.Error("expected error for invalid database URL, got nil")
	}
}

