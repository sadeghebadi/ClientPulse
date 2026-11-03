package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/logger"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isValidUUID(u string) bool {
	return uuidRegex.MatchString(u)
}

func printUsage() {
	fmt.Print(`Platform Admin Management CLI (ClientPulse)

Usage:
  platformadmin grant --user-id <uuid> [--reason <reason>] [--granted-by <uuid>]
  platformadmin revoke --user-id <uuid> [--reason <reason>] [--revoked-by <uuid>]
  platformadmin list

Commands:
  grant   Explicitly grants global platform_admin privileges to an existing user
  revoke  Revokes active platform_admin privileges (enforces last-admin protection)
  list    Displays all active and historical platform admin grants

Options:
  --user-id     Target User ID (required, must be a valid UUID)
  --reason      Audit justification for the privilege change
  --granted-by  UUID of the administrator executing the grant (optional)
  --revoked-by  UUID of the administrator executing the revocation (optional)
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	migrationDBURL := os.Getenv("MIGRATION_DATABASE_URL")
	if migrationDBURL == "" {
		migrationDBURL = cfg.DatabaseURL
	}

	db, err := database.New(ctx, database.Config{
		DatabaseURL: migrationDBURL,
	})
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	switch command {
	case "grant":
		grantCmd := flag.NewFlagSet("grant", flag.ExitOnError)
		userIDStr := grantCmd.String("user-id", "", "Target user UUID (required)")
		reason := grantCmd.String("reason", "Administrative bootstrap grant", "Audit justification")
		grantedByStr := grantCmd.String("granted-by", "", "Admin user UUID granting privileges (optional)")
		_ = grantCmd.Parse(os.Args[2:])

		trimmedUserID := strings.TrimSpace(*userIDStr)
		if trimmedUserID == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required")
			os.Exit(1)
		}

		if !isValidUUID(trimmedUserID) {
			fmt.Fprintf(os.Stderr, "Error: invalid user-id UUID: %s\n", trimmedUserID)
			os.Exit(1)
		}

		var grantedBy *string
		if strings.TrimSpace(*grantedByStr) != "" {
			trimmedGrantedBy := strings.TrimSpace(*grantedByStr)
			if !isValidUUID(trimmedGrantedBy) {
				fmt.Fprintf(os.Stderr, "Error: invalid granted-by UUID: %s\n", trimmedGrantedBy)
				os.Exit(1)
			}
			grantedBy = &trimmedGrantedBy
		}

		if err := executeGrant(ctx, db.Pool, trimmedUserID, *reason, grantedBy); err != nil {
			log.Error("failed to grant platform_admin privilege", "error", err)
			os.Exit(1)
		}

		fmt.Printf("✔ Successfully granted platform_admin privilege to user %s\n", trimmedUserID)

	case "revoke":
		revokeCmd := flag.NewFlagSet("revoke", flag.ExitOnError)
		userIDStr := revokeCmd.String("user-id", "", "Target user UUID (required)")
		reason := revokeCmd.String("reason", "Administrative privilege revocation", "Audit justification")
		revokedByStr := revokeCmd.String("revoked-by", "", "Admin user UUID revoking privileges (optional)")
		_ = revokeCmd.Parse(os.Args[2:])

		trimmedUserID := strings.TrimSpace(*userIDStr)
		if trimmedUserID == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required")
			os.Exit(1)
		}

		if !isValidUUID(trimmedUserID) {
			fmt.Fprintf(os.Stderr, "Error: invalid user-id UUID: %s\n", trimmedUserID)
			os.Exit(1)
		}

		var revokedBy *string
		if strings.TrimSpace(*revokedByStr) != "" {
			trimmedRevokedBy := strings.TrimSpace(*revokedByStr)
			if !isValidUUID(trimmedRevokedBy) {
				fmt.Fprintf(os.Stderr, "Error: invalid revoked-by UUID: %s\n", trimmedRevokedBy)
				os.Exit(1)
			}
			revokedBy = &trimmedRevokedBy
		}

		if err := executeRevoke(ctx, db.Pool, trimmedUserID, *reason, revokedBy); err != nil {
			log.Error("failed to revoke platform_admin privilege", "error", err)
			os.Exit(1)
		}

		fmt.Printf("✔ Successfully revoked platform_admin privilege for user %s\n", trimmedUserID)

	case "list":
		if err := executeList(ctx, db.Pool); err != nil {
			log.Error("failed to list platform admin grants", "error", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

const PlatformAdminLockID = 982347103

func executeGrant(ctx context.Context, pool *pgxpool.Pool, targetUserID string, reason string, grantedBy *string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Acquire advisory lock to serialize grant operations
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", PlatformAdminLockID); err != nil {
		return fmt.Errorf("acquire admin lock: %w", err)
	}

	// 1. Verify user exists
	var email, displayName string
	err = tx.QueryRow(ctx, "SELECT email, display_name FROM users WHERE id = $1", targetUserID).Scan(&email, &displayName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("user %s does not exist", targetUserID)
		}
		return fmt.Errorf("query user: %w", err)
	}

	// 2. Check if already granted (Idempotency)
	var activeGrantID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM platform_admin_grants 
		WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL
	`, targetUserID).Scan(&activeGrantID)
	if err == nil {
		fmt.Printf("Notice: user %s (%s) already has an active platform_admin grant (idempotent no-op)\n", targetUserID, email)
		return nil
	} else if err != pgx.ErrNoRows {
		return fmt.Errorf("check active grant: %w", err)
	}

	// 3. Insert grant
	var grantID string
	err = tx.QueryRow(ctx, `
		INSERT INTO platform_admin_grants (user_id, role, granted_at, granted_by, reason)
		VALUES ($1, 'platform_admin', NOW(), $2, $3)
		RETURNING id
	`, targetUserID, grantedBy, reason).Scan(&grantID)
	if err != nil {
		return fmt.Errorf("insert grant: %w", err)
	}

	// 4. Log in platform admin audit log
	actorID := targetUserID
	if grantedBy != nil {
		actorID = *grantedBy
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO platform_admin_audit_logs (actor_user_id, action, target_type, target_id, reason, metadata_json)
		VALUES ($1, 'platform_admin.granted', 'user', $2, $3, $4::jsonb)
	`, actorID, targetUserID, reason, fmt.Sprintf(`{"email":%q,"grant_id":%q}`, email, grantID))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func executeRevoke(ctx context.Context, pool *pgxpool.Pool, targetUserID string, reason string, revokedBy *string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Acquire advisory lock to serialize revocation operations and prevent concurrent last-admin races
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", PlatformAdminLockID); err != nil {
		return fmt.Errorf("acquire admin lock: %w", err)
	}

	// 1. Last-Administrator Protection
	var activeCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM platform_admin_grants 
		WHERE role = 'platform_admin' AND revoked_at IS NULL
	`).Scan(&activeCount)
	if err != nil {
		return fmt.Errorf("count active admins: %w", err)
	}

	// Check if target user is active
	var activeGrantID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM platform_admin_grants 
		WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL
	`, targetUserID).Scan(&activeGrantID)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("user %s does not have an active platform_admin grant", targetUserID)
	} else if err != nil {
		return fmt.Errorf("query target grant: %w", err)
	}

	if activeCount <= 1 {
		return fmt.Errorf("CANNOT REVOKE: last-administrator protection active (only %d platform administrator remains)", activeCount)
	}

	// 2. Revoke grant
	_, err = tx.Exec(ctx, `
		UPDATE platform_admin_grants 
		SET revoked_at = NOW(), revoked_by = $1, reason = COALESCE($2, reason), updated_at = NOW()
		WHERE id = $3
	`, revokedBy, reason, activeGrantID)
	if err != nil {
		return fmt.Errorf("update grant: %w", err)
	}

	// 3. Log in platform admin audit log
	actorID := targetUserID
	if revokedBy != nil {
		actorID = *revokedBy
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO platform_admin_audit_logs (actor_user_id, action, target_type, target_id, reason, metadata_json)
		VALUES ($1, 'platform_admin.revoked', 'user', $2, $3, $4::jsonb)
	`, actorID, targetUserID, reason, fmt.Sprintf(`{"grant_id":%q}`, activeGrantID))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func executeList(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT g.id, g.user_id, u.email, u.display_name, g.role, g.granted_at, g.revoked_at, COALESCE(g.reason, '')
		FROM platform_admin_grants g
		JOIN users u ON u.id = g.user_id
		ORDER BY g.granted_at DESC
	`)
	if err != nil {
		return fmt.Errorf("query grants: %w", err)
	}
	defer rows.Close()

	fmt.Println("\n--- Platform Admin Grants ---")
	fmt.Printf("%-36s | %-36s | %-28s | %-14s | %-10s | %s\n", "Grant ID", "User ID", "Email", "Role", "Status", "Granted At")
	fmt.Println(strings.Repeat("-", 140))

	count := 0
	for rows.Next() {
		count++
		var grantID, userID, email, displayName, role, reason string
		var grantedAt time.Time
		var revokedAt *time.Time

		if err := rows.Scan(&grantID, &userID, &email, &displayName, &role, &grantedAt, &revokedAt, &reason); err != nil {
			return err
		}

		status := "ACTIVE"
		if revokedAt != nil {
			status = "REVOKED"
		}

		fmt.Printf("%-36s | %-36s | %-28s | %-14s | %-10s | %s\n",
			grantID, userID, email, role, status, grantedAt.Format(time.RFC3339))
	}

	if count == 0 {
		fmt.Println("No platform admin grants found.")
	}
	fmt.Println()
	return nil
}
