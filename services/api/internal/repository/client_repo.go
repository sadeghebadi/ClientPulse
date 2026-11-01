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
	ErrClientNotFound = errors.New("client not found")
)

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{pool: pool}
}

func (r *ClientRepository) CreateClient(ctx context.Context, orgID string, name string, contactName, contactEmail, website, notes *string) (*Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("client name is required")
	}

	var client Client
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO clients (organization_id, name, contact_name, contact_email, website, notes, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW(), NOW())
			RETURNING id, organization_id, name, contact_name, contact_email, website, notes, status, created_at, updated_at, archived_at
		`
		return tx.QueryRow(ctx, query, orgID, name, contactName, contactEmail, website, notes).Scan(
			&client.ID, &client.OrganizationID, &client.Name,
			&client.ContactName, &client.ContactEmail, &client.Website, &client.Notes,
			&client.Status, &client.CreatedAt, &client.UpdatedAt, &client.ArchivedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &client, nil
}

func (r *ClientRepository) GetClientByID(ctx context.Context, orgID, clientID string) (*Client, error) {
	var client Client
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT c.id, c.organization_id, c.name, c.contact_name, c.contact_email, c.website, c.notes,
			       c.status, c.created_at, c.updated_at, c.archived_at,
			       COUNT(DISTINCT a.id) AS asset_count,
			       COUNT(DISTINCT m.id) AS monitor_count
			FROM clients c
			LEFT JOIN assets a ON c.id = a.client_id
			LEFT JOIN monitors m ON a.id = m.asset_id
			WHERE c.id = $1 AND c.organization_id = $2
			GROUP BY c.id
		`
		err := tx.QueryRow(ctx, query, clientID, orgID).Scan(
			&client.ID, &client.OrganizationID, &client.Name,
			&client.ContactName, &client.ContactEmail, &client.Website, &client.Notes,
			&client.Status, &client.CreatedAt, &client.UpdatedAt, &client.ArchivedAt,
			&client.AssetCount, &client.MonitorCount,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClientNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *ClientRepository) ListClients(ctx context.Context, orgID string, search, status string, limit, offset int) ([]*Client, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var clients []*Client
	var total int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		baseFilter := "WHERE c.organization_id = $1"
		args := []interface{}{orgID}
		argIdx := 2

		if status != "" && status != "all" {
			baseFilter += fmt.Sprintf(" AND c.status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}

		if strings.TrimSpace(search) != "" {
			baseFilter += fmt.Sprintf(" AND (c.name ILIKE $%d OR COALESCE(c.contact_name, '') ILIKE $%d OR COALESCE(c.contact_email, '') ILIKE $%d)", argIdx, argIdx, argIdx)
			args = append(args, "%"+strings.TrimSpace(search)+"%")
			argIdx++
		}

		// Count total
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM clients c %s", baseFilter)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		// Query page
		query := fmt.Sprintf(`
			SELECT c.id, c.organization_id, c.name, c.contact_name, c.contact_email, c.website, c.notes,
			       c.status, c.created_at, c.updated_at, c.archived_at,
			       COUNT(DISTINCT a.id) AS asset_count,
			       COUNT(DISTINCT m.id) AS monitor_count
			FROM clients c
			LEFT JOIN assets a ON c.id = a.client_id
			LEFT JOIN monitors m ON a.id = m.asset_id
			%s
			GROUP BY c.id
			ORDER BY c.created_at DESC
			LIMIT $%d OFFSET $%d
		`, baseFilter, argIdx, argIdx+1)

		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var c Client
			if err := rows.Scan(
				&c.ID, &c.OrganizationID, &c.Name,
				&c.ContactName, &c.ContactEmail, &c.Website, &c.Notes,
				&c.Status, &c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt,
				&c.AssetCount, &c.MonitorCount,
			); err != nil {
				return err
			}
			clients = append(clients, &c)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return clients, total, nil
}

func (r *ClientRepository) UpdateClient(ctx context.Context, orgID, clientID string, name string, contactName, contactEmail, website, notes *string) (*Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("client name is required")
	}

	var client Client
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			UPDATE clients
			SET name = $1,
			    contact_name = $2,
			    contact_email = $3,
			    website = $4,
			    notes = $5,
			    updated_at = NOW()
			WHERE id = $6 AND organization_id = $7
			RETURNING id, organization_id, name, contact_name, contact_email, website, notes, status, created_at, updated_at, archived_at
		`
		err := tx.QueryRow(ctx, query, name, contactName, contactEmail, website, notes, clientID, orgID).Scan(
			&client.ID, &client.OrganizationID, &client.Name,
			&client.ContactName, &client.ContactEmail, &client.Website, &client.Notes,
			&client.Status, &client.CreatedAt, &client.UpdatedAt, &client.ArchivedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClientNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *ClientRepository) ArchiveClient(ctx context.Context, orgID, clientID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		res, err := tx.Exec(ctx, `
			UPDATE clients
			SET status = 'archived', archived_at = $1, updated_at = $1
			WHERE id = $2 AND organization_id = $3
		`, now, clientID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrClientNotFound
		}

		// Also pause monitors and deactivate assets belonging to this client
		_, _ = tx.Exec(ctx, `
			UPDATE assets
			SET is_active = false, archived_at = $1, updated_at = $1
			WHERE client_id = $2 AND organization_id = $3
		`, now, clientID, orgID)

		_, _ = tx.Exec(ctx, `
			UPDATE monitors
			SET status = 'paused', is_active = false, updated_at = $1
			WHERE asset_id IN (SELECT id FROM assets WHERE client_id = $2) AND organization_id = $3
		`, now, clientID, orgID)

		return nil
	})
}

func (r *ClientRepository) RestoreClient(ctx context.Context, orgID, clientID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		res, err := tx.Exec(ctx, `
			UPDATE clients
			SET status = 'active', archived_at = NULL, updated_at = $1
			WHERE id = $2 AND organization_id = $3
		`, now, clientID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrClientNotFound
		}
		return nil
	})
}

func (r *ClientRepository) DeleteClient(ctx context.Context, orgID, clientID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			DELETE FROM clients
			WHERE id = $1 AND organization_id = $2
		`, clientID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrClientNotFound
		}
		return nil
	})
}

