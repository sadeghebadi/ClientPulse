package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"clientpulse/services/api/internal/pdf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrReportNotFound     = errors.New("report not found")
	ErrShareTokenNotFound = errors.New("report share link not found or has expired")
	ErrInvalidPeriod      = errors.New("period end must be strictly after period start")
)

type ReportRepository struct {
	pool         *pgxpool.Pool
	brandingRepo *BrandingRepository
}

func NewReportRepository(pool *pgxpool.Pool, brandingRepo *BrandingRepository) *ReportRepository {
	return &ReportRepository{
		pool:         pool,
		brandingRepo: brandingRepo,
	}
}

type ReportFilter struct {
	ClientID   *string
	ReportType *string
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

// GenerateShareTokenMaterial generates at least 32 cryptographically secure random bytes.
func GenerateShareTokenMaterial() (plaintextToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed reading random bytes: %w", err)
	}
	plaintextToken = "cp_sh_" + hex.EncodeToString(bytes)

	hashBytes := sha256.Sum256([]byte(plaintextToken))
	tokenHash = hex.EncodeToString(hashBytes[:])
	return plaintextToken, tokenHash, nil
}

// HashShareToken calculates SHA-256 digest of incoming plaintext share token.
func HashShareToken(token string) string {
	hashBytes := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(hashBytes[:])
}

// CreateReport inserts a brand-new immutable report version with full calculated snapshot JSON.
func (r *ReportRepository) CreateReport(
	ctx context.Context,
	orgID, clientID string,
	reportType string,
	periodStart, periodEnd time.Time,
	orgTimezone, locale, title string,
	snapshot pdf.ReportData,
	generatedBy string,
) (*Report, error) {
	if !periodEnd.After(periodStart) {
		return nil, ErrInvalidPeriod
	}

	if reportType == "" {
		reportType = "monthly_care"
	}
	if locale != "fa" {
		locale = "en"
	}
	if orgTimezone == "" {
		orgTimezone = "UTC"
	}
	if title == "" {
		title = fmt.Sprintf("Website Care Report - %s (%s)", snapshot.Client.Name, periodStart.Format("2006-01"))
	}

	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed serializing report snapshot: %w", err)
	}

	lockKey := fmt.Sprintf("report_ver:%s:%s:%s:%s", orgID, clientID, periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339))

	var rep Report
	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// 1. Acquire transaction-level advisory lock to serialize concurrent report version generation
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
			return fmt.Errorf("failed acquiring advisory lock for report versioning: %w", err)
		}

		// 2. Determine next version number atomically for this exact client & period
		var nextVersion int
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(version_number), 0) + 1
			FROM reports
			WHERE organization_id = $1 AND client_id = $2
			  AND period_start = $3 AND period_end = $4
		`, orgID, clientID, periodStart, periodEnd).Scan(&nextVersion)
		if err != nil {
			return fmt.Errorf("failed calculating next version number: %w", err)
		}

		query := `
			INSERT INTO reports (
				organization_id, client_id, report_type, period_start, period_end,
				organization_timezone, locale, version_number, template_version,
				title, snapshot_json, generated_by, generated_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW(), NOW())
			RETURNING id, organization_id, client_id, report_type, period_start, period_end,
			          organization_timezone, locale, version_number, template_version,
			          title, snapshot_json, pdf_storage_key, generated_by, generated_at, created_at, updated_at
		`

		return tx.QueryRow(ctx, query,
			orgID, clientID, reportType, periodStart, periodEnd,
			orgTimezone, locale, nextVersion, "v1.0",
			title, snapshotJSON, generatedBy,
		).Scan(
			&rep.ID, &rep.OrganizationID, &rep.ClientID, &rep.ReportType, &rep.PeriodStart, &rep.PeriodEnd,
			&rep.OrganizationTimezone, &rep.Locale, &rep.VersionNumber, &rep.TemplateVersion,
			&rep.Title, &rep.SnapshotJSON, &rep.PDFStorageKey, &rep.GeneratedBy, &rep.GeneratedAt,
			&rep.CreatedAt, &rep.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}
	return &rep, nil
}

// GetReport retrieves a single report version by ID.
func (r *ReportRepository) GetReport(ctx context.Context, orgID, reportID string) (*Report, error) {
	var rep Report
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT r.id, r.organization_id, r.client_id, r.report_type, r.period_start, r.period_end,
			       r.organization_timezone, r.locale, r.version_number, r.template_version,
			       r.title, r.snapshot_json, r.pdf_storage_key, r.generated_by, r.generated_at,
			       r.created_at, r.updated_at,
			       c.name AS client_name,
			       u.display_name AS generated_by_name,
			       u.email AS generated_by_email
			FROM reports r
			JOIN clients c ON c.id = r.client_id AND c.organization_id = r.organization_id
			JOIN users u ON u.id = r.generated_by
			WHERE r.id = $1 AND r.organization_id = $2
		`

		return tx.QueryRow(ctx, query, reportID, orgID).Scan(
			&rep.ID, &rep.OrganizationID, &rep.ClientID, &rep.ReportType, &rep.PeriodStart, &rep.PeriodEnd,
			&rep.OrganizationTimezone, &rep.Locale, &rep.VersionNumber, &rep.TemplateVersion,
			&rep.Title, &rep.SnapshotJSON, &rep.PDFStorageKey, &rep.GeneratedBy, &rep.GeneratedAt,
			&rep.CreatedAt, &rep.UpdatedAt,
			&rep.ClientName, &rep.GeneratedByName, &rep.GeneratedByEmail,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportNotFound
		}
		return nil, err
	}
	return &rep, nil
}

// ListReports queries reports with pagination and filtering.
func (r *ReportRepository) ListReports(ctx context.Context, orgID string, filter ReportFilter) ([]*Report, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var reports []*Report
	var totalCount int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		conditions := []string{"r.organization_id = $1"}
		args := []interface{}{orgID}
		argIdx := 2

		if filter.ClientID != nil && *filter.ClientID != "" {
			conditions = append(conditions, fmt.Sprintf("r.client_id = $%d", argIdx))
			args = append(args, *filter.ClientID)
			argIdx++
		}

		if filter.ReportType != nil && *filter.ReportType != "" {
			conditions = append(conditions, fmt.Sprintf("r.report_type = $%d", argIdx))
			args = append(args, *filter.ReportType)
			argIdx++
		}

		if filter.From != nil {
			conditions = append(conditions, fmt.Sprintf("r.period_start >= $%d", argIdx))
			args = append(args, *filter.From)
			argIdx++
		}

		if filter.To != nil {
			conditions = append(conditions, fmt.Sprintf("r.period_end <= $%d", argIdx))
			args = append(args, *filter.To)
			argIdx++
		}

		whereClause := strings.Join(conditions, " AND ")

		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reports r WHERE %s", whereClause)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
			return fmt.Errorf("failed to count reports: %w", err)
		}

		query := fmt.Sprintf(`
			SELECT r.id, r.organization_id, r.client_id, r.report_type, r.period_start, r.period_end,
			       r.organization_timezone, r.locale, r.version_number, r.template_version,
			       r.title, r.pdf_storage_key, r.generated_by, r.generated_at,
			       r.created_at, r.updated_at,
			       c.name AS client_name,
			       u.display_name AS generated_by_name,
			       u.email AS generated_by_email
			FROM reports r
			JOIN clients c ON c.id = r.client_id AND c.organization_id = r.organization_id
			JOIN users u ON u.id = r.generated_by
			WHERE %s
			ORDER BY r.generated_at DESC
			LIMIT $%d OFFSET $%d
		`, whereClause, argIdx, argIdx+1)

		args = append(args, filter.Limit, filter.Offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed querying reports: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var rep Report
			if err := rows.Scan(
				&rep.ID, &rep.OrganizationID, &rep.ClientID, &rep.ReportType, &rep.PeriodStart, &rep.PeriodEnd,
				&rep.OrganizationTimezone, &rep.Locale, &rep.VersionNumber, &rep.TemplateVersion,
				&rep.Title, &rep.PDFStorageKey, &rep.GeneratedBy, &rep.GeneratedAt,
				&rep.CreatedAt, &rep.UpdatedAt,
				&rep.ClientName, &rep.GeneratedByName, &rep.GeneratedByEmail,
			); err != nil {
				return fmt.Errorf("failed scanning report: %w", err)
			}
			reports = append(reports, &rep)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return reports, totalCount, nil
}

// UpdateReportPDFStorageKey sets the PDF storage key once the PDF is rendered.
func (r *ReportRepository) UpdateReportPDFStorageKey(ctx context.Context, orgID, reportID, storageKey string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE reports
			SET pdf_storage_key = $1, updated_at = NOW()
			WHERE id = $2 AND organization_id = $3
		`, storageKey, reportID, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrReportNotFound
		}
		return nil
	})
}

// CreateShareToken generates an unauthenticated read-only share link token.
func (r *ReportRepository) CreateShareToken(ctx context.Context, orgID, reportID string, expiresAt *time.Time, createdBy string) (string, *ReportShareToken, error) {
	plaintextToken, tokenHash, err := GenerateShareTokenMaterial()
	if err != nil {
		return "", nil, err
	}

	var st ReportShareToken
	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Verify report exists and belongs to org
		var reportExists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM reports WHERE id = $1 AND organization_id = $2)
		`, reportID, orgID).Scan(&reportExists)
		if err != nil {
			return err
		}
		if !reportExists {
			return ErrReportNotFound
		}

		query := `
			INSERT INTO report_share_tokens (
				organization_id, report_id, token_hash, expires_at, created_by, created_at
			)
			VALUES ($1, $2, $3, $4, $5, NOW())
			RETURNING id, organization_id, report_id, expires_at, revoked_at, created_by, created_at
		`

		return tx.QueryRow(ctx, query,
			orgID, reportID, tokenHash, expiresAt, createdBy,
		).Scan(
			&st.ID, &st.OrganizationID, &st.ReportID, &st.ExpiresAt, &st.RevokedAt, &st.CreatedBy, &st.CreatedAt,
		)
	})

	if err != nil {
		return "", nil, err
	}
	st.IsActive = true
	return plaintextToken, &st, nil
}

// RevokeShareToken immediately invalidates an active share link.
func (r *ReportRepository) RevokeShareToken(ctx context.Context, orgID, shareID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE report_share_tokens
			SET revoked_at = NOW()
			WHERE id = $1 AND organization_id = $2 AND revoked_at IS NULL
		`, shareID, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrShareTokenNotFound
		}
		return nil
	})
}

// ListShareTokens returns all share tokens created for a report.
func (r *ReportRepository) ListShareTokens(ctx context.Context, orgID, reportID string) ([]*ReportShareToken, error) {
	var tokens []*ReportShareToken
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, organization_id, report_id, expires_at, revoked_at, created_by, created_at
			FROM report_share_tokens
			WHERE report_id = $1 AND organization_id = $2
			ORDER BY created_at DESC
		`, reportID, orgID)
		if err != nil {
			return err
		}
		defer rows.Close()

		now := time.Now().UTC()
		for rows.Next() {
			var st ReportShareToken
			if err := rows.Scan(
				&st.ID, &st.OrganizationID, &st.ReportID, &st.ExpiresAt, &st.RevokedAt, &st.CreatedBy, &st.CreatedAt,
			); err != nil {
				return err
			}
			st.IsActive = st.RevokedAt == nil && (st.ExpiresAt == nil || st.ExpiresAt.After(now))
			tokens = append(tokens, &st)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// LookupReportByShareToken resolves an unauthenticated public share token via SECURITY DEFINER procedure.
func (r *ReportRepository) LookupReportByShareToken(ctx context.Context, plaintextToken string) (*PublicReportView, string, error) {
	tokenHash := HashShareToken(plaintextToken)

	var v PublicReportView
	var pdfStorageKey *string
	var orgID, shareID, clientID string

	query := `
		SELECT share_id, report_id, organization_id, client_id, report_type,
		       period_start, period_end, organization_timezone, locale,
		       version_number, template_version, title, snapshot_json,
		       pdf_storage_key, generated_at, expires_at, revoked_at
		FROM lookup_report_by_share_token_hash($1)
	`

	var revokedAt *time.Time
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&shareID, &v.ReportID, &orgID, &clientID, &v.ReportType,
		&v.PeriodStart, &v.PeriodEnd, &v.OrganizationTimezone, &v.Locale,
		&v.VersionNumber, &v.TemplateVersion, &v.Title, &v.SnapshotJSON,
		&pdfStorageKey, &v.GeneratedAt, &v.ExpiresAt, &revokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrShareTokenNotFound
		}
		return nil, "", err
	}

	if revokedAt != nil || (v.ExpiresAt != nil && v.ExpiresAt.Before(time.Now().UTC())) {
		return nil, "", ErrShareTokenNotFound
	}

	v.HasPDF = pdfStorageKey != nil && *pdfStorageKey != ""
	key := ""
	if pdfStorageKey != nil {
		key = *pdfStorageKey
	}
	return &v, key, nil
}

// GetOrganizationTimezone retrieves the organization's configured IANA timezone, defaulting to UTC.
func (r *ReportRepository) GetOrganizationTimezone(ctx context.Context, orgID string) (string, error) {
	var tz string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(timezone, 'UTC') FROM organizations WHERE id = $1`, orgID).Scan(&tz)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "UTC", nil
		}
		return "UTC", err
	}
	if tz == "" {
		return "UTC", nil
	}
	return tz, nil
}

// CalculateReportData performs deterministic aggregation of monitoring, maintenance, incident, and heartbeat data.
//
// Formula & Methodology Specification:
// 1. Monitoring Eligible Period (Monitored Sec):
//    The actual time window within [PeriodStart, PeriodEnd] during which the monitor was eligible to run checks.
//    - CovStart = max(PeriodStart, Monitor.CreatedAt)
//    - CovEnd = min(PeriodEnd, Asset.ArchivedAt if archived, DeactivationTime if deactivated)
//    - Paused periods (tracked via audit logs and status) are strictly subtracted from the eligible window.
// 2. Monitoring Coverage Percentage:
//    Proportion of expected scheduled checks actually executed and recorded:
//    Coverage% = (Executed Scheduled Checks / Expected Scheduled Checks) * 100.
//    - Expected Checks = Eligible Seconds / Monitor.IntervalSeconds.
//    - Executed Scheduled Checks include both successful and failed checks (since monitoring actively occurred).
//    - Missing scheduled checks (e.g. worker outages) directly reduce coverage.
//    - Paused intervals are excluded from expected checks.
// 3. Uptime Percentage:
//    Proportion of eligible monitored time during which the service was healthy:
//    Uptime% = ((Eligible Seconds - Outage Seconds) / Eligible Seconds) * 100.
//    - Outage Seconds include overlapping confirmed incident intervals and executed failed checks.
//    - Failed executed checks do NOT count as uptime.
//    - Paused periods are strictly excluded from the uptime denominator.
//    - If 0 scheduled checks were executed or eligible seconds == 0, Uptime is null (InsufficientData = true).
// 4. Manual checks and synthetic tests are strictly excluded from all uptime and coverage calculations.
func (r *ReportRepository) CalculateReportData(ctx context.Context, orgID, clientID string, periodStart, periodEnd time.Time, orgTimezone, locale string) (*pdf.ReportData, error) {
	if !periodEnd.After(periodStart) {
		return nil, ErrInvalidPeriod
	}

	// 1. Fetch Branding
	branding, err := r.brandingRepo.GetBranding(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching branding: %w", err)
	}

	if locale == "" {
		locale = branding.DefaultReportLocale
	}
	if locale != "fa" {
		locale = "en"
	}
	if orgTimezone == "" {
		orgTimezone = "UTC"
	}

	var data pdf.ReportData
	data.ReportType = "monthly_care"
	data.Locale = locale
	data.PeriodStart = periodStart
	data.PeriodEnd = periodEnd
	data.OrganizationTimezone = orgTimezone
	data.GeneratedAt = time.Now().UTC()
	data.TemplateVersion = "v1.0"
	data.Branding = pdf.BrandingData{
		DisplayName:  branding.DisplayName,
		PrimaryColor: branding.PrimaryColor,
		AccentColor:  branding.AccentColor,
		FooterText:   branding.ReportFooterText,
		ContactEmail: branding.ContactEmail,
		WebsiteURL:   branding.WebsiteURL,
	}

	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// 2. Fetch Client Info
		var contactName, contactEmail *string
		err := tx.QueryRow(ctx, `
			SELECT id, name, contact_name, contact_email
			FROM clients
			WHERE id = $1 AND organization_id = $2
		`, clientID, orgID).Scan(
			&data.Client.ID, &data.Client.Name, &contactName, &contactEmail,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrClientNotFound
			}
			return err
		}
		if contactName != nil {
			data.Client.Contact = *contactName
		}
		if contactEmail != nil {
			data.Client.Email = *contactEmail
		}

		data.Title = fmt.Sprintf("Website Care Report - %s (%s)", data.Client.Name, periodStart.Format("2006-01"))
		if locale == "fa" {
			data.Title = fmt.Sprintf("گزارش ماهانه پایش و نگهداری - %s", data.Client.Name)
		}

		// 3. Fetch Assets
		assetRows, err := tx.Query(ctx, `
			SELECT id, name, type, url, environment, is_active
			FROM assets
			WHERE client_id = $1 AND organization_id = $2
			  AND (archived_at IS NULL OR archived_at >= $3)
			ORDER BY name ASC
		`, clientID, orgID, periodStart)
		if err != nil {
			return fmt.Errorf("failed querying assets: %w", err)
		}
		defer assetRows.Close()

		assetMap := make(map[string]string)
		for assetRows.Next() {
			var a pdf.AssetReportData
			var isActive bool
			if err := assetRows.Scan(&a.ID, &a.Name, &a.Type, &a.URL, &a.Environment, &isActive); err != nil {
				return err
			}
			a.Status = "active"
			if !isActive {
				a.Status = "paused"
			}
			a.SSLStatus = "valid"
			assetMap[a.ID] = a.Name
			data.Assets = append(data.Assets, a)
		}
		if err := assetRows.Err(); err != nil {
			return err
		}

		// 4. Fetch Monitors and Calculate Scheduled Uptime & Latency
		monitorRows, err := tx.Query(ctx, `
			SELECT m.id, m.asset_id, m.created_at, m.updated_at, m.status, m.is_active, m.interval_seconds,
			       a.name AS asset_name, a.url AS asset_url, a.archived_at AS asset_archived_at
			FROM monitors m
			JOIN assets a ON a.id = m.asset_id AND a.organization_id = m.organization_id
			WHERE a.client_id = $1 AND m.organization_id = $2
			  AND (a.archived_at IS NULL OR a.archived_at >= $3)
			  AND m.created_at <= $4
		`, clientID, orgID, periodStart, periodEnd)
		if err != nil {
			return fmt.Errorf("failed querying monitors: %w", err)
		}
		defer monitorRows.Close()

		type monMeta struct {
			id              string
			assetID         string
			createdAt       time.Time
			updatedAt       time.Time
			status          string
			isActive        bool
			intervalSeconds int
			assetName       string
			assetURL        string
			assetArchivedAt *time.Time
		}
		var mons []monMeta
		for monitorRows.Next() {
			var m monMeta
			if err := monitorRows.Scan(
				&m.id, &m.assetID, &m.createdAt, &m.updatedAt, &m.status, &m.isActive, &m.intervalSeconds,
				&m.assetName, &m.assetURL, &m.assetArchivedAt,
			); err != nil {
				return err
			}
			mons = append(mons, m)
		}
		monitorRows.Close()

		var uptimeSum float64
		var uptimeCount int

		for _, m := range mons {
			// CovStart accounts for monitor creation during the reporting period
			covStart := periodStart
			if m.createdAt.After(covStart) {
				covStart = m.createdAt
			}

			// CovEnd accounts for monitor deletion / asset archiving during the period
			covEnd := periodEnd
			if m.assetArchivedAt != nil && m.assetArchivedAt.Before(covEnd) {
				covEnd = *m.assetArchivedAt
			}
			if !m.isActive && m.status != "paused" && m.updatedAt.After(covStart) && m.updatedAt.Before(covEnd) {
				covEnd = m.updatedAt
			}

			// Calculate paused spans within [covStart, covEnd]
			var pausedSpans []timeSpan
			auditRows, aErr := tx.Query(ctx, `
				SELECT action, created_at
				FROM audit_logs
				WHERE organization_id = $1 AND entity_type = 'monitor' AND entity_id = $2
				  AND action IN ('monitor.paused', 'monitor.resumed')
				ORDER BY created_at ASC
			`, orgID, m.id)
			if aErr == nil {
				type pauseEvent struct {
					action string
					at     time.Time
				}
				var events []pauseEvent
				for auditRows.Next() {
					var pe pauseEvent
					if err := auditRows.Scan(&pe.action, &pe.at); err == nil {
						events = append(events, pe)
					}
				}
				auditRows.Close()

				var activePauseStart *time.Time
				for _, ev := range events {
					if ev.action == "monitor.paused" {
						pauseTime := ev.at
						activePauseStart = &pauseTime
					} else if ev.action == "monitor.resumed" && activePauseStart != nil {
						pStart := *activePauseStart
						pEnd := ev.at
						activePauseStart = nil
						if pStart.Before(covStart) {
							pStart = covStart
						}
						if pEnd.After(covEnd) {
							pEnd = covEnd
						}
						if pEnd.After(pStart) {
							pausedSpans = append(pausedSpans, timeSpan{start: pStart, end: pEnd})
						}
					}
				}
				if activePauseStart != nil {
					pStart := *activePauseStart
					pEnd := covEnd
					if pStart.Before(covStart) {
						pStart = covStart
					}
					if pEnd.After(pStart) {
						pausedSpans = append(pausedSpans, timeSpan{start: pStart, end: pEnd})
					}
				}
			}

			// Fallback: If no audit log records exist and monitor status is paused
			if len(pausedSpans) == 0 && (m.status == "paused" || (!m.isActive && m.updatedAt.After(covStart))) {
				pStart := m.updatedAt
				if pStart.Before(covStart) {
					pStart = covStart
				}
				if covEnd.After(pStart) {
					pausedSpans = append(pausedSpans, timeSpan{start: pStart, end: covEnd})
				}
			}
			pausedSpans = unionTimeSpans(pausedSpans)

			var pausedSec int64
			for _, ps := range pausedSpans {
				pausedSec += int64(ps.end.Sub(ps.start).Seconds())
			}

			// Monitoring Eligible Period: Total unpaused active seconds
			var unpausedSec int64
			if covEnd.After(covStart) {
				rawSec := int64(covEnd.Sub(covStart).Seconds())
				// Inclusive calendar boundaries ending at 23:59:59 account for the final second of the day
				if covEnd.Second() == 59 {
					rawSec++
				}
				if pausedSec > rawSec {
					pausedSec = rawSec
				}
				unpausedSec = rawSec - pausedSec
			}
			if unpausedSec < 0 {
				unpausedSec = 0
			}

			// Expected scheduled checks based on scheduler cadence and anchor
			intervalSec := int64(m.intervalSeconds)
			if intervalSec <= 0 {
				intervalSec = 300
			}
			expectedChecks := calculateExpectedSlots(m.createdAt, covStart, covEnd, intervalSec, pausedSpans)
			if expectedChecks <= 0 && unpausedSec > 0 {
				expectedChecks = 1
			}

			// Query scheduled check metrics (strictly excluding manual checks)
			type scheduledCheckItem struct {
				id        string
				result    string
				latencyMs int64
				createdAt time.Time
			}
			var checks []scheduledCheckItem
			checkRows, err := tx.Query(ctx, `
				SELECT id, result, latency_ms, created_at
				FROM check_results
				WHERE monitor_id = $1 AND organization_id = $2
				  AND execution_type = 'scheduled'
				  AND created_at >= $3 AND created_at <= $4
				ORDER BY created_at ASC
			`, m.id, orgID, covStart, covEnd)
			if err != nil {
				return err
			}
			defer checkRows.Close()

			var checkCount, failedCount int
			var latencySum int64
			var successfulChecks int
			for checkRows.Next() {
				var it scheduledCheckItem
				if err := checkRows.Scan(&it.id, &it.result, &it.latencyMs, &it.createdAt); err != nil {
					return err
				}
				checks = append(checks, it)
				checkCount++
				if it.result == "failure" || it.result == "timeout" || it.result == "blocked" || it.result == "error" {
					failedCount++
				}
				if it.result == "success" && it.latencyMs > 0 {
					latencySum += it.latencyMs
					successfulChecks++
				}
			}
			checkRows.Close()

			var avgRespMs *int64
			if successfulChecks > 0 {
				avgVal := latencySum / int64(successfulChecks)
				avgRespMs = &avgVal
			}

			// Median Latency query
			var medianResp *int64
			if successfulChecks > 0 {
				_ = tx.QueryRow(ctx, `
					SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms)::BIGINT
					FROM check_results
					WHERE monitor_id = $1 AND organization_id = $2
					  AND execution_type = 'scheduled'
					  AND result = 'success'
					  AND created_at >= $3 AND created_at <= $4
				`, m.id, orgID, covStart, covEnd).Scan(&medianResp)
			}

			// Outage calculation: Mathematical union of incident durations and failed-check intervals
			var rawOutages []timeSpan
			var incCount int
			incRows, err := tx.Query(ctx, `
				SELECT confirmed_at, resolved_at
				FROM incidents
				WHERE monitor_id = $1 AND organization_id = $2
				  AND confirmed_at <= $3
				  AND (resolved_at IS NULL OR resolved_at >= $4)
			`, m.id, orgID, covEnd, covStart)
			if err == nil {
				for incRows.Next() {
					var cAt time.Time
					var rAt *time.Time
					if err := incRows.Scan(&cAt, &rAt); err == nil {
						incCount++
						oStart := cAt
						if oStart.Before(covStart) {
							oStart = covStart
						}
						// Open incident without resolution caps outage at covEnd
						oEnd := covEnd
						if rAt != nil && rAt.Before(covEnd) {
							oEnd = *rAt
						}
						if oEnd.After(oStart) {
							rawOutages = append(rawOutages, timeSpan{start: oStart, end: oEnd})
						}
					}
				}
				incRows.Close()
			}

			// Add failed check intervals (each failed check covers 1 interval slot)
			for _, chk := range checks {
				if chk.result == "failure" || chk.result == "timeout" || chk.result == "blocked" || chk.result == "error" {
					fStart := chk.createdAt
					fEnd := fStart.Add(time.Duration(intervalSec) * time.Second)
					if fEnd.After(covEnd) {
						fEnd = covEnd
					}
					if fEnd.After(fStart) {
						rawOutages = append(rawOutages, timeSpan{start: fStart, end: fEnd})
					}
				}
			}

			// Merge outage intervals into a mathematical union (completely eliminating double-counting)
			mergedOutages := unionTimeSpans(rawOutages)

			var totalOutageSec int64
			for _, o := range mergedOutages {
				// Subtract any overlap with paused spans
				oStart := o.start
				oEnd := o.end
				var pausedOverlap int64
				for _, ps := range pausedSpans {
					interStart := oStart
					if ps.start.After(interStart) {
						interStart = ps.start
					}
					interEnd := oEnd
					if ps.end.Before(interEnd) {
						interEnd = ps.end
					}
					if interEnd.After(interStart) {
						pausedOverlap += int64(interEnd.Sub(interStart).Seconds())
					}
				}
				spanSec := int64(oEnd.Sub(oStart).Seconds()) - pausedOverlap
				if spanSec > 0 {
					totalOutageSec += spanSec
				}
			}

			// Observed Monitoring Time: Time interval supported by actual recorded check evidence
			observedSec := int64(checkCount) * intervalSec
			if observedSec > unpausedSec {
				observedSec = unpausedSec
			}
			unknownSec := unpausedSec - observedSec
			if unknownSec < 0 {
				unknownSec = 0
			}

			// Outage cannot exceed observed time
			outageInObservedSec := totalOutageSec
			if outageInObservedSec > observedSec {
				outageInObservedSec = observedSec
			}

			// Monitoring Coverage: proportion of expected scheduled checks executed and recorded
			var covPct float64
			if expectedChecks > 0 {
				covPct = (float64(checkCount) / float64(expectedChecks)) * 100.0
				if covPct > 100.0 {
					covPct = 100.0
				}
				if covPct < 0.0 {
					covPct = 0.0
				}
			}

			// Observed Uptime: proportion of observed monitored time service was healthy
			var uptimePct *float64
			insufficientData := checkCount == 0 || observedSec == 0

			var monNotice string
			if !insufficientData && observedSec > 0 {
				healthyObservedSec := observedSec - outageInObservedSec
				if healthyObservedSec < 0 {
					healthyObservedSec = 0
				}
				pct := (float64(healthyObservedSec) / float64(observedSec)) * 100.0
				if pct > 100.0 {
					pct = 100.0
				}
				if pct < 0.0 {
					pct = 0.0
				}
				uptimePct = &pct
				uptimeSum += pct
				uptimeCount++

				observedDays := float64(observedSec) / 86400.0
				unknownDays := float64(unknownSec) / 86400.0
				if locale == "fa" {
					if unknownSec > 0 {
						monNotice = fmt.Sprintf("ضریب پایداری مشاهده‌شده: %.2f%% بر اساس %.1f روز شواهد پایش واقعی (%.1f روز فاقد داده / نامشخص از مخرج پایداری کسر شد).", pct, observedDays, unknownDays)
					} else {
						monNotice = fmt.Sprintf("ضریب پایداری مشاهده‌شده: %.2f%% در کل دوره پایش.", pct)
					}
				} else {
					if unknownSec > 0 {
						monNotice = fmt.Sprintf("Observed Uptime: %.2f%% across %.1f observed days of monitoring evidence (%.1f days missing / unmonitored time excluded from uptime calculation).", pct, observedDays, unknownDays)
					} else {
						monNotice = fmt.Sprintf("Observed Uptime: %.2f%% across full eligible monitoring period.", pct)
					}
				}
			} else {
				if locale == "fa" {
					monNotice = "داده‌های پایش در این دوره کافی نیست. هیچ تست زمان‌بندی‌شده‌ای ثبت نشده است."
				} else {
					monNotice = "Insufficient monitoring data. No scheduled checks were executed or recorded for this period."
				}
			}

			data.Monitors = append(data.Monitors, pdf.MonitorReportData{
				ID:                   m.id,
				Name:                 "HTTP Monitor (" + m.assetName + ")",
				AssetName:            m.assetName,
				URL:                  m.assetURL,
				UptimePercentage:     uptimePct,
				CoveragePercentage:   covPct,
				InsufficientData:     insufficientData,
				AvgResponseTimeMs:    avgRespMs,
				MedianResponseTimeMs: medianResp,
				TotalChecks:          checkCount,
				FailedChecks:         failedCount,
				IncidentCount:        incCount,
				EligibleDurationSec:  unpausedSec,
				ObservedDurationSec:  observedSec,
				UnknownDurationSec:   unknownSec,
				LimitationNotice:     monNotice,
			})
		}

		// 5. Incidents during the period
		incRows, err := tx.Query(ctx, `
			SELECT i.id, 
			       CASE WHEN i.monitor_id IS NOT NULL THEN 'monitor' ELSE 'heartbeat' END AS target_type,
			       i.confirmed_at, i.resolved_at,
			       i.failure_category, i.duration_seconds, i.summary,
			       COALESCE(m.asset_id, hm.asset_id) AS asset_id,
			       COALESCE(a.name, hm_a.name, 'Endpoint') AS asset_name,
			       COALESCE(hm.name, a.name, 'Monitor') AS target_name
			FROM incidents i
			LEFT JOIN monitors m ON m.id = i.monitor_id AND m.organization_id = i.organization_id
			LEFT JOIN assets a ON a.id = m.asset_id AND a.organization_id = i.organization_id
			LEFT JOIN heartbeat_monitors hm ON hm.id = i.heartbeat_monitor_id AND hm.organization_id = i.organization_id
			LEFT JOIN assets hm_a ON hm_a.id = hm.asset_id AND hm_a.organization_id = i.organization_id
			WHERE i.organization_id = $1 
			  AND COALESCE(a.client_id, hm.client_id) = $2
			  AND i.confirmed_at <= $3
			  AND (i.resolved_at IS NULL OR i.resolved_at >= $4)
			ORDER BY i.confirmed_at DESC
		`, orgID, clientID, periodEnd, periodStart)
		if err != nil {
			return fmt.Errorf("failed querying incidents: %w", err)
		}
		defer incRows.Close()

		var totalDowntimeSec int64
		for incRows.Next() {
			var i pdf.IncidentReportData
			var resolvedAt *time.Time
			var confirmedAt time.Time
			var durSec *int
			var assetID *string
			var assetName, sumText string

			if err := incRows.Scan(
				&i.ID, &i.TargetType, &confirmedAt, &resolvedAt,
				&i.FailureCategory, &durSec, &sumText,
				&assetID, &assetName, &i.TargetName,
			); err != nil {
				return err
			}
			i.ConfirmedAt = confirmedAt.Format("2006-01-02 15:04")
			if resolvedAt != nil {
				i.ResolvedAt = resolvedAt.Format("2006-01-02 15:04")
			}
			if durSec != nil {
				i.DurationMinutes = int64(*durSec) / 60
				totalDowntimeSec += int64(*durSec)
			}
			i.Summary = sumText
			data.Incidents = append(data.Incidents, i)
		}
		incRows.Close()

		// 6. Heartbeats (Cron / Backup Workers)
		hbRows, err := tx.Query(ctx, `
			SELECT hm.id, hm.name, hm.status, hm.expected_interval_seconds, hm.last_received_at,
			       a.name AS asset_name
			FROM heartbeat_monitors hm
			LEFT JOIN assets a ON a.id = hm.asset_id AND a.organization_id = hm.organization_id
			WHERE hm.client_id = $1 AND hm.organization_id = $2 AND hm.archived_at IS NULL
			ORDER BY hm.name ASC
		`, clientID, orgID)
		if err != nil {
			return fmt.Errorf("failed querying heartbeats: %w", err)
		}
		defer hbRows.Close()

		for hbRows.Next() {
			var hb pdf.HeartbeatReportData
			var intervalSec int
			var lastRec *time.Time
			var assetName *string

			if err := hbRows.Scan(&hb.ID, &hb.Name, &hb.Status, &intervalSec, &lastRec, &assetName); err != nil {
				return err
			}
			if assetName != nil {
				hb.AssetName = *assetName
			}
			if lastRec != nil {
				hb.LastReceivedAt = lastRec.Format("2006-01-02 15:04")
			}
			hb.ExpectedInterval = formatInterval(intervalSec)
			hb.LimitationNotice = "Heartbeat received indicates the script ran, but does not verify backup restoration validity or completeness."
			data.Heartbeats = append(data.Heartbeats, hb)
		}
		hbRows.Close()

		// 7. Maintenance Activities
		actRows, err := tx.Query(ctx, `
			SELECT ma.id, ma.title, ma.category, ma.performed_at, ma.duration_minutes, ma.description,
			       COALESCE(a.name, 'General Client System') AS asset_name
			FROM maintenance_activities ma
			LEFT JOIN assets a ON a.id = ma.asset_id AND a.organization_id = ma.organization_id
			WHERE ma.client_id = $1 AND ma.organization_id = $2
			  AND ma.performed_at >= $3 AND ma.performed_at <= $4
			ORDER BY ma.performed_at DESC
		`, clientID, orgID, periodStart, periodEnd)
		if err != nil {
			return fmt.Errorf("failed querying maintenance activities: %w", err)
		}
		defer actRows.Close()

		var totalMaintenanceMin int
		for actRows.Next() {
			var act pdf.MaintenanceReportData
			var perfAt time.Time
			if err := actRows.Scan(&act.ID, &act.Title, &act.Category, &perfAt, &act.DurationMinutes, &act.Description, &act.AssetName); err != nil {
				return err
			}
			act.PerformedAt = perfAt.Format("2006-01-02 15:04")
			if act.DurationMinutes != nil {
				totalMaintenanceMin += *act.DurationMinutes
			}
			data.Maintenance = append(data.Maintenance, act)
		}
		actRows.Close()

		// 8. Deterministic Summary Calculations
		data.Summary.TotalMonitors = len(data.Monitors)
		data.Summary.TotalIncidents = len(data.Incidents)
		data.Summary.TotalDowntimeSec = totalDowntimeSec
		data.Summary.TotalActivities = len(data.Maintenance)
		data.Summary.TotalMaintenanceMin = totalMaintenanceMin

		if uptimeCount > 0 {
			avg := uptimeSum / float64(uptimeCount)
			data.Summary.AverageUptime = &avg
		}

		// Executive Statement
		if len(data.Incidents) == 0 && data.Summary.AverageUptime != nil && *data.Summary.AverageUptime >= 99.9 {
			data.Summary.OverallStatus = "optimal"
			if locale == "fa" {
				data.Summary.ExecutiveStatement = fmt.Sprintf("در این دوره تمامی سرویس‌های %s با ضریب پایداری %.2f%% فعال بودند و هیچ‌گونه قطعی شناسایی نشد.", data.Client.Name, *data.Summary.AverageUptime)
			} else {
				data.Summary.ExecutiveStatement = fmt.Sprintf("All monitored endpoints for %s maintained exceptional stability at %.2f%% uptime with zero recorded incidents.", data.Client.Name, *data.Summary.AverageUptime)
			}
		} else if len(data.Incidents) > 0 {
			data.Summary.OverallStatus = "remediated"
			if locale == "fa" {
				data.Summary.ExecutiveStatement = fmt.Sprintf("در این دوره تعداد %d رخداد قطعی ثبت شد که مجموعاً %d دقیقه به طول انجامید و با موفقیت توسط تیم فنی برطرف گردید.", len(data.Incidents), totalDowntimeSec/60)
			} else {
				data.Summary.ExecutiveStatement = fmt.Sprintf("%d incident(s) occurred during this period totaling %d minutes of downtime, which were investigated and resolved.", len(data.Incidents), totalDowntimeSec/60)
			}
		} else {
			data.Summary.OverallStatus = "normal"
			if locale == "fa" {
				data.Summary.ExecutiveStatement = "سرویس‌ها به صورت عادی و پایدار در طول دوره گزارش فعالیت کردند."
			} else {
				data.Summary.ExecutiveStatement = "Operational endpoints operated normally throughout the reported billing period."
			}
		}

		// 9. Deterministic Recommendations
		for _, m := range data.Monitors {
			if m.UptimePercentage != nil && *m.UptimePercentage < 99.0 {
				data.Recommendations = append(data.Recommendations, pdf.RecommendationData{
					Severity:    "warning",
					Title:       ifElse(locale == "fa", "بررسی نوسان پایداری سرور", "Investigate Availability Fluctuations"),
					Description: ifElse(locale == "fa", fmt.Sprintf("پایداری %s زیر ۹۹٪ بوده است. پیشنهاد می‌شود لاگ‌های سرور بررسی شوند.", m.Name), fmt.Sprintf("Availability on %s was %.2f%%. Server resource contention should be examined.", m.Name, *m.UptimePercentage)),
				})
			}
			if m.AvgResponseTimeMs != nil && *m.AvgResponseTimeMs > 1000 {
				data.Recommendations = append(data.Recommendations, pdf.RecommendationData{
					Severity:    "info",
					Title:       ifElse(locale == "fa", "بهینه‌سازی زمان پاسخ سرور", "Optimize Latency and Response Time"),
					Description: ifElse(locale == "fa", fmt.Sprintf("میانگین پاسخ %s معادل %d میلی‌ثانیه است؛ فعال‌سازی Caching و CDN توصیه می‌شود.", m.Name, *m.AvgResponseTimeMs), fmt.Sprintf("Average response time for %s was %d ms. CDN caching is recommended.", m.Name, *m.AvgResponseTimeMs)),
				})
			}
		}

		if len(data.Heartbeats) == 0 {
			data.Recommendations = append(data.Recommendations, pdf.RecommendationData{
				Severity:    "info",
				Title:       ifElse(locale == "fa", "فعال‌سازی مانیتورینگ بکاپ‌های خودکار", "Configure Automated Backup Monitoring"),
				Description: ifElse(locale == "fa", "پیشنهاد می‌شود برای جاب‌های دوره‌ای دیتابیس یا بکاپ، Heartbeat فعال شود.", "Configure heartbeat monitoring for automated backups to track execution health."),
			})
		}

		// 10. Limitation notice & methodology disclosure
		if locale == "fa" {
			data.LimitationNotice = "شفاف‌سازی متدولوژی: درصد پایداری بر اساس زمان مجاز پایش و کسر زمان قطعی ناشی از رخدادهای تأییدشده محاسبه شده است. پوشش پایش نسبت تست‌های زمان‌بندی‌شده اجرا شده به تست‌های مورد انتظار است. دوره‌های تعلیق و تست‌های دستی به طور کامل از مخرج محاسبات پایداری حذف شده‌اند. سیگنال سلامت به تنهایی صحت بازگردانی بکاپ را اثبات نمی‌کند."
		} else {
			data.LimitationNotice = "Methodology Disclosure: Monitoring Eligible Period excludes paused intervals and periods prior to creation or after deactivation. Monitoring Coverage is (Executed Scheduled Checks / Expected Scheduled Checks) * 100; missing scheduled checks reduce coverage while executed failed checks count toward coverage. Uptime is ((Eligible Monitored Seconds - Outage Seconds) / Eligible Monitored Seconds) * 100; failed checks and confirmed incident outages reduce uptime. Paused periods and manual tests are strictly excluded from uptime denominators. Heartbeat receipt confirms task execution but does not verify backup restoration validity or completeness."
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &data, nil
}

func formatInterval(seconds int) string {
	if seconds >= 86400 {
		return fmt.Sprintf("%d days", seconds/86400)
	}
	if seconds >= 3600 {
		return fmt.Sprintf("%d hours", seconds/3600)
	}
	if seconds >= 60 {
		return fmt.Sprintf("%d minutes", seconds/60)
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func ifElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

type timeSpan struct {
	start time.Time
	end   time.Time
}

func unionTimeSpans(spans []timeSpan) []timeSpan {
	if len(spans) == 0 {
		return nil
	}
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].start.Before(spans[j].start)
	})
	merged := []timeSpan{spans[0]}
	for _, current := range spans[1:] {
		last := &merged[len(merged)-1]
		if !current.start.After(last.end) {
			if current.end.After(last.end) {
				last.end = current.end
			}
		} else {
			merged = append(merged, current)
		}
	}
	return merged
}

func calculateExpectedSlots(anchor, covStart, covEnd time.Time, intervalSec int64, pausedSpans []timeSpan) int64 {
	if !covEnd.After(covStart) || intervalSec <= 0 {
		return 0
	}

	var firstSlot time.Time
	if !anchor.Before(covStart) {
		firstSlot = anchor
	} else {
		offset := covStart.Sub(anchor)
		intervalDur := time.Duration(intervalSec) * time.Second
		rem := offset % intervalDur
		if rem == 0 {
			firstSlot = covStart
		} else {
			firstSlot = covStart.Add(intervalDur - rem)
		}
	}

	var count int64
	for slot := firstSlot; !slot.After(covEnd); slot = slot.Add(time.Duration(intervalSec) * time.Second) {
		inPaused := false
		for _, p := range pausedSpans {
			if (slot.Equal(p.start) || slot.After(p.start)) && slot.Before(p.end) {
				inPaused = true
				break
			}
		}
		if !inPaused {
			count++
		}
	}
	return count
}
