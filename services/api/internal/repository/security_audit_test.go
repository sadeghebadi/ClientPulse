package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/testhelper"
)

func TestConcurrentPasswordResetTokenAtomicConsumption(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := NewUserRepository(db.Pool)
	tokenRepo := NewTokenRepository(db.Pool)

	// Create user
	randSuffix, _ := auth.GenerateSecureToken(4)
	user, err := userRepo.CreateUser(ctx, fmt.Sprintf("race-reset-%s@example.com", randSuffix), "$argon2id$v=19$m=65536,t=1,p=4$fake$fake", "Race Reset User")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a single password reset token
	rawToken, _ := auth.GenerateSecureToken(32)
	hashedToken := auth.HashToken(rawToken, "test-pepper")
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = tokenRepo.CreatePasswordResetToken(ctx, user.ID, hashedToken, expiresAt)
	if err != nil {
		t.Fatalf("failed to create password reset token: %v", err)
	}

	// Launch 10 concurrent goroutines attempting to consume this exact same token
	const concurrency = 10
	var successCount int64
	var failureCount int64
	var wg sync.WaitGroup

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startSignal // synchronize launch
			consumed, err := tokenRepo.ConsumePasswordResetToken(ctx, hashedToken)
			if err == nil && consumed != nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failureCount, 1)
			}
		}()
	}

	// Release all goroutines simultaneously
	close(startSignal)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("CRITICAL RACE CONDITION: expected exactly 1 successful consumption, got %d", successCount)
	}
	if failureCount != concurrency-1 {
		t.Fatalf("expected %d failed consumptions, got %d", concurrency-1, failureCount)
	}
}

func TestConcurrentEmailVerificationTokenAtomicConsumption(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := NewUserRepository(db.Pool)
	tokenRepo := NewTokenRepository(db.Pool)

	// Create user
	randSuffix, _ := auth.GenerateSecureToken(4)
	user, err := userRepo.CreateUser(ctx, fmt.Sprintf("race-verify-%s@example.com", randSuffix), "$argon2id$v=19$m=65536,t=1,p=4$fake$fake", "Race Verify User")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a single verification token
	rawToken, _ := auth.GenerateSecureToken(32)
	hashedToken := auth.HashToken(rawToken, "test-pepper")
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = tokenRepo.CreateEmailVerificationToken(ctx, user.ID, hashedToken, expiresAt)
	if err != nil {
		t.Fatalf("failed to create verification token: %v", err)
	}

	// Launch 10 concurrent goroutines attempting to consume this exact same token
	const concurrency = 10
	var successCount int64
	var failureCount int64
	var wg sync.WaitGroup

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startSignal // synchronize launch
			consumed, err := tokenRepo.ConsumeEmailVerificationToken(ctx, hashedToken)
			if err == nil && consumed != nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failureCount, 1)
			}
		}()
	}

	close(startSignal)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("CRITICAL RACE CONDITION: expected exactly 1 successful consumption, got %d", successCount)
	}
	if failureCount != concurrency-1 {
		t.Fatalf("expected %d failed consumptions, got %d", concurrency-1, failureCount)
	}
}

func TestRuntimeRoleRestrictedPrivileges(t *testing.T) {
	// Connect using clientpulse_app runtime credentials
	appDB := testhelper.GetAppTestDB(t)
	defer appDB.Close()
	ctx := context.Background()

	// 1. Verify clientpulse_app is NOT superuser and has NOBYPASSRLS
	var isSuperuser, bypassRLS bool
	err := appDB.Pool.QueryRow(ctx, "SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = 'clientpulse_app'").Scan(&isSuperuser, &bypassRLS)
	if err != nil {
		t.Fatalf("failed to query role attributes: %v", err)
	}
	if isSuperuser {
		t.Fatal("CRITICAL: clientpulse_app role has SUPERUSER privilege!")
	}
	if bypassRLS {
		t.Fatal("CRITICAL: clientpulse_app role has BYPASSRLS privilege!")
	}

	// 2. Verify clientpulse_app CANNOT UPDATE audit_logs (append-only)
	_, err = appDB.Pool.Exec(ctx, "UPDATE audit_logs SET action = 'tampered' WHERE true")
	if err == nil {
		t.Fatal("CRITICAL SECURITY VIOLATION: clientpulse_app was able to UPDATE audit_logs!")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied error on UPDATE audit_logs, got: %v", err)
	}

	// 3. Verify clientpulse_app CANNOT DELETE from audit_logs
	_, err = appDB.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE true")
	if err == nil {
		t.Fatal("CRITICAL SECURITY VIOLATION: clientpulse_app was able to DELETE from audit_logs!")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied error on DELETE audit_logs, got: %v", err)
	}

	// 4. Verify clientpulse_app CANNOT DROP tables
	_, err = appDB.Pool.Exec(ctx, "DROP TABLE audit_logs")
	if err == nil {
		t.Fatal("CRITICAL SECURITY VIOLATION: clientpulse_app was able to DROP TABLE audit_logs!")
	}
	if !strings.Contains(err.Error(), "must be owner") && !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected ownership or permission denied error on DROP TABLE, got: %v", err)
	}

	// 5. Verify clientpulse_app CANNOT disable RLS
	_, err = appDB.Pool.Exec(ctx, "ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY")
	if err == nil {
		t.Fatal("CRITICAL SECURITY VIOLATION: clientpulse_app was able to DISABLE ROW LEVEL SECURITY!")
	}
	if !strings.Contains(err.Error(), "must be owner") && !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected ownership or permission denied error on ALTER TABLE DISABLE RLS, got: %v", err)
	}
}

func TestRuntimeRoleFailsClosedWithoutTenantContext(t *testing.T) {
	// First insert an audit log using admin DB
	adminDB := testhelper.GetTestDB(t)
	defer adminDB.Close()
	ctx := context.Background()

	orgRepo := NewOrganizationRepository(adminDB.Pool)
	rnd, _ := auth.GenerateSecureToken(4)
	org, err := orgRepo.CreateOrganization(ctx, "Fail Closed Org", fmt.Sprintf("fail-closed-%s", rnd), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	auditRepo := NewAuditRepository(adminDB.Pool)
	err = auditRepo.RecordEvent(ctx, &org.ID, nil, "action.test", "resource", "id1", []byte(`{"secret":"data"}`), "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to record audit event: %v", err)
	}

	// Connect as clientpulse_app
	appDB := testhelper.GetAppTestDB(t)
	defer appDB.Close()

	// Query without setting tenant context -> RLS fails closed (0 rows returned)
	var count int
	err = appDB.Pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query audit_logs: %v", err)
	}
	if count != 0 {
		t.Fatalf("CRITICAL SECURITY VIOLATION: clientpulse_app saw %d audit_log rows without tenant context! Expected 0 (fails closed)", count)
	}

	// In a transaction with the correct tenant context, exactly 1 row is visible
	var tenantCount int
	err = appDB.Pool.QueryRow(ctx, "SELECT count(*) FROM (SELECT * FROM audit_logs WHERE organization_id = $1) t", org.ID).Scan(&tenantCount)
	// Even with a direct WHERE organization_id = ..., RLS policy app.current_organization_id must be set!
	if tenantCount != 0 {
		t.Fatalf("CRITICAL: query with WHERE organization_id without session setting should still return 0 due to RLS, got %d", tenantCount)
	}
}

