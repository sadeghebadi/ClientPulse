package testhelper

import (
	"context"
	"os"
	"testing"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
)

// GetTestDB connects to the running PostgreSQL test instance.
func GetTestDB(t *testing.T) *database.DB {
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

// GetTestConfig creates a test configuration.
func GetTestConfig() *config.Config {
	return &config.Config{
		AppEnv:          "testing",
		AppBaseURL:      "http://localhost:3000",
		APIBaseURL:      "http://localhost:8080",
		DefaultLocale:   "en",
		DefaultTimezone: "UTC",
		SessionSecret:   "test-session-secret-for-integration-tests",
		TokenHashPepper: "test-token-hash-pepper-value-for-testing-12345",
	}
}

