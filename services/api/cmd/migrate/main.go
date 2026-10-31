package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/logger"
	"clientpulse/services/api/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, os.Stdout)

	// Resolve migrations directory path
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		// Look for migrations in repo root or relative path
		candidates := []string{
			"migrations",
			"../../migrations",
			"/app/migrations",
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				migrationsDir = c
				break
			}
		}
		if migrationsDir == "" {
			migrationsDir = "migrations"
		}
	}

	absPath, _ := filepath.Abs(migrationsDir)
	log.Info("migration CLI initialized", "command", command, "migrations_dir", absPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DatabaseURL: cfg.DatabaseURL,
	})
	if err != nil {
		log.Error("failed to connect to database for migration", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	engine := migrate.New(db.Pool, migrationsDir)

	switch command {
	case "up":
		count, err := engine.Up(ctx)
		if err != nil {
			log.Error("failed applying migrations", "error", err)
			os.Exit(1)
		}
		log.Info("migrations applied successfully", "applied_count", count)

	case "down":
		rolledBack, err := engine.Down(ctx)
		if err != nil {
			log.Error("failed rolling back migration", "error", err)
			os.Exit(1)
		}
		if rolledBack == 0 {
			log.Info("no applied migrations to roll back")
		} else {
			log.Info("successfully rolled back migration", "version", rolledBack)
		}

	case "status":
		statuses, err := engine.Status(ctx)
		if err != nil {
			log.Error("failed to retrieve migration status", "error", err)
			os.Exit(1)
		}
		fmt.Printf("\n--- Migration Status ---\n")
		for _, s := range statuses {
			appliedText := "[PENDING]"
			if s.Applied {
				appliedText = fmt.Sprintf("[APPLIED at %s]", s.AppliedAt.Format(time.RFC3339))
			}
			fmt.Printf("Version %06d | %-20s | %s\n", s.Version, s.Name, appliedText)
		}
		fmt.Println()

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command>")
	fmt.Println("Commands:")
	fmt.Println("  up      Apply all pending migrations")
	fmt.Println("  down    Roll back the most recent migration")
	fmt.Println("  status  Show migration status")
}

