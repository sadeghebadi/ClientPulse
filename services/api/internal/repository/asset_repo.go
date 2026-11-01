package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAssetNotFound       = errors.New("asset not found")
	ErrClientNotInOrg      = errors.New("client does not belong to the active organization")
	ErrInvalidEnvironment  = errors.New("invalid environment")
	ErrDevelopmentInHosted = errors.New("private development URLs cannot be monitored in hosted product")
)

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

func (r *AssetRepository) CreateAsset(ctx context.Context, orgID, clientID, name, urlStr, assetType, environment string, description *string) (*Asset, error) {
	name = strings.TrimSpace(name)
	urlStr = strings.TrimSpace(urlStr)
	if name == "" {
		return nil, errors.New("asset name is required")
	}
	if urlStr == "" {
		return nil, errors.New("asset URL is required")
	}
	if assetType != "website" && assetType != "api" {
		assetType = "website"
	}
	if environment != "production" && environment != "staging" && environment != "development" {
		environment = "production"
	}

	var asset Asset
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Strict parent-child tenant ownership check: verify client exists in this org
		var clientExists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM clients WHERE id = $1 AND organization_id = $2)", clientID, orgID).Scan(&clientExists)
		if err != nil {
			return err
		}
		if !clientExists {
			return ErrClientNotInOrg
		}

		query := `
			INSERT INTO assets (organization_id, client_id, name, url, type, environment, description, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())
			RETURNING id, organization_id, client_id, name, url, type, environment, description, is_active, created_at, updated_at, archived_at
		`
		return tx.QueryRow(ctx, query, orgID, clientID, name, urlStr, assetType, environment, description).Scan(
			&asset.ID, &asset.OrganizationID, &asset.ClientID,
			&asset.Name, &asset.URL, &asset.Type, &asset.Environment,
			&asset.Description, &asset.IsActive, &asset.CreatedAt, &asset.UpdatedAt, &asset.ArchivedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	return &asset, nil
}

func (r *AssetRepository) GetAssetByID(ctx context.Context, orgID, assetID string) (*Asset, error) {
	var asset Asset
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT a.id, a.organization_id, a.client_id, a.name, a.url, a.type, a.environment,
			       a.description, a.is_active, a.created_at, a.updated_at, a.archived_at,
			       c.name AS client_name,
			       m.id AS monitor_id,
			       m.status AS monitor_status,
			       m.last_checked_at
			FROM assets a
			JOIN clients c ON a.client_id = c.id
			LEFT JOIN monitors m ON a.id = m.asset_id
			WHERE a.id = $1 AND a.organization_id = $2
		`
		err := tx.QueryRow(ctx, query, assetID, orgID).Scan(
			&asset.ID, &asset.OrganizationID, &asset.ClientID,
			&asset.Name, &asset.URL, &asset.Type, &asset.Environment,
			&asset.Description, &asset.IsActive, &asset.CreatedAt, &asset.UpdatedAt, &asset.ArchivedAt,
			&asset.ClientName, &asset.MonitorID, &asset.MonitorStatus, &asset.LastCheckedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssetNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) ListAssets(ctx context.Context, orgID string, clientID, assetType, environment string, isActive *bool, limit, offset int) ([]*Asset, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var assets []*Asset
	var total int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		baseFilter := "WHERE a.organization_id = $1"
		args := []interface{}{orgID}
		argIdx := 2

		if clientID != "" {
			baseFilter += fmt.Sprintf(" AND a.client_id = $%d", argIdx)
			args = append(args, clientID)
			argIdx++
		}
		if assetType != "" {
			baseFilter += fmt.Sprintf(" AND a.type = $%d", argIdx)
			args = append(args, assetType)
			argIdx++
		}
		if environment != "" {
			baseFilter += fmt.Sprintf(" AND a.environment = $%d", argIdx)
			args = append(args, environment)
			argIdx++
		}
		if isActive != nil {
			baseFilter += fmt.Sprintf(" AND a.is_active = $%d", argIdx)
			args = append(args, *isActive)
			argIdx++
		}

		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM assets a %s", baseFilter)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		query := fmt.Sprintf(`
			SELECT a.id, a.organization_id, a.client_id, a.name, a.url, a.type, a.environment,
			       a.description, a.is_active, a.created_at, a.updated_at, a.archived_at,
			       c.name AS client_name,
			       m.id AS monitor_id,
			       m.status AS monitor_status,
			       m.last_checked_at
			FROM assets a
			JOIN clients c ON a.client_id = c.id
			LEFT JOIN monitors m ON a.id = m.asset_id
			%s
			ORDER BY a.created_at DESC
			LIMIT $%d OFFSET $%d
		`, baseFilter, argIdx, argIdx+1)

		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var a Asset
			if err := rows.Scan(
				&a.ID, &a.OrganizationID, &a.ClientID,
				&a.Name, &a.URL, &a.Type, &a.Environment,
				&a.Description, &a.IsActive, &a.CreatedAt, &a.UpdatedAt, &a.ArchivedAt,
				&a.ClientName, &a.MonitorID, &a.MonitorStatus, &a.LastCheckedAt,
			); err != nil {
				return err
			}
			assets = append(assets, &a)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

func (r *AssetRepository) UpdateAsset(ctx context.Context, orgID, assetID, name, urlStr, assetType, environment string, description *string) (*Asset, error) {
	name = strings.TrimSpace(name)
	urlStr = strings.TrimSpace(urlStr)
	if name == "" {
		return nil, errors.New("asset name is required")
	}
	if urlStr == "" {
		return nil, errors.New("asset URL is required")
	}

	var asset Asset
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			UPDATE assets
			SET name = $1,
			    url = $2,
			    type = $3,
			    environment = $4,
			    description = $5,
			    updated_at = NOW()
			WHERE id = $6 AND organization_id = $7
			RETURNING id, organization_id, client_id, name, url, type, environment, description, is_active, created_at, updated_at, archived_at
		`
		err := tx.QueryRow(ctx, query, name, urlStr, assetType, environment, description, assetID, orgID).Scan(
			&asset.ID, &asset.OrganizationID, &asset.ClientID,
			&asset.Name, &asset.URL, &asset.Type, &asset.Environment,
			&asset.Description, &asset.IsActive, &asset.CreatedAt, &asset.UpdatedAt, &asset.ArchivedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssetNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) ArchiveAsset(ctx context.Context, orgID, assetID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		res, err := tx.Exec(ctx, `
			UPDATE assets
			SET is_active = false, archived_at = $1, updated_at = $1
			WHERE id = $2 AND organization_id = $3
		`, now, assetID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrAssetNotFound
		}

		// Also pause and stop scheduled monitoring jobs
		_, _ = tx.Exec(ctx, `
			UPDATE monitors
			SET status = 'paused', is_active = false, updated_at = $1
			WHERE asset_id = $2 AND organization_id = $3
		`, now, assetID, orgID)

		return nil
	})
}

func (r *AssetRepository) RestoreAsset(ctx context.Context, orgID, assetID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		res, err := tx.Exec(ctx, `
			UPDATE assets
			SET is_active = true, archived_at = NULL, updated_at = $1
			WHERE id = $2 AND organization_id = $3
		`, now, assetID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrAssetNotFound
		}

		// Resume associated monitors
		_, _ = tx.Exec(ctx, `
			UPDATE monitors
			SET is_active = true, status = 'pending', next_check_at = NOW(), updated_at = $1
			WHERE asset_id = $2 AND organization_id = $3
		`, now, assetID, orgID)

		return nil
	})
}

func (r *AssetRepository) CheckDuplicateURL(ctx context.Context, orgID, urlStr, excludeAssetID string) (bool, error) {
	var exists bool
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT EXISTS(
				SELECT 1 FROM assets
				WHERE organization_id = $1
				  AND LOWER(url) = LOWER($2)
				  AND ($3 = '' OR id != $3::uuid)
				  AND is_active = true
			)
		`
		return tx.QueryRow(ctx, query, orgID, strings.TrimSpace(urlStr), excludeAssetID).Scan(&exists)
	})
	return exists, err
}
