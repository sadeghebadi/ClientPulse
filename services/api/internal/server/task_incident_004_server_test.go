package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5"
)

func TestTaskIncident004_ServerEndpoints(t *testing.T) {
	srv, authService, userRepo, _, incidentRepo, cleanup := setupPhase3Server(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create and verify user
	suffix, _ := auth.GenerateSecureToken(4)
	email := fmt.Sprintf("incident004-api-%s@testpulse.local", suffix)
	signupRes, err := authService.Signup(ctx, service.SignupRequest{
		Email:            email,
		Password:         "SecurePassword123!",
		DisplayName:      "Incident Manager",
		OrganizationName: "Lifecycle Corp",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	if err := userRepo.SetEmailVerified(ctx, signupRes.User.ID); err != nil {
		t.Fatalf("failed to verify email: %v", err)
	}

	doAuthedReq := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, reqBody)
		req.Header.Set("X-Organization-ID", signupRes.Organization.ID)
		req.Header.Set("X-CSRF-Token", signupRes.RawCSRF)
		req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: signupRes.RawSession})
		req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: signupRes.RawCSRF})
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(w, req)
		return w
	}

	orgID := signupRes.Organization.ID

	// Create Client
	clientW := doAuthedReq("POST", "/api/v1/clients", map[string]interface{}{
		"name": "Acme Monitoring Corp",
	})
	if clientW.Code != http.StatusCreated {
		t.Fatalf("client creation failed: %d", clientW.Code)
	}
	var clientResp struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
	}
	_ = json.NewDecoder(clientW.Body).Decode(&clientResp)
	clientID := clientResp.Client.ID

	// Create Heartbeat Monitor
	hbW := doAuthedReq("POST", "/api/v1/heartbeats", map[string]interface{}{
		"client_id":                 clientID,
		"name":                      "Billing Ingestion Cron",
		"expected_interval_seconds": 3600,
		"grace_period_seconds":      300,
	})
	if hbW.Code != http.StatusCreated {
		t.Fatalf("heartbeat creation failed: %d", hbW.Code)
	}
	var hbResp struct {
		Heartbeat struct {
			ID string `json:"id"`
		} `json:"heartbeat"`
	}
	_ = json.NewDecoder(hbW.Body).Decode(&hbResp)
	hbID := hbResp.Heartbeat.ID

	// Create an open incident directly for this heartbeat
	var inc1 *repository.Incident
	err = repository.WithTenantTx(ctx, testhelper.GetTestDB(t).Pool, orgID, func(tx pgx.Tx) error {
		var created bool
		var err error
		inc1, created, err = incidentRepo.CreateHeartbeatIncidentTx(
			ctx, tx, orgID, hbID, time.Now().UTC().Add(-15*time.Minute), time.Now().UTC().Add(-5*time.Minute),
			"heartbeat_missed", "Billing cron missed expected schedule",
		)
		if !created {
			return fmt.Errorf("failed to create incident")
		}
		return err
	})
	if err != nil {
		t.Fatalf("error creating inc1: %v", err)
	}

	// 1. Test Manual Resolution via POST /api/v1/incidents/{id}/resolve
	t.Run("Manual Resolution Endpoint", func(t *testing.T) {
		resW := doAuthedReq("POST", fmt.Sprintf("/api/v1/incidents/%s/resolve", inc1.ID), map[string]interface{}{
			"note": "Resolved by on-call operator after restarting billing service",
		})
		if resW.Code != http.StatusOK {
			t.Fatalf("expected 200 from manual resolve, got %d: %s", resW.Code, resW.Body.String())
		}

		var resResp struct {
			Status   string `json:"status"`
			Incident struct {
				ID             string  `json:"id"`
				Status         string  `json:"status"`
				ResolutionType *string `json:"resolution_type"`
				ResolutionNote *string `json:"resolution_note"`
				ResolvedBy     *string `json:"resolved_by"`
			} `json:"incident"`
		}
		_ = json.NewDecoder(resW.Body).Decode(&resResp)

		if resResp.Incident.Status != "resolved" {
			t.Fatalf("expected status 'resolved', got %s", resResp.Incident.Status)
		}
		if resResp.Incident.ResolutionType == nil || *resResp.Incident.ResolutionType != "manual" {
			t.Fatalf("expected resolution_type 'manual', got %v", resResp.Incident.ResolutionType)
		}
		if resResp.Incident.ResolutionNote == nil || *resResp.Incident.ResolutionNote != "Resolved by on-call operator after restarting billing service" {
			t.Fatalf("resolution note mismatch: %v", resResp.Incident.ResolutionNote)
		}
		if resResp.Incident.ResolvedBy == nil || *resResp.Incident.ResolvedBy != signupRes.User.ID {
			t.Fatalf("resolved_by mismatch: expected %s, got %v", signupRes.User.ID, resResp.Incident.ResolvedBy)
		}

		// Second resolve attempt must return 400 ALREADY_RESOLVED
		resW2 := doAuthedReq("POST", fmt.Sprintf("/api/v1/incidents/%s/resolve", inc1.ID), nil)
		if resW2.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for repeated resolve, got %d", resW2.Code)
		}

		// Acknowledge on resolved incident must return 400 ALREADY_RESOLVED
		ackW := doAuthedReq("POST", fmt.Sprintf("/api/v1/incidents/%s/acknowledge", inc1.ID), nil)
		if ackW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for ack on resolved incident, got %d", ackW.Code)
		}
	})

	// 2. Test Heartbeat Deletion Resolving Open Incidents
	t.Run("Heartbeat Deletion Endpoint Resolves Open Incident", func(t *testing.T) {
		// Create a second heartbeat
		hbW2 := doAuthedReq("POST", "/api/v1/heartbeats", map[string]interface{}{
			"client_id":                 clientID,
			"name":                      "Log Pruner Daemon",
			"expected_interval_seconds": 1800,
			"grace_period_seconds":      120,
		})
		if hbW2.Code != http.StatusCreated {
			t.Fatalf("hb2 creation failed: %d", hbW2.Code)
		}
		var hbResp2 struct {
			Heartbeat struct {
				ID string `json:"id"`
			} `json:"heartbeat"`
		}
		_ = json.NewDecoder(hbW2.Body).Decode(&hbResp2)
		hbID2 := hbResp2.Heartbeat.ID

		// Create an open incident for hbID2
		var inc2 *repository.Incident
		err = repository.WithTenantTx(ctx, testhelper.GetTestDB(t).Pool, orgID, func(tx pgx.Tx) error {
			var created bool
			var err error
			inc2, created, err = incidentRepo.CreateHeartbeatIncidentTx(
				ctx, tx, orgID, hbID2, time.Now().UTC().Add(-20*time.Minute), time.Now().UTC().Add(-10*time.Minute),
				"heartbeat_missed", "Log pruner missed schedule",
			)
			if !created {
				return fmt.Errorf("failed to create inc2")
			}
			return err
		})
		if err != nil {
			t.Fatalf("error creating inc2: %v", err)
		}

		// Delete Heartbeat Monitor via DELETE /api/v1/heartbeats/{id}
		delW := doAuthedReq("DELETE", fmt.Sprintf("/api/v1/heartbeats/%s", hbID2), nil)
		if delW.Code != http.StatusOK {
			t.Fatalf("expected 200 from delete heartbeat, got %d: %s", delW.Code, delW.Body.String())
		}

		// Verify Incident detail now reflects resolved status and resolution_type = 'source_monitor_deleted'
		detailW := doAuthedReq("GET", fmt.Sprintf("/api/v1/incidents/%s", inc2.ID), nil)
		if detailW.Code != http.StatusOK {
			t.Fatalf("expected 200 from get incident detail, got %d", detailW.Code)
		}

		var detailResp struct {
			Incident struct {
				ID                  string  `json:"id"`
				Status              string  `json:"status"`
				ResolutionType      *string `json:"resolution_type"`
				ResolvedBy          *string `json:"resolved_by"`
				HeartbeatArchivedAt *string `json:"heartbeat_archived_at"`
			} `json:"incident"`
		}
		_ = json.NewDecoder(detailW.Body).Decode(&detailResp)

		if detailResp.Incident.Status != "resolved" {
			t.Fatalf("expected inc2 to be resolved after heartbeat deletion, got %s", detailResp.Incident.Status)
		}
		if detailResp.Incident.ResolutionType == nil || *detailResp.Incident.ResolutionType != "source_monitor_deleted" {
			t.Fatalf("expected resolution_type 'source_monitor_deleted', got %v", detailResp.Incident.ResolutionType)
		}
		if detailResp.Incident.ResolvedBy == nil || *detailResp.Incident.ResolvedBy != signupRes.User.ID {
			t.Fatalf("expected resolved_by to be deleting user %s, got %v", signupRes.User.ID, detailResp.Incident.ResolvedBy)
		}
		if detailResp.Incident.HeartbeatArchivedAt == nil {
			t.Fatalf("expected heartbeat_archived_at to be populated")
		}
	})

	// 3. Test Query Filters for Incidents List
	t.Run("Incident List Filters", func(t *testing.T) {
		// Filter by target_type=heartbeat
		hbListW := doAuthedReq("GET", "/api/v1/incidents?target_type=heartbeat", nil)
		if hbListW.Code != http.StatusOK {
			t.Fatalf("expected 200 from filtered incidents, got %d", hbListW.Code)
		}

		// Filter by ack_state=unack
		unackListW := doAuthedReq("GET", "/api/v1/incidents?ack_state=unack", nil)
		if unackListW.Code != http.StatusOK {
			t.Fatalf("expected 200 from unack filter, got %d", unackListW.Code)
		}
	})
}

