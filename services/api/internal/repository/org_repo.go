package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrgNotFound          = errors.New("organization not found")
	ErrOrgSlugExists        = errors.New("organization with this slug already exists")
	ErrMembershipNotFound   = errors.New("organization membership not found")
	ErrMembershipExists     = errors.New("user is already a member of this organization")
	ErrCannotRemoveLastOwner = errors.New("cannot remove or demote the last remaining owner")
)

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// CreateOrganization creates a new organization record.
func (r *OrganizationRepository) CreateOrganization(ctx context.Context, name, slug, locale, timezone string) (*Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.ToLower(strings.TrimSpace(slug))
	if locale == "" {
		locale = "en"
	}
	if timezone == "" {
		timezone = "UTC"
	}

	query := `
		INSERT INTO organizations (name, slug, locale, timezone)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, locale, timezone, status, created_at, updated_at, deleted_at
	`

	org := &Organization{}
	err := r.pool.QueryRow(ctx, query, name, slug, locale, timezone).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Locale,
		&org.Timezone,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.DeletedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrOrgSlugExists
		}
		return nil, fmt.Errorf("failed to insert organization: %w", err)
	}

	return org, nil
}

// GetOrganizationByID retrieves an organization by its ID.
func (r *OrganizationRepository) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	query := `
		SELECT id, name, slug, locale, timezone, status, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`

	org := &Organization{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Locale,
		&org.Timezone,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("failed to query organization: %w", err)
	}

	return org, nil
}

// GetOrganizationBySlug retrieves an organization by its slug.
func (r *OrganizationRepository) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT id, name, slug, locale, timezone, status, created_at, updated_at, deleted_at
		FROM organizations
		WHERE LOWER(slug) = LOWER($1) AND deleted_at IS NULL
	`

	org := &Organization{}
	err := r.pool.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(slug))).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Locale,
		&org.Timezone,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("failed to query organization by slug: %w", err)
	}

	return org, nil
}

// UpdateOrganization updates organization details.
func (r *OrganizationRepository) UpdateOrganization(ctx context.Context, id, name, locale, timezone string) (*Organization, error) {
	query := `
		UPDATE organizations
		SET name = $1, locale = $2, timezone = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING id, name, slug, locale, timezone, status, created_at, updated_at, deleted_at
	`

	org := &Organization{}
	err := r.pool.QueryRow(ctx, query, name, locale, timezone, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Locale,
		&org.Timezone,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// AddMember adds a user to an organization with a specified role.
func (r *OrganizationRepository) AddMember(ctx context.Context, orgID, userID, role string) (*Membership, error) {
	query := `
		INSERT INTO organization_memberships (organization_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, organization_id, user_id, role, created_at, updated_at
	`

	m := &Membership{}
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, orgID, userID, role).Scan(
			&m.ID,
			&m.OrganizationID,
			&m.UserID,
			&m.Role,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrMembershipExists
		}
		return nil, fmt.Errorf("failed to add membership: %w", err)
	}

	return m, nil
}

// GetMembership retrieves a specific user's membership in an organization.
func (r *OrganizationRepository) GetMembership(ctx context.Context, orgID, userID string) (*Membership, error) {
	query := `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, m.updated_at,
		       o.name as org_name, o.slug as org_slug,
		       u.email as user_email, u.display_name as user_display_name
		FROM organization_memberships m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND m.user_id = $2 AND o.deleted_at IS NULL
	`

	m := &Membership{}
	err := WithUserTx(ctx, r.pool, userID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, orgID, userID).Scan(
			&m.ID,
			&m.OrganizationID,
			&m.UserID,
			&m.Role,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.OrgName,
			&m.OrgSlug,
			&m.UserEmail,
			&m.UserDisplayName,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMembershipNotFound
		}
		return nil, fmt.Errorf("failed to query membership: %w", err)
	}

	return m, nil
}

// ListUserMemberships lists all organizations the user is a member of.
func (r *OrganizationRepository) ListUserMemberships(ctx context.Context, userID string) ([]Membership, error) {
	query := `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, m.updated_at,
		       o.name as org_name, o.slug as org_slug,
		       u.email as user_email, u.display_name as user_display_name
		FROM organization_memberships m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1 AND o.deleted_at IS NULL
		ORDER BY m.created_at ASC
	`

	var list []Membership
	err := WithUserTx(ctx, r.pool, userID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, userID)
		if err != nil {
			return fmt.Errorf("failed to query user memberships: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var m Membership
			err := rows.Scan(
				&m.ID,
				&m.OrganizationID,
				&m.UserID,
				&m.Role,
				&m.CreatedAt,
				&m.UpdatedAt,
				&m.OrgName,
				&m.OrgSlug,
				&m.UserEmail,
				&m.UserDisplayName,
			)
			if err != nil {
				return fmt.Errorf("failed to scan membership: %w", err)
			}
			list = append(list, m)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListOrganizationMembers lists all members of an organization.
func (r *OrganizationRepository) ListOrganizationMembers(ctx context.Context, orgID string) ([]Membership, error) {
	query := `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, m.updated_at,
		       o.name as org_name, o.slug as org_slug,
		       u.email as user_email, u.display_name as user_display_name
		FROM organization_memberships m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY m.created_at ASC
	`

	var list []Membership
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, orgID)
		if err != nil {
			return fmt.Errorf("failed to list organization members: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var m Membership
			err := rows.Scan(
				&m.ID,
				&m.OrganizationID,
				&m.UserID,
				&m.Role,
				&m.CreatedAt,
				&m.UpdatedAt,
				&m.OrgName,
				&m.OrgSlug,
				&m.UserEmail,
				&m.UserDisplayName,
			)
			if err != nil {
				return fmt.Errorf("failed to scan membership: %w", err)
			}
			list = append(list, m)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateMemberRole modifies a member's role in the organization.
func (r *OrganizationRepository) UpdateMemberRole(ctx context.Context, orgID, userID, newRole string) error {
	query := `
		UPDATE organization_memberships
		SET role = $1, updated_at = NOW()
		WHERE organization_id = $2 AND user_id = $3
	`
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		cmdTag, err := tx.Exec(ctx, query, newRole, orgID, userID)
		if err != nil {
			return fmt.Errorf("failed to update member role: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return ErrMembershipNotFound
		}
		return nil
	})
}

// RemoveMember deletes a membership from an organization.
func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	query := `
		DELETE FROM organization_memberships
		WHERE organization_id = $1 AND user_id = $2
	`
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		cmdTag, err := tx.Exec(ctx, query, orgID, userID)
		if err != nil {
			return fmt.Errorf("failed to remove member: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return ErrMembershipNotFound
		}
		return nil
	})
}

// CountOwners returns the number of owners in an organization.
func (r *OrganizationRepository) CountOwners(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM organization_memberships WHERE organization_id = $1 AND role = 'owner'`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	return count, err
}

