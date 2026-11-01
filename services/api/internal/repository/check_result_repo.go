package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckResultRepository struct {
	pool *pgxpool.Pool
}

func NewCheckResultRepository(pool *pgxpool.Pool) *CheckResultRepository {
	return &CheckResultRepository{pool: pool}
}

type CreateCheckResultParams struct {
	OrganizationID   string
	MonitorID        string
	ScheduledAt      time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
	LatencyMs        int64
	Result           string
	HTTPStatusCode   *int
	ErrorCategory    *string
	ErrorMessage     *string
	TLSVersion       *string
	TLSCipher        *string
	TLSCertIssuer    *string
	TLSCertSubject   *string
	TLSCertNotBefore *time.Time
	TLSCertNotAfter  *time.Time
	TLSDaysRemaining *int
	TLSIsValid       *bool
	ResolvedIP       *string
	WorkerID         string
	ExecutionType    string
}

func (r *CheckResultRepository) CreateCheckResult(ctx context.Context, params CreateCheckResultParams) (*CheckResult, error) {
	if params.ExecutionType == "" {
		params.ExecutionType = "scheduled"
	}

	var res CheckResult
	err := WithTenantTx(ctx, r.pool, params.OrganizationID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO check_results (
				organization_id, monitor_id, scheduled_at, started_at, completed_at,
				latency_ms, result, http_status_code, error_category, error_message,
				tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject,
				tls_cert_not_before, tls_cert_not_after, tls_days_remaining, tls_is_valid,
				resolved_ip, worker_id, execution_type, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW())
			RETURNING id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
			          latency_ms, result, http_status_code, error_category, error_message,
			          tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject,
			          tls_cert_not_before, tls_cert_not_after, tls_days_remaining, tls_is_valid,
			          resolved_ip, worker_id, execution_type, created_at
		`
		return tx.QueryRow(ctx, query,
			params.OrganizationID, params.MonitorID, params.ScheduledAt, params.StartedAt, params.CompletedAt,
			params.LatencyMs, params.Result, params.HTTPStatusCode, params.ErrorCategory, params.ErrorMessage,
			params.TLSVersion, params.TLSCipher, params.TLSCertIssuer, params.TLSCertSubject,
			params.TLSCertNotBefore, params.TLSCertNotAfter, params.TLSDaysRemaining, params.TLSIsValid,
			params.ResolvedIP, params.WorkerID, params.ExecutionType,
		).Scan(
			&res.ID, &res.OrganizationID, &res.MonitorID, &res.ScheduledAt, &res.StartedAt, &res.CompletedAt,
			&res.LatencyMs, &res.Result, &res.HTTPStatusCode, &res.ErrorCategory, &res.ErrorMessage,
			&res.TLSVersion, &res.TLSCipher, &res.TLSCertIssuer, &res.TLSCertSubject,
			&res.TLSCertNotBefore, &res.TLSCertNotAfter, &res.TLSDaysRemaining, &res.TLSIsValid,
			&res.ResolvedIP, &res.WorkerID, &res.ExecutionType, &res.CreatedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create check result: %w", err)
	}

	return &res, nil
}

func (r *CheckResultRepository) ListMonitorResults(ctx context.Context, orgID, monitorID string, limit, offset int) ([]*CheckResult, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var results []*CheckResult
	var total int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		countQuery := `SELECT COUNT(*) FROM check_results WHERE monitor_id = $1 AND organization_id = $2`
		if err := tx.QueryRow(ctx, countQuery, monitorID, orgID).Scan(&total); err != nil {
			return err
		}

		query := `
			SELECT id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
			       latency_ms, result, http_status_code, error_category, error_message,
			       tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject,
			       tls_cert_not_before, tls_cert_not_after, tls_days_remaining, tls_is_valid,
			       resolved_ip, worker_id, execution_type, created_at
			FROM check_results
			WHERE monitor_id = $1 AND organization_id = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		rows, err := tx.Query(ctx, query, monitorID, orgID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var res CheckResult
			if err := rows.Scan(
				&res.ID, &res.OrganizationID, &res.MonitorID, &res.ScheduledAt, &res.StartedAt, &res.CompletedAt,
				&res.LatencyMs, &res.Result, &res.HTTPStatusCode, &res.ErrorCategory, &res.ErrorMessage,
				&res.TLSVersion, &res.TLSCipher, &res.TLSCertIssuer, &res.TLSCertSubject,
				&res.TLSCertNotBefore, &res.TLSCertNotAfter, &res.TLSDaysRemaining, &res.TLSIsValid,
				&res.ResolvedIP, &res.WorkerID, &res.ExecutionType, &res.CreatedAt,
			); err != nil {
				return err
			}
			results = append(results, &res)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *CheckResultRepository) GetDashboardStats(ctx context.Context, orgID string) (*DashboardStats, error) {
	stats := &DashboardStats{
		RecentChecks: []*CheckResult{},
	}

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Total Clients
		_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM clients WHERE organization_id = $1 AND status = 'active'", orgID).Scan(&stats.TotalClients)

		// Total Assets
		_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE organization_id = $1 AND is_active = true", orgID).Scan(&stats.TotalAssets)

		// Monitors counts by status
		rows, err := tx.Query(ctx, "SELECT status, COUNT(*) FROM monitors WHERE organization_id = $1 AND is_active = true GROUP BY status", orgID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var status string
				var count int
				if err := rows.Scan(&status, &count); err == nil {
					switch status {
					case "up":
						stats.MonitorsUp = count
					case "down":
						stats.MonitorsDown = count
					case "pending":
						stats.MonitorsPending = count
					case "paused":
						stats.MonitorsPaused = count
					}
				}
			}
		}

		// Recent 10 check results
		recentQuery := `
			SELECT id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
			       latency_ms, result, http_status_code, error_category, error_message,
			       tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject,
			       tls_cert_not_before, tls_cert_not_after, tls_days_remaining, tls_is_valid,
			       resolved_ip, worker_id, execution_type, created_at
			FROM check_results
			WHERE organization_id = $1
			ORDER BY created_at DESC
			LIMIT 10
		`
		recentRows, err := tx.Query(ctx, recentQuery, orgID)
		if err == nil {
			defer recentRows.Close()
			for recentRows.Next() {
				var res CheckResult
				if err := recentRows.Scan(
					&res.ID, &res.OrganizationID, &res.MonitorID, &res.ScheduledAt, &res.StartedAt, &res.CompletedAt,
					&res.LatencyMs, &res.Result, &res.HTTPStatusCode, &res.ErrorCategory, &res.ErrorMessage,
					&res.TLSVersion, &res.TLSCipher, &res.TLSCertIssuer, &res.TLSCertSubject,
					&res.TLSCertNotBefore, &res.TLSCertNotAfter, &res.TLSDaysRemaining, &res.TLSIsValid,
					&res.ResolvedIP, &res.WorkerID, &res.ExecutionType, &res.CreatedAt,
				); err == nil {
					stats.RecentChecks = append(stats.RecentChecks, &res)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}
	return stats, nil
}

// PruneOldResults deletes check results older than the given duration (retention policy).
func (r *CheckResultRepository) PruneOldResults(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-retention)
	res, err := r.pool.Exec(ctx, "DELETE FROM check_results WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old check results: %w", err)
	}
	return res.RowsAffected(), nil
}
