package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/testhelper"
)

func TestPhase2TenantIsolationAndScheduler(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	ctx := context.Background()

	orgRepo := NewOrganizationRepository(db.Pool)
	clientRepo := NewClientRepository(db.Pool)
	assetRepo := NewAssetRepository(db.Pool)
	monitorRepo := NewMonitorRepository(db.Pool)
	resultRepo := NewCheckResultRepository(db.Pool)

	// 1. Create Organization A and Organization B
	randA, _ := auth.GenerateSecureToken(4)
	orgA, err := orgRepo.CreateOrganization(ctx, "Org Alpha", fmt.Sprintf("alpha-%s", randA), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Org A: %v", err)
	}

	randB, _ := auth.GenerateSecureToken(4)
	orgB, err := orgRepo.CreateOrganization(ctx, "Org Beta", fmt.Sprintf("beta-%s", randB), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Org B: %v", err)
	}

	// 2. Org A creates Client A and Asset A
	clientA, err := clientRepo.CreateClient(ctx, orgA.ID, "Alpha Client Corp", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create Client A: %v", err)
	}

	assetA, err := assetRepo.CreateAsset(ctx, orgA.ID, clientA.ID, "Alpha Website", "https://alpha.example.com", "website", "production", nil)
	if err != nil {
		t.Fatalf("failed to create Asset A: %v", err)
	}

	monA, err := monitorRepo.CreateMonitor(ctx, orgA.ID, CreateMonitorParams{
		AssetID:         assetA.ID,
		Method:          "GET",
		IntervalSeconds: 60,
		TimeoutSeconds:  10,
	})
	if err != nil {
		t.Fatalf("failed to create Monitor A: %v", err)
	}

	// 3. Org B creates Client B and Asset B
	clientB, err := clientRepo.CreateClient(ctx, orgB.ID, "Beta Client Ltd", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create Client B: %v", err)
	}

	assetB, err := assetRepo.CreateAsset(ctx, orgB.ID, clientB.ID, "Beta Website", "https://beta.example.com", "website", "production", nil)
	if err != nil {
		t.Fatalf("failed to create Asset B: %v", err)
	}

	monB, err := monitorRepo.CreateMonitor(ctx, orgB.ID, CreateMonitorParams{
		AssetID:         assetB.ID,
		Method:          "HEAD",
		IntervalSeconds: 120,
		TimeoutSeconds:  15,
	})
	if err != nil {
		t.Fatalf("failed to create Monitor B: %v", err)
	}

	// Record check results for both
	now := time.Now().UTC()
	_, err = resultRepo.CreateCheckResult(ctx, CreateCheckResultParams{
		OrganizationID: orgA.ID,
		MonitorID:      monA.ID,
		ScheduledAt:    now,
		StartedAt:      now,
		CompletedAt:    now.Add(150 * time.Millisecond),
		LatencyMs:      150,
		Result:         "success",
		WorkerID:       "worker-test",
		ExecutionType:  "scheduled",
	})
	if err != nil {
		t.Fatalf("failed to create CheckResult A: %v", err)
	}

	_, err = resultRepo.CreateCheckResult(ctx, CreateCheckResultParams{
		OrganizationID: orgB.ID,
		MonitorID:      monB.ID,
		ScheduledAt:    now,
		StartedAt:      now,
		CompletedAt:    now.Add(200 * time.Millisecond),
		LatencyMs:      200,
		Result:         "failure",
		WorkerID:       "worker-test",
		ExecutionType:  "scheduled",
	})
	if err != nil {
		t.Fatalf("failed to create CheckResult B: %v", err)
	}

	// 4. Test RLS Isolation: Org A cannot read Org B's records
	clientsA, _, err := clientRepo.ListClients(ctx, orgA.ID, "", "", 100, 0)
	if err != nil {
		t.Fatalf("failed to list Org A clients: %v", err)
	}
	for _, c := range clientsA {
		if c.ID == clientB.ID {
			t.Fatalf("SECURITY VIOLATION: Org A list returned Org B client %s", c.ID)
		}
	}

	assetsA, _, err := assetRepo.ListAssets(ctx, orgA.ID, "", "", "", nil, 100, 0)
	if err != nil {
		t.Fatalf("failed to list Org A assets: %v", err)
	}
	for _, a := range assetsA {
		if a.ID == assetB.ID {
			t.Fatalf("SECURITY VIOLATION: Org A list returned Org B asset %s", a.ID)
		}
	}

	monitorsA, _, err := monitorRepo.ListMonitors(ctx, orgA.ID, "", "", "", 100, 0)
	if err != nil {
		t.Fatalf("failed to list Org A monitors: %v", err)
	}
	for _, m := range monitorsA {
		if m.ID == monB.ID {
			t.Fatalf("SECURITY VIOLATION: Org A list returned Org B monitor %s", m.ID)
		}
	}

	// 5. Direct Get by ID cross-tenant rejection
	_, err = clientRepo.GetClientByID(ctx, orgA.ID, clientB.ID)
	if err == nil {
		t.Fatal("SECURITY VIOLATION: Org A successfully fetched Org B's client by ID")
	}

	_, err = assetRepo.GetAssetByID(ctx, orgA.ID, assetB.ID)
	if err == nil {
		t.Fatal("SECURITY VIOLATION: Org A successfully fetched Org B's asset by ID")
	}

	_, err = monitorRepo.GetMonitorByID(ctx, orgA.ID, monB.ID)
	if err == nil {
		t.Fatal("SECURITY VIOLATION: Org A successfully fetched Org B's monitor by ID")
	}

	// 6. Cross-tenant parent-child creation rejection
	_, err = assetRepo.CreateAsset(ctx, orgA.ID, clientB.ID, "Malicious Asset", "https://malicious.example.com", "website", "production", nil)
	if err == nil {
		t.Fatal("SECURITY VIOLATION: Org A created asset linked to Org B's client")
	}

	_, err = monitorRepo.CreateMonitor(ctx, orgA.ID, CreateMonitorParams{
		AssetID: assetB.ID,
		Method:  "GET",
	})
	if err == nil {
		t.Fatal("SECURITY VIOLATION: Org A created monitor linked to Org B's asset")
	}

	// 7. Test Atomic Scheduler Claiming
	claimed, err := monitorRepo.ClaimDueMonitors(ctx, "worker-1", 10, 60)
	if err != nil {
		t.Fatalf("failed to claim due monitors: %v", err)
	}
	if len(claimed) < 2 {
		t.Fatalf("expected at least 2 due monitors to be claimed, got %d", len(claimed))
	}

	// Immediate re-claim must return 0 because monitors are locked with active lease!
	reclaimed, err := monitorRepo.ClaimDueMonitors(ctx, "worker-2", 10, 60)
	if err != nil {
		t.Fatalf("failed to re-claim due monitors: %v", err)
	}
	// Neither monA nor monB should be re-claimed while lease is active
	for _, job := range reclaimed {
		if job.ID == monA.ID || job.ID == monB.ID {
			t.Fatalf("DUPLICATE CLAIM VIOLATION: Monitor %s was claimed while lease was active", job.ID)
		}
	}

	// 8. Test UpdateAfterCheck with misfire drift prevention
	completedAt := time.Now().UTC()
	err = monitorRepo.UpdateAfterCheck(ctx, orgA.ID, monA.ID, "up", completedAt, now, 60)
	if err != nil {
		t.Fatalf("failed to update monitor after check: %v", err)
	}

	updatedMonA, err := monitorRepo.GetMonitorByID(ctx, orgA.ID, monA.ID)
	if err != nil {
		t.Fatalf("failed to get updated monitor A: %v", err)
	}
	if updatedMonA.Status != "up" {
		t.Fatalf("expected status 'up', got %s", updatedMonA.Status)
	}
	if updatedMonA.LeaseWorkerID != nil {
		t.Fatalf("expected lease to be cleared, got %v", updatedMonA.LeaseWorkerID)
	}

	// 9. Test Client Archiving stops child monitors and assets
	err = clientRepo.ArchiveClient(ctx, orgA.ID, clientA.ID)
	if err != nil {
		t.Fatalf("failed to archive client A: %v", err)
	}

	archivedAssetA, err := assetRepo.GetAssetByID(ctx, orgA.ID, assetA.ID)
	if err != nil {
		t.Fatalf("failed to fetch asset A after client archive: %v", err)
	}
	if archivedAssetA.IsActive {
		t.Fatal("expected asset A to be inactive after client archive")
	}

	archivedMonA, err := monitorRepo.GetMonitorByID(ctx, orgA.ID, monA.ID)
	if err != nil {
		t.Fatalf("failed to fetch monitor A after client archive: %v", err)
	}
	if archivedMonA.Status != "paused" {
		t.Fatalf("expected monitor A to be paused, got %s", archivedMonA.Status)
	}
}
