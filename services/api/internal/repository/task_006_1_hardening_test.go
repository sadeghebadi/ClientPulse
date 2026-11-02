package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// 1. Telegram Group Connection Authorization Tests
// ---------------------------------------------------------------------------

func TestTelegramGroup_AuthorizedAdminConnection(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("tg-auth-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "tg-auth", slug)
	adminUserID := createTestUser(t, adminPool, fmt.Sprintf("tg-admin-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	// Step 1: Link admin's private Telegram account (User ID: 998877)
	privChatID := int64(998877)
	privTelegramUserID := int64(998877)
	_, err := notifRepo.ConnectTelegramDestination(ctx, orgID, adminUserID, "telegram_private", "Admin Phone", privChatID, privTelegramUserID, nil)
	if err != nil {
		t.Fatalf("failed linking admin private telegram: %v", err)
	}

	// Verify sender identity check matches
	isVerified, err := notifRepo.IsUserVerifiedTelegramIdentity(ctx, orgID, adminUserID, privTelegramUserID)
	if err != nil || !isVerified {
		t.Fatalf("expected admin telegram identity to be verified, got: %v, err: %v", isVerified, err)
	}

	// Step 2: Connect group by the verified admin -> immediately active
	groupChatID := -int64(3000000000 + time.Now().UnixNano()%100000000)
	groupDest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, adminUserID, "telegram_group", "Engineering Group", groupChatID, privTelegramUserID, nil)
	if err != nil {
		t.Fatalf("failed connecting authorized group: %v", err)
	}
	if groupDest.Status != "active" {
		t.Errorf("expected group status 'active', got: %s", groupDest.Status)
	}
	if groupDest.PendingGroupApproval {
		t.Errorf("expected pending_group_approval=false for verified admin")
	}
}

func TestTelegramGroup_UnlinkedSender_RequiresPendingApproval(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("tg-unlinked-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "tg-unlinked", slug)
	adminUserID := createTestUser(t, adminPool, fmt.Sprintf("tg-initiator-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	// Sender 554433 has NOT linked a telegram_private destination with adminUserID
	unlinkedSenderID := int64(554433)
	isVerified, err := notifRepo.IsUserVerifiedTelegramIdentity(ctx, orgID, adminUserID, unlinkedSenderID)
	if err != nil {
		t.Fatalf("error checking sender identity: %v", err)
	}
	if isVerified {
		t.Fatalf("unlinked sender should NOT be verified for admin")
	}

	// Group connection must be placed in pending_approval
	groupChatID := -int64(3100000000 + time.Now().UnixNano()%100000000)
	username := "unlinked_member"
	pendingDest, err := notifRepo.CreatePendingGroupDestination(ctx, orgID, adminUserID, "DevOps Channel", groupChatID, unlinkedSenderID, &username)
	if err != nil {
		t.Fatalf("failed creating pending group destination: %v", err)
	}

	if pendingDest.Status != "pending_approval" {
		t.Errorf("expected status 'pending_approval', got: %s", pendingDest.Status)
	}
	if !pendingDest.PendingGroupApproval {
		t.Errorf("expected PendingGroupApproval=true")
	}

	// Ensure pending group is NOT eligible for receiving notifications in fan-out
	destList, err := notifRepo.ListDestinations(ctx, orgID)
	if err != nil {
		t.Fatalf("failed listing destinations: %v", err)
	}
	var found *NotificationDestination
	for _, d := range destList {
		if d.ID == pendingDest.ID {
			found = d
			break
		}
	}
	if found == nil {
		t.Fatalf("pending destination not found in list")
	}
	if found.Status != "pending_approval" {
		t.Errorf("expected list status 'pending_approval', got: %s", found.Status)
	}

	// Admin approves the pending group
	approvedDest, err := notifRepo.ApprovePendingGroupDestination(ctx, orgID, pendingDest.ID, adminUserID)
	if err != nil {
		t.Fatalf("failed approving pending group: %v", err)
	}
	if approvedDest.Status != "active" {
		t.Errorf("expected status 'active' after approval, got: %s", approvedDest.Status)
	}
	if approvedDest.PendingGroupApproval {
		t.Errorf("expected PendingGroupApproval=false after approval")
	}
	if approvedDest.AuthorizedByUserID == nil || *approvedDest.AuthorizedByUserID != adminUserID {
		t.Errorf("expected AuthorizedByUserID=%s, got %v", adminUserID, approvedDest.AuthorizedByUserID)
	}
}

func TestTelegramGroup_PendingConnectionCancellation(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("tg-cancel-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "tg-cancel", slug)
	adminUserID := createTestUser(t, adminPool, fmt.Sprintf("tg-admin-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	groupChatID := -int64(3200000000 + time.Now().UnixNano()%100000000)
	pendingDest, err := notifRepo.CreatePendingGroupDestination(ctx, orgID, adminUserID, "Cancelled Group", groupChatID, 12345, nil)
	if err != nil {
		t.Fatalf("failed creating pending group: %v", err)
	}

	// Admin cancels the pending connection
	err = notifRepo.CancelPendingGroupDestination(ctx, orgID, pendingDest.ID)
	if err != nil {
		t.Fatalf("failed cancelling pending group: %v", err)
	}

	// Should no longer exist
	_, err = notifRepo.GetDestinationByID(ctx, orgID, pendingDest.ID)
	if err != ErrDestinationNotFound {
		t.Errorf("expected ErrDestinationNotFound after cancellation, got: %v", err)
	}
}

func TestTelegramGroup_CrossTenantHijackingPrevention(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slugA := fmt.Sprintf("tg-hijack-a-%d", time.Now().UnixNano())
	orgA, _ := createTestOrgAndClient(t, adminPool, "tg-hijack-a", slugA)
	userA := createTestUser(t, adminPool, fmt.Sprintf("admin-a-%d@test.org", time.Now().UnixNano()))

	slugB := fmt.Sprintf("tg-hijack-b-%d", time.Now().UnixNano())
	orgB, _ := createTestOrgAndClient(t, adminPool, "tg-hijack-b", slugB)
	userB := createTestUser(t, adminPool, fmt.Sprintf("admin-b-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	// Org A connects group
	sharedGroupChatID := -int64(3300000000 + time.Now().UnixNano()%100000000)
	_, err := notifRepo.ConnectTelegramDestination(ctx, orgA, userA, "telegram_group", "Company Alerts", sharedGroupChatID, 888, nil)
	if err != nil {
		t.Fatalf("failed connecting group to Org A: %v", err)
	}

	// Org B checks if group is available
	existing, err := notifRepo.GetGroupDestinationByChatID(ctx, sharedGroupChatID)
	if err != nil {
		t.Fatalf("failed checking group: %v", err)
	}
	if existing == nil {
		t.Fatalf("expected existing group destination, got nil")
	}
	if existing.OrganizationID != orgA {
		t.Errorf("expected existing group to belong to orgA, got: %s", existing.OrganizationID)
	}

	// Org B attempting to claim the same group MUST fail at DB level (uq_active_telegram_group constraint)
	_, err = notifRepo.ConnectTelegramDestination(ctx, orgB, userB, "telegram_group", "Hijacked Group", sharedGroupChatID, 999, nil)
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: Org B was able to claim a group already bound to Org A!")
	}
}

// ---------------------------------------------------------------------------
// 2. Email Destination Ownership Verification Tests
// ---------------------------------------------------------------------------

func TestEmailDestination_OwnershipVerificationAndFanout(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("email-verify-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "email-verify", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("email-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	// 1. Create email destination -> Must be 'pending' with token hash
	targetEmail := fmt.Sprintf("alerts-%d@example.com", time.Now().UnixNano())
	dest, rawToken, err := notifRepo.CreateEmailDestination(ctx, orgID, userID, targetEmail, "SecOps Email", map[string]bool{
		"incident.opened": true,
	})
	if err != nil {
		t.Fatalf("failed creating email destination: %v", err)
	}
	if rawToken == "" {
		t.Fatalf("expected raw verification token, got empty string")
	}
	if dest.Status != "pending" {
		t.Errorf("expected status 'pending', got: %s", dest.Status)
	}
	if dest.VerifiedAt != nil {
		t.Errorf("verified_at should be nil before verification")
	}

	// 2. Insert notification event and dispatch pending events
	// UNVERIFIED destination MUST NOT receive any deliveries!
	eventID := fmt.Sprintf("ev-unverified-%d", time.Now().UnixNano())
	_, err = adminPool.Exec(ctx, `
		INSERT INTO notification_events (id, organization_id, event_type, idempotency_key, payload, status)
		VALUES (gen_random_uuid(), $1, 'incident.opened', $2, '{"summary":"Test"}'::jsonb, 'pending')
	`, orgID, eventID)
	if err != nil {
		t.Fatalf("failed inserting event: %v", err)
	}

	dispatched, err := workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("DispatchPendingEvents failed: %v", err)
	}
	if dispatched == 0 {
		t.Fatalf("expected 1 event dispatched")
	}

	// Deliveries for Org should be 0 because destination is unverified!
	delivs, total, err := notifRepo.ListDeliveries(ctx, orgID, 10, 0, "", dest.ID)
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if total > 0 || len(delivs) > 0 {
		t.Fatalf("SECURITY VIOLATION: Unverified email destination received %d deliveries!", total)
	}

	// 3. Consume verification token -> transitions to 'active'
	verifiedDest, err := notifRepo.ConsumeEmailVerificationToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("failed consuming verification token: %v", err)
	}
	if verifiedDest.Status != "active" {
		t.Errorf("expected status 'active' after verification, got: %s", verifiedDest.Status)
	}
	if verifiedDest.VerifiedAt == nil {
		t.Errorf("expected verified_at to be populated")
	}

	// 4. Token replay prevention: second consumption MUST fail
	_, err = notifRepo.ConsumeEmailVerificationToken(ctx, rawToken)
	if err != ErrTokenInvalidOrExpired {
		t.Errorf("expected ErrTokenInvalidOrExpired on token reuse, got: %v", err)
	}

	// 5. Rate limiting: requesting resend immediately MUST return ErrVerificationRateLimited
	// (Create a new pending destination to test resend cooldown)
	pendingDest2, _, err := notifRepo.CreateEmailDestination(ctx, orgID, userID, "pending2@example.com", "Pending 2", nil)
	if err != nil {
		t.Fatalf("failed creating pending2: %v", err)
	}

	_, err = notifRepo.ResendEmailVerificationToken(ctx, orgID, pendingDest2.ID)
	if err != ErrVerificationRateLimited {
		t.Errorf("expected ErrVerificationRateLimited within 60s cooldown, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 3. Telegram Webhook Replay Protection Tests
// ---------------------------------------------------------------------------

func TestTelegramWebhook_ReplayProtection(t *testing.T) {
	_, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	notifRepo := NewNotificationDestRepository(appPool)
	updateID := int64(987654321000 + time.Now().UnixNano()%1000000)

	// First attempt -> must be recorded as new
	isNew, err := notifRepo.RecordTelegramUpdateID(ctx, updateID)
	if err != nil {
		t.Fatalf("failed recording update id: %v", err)
	}
	if !isNew {
		t.Fatalf("expected first recording to return isNew=true")
	}

	// Second attempt with identical update_id (replay attack) -> must return isNew=false
	isNewSecond, err := notifRepo.RecordTelegramUpdateID(ctx, updateID)
	if err != nil {
		t.Fatalf("failed checking replayed update id: %v", err)
	}
	if isNewSecond {
		t.Fatalf("SECURITY VIOLATION: Replayed update_id was accepted as new!")
	}
}

// ---------------------------------------------------------------------------
// 4. Delivery Guarantee Semantics & Stale Worker Recovery Tests
// ---------------------------------------------------------------------------

func TestDeliverySemantics_StaleWorkerRecoveryToUncertain(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("stale-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "stale-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("stale-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	dest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "My Bot", 112233, 112233, nil)
	if err != nil {
		t.Fatalf("failed creating dest: %v", err)
	}

	// 1. Create a delivery and simulate worker claiming it (status='sending', last_attempt_at in the past)
	idempKey := fmt.Sprintf("stale-deliv-%d", time.Now().UnixNano())
	deliv, err := notifRepo.CreateDelivery(ctx, orgID, nil, dest.ID, "telegram_private", "112233", "incident.opened", idempKey, map[string]interface{}{"test": true})
	if err != nil {
		t.Fatalf("failed creating delivery: %v", err)
	}

	// Simulate worker crash: worker set status='sending', incremented retry_count=1, and crashed 10 minutes ago
	staleTime := time.Now().UTC().Add(-10 * time.Minute)
	_, err = adminPool.Exec(ctx, `
		UPDATE notification_deliveries
		SET status = 'sending',
		    retry_count = 1,
		    last_attempt_at = $1
		WHERE id = $2
	`, staleTime, deliv.ID)
	if err != nil {
		t.Fatalf("failed simulating crash: %v", err)
	}

	// 2. Run stale sweep (5 minute threshold)
	requeued, err := workerNotifRepo.RequeueStaleDeliveries(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("RequeueStaleDeliveries failed: %v", err)
	}
	if requeued == 0 {
		t.Fatalf("expected at least 1 delivery recovered by stale sweep")
	}

	// 3. Because retry_count was > 0, the crashed in-flight delivery must transition to 'uncertain'
	// and record the recovery note in delivery_attempts audit trail.
	recovered, _, err := notifRepo.ListDeliveries(ctx, orgID, 1, 0, "", dest.ID)
	if err != nil || len(recovered) == 0 {
		t.Fatalf("failed fetching recovered delivery: %v", err)
	}

	if recovered[0].Status != "uncertain" {
		t.Errorf("expected status 'uncertain' after worker crash during sending, got: %s", recovered[0].Status)
	}
	if len(recovered[0].DeliveryAttempts) == 0 {
		t.Errorf("expected delivery_attempts audit trail to be recorded")
	}
}

func TestDeliverySemantics_ConcurrentWorkerClaims_SkipLocked(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("conc-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "conc-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("conc-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	dest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "Bot", 445566, 445566, nil)
	if err != nil {
		t.Fatalf("failed creating dest: %v", err)
	}

	// Queue 10 deliveries
	for i := 0; i < 10; i++ {
		idemp := fmt.Sprintf("conc-queue-%d-%d", time.Now().UnixNano(), i)
		_, err := notifRepo.CreateDelivery(ctx, orgID, nil, dest.ID, "telegram_private", "445566", "incident.opened", idemp, nil)
		if err != nil {
			t.Fatalf("failed queuing delivery %d: %v", i, err)
		}
	}

	// Run 5 concurrent workers claiming deliveries simultaneously
	var claimedIDs sync.Map
	var duplicateClaims int64
	var wg sync.WaitGroup

	for w := 0; w < 5; w++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()
			workerID := fmt.Sprintf("worker-conc-%d", workerIndex)
			deliveries, err := workerNotifRepo.ClaimDueDeliveries(ctx, workerID, 10)
			if err != nil {
				t.Errorf("worker claim error: %v", err)
				return
			}
			for _, d := range deliveries {
				if _, exists := claimedIDs.LoadOrStore(d.ID, true); exists {
					atomic.AddInt64(&duplicateClaims, 1)
				}
			}
		}(w)
	}

	wg.Wait()

	if duplicateClaims > 0 {
		t.Fatalf("RACE CONDITION: %d deliveries were claimed by more than one worker simultaneously!", duplicateClaims)
	}
}

