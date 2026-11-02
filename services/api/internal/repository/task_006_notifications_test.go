package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"clientpulse/services/api/internal/notification"
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

func TestIncidentTimeline_EventDeliveryCorrelationAndConsistency(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	// 1. Organization with Asia/Tehran timezone
	slug := fmt.Sprintf("timeline-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Timeline Agency", slug)

	// Set organization timezone to Asia/Tehran
	_, err := adminPool.Exec(ctx, `UPDATE organizations SET timezone = 'Asia/Tehran', locale = 'fa' WHERE id = $1`, orgID)
	if err != nil {
		t.Fatalf("failed setting organization timezone: %v", err)
	}

	userID := createTestUser(t, adminPool, fmt.Sprintf("timeline-user-%d@test.org", time.Now().UnixNano()))
	notifRepo := NewNotificationDestRepository(appPool)
	workerNotifRepo := NewNotificationDestRepository(workerPool)

	// Create active Telegram destination
	chatId := -int64(999000000 + time.Now().UnixNano()%100000)
	_, err = notifRepo.ConnectTelegramDestination(ctx, orgID, userID, "telegram_group", "Ops Alerts", chatId, 777, nil)
	if err != nil {
		t.Fatalf("failed creating test destination: %v", err)
	}

	// Create Asset and Monitor
	var assetID, monitorID string
	err = adminPool.QueryRow(ctx, "INSERT INTO assets (organization_id, client_id, name, url) VALUES ($1, $2, 'Main E-Commerce Store', 'https://store.digitech.com') RETURNING id", orgID, clientID).Scan(&assetID)
	if err != nil {
		t.Fatalf("failed creating asset: %v", err)
	}
	err = adminPool.QueryRow(ctx, "INSERT INTO monitors (organization_id, asset_id, method) VALUES ($1, $2, 'GET') RETURNING id", orgID, assetID).Scan(&monitorID)
	if err != nil {
		t.Fatalf("failed creating monitor: %v", err)
	}

	// 2. Incident Opened phase
	// T0: First failure at 12:56:21 UTC (16:26:21 +03:30)
	tFirstFail, _ := time.Parse(time.RFC3339, "2026-11-02T12:56:21Z")
	tConfirmed, _ := time.Parse(time.RFC3339, "2026-11-02T12:58:21Z")

	var incidentID string
	err = adminPool.QueryRow(ctx, `
		INSERT INTO incidents (organization_id, monitor_id, status, failure_category, first_failure_at, confirmed_at, summary)
		VALUES ($1, $2, 'open', 'http_5xx', $3, $4, 'HTTP 503: database connection limit exceeded')
		RETURNING id
	`, orgID, monitorID, tFirstFail, tConfirmed).Scan(&incidentID)
	if err != nil {
		t.Fatalf("failed creating incident: %v", err)
	}

	// Create incident.opened event (Event E1) at 13:10:21 UTC
	tE1Created, _ := time.Parse(time.RFC3339, "2026-11-02T13:10:21Z")
	var eventOpenedID string
	openedIdemp := fmt.Sprintf("test_manual_inc_%d", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, `
		INSERT INTO notification_events (
			organization_id, event_type, incident_id, idempotency_key, payload, status, created_at
		) VALUES (
			$1, 'incident.opened', $2, $3,
			$4::jsonb, 'pending', $5
		) RETURNING id
	`, orgID, incidentID, openedIdemp, fmt.Sprintf(`{
		"incident_id": "%s",
		"client_name": "DigiTech Solutions",
		"client_id": "%s",
		"asset_name": "Main E-Commerce Store",
		"monitor_name": "Checkout API Monitor",
		"first_failure_at": "%s",
		"confirmed_at": "%s",
		"failure_reason": "HTTP 503: database connection limit exceeded",
		"timezone": "Asia/Tehran",
		"locale": "fa"
	}`, incidentID, clientID, tFirstFail.Format(time.RFC3339), tConfirmed.Format(time.RFC3339)), tE1Created).Scan(&eventOpenedID)
	if err != nil {
		t.Fatalf("failed inserting incident.opened event: %v", err)
	}

	// Fan out to delivery D1
	_, err = workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("dispatch opened event failed: %v", err)
	}

	// Fetch delivery D1
	var delivOpenedID string
	var d1CreatedAt time.Time
	err = workerPool.QueryRow(ctx, `
		SELECT id, created_at FROM notification_deliveries WHERE event_id = $1
	`, eventOpenedID).Scan(&delivOpenedID, &d1CreatedAt)
	if err != nil {
		t.Fatalf("failed fetching delivery D1: %v", err)
	}

	// Simulate provider acceptance for D1 at 13:10:24 UTC
	tD1Sent, _ := time.Parse(time.RFC3339, "2026-11-02T13:10:24Z")
	_, err = adminPool.Exec(ctx, `
		UPDATE notification_deliveries SET status = 'sent', sent_at = $1 WHERE id = $2
	`, tD1Sent, delivOpenedID)
	if err != nil {
		t.Fatalf("failed updating delivery D1: %v", err)
	}

	// 3. Incident Resolved phase
	// Resolved at 13:14:40 UTC (16:44:40 +03:30)
	tResolved, _ := time.Parse(time.RFC3339, "2026-11-02T13:14:40Z")
	durationSec := int(tResolved.Sub(tFirstFail).Seconds()) // 1100 seconds (or 720 from secondary check)

	_, err = adminPool.Exec(ctx, `
		UPDATE incidents
		SET status = 'resolved', resolved_at = $1, duration_seconds = $2
		WHERE id = $3
	`, tResolved, durationSec, incidentID)
	if err != nil {
		t.Fatalf("failed updating incident to resolved: %v", err)
	}

	// Create incident.resolved event (Event E2) at 13:16:40 UTC
	tE2Created, _ := time.Parse(time.RFC3339, "2026-11-02T13:16:40Z")
	var eventResolvedID string
	resolvedIdemp := fmt.Sprintf("test_manual_res_%d", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, `
		INSERT INTO notification_events (
			organization_id, event_type, incident_id, idempotency_key, payload, status, created_at
		) VALUES (
			$1, 'incident.resolved', $2, $3,
			$4::jsonb, 'pending', $5
		) RETURNING id
	`, orgID, incidentID, resolvedIdemp, fmt.Sprintf(`{
		"incident_id": "%s",
		"client_name": "DigiTech Solutions",
		"client_id": "%s",
		"asset_name": "Main E-Commerce Store",
		"monitor_name": "Checkout API Monitor",
		"first_failure_at": "%s",
		"resolved_at": "%s",
		"duration_sec": %d,
		"recovery_status": "HTTP 200 OK (passed in 128ms)",
		"timezone": "Asia/Tehran",
		"locale": "fa"
	}`, incidentID, clientID, tFirstFail.Format(time.RFC3339), tResolved.Format(time.RFC3339), durationSec), tE2Created).Scan(&eventResolvedID)
	if err != nil {
		t.Fatalf("failed inserting incident.resolved event: %v", err)
	}

	// Fan out to delivery D2
	_, err = workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("dispatch resolved event failed: %v", err)
	}

	// Fetch delivery D2
	var delivResolvedID string
	var d2CreatedAt time.Time
	err = workerPool.QueryRow(ctx, `
		SELECT id, created_at FROM notification_deliveries WHERE event_id = $1
	`, eventResolvedID).Scan(&delivResolvedID, &d2CreatedAt)
	if err != nil {
		t.Fatalf("failed fetching delivery D2: %v", err)
	}

	// Simulate provider acceptance for D2 at 13:16:43 UTC
	tD2Sent, _ := time.Parse(time.RFC3339, "2026-11-02T13:16:43Z")
	_, err = adminPool.Exec(ctx, `
		UPDATE notification_deliveries SET status = 'sent', sent_at = $1 WHERE id = $2
	`, tD2Sent, delivResolvedID)
	if err != nil {
		t.Fatalf("failed updating delivery D2: %v", err)
	}

	// 4. Strict Timeline Correlation & Verification Assertions
	if eventOpenedID == eventResolvedID {
		t.Fatalf("eventOpenedID and eventResolvedID must be distinct events")
	}
	if delivOpenedID == delivResolvedID {
		t.Fatalf("delivOpenedID and delivResolvedID must be distinct deliveries")
	}

	// Verify Delivery D1 strictly belongs to Event E1 (incident.opened)
	var e1Type, d1Status string
	var d1SentAt time.Time
	err = workerPool.QueryRow(ctx, `
		SELECT e.event_type, d.status, d.sent_at
		FROM notification_deliveries d
		JOIN notification_events e ON d.event_id = e.id
		WHERE d.id = $1
	`, delivOpenedID).Scan(&e1Type, &d1Status, &d1SentAt)
	if err != nil || e1Type != "incident.opened" || d1Status != "sent" {
		t.Errorf("delivery D1 correlation failure: type=%s, status=%s, err=%v", e1Type, d1Status, err)
	}

	// Verify Delivery D2 strictly belongs to Event E2 (incident.resolved)
	var e2Type, d2Status string
	var d2SentAt time.Time
	err = workerPool.QueryRow(ctx, `
		SELECT e.event_type, d.status, d.sent_at
		FROM notification_deliveries d
		JOIN notification_events e ON d.event_id = e.id
		WHERE d.id = $1
	`, delivResolvedID).Scan(&e2Type, &d2Status, &d2SentAt)
	if err != nil || e2Type != "incident.resolved" || d2Status != "sent" {
		t.Errorf("delivery D2 correlation failure: type=%s, status=%s, err=%v", e2Type, d2Status, err)
	}

	// Timestamps consistency:
	// Event E1 created (13:10:21) <= D1 sent (13:10:24)
	// Resolved time (13:14:40) happens after E1 and D1!
	// Event E2 created (13:16:40) >= Resolved time (13:14:40)
	// D2 sent (13:16:43) >= Event E2 created (13:16:40)
	if d1SentAt.After(tResolved) {
		t.Errorf("expected D1 sent_at (%v) to precede incident resolution (%v)", d1SentAt, tResolved)
	}
	if d2SentAt.Before(tResolved) {
		t.Errorf("expected D2 sent_at (%v) to follow incident resolution (%v)", d2SentAt, tResolved)
	}
	if d2SentAt.Before(d1SentAt) {
		t.Errorf("expected D2 sent_at (%v) to follow D1 sent_at (%v)", d2SentAt, d1SentAt)
	}

	// 5. Verify Timezone Display in Asia/Tehran
	tehranLoc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatalf("failed loading Asia/Tehran timezone: %v", err)
	}
	tResolvedTehran := tResolved.In(tehranLoc)
	if tResolvedTehran.Hour() != 16 || tResolvedTehran.Minute() != 44 || tResolvedTehran.Second() != 40 {
		t.Errorf("expected 13:14:40 UTC to be 16:44:40 in Asia/Tehran, got %s", tResolvedTehran.Format("15:04:05"))
	}

	tD1SentTehran := tD1Sent.In(tehranLoc)
	if tD1SentTehran.Hour() != 16 || tD1SentTehran.Minute() != 40 || tD1SentTehran.Second() != 24 {
		t.Errorf("expected 13:10:24 UTC to be 16:40:24 in Asia/Tehran, got %s", tD1SentTehran.Format("15:04:05"))
	}

	// 6. Verify Delayed Delivery Handling:
	// If D1 (for incident.opened) was delayed in queue and processed AFTER the incident is resolved in DB:
	// ResolveAndRefreshContext MUST check current incident status in DB and flag IsAlreadyResolved = true!
	resolver := notification.NewNotificationContextResolver(workerPool)
	delayedPayload := map[string]interface{}{
		"incident_id":      incidentID,
		"client_name":      "DigiTech Solutions",
		"asset_name":       "Main E-Commerce Store",
		"monitor_name":     "Checkout API Monitor",
		"first_failure_at": tFirstFail.Format(time.RFC3339),
		"confirmed_at":     tConfirmed.Format(time.RFC3339),
		"failure_reason":   "HTTP 503: database connection limit exceeded",
	}
	delayedCtx := resolver.ResolveAndRefreshContext(ctx, orgID, "incident.opened", delayedPayload, "https://mindu.ir", "en")
	if !delayedCtx.IsAlreadyResolved {
		t.Errorf("expected delayed incident.opened delivery to be flagged as IsAlreadyResolved = true")
	}
	if delayedCtx.ResolvedAt == nil {
		t.Errorf("expected delayedCtx.ResolvedAt to be populated from DB")
	}
	if delayedCtx.DurationSeconds <= 0 {
		t.Errorf("expected positive DurationSeconds from resolved incident, got %d", delayedCtx.DurationSeconds)
	}
	if delayedCtx.Timezone != "Asia/Tehran" {
		t.Errorf("expected delayedCtx.Timezone to be Asia/Tehran, got %s", delayedCtx.Timezone)
	}
	// With destination override "en", destination locale must be preserved
	if delayedCtx.Locale != "en" {
		t.Errorf("expected destination locale override 'en' to be preserved, got %s", delayedCtx.Locale)
	}

	// Without destination override and no payload locale, org locale "fa" must be used as fallback
	delayedCtxNoDest := resolver.ResolveAndRefreshContext(ctx, orgID, "incident.opened", delayedPayload, "https://mindu.ir", "")
	if delayedCtxNoDest.Locale != "fa" {
		t.Errorf("expected fallback to organization locale 'fa', got %s", delayedCtxNoDest.Locale)
	}
}

