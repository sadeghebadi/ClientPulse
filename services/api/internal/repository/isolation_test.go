package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/testhelper"
)

func TestTenantIsolationAgencyAandAgencyB(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	orgRepo := NewOrganizationRepository(db.Pool)
	userRepo := NewUserRepository(db.Pool)
	auditRepo := NewAuditRepository(db.Pool)

	// 1. Create Agency A and User A
	randSuffix, _ := auth.GenerateSecureToken(4)
	orgA, err := orgRepo.CreateOrganization(ctx, "Agency A Tech", fmt.Sprintf("agency-a-%s", randSuffix), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Agency A: %v", err)
	}

	userA, err := userRepo.CreateUser(ctx, fmt.Sprintf("user-a-%s@example.com", randSuffix), "$argon2id$v=19$m=65536,t=1,p=4$fake$fake", "Alice Agency A")
	if err != nil {
		t.Fatalf("failed to create User A: %v", err)
	}
	_, err = orgRepo.AddMember(ctx, orgA.ID, userA.ID, "owner")
	if err != nil {
		t.Fatalf("failed to add User A to Agency A: %v", err)
	}

	// 2. Create Agency B and User B
	randSuffixB, _ := auth.GenerateSecureToken(4)
	orgB, err := orgRepo.CreateOrganization(ctx, "Agency B Creative", fmt.Sprintf("agency-b-%s", randSuffixB), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Agency B: %v", err)
	}

	userB, err := userRepo.CreateUser(ctx, fmt.Sprintf("user-b-%s@example.com", randSuffixB), "$argon2id$v=19$m=65536,t=1,p=4$fake$fake", "Bob Agency B")
	if err != nil {
		t.Fatalf("failed to create User B: %v", err)
	}
	_, err = orgRepo.AddMember(ctx, orgB.ID, userB.ID, "owner")
	if err != nil {
		t.Fatalf("failed to add User B to Agency B: %v", err)
	}

	// 3. Record Audit Events in Agency A and Agency B
	err = auditRepo.RecordEvent(ctx, &orgA.ID, &userA.ID, "project.created", "project", "proj-a-101", []byte(`{"title":"Agency A Confidential"}`), "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to record event for Agency A: %v", err)
	}

	err = auditRepo.RecordEvent(ctx, &orgB.ID, &userB.ID, "project.created", "project", "proj-b-202", []byte(`{"title":"Agency B Secret Plan"}`), "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to record event for Agency B: %v", err)
	}

	// 4. Test PostgreSQL Row-Level Security (RLS) under Tenant Transaction
	// In Agency A's scoped transaction, User A MUST NOT be able to see Agency B's records!
	err = WithTenantTx(ctx, db.Pool, orgA.ID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT organization_id, action, metadata_json FROM audit_logs")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var rowOrgID string
			var action string
			var meta []byte
			if err := rows.Scan(&rowOrgID, &action, &meta); err != nil {
				return err
			}
			if rowOrgID == orgB.ID {
				t.Fatalf("CRITICAL SECURITY VIOLATION: Agency A tenant query returned Agency B record: %s", string(meta))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Agency A tenant tx failed: %v", err)
	}

	// In Agency B's scoped transaction, User B MUST NOT be able to see Agency A's records!
	err = WithTenantTx(ctx, db.Pool, orgB.ID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT organization_id, action, metadata_json FROM audit_logs")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var rowOrgID string
			var action string
			var meta []byte
			if err := rows.Scan(&rowOrgID, &action, &meta); err != nil {
				return err
			}
			if rowOrgID == orgA.ID {
				t.Fatalf("CRITICAL SECURITY VIOLATION: Agency B tenant query returned Agency A record: %s", string(meta))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Agency B tenant tx failed: %v", err)
	}

	// 5. Test Membership Boundary
	// User A cannot retrieve membership in Agency B
	_, err = orgRepo.GetMembership(ctx, orgB.ID, userA.ID)
	if err != ErrMembershipNotFound {
		t.Fatalf("expected ErrMembershipNotFound when User A checks membership in Agency B, got: %v", err)
	}

	// User B cannot retrieve membership in Agency A
	_, err = orgRepo.GetMembership(ctx, orgA.ID, userB.ID)
	if err != ErrMembershipNotFound {
		t.Fatalf("expected ErrMembershipNotFound when User B checks membership in Agency A, got: %v", err)
	}
}

func TestTenantContextCleanupAndConnectionPoolSafety(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	orgRepo := NewOrganizationRepository(db.Pool)
	rnd, _ := auth.GenerateSecureToken(4)
	org, err := orgRepo.CreateOrganization(ctx, "Pool Safety Org", fmt.Sprintf("pool-org-%s", rnd), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	// 1. Acquire dedicated connection from pool
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	// 2. Start a transaction on this connection and set tenant context
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	_, err = tx.Exec(ctx, "SET LOCAL ROLE clientpulse_app")
	if err != nil {
		t.Fatalf("failed to set role: %v", err)
	}
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_organization_id', $1, true)", org.ID)
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Verify setting is visible INSIDE transaction
	var insideSetting string
	err = tx.QueryRow(ctx, "SELECT current_setting('app.current_organization_id', true)").Scan(&insideSetting)
	if err != nil || insideSetting != org.ID {
		t.Fatalf("expected setting to be %s inside tx, got: %s", org.ID, insideSetting)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// 3. CRITICAL: On the exact same physical connection, verify the setting has been completely reset!
	var outsideSetting string
	err = conn.QueryRow(ctx, "SELECT current_setting('app.current_organization_id', true)").Scan(&outsideSetting)
	if err != nil {
		t.Fatalf("failed to query setting outside tx: %v", err)
	}
	if outsideSetting != "" {
		t.Fatalf("CRITICAL CONTEXT LEAK: setting 'app.current_organization_id' persisted on pooled connection after commit: '%s'", outsideSetting)
	}

	// 4. Test Rollback cleanup as well
	tx2, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin tx2: %v", err)
	}
	_, _ = tx2.Exec(ctx, "SET LOCAL ROLE clientpulse_app")
	_, _ = tx2.Exec(ctx, "SELECT set_config('app.current_organization_id', $1, true)", org.ID)
	_ = tx2.Rollback(ctx)

	err = conn.QueryRow(ctx, "SELECT current_setting('app.current_organization_id', true)").Scan(&outsideSetting)
	if err != nil {
		t.Fatalf("failed to query setting outside tx after rollback: %v", err)
	}
	if outsideSetting != "" {
		t.Fatalf("CRITICAL CONTEXT LEAK: setting 'app.current_organization_id' persisted after rollback: '%s'", outsideSetting)
	}
}

func TestConcurrentTenantQueries(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	orgRepo := NewOrganizationRepository(db.Pool)
	rndA, _ := auth.GenerateSecureToken(4)
	rndB, _ := auth.GenerateSecureToken(4)
	orgA, err := orgRepo.CreateOrganization(ctx, "Concurrent Org A", fmt.Sprintf("concurrent-a-%s", rndA), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Concurrent Org A: %v", err)
	}
	orgB, err := orgRepo.CreateOrganization(ctx, "Concurrent Org B", fmt.Sprintf("concurrent-b-%s", rndB), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Concurrent Org B: %v", err)
	}

	auditRepo := NewAuditRepository(db.Pool)
	_ = auditRepo.RecordEvent(ctx, &orgA.ID, nil, "action.a", "test", "1", []byte(`{"org":"A"}`), "")
	_ = auditRepo.RecordEvent(ctx, &orgB.ID, nil, "action.b", "test", "2", []byte(`{"org":"B"}`), "")

	const numWorkers = 20
	const iterations = 10
	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers*iterations)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		targetOrgID := orgA.ID
		forbiddenOrgID := orgB.ID
		if w%2 == 1 {
			targetOrgID = orgB.ID
			forbiddenOrgID = orgA.ID
		}

		go func(workerID int, targetID, forbiddenID string) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				err := WithTenantTx(ctx, db.Pool, targetID, func(tx pgx.Tx) error {
					rows, err := tx.Query(ctx, "SELECT organization_id FROM audit_logs")
					if err != nil {
						return err
					}
					defer rows.Close()

					for rows.Next() {
						var seenOrgID string
						if err := rows.Scan(&seenOrgID); err != nil {
							return err
						}
						if seenOrgID == forbiddenID {
							return fmt.Errorf("concurrency violation: worker target %s saw forbidden %s", targetID, forbiddenID)
						}
					}
					return nil
				})
				if err != nil {
					errChan <- err
					return
				}
			}
		}(w, targetOrgID, forbiddenOrgID)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Fatalf("concurrent tenant isolation error: %v", err)
	}
}
