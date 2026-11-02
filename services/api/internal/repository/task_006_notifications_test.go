package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTelegramConnectionToken_SingleUseAndExpiry(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("token-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "token-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("token-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	// 1. Generate single-use token with 2-second TTL
	rawToken, expiresAt, err := notifRepo.CreateConnectionToken(ctx, orgID, userID, "telegram_private", "My Phone", 2*time.Second)
	if err != nil {
		t.Fatalf("failed creating connection token: %v", err)
	}
	if rawToken == "" || expiresAt.IsZero() {
		t.Fatalf("token or expiresAt is empty")
	}

	// 2. Consume token first time -> MUST SUCCEED
	tokenInfo, err := notifRepo.ConsumeConnectionToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("first token consumption failed: %v", err)
	}
	if tokenInfo.OrganizationID != orgID || tokenInfo.UserID != userID || tokenInfo.DestinationType != "telegram_private" {
		t.Errorf("mismatched consumed token info: %+v", tokenInfo)
	}

	// 3. Re-consumption of same token -> MUST BE REJECTED (single use)
	_, err = notifRepo.ConsumeConnectionToken(ctx, rawToken)
	if err != ErrTokenInvalidOrExpired {
		t.Errorf("expected ErrTokenInvalidOrExpired for re-used token, got: %v", err)
	}

	// 4. Test expired token
	expiredToken, _, err := notifRepo.CreateConnectionToken(ctx, orgID, userID, "telegram_private", "Expired Token", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed creating short-lived token: %v", err)
	}
	time.Sleep(150 * time.Millisecond) // Wait for token to expire

	_, err = notifRepo.ConsumeConnectionToken(ctx, expiredToken)
	if err != ErrTokenInvalidOrExpired {
		t.Errorf("expected ErrTokenInvalidOrExpired for expired token, got: %v", err)
	}
}

func TestTelegramDestinationBinding_PrivateAndGroup(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("binding-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "binding-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("binding-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)

	// 1. Bind private chat
	username := "johndoe"
	privDest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "John's Telegram", 987654321, 11223344, &username)
	if err != nil {
		t.Fatalf("failed connecting private telegram destination: %v", err)
	}
	if privDest.TelegramChatID == nil || *privDest.TelegramChatID != 987654321 {
		t.Errorf("expected chat ID 987654321, got %v", privDest.TelegramChatID)
	}

	// 2. Bind group chat
	groupChatID := -int64(2000000000 + time.Now().UnixNano()%100000000)
	groupDest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_group", "DevOps Alerts Group", groupChatID, 11223344, nil)
	if err != nil {
		t.Fatalf("failed connecting group telegram destination: %v", err)
	}
	if groupDest.TelegramChatID == nil || *groupDest.TelegramChatID != groupChatID {
		t.Errorf("expected group chat ID %d, got %v", groupChatID, groupDest.TelegramChatID)
	}

	// 3. Bind email destination
	emailDest, rawVerifyToken, err := notifRepo.CreateEmailDestination(ctx, orgID, userID, "alerts@binding-org.com", "DevOps Email", map[string]bool{
		"incident.opened":   true,
		"incident.resolved": true,
		"heartbeat.missed":  false,
	})
	if err != nil {
		t.Fatalf("failed creating email destination: %v", err)
	}
	// Verify email destination so it is active
	_, err = notifRepo.ConsumeEmailVerificationToken(ctx, rawVerifyToken)
	if err != nil {
		t.Fatalf("failed verifying email destination: %v", err)
	}

	// 4. List all destinations for org
	list, err := notifRepo.ListDestinations(ctx, orgID)
	if err != nil {
		t.Fatalf("failed listing destinations: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 destinations, got %d", len(list))
	}

	// Verify preferences loaded properly
	var foundEmail *NotificationDestination
	for _, d := range list {
		if d.ID == emailDest.ID {
			foundEmail = d
			break
		}
	}
	if foundEmail == nil {
		t.Fatalf("email destination not found in list")
	}
	if !foundEmail.Preferences["incident.opened"] || foundEmail.Preferences["heartbeat.missed"] {
		t.Errorf("email destination preferences mismatch: %+v", foundEmail.Preferences)
	}
}

func TestNotificationEvent_FanOutAndDeduplication(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("fanout-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "fanout-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("fanout-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	// Drain any legacy pending events in test database
	for {
		c, err := workerNotifRepo.DispatchPendingEvents(ctx, 100)
		if err != nil || c == 0 {
			break
		}
	}

	// Create 2 destinations: Dest1 wants incident.opened, Dest2 only wants heartbeat.missed
	dest1, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "Alerts 1", 111111, 111, nil)
	if err != nil {
		t.Fatalf("failed creating dest1: %v", err)
	}

	dest2ChatID := -int64(222222000 + time.Now().UnixNano()%100000)
	dest2, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_group", "Alerts 2", dest2ChatID, 222, nil)
	if err != nil {
		t.Fatalf("failed creating dest2: %v", err)
	}

	// Set Dest2 to NOT receive incident.opened
	_, err = notifRepo.UpdateDestination(ctx, orgID, dest2.ID, nil, nil, nil, map[string]bool{
		"incident.opened": false,
	})
	if err != nil {
		t.Fatalf("failed updating dest2 preferences: %v", err)
	}

	// Insert incident.opened event directly into notification_events
	eventID := ""
	idemp := fmt.Sprintf("fanout-event-%d", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, `
		INSERT INTO notification_events (organization_id, event_type, idempotency_key, payload, status)
		VALUES ($1, 'incident.opened', $2, '{"summary": "Test Outage"}'::jsonb, 'pending')
		RETURNING id
	`, orgID, idemp).Scan(&eventID)
	if err != nil {
		t.Fatalf("failed inserting test notification event: %v", err)
	}

	// Dispatch pending events
	dispatched, err := workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("DispatchPendingEvents failed: %v", err)
	}
	if dispatched != 1 {
		t.Errorf("expected 1 dispatched event, got %d", dispatched)
	}

	// Check notification_deliveries: Only Dest1 should have received a delivery row! Dest2 unsubscribed.
	deliveries, total, err := notifRepo.ListDeliveries(ctx, orgID, 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if total != 1 || len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery record, got total=%d, len=%d", total, len(deliveries))
	}
	if deliveries[0].DestinationID != dest1.ID {
		t.Errorf("expected delivery to dest1 (%s), got %s", dest1.ID, deliveries[0].DestinationID)
	}
	if deliveries[0].Status != "queued" {
		t.Errorf("expected initial status 'queued', got %s", deliveries[0].Status)
	}

	// Re-running dispatch should be idempotent (event status is already 'delivered')
	dispatchedAgain, err := workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("DispatchPendingEvents 2nd run failed: %v", err)
	}
	if dispatchedAgain != 0 {
		t.Errorf("expected 0 dispatched on 2nd run, got %d", dispatchedAgain)
	}
}

func TestClaimDueDeliveries_ConcurrencySkipLocked(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("concurrency-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "concurrency-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("conc-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	// Drain any legacy pending events in test database
	for {
		c, err := workerNotifRepo.DispatchPendingEvents(ctx, 100)
		if err != nil || c == 0 {
			break
		}
	}

	dest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "Conc Dest", 55555, 55, nil)
	if err != nil {
		t.Fatalf("failed creating dest: %v", err)
	}

	// Create 20 pending deliveries
	for i := 0; i < 20; i++ {
		idempKey := fmt.Sprintf("conc-deliv-%s-%d", orgID, i)
		_, err := notifRepo.CreateDelivery(ctx, orgID, nil, dest.ID, "telegram_private", "55555", "incident.opened", idempKey, nil)
		if err != nil {
			t.Fatalf("failed creating delivery %d: %v", i, err)
		}
	}

	// Concurrently claim deliveries from 5 workers
	var wg sync.WaitGroup
	var claimedCount int64
	claimedIDs := sync.Map{}

	for w := 0; w < 5; w++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			wID := fmt.Sprintf("worker-%d", workerIdx)
			claimed, err := workerNotifRepo.ClaimDueDeliveries(ctx, wID, 10)
			if err != nil {
				t.Errorf("worker %d claim failed: %v", workerIdx, err)
				return
			}
			for _, d := range claimed {
				if d.OrganizationID == orgID {
					if _, loaded := claimedIDs.LoadOrStore(d.ID, true); loaded {
						t.Errorf("DUPLICATE CLAIM DETECTED: delivery %s claimed twice!", d.ID)
					}
					atomic.AddInt64(&claimedCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()

	if claimedCount == 0 {
		t.Errorf("expected at least 1 claimed delivery, got 0")
	}

	// Verify all 20 deliveries were claimed out of 'queued' state with no duplicate claims
	deliveries, _, err := notifRepo.ListDeliveries(ctx, orgID, 20, 0, "queued", "")
	if err != nil {
		t.Fatalf("failed checking queued deliveries: %v", err)
	}
	if len(deliveries) != 0 {
		t.Errorf("expected 0 queued deliveries remaining, found %d", len(deliveries))
	}
}

func TestDeliveryRetryAndTerminalFailure(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("retry-org-%d", time.Now().UnixNano())
	orgID, _ := createTestOrgAndClient(t, adminPool, "retry-org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("retry-user-%d@test.org", time.Now().UnixNano()))

	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	// Drain any legacy pending events in test database
	for {
		c, err := workerNotifRepo.DispatchPendingEvents(ctx, 100)
		if err != nil || c == 0 {
			break
		}
	}

	dest, err := notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_private", "Retry Dest", 77777, 77, nil)
	if err != nil {
		t.Fatalf("failed creating dest: %v", err)
	}

	retryIdemp := fmt.Sprintf("retry-deliv-%s", orgID)
	deliv, err := notifRepo.CreateDelivery(ctx, orgID, nil, dest.ID, "telegram_private", "77777", "incident.opened", retryIdemp, nil)
	if err != nil {
		t.Fatalf("failed creating delivery: %v", err)
	}

	// 1. Temporary failure -> marks 'retrying' with backoff
	err = workerNotifRepo.MarkDeliveryFailed(ctx, deliv.ID, "worker-test-1", "Temporary network timeout", 5*time.Second, false)
	if err != nil {
		t.Fatalf("MarkDeliveryFailed failed: %v", err)
	}

	d, _, err := notifRepo.ListDeliveries(ctx, orgID, 1, 0, "", "")
	if err != nil || len(d) == 0 {
		t.Fatalf("failed getting delivery: %v", err)
	}
	if d[0].Status != "retrying" || d[0].RetryCount != 1 {
		t.Errorf("expected status 'retrying' with retry_count 1, got status=%s, count=%d", d[0].Status, d[0].RetryCount)
	}

	// 2. Permanent failure (e.g. 403 Bot blocked) -> immediately marks terminal 'failed'
	err = workerNotifRepo.MarkDeliveryFailed(ctx, deliv.ID, "worker-test-1", "HTTP 403 Bot was blocked by user", 0, true)
	if err != nil {
		t.Fatalf("MarkDeliveryFailed permanent failed: %v", err)
	}

	d, _, err = notifRepo.ListDeliveries(ctx, orgID, 1, 0, "", "")
	if err != nil || len(d) == 0 {
		t.Fatalf("failed getting delivery: %v", err)
	}
	if d[0].Status != "failed" {
		t.Errorf("expected status 'failed', got %s", d[0].Status)
	}
}

func TestNotificationCrossTenantIsolation(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slugA := fmt.Sprintf("notif-org-a-%d", time.Now().UnixNano())
	orgA, _ := createTestOrgAndClient(t, adminPool, "notif-org-a", slugA)
	userA := createTestUser(t, adminPool, fmt.Sprintf("user-a-%d@notif.test", time.Now().UnixNano()))

	slugB := fmt.Sprintf("notif-org-b-%d", time.Now().UnixNano())
	orgB, _ := createTestOrgAndClient(t, adminPool, "notif-org-b", slugB)

	notifRepo := NewNotificationDestRepository(appPool)

	// Org A creates a destination
	destA, err := notifRepo.ConnectTelegramDestination(ctx, orgA, userA, "telegram_private", "Org A Bot", 101010, 10, nil)
	if err != nil {
		t.Fatalf("failed creating dest in Org A: %v", err)
	}

	// Org B queries destinations -> Org A destination MUST NOT be visible!
	listB, err := notifRepo.ListDestinations(ctx, orgB)
	if err != nil {
		t.Fatalf("failed listing Org B destinations: %v", err)
	}
	for _, d := range listB {
		if d.ID == destA.ID {
			t.Errorf("SECURITY LEAK: Org B can see Org A's notification destination %s", destA.ID)
		}
	}

	// Org B attempting to fetch Org A's destination directly -> MUST RETURN ErrDestinationNotFound
	_, err = notifRepo.GetDestinationByID(ctx, orgB, destA.ID)
	if err != ErrDestinationNotFound {
		t.Errorf("expected ErrDestinationNotFound for cross-tenant get, got: %v", err)
	}

	// Org B attempting to disconnect Org A's destination -> MUST RETURN ErrDestinationNotFound
	err = notifRepo.DisconnectDestination(ctx, orgB, destA.ID)
	if err != ErrDestinationNotFound {
		t.Errorf("expected ErrDestinationNotFound for cross-tenant delete, got: %v", err)
	}
}
