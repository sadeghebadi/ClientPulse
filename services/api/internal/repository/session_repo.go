package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("session not found or expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrSessionExpired  = errors.New("session has expired")
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// CreateSession inserts a new hashed session record.
func (r *SessionRepository) CreateSession(ctx context.Context, userID, tokenHash, ip, userAgent string, expiresAt time.Time, activeOrgID *string) (*Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, ip_address, user_agent, expires_at, active_organization_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, token_hash, active_organization_id, ip_address, user_agent, expires_at, revoked_at, created_at, last_seen_at
	`

	s := &Session{}
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, ip, userAgent, expiresAt, activeOrgID).Scan(
		&s.ID,
		&s.UserID,
		&s.TokenHash,
		&s.ActiveOrganizationID,
		&s.IPAddress,
		&s.UserAgent,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.CreatedAt,
		&s.LastSeenAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert session: %w", err)
	}

	return s, nil
}

// GetSessionByTokenHash retrieves a session by its token hash.
func (r *SessionRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	query := `
		SELECT id, user_id, token_hash, active_organization_id, ip_address, user_agent, expires_at, revoked_at, created_at, last_seen_at
		FROM sessions
		WHERE token_hash = $1
	`

	s := &Session{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.TokenHash,
		&s.ActiveOrganizationID,
		&s.IPAddress,
		&s.UserAgent,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.CreatedAt,
		&s.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	if s.RevokedAt != nil {
		return nil, ErrSessionRevoked
	}

	if time.Now().After(s.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return s, nil
}

// TouchSession updates the last_seen_at timestamp for an active session.
func (r *SessionRepository) TouchSession(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET last_seen_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, sessionID)
	return err
}

// SetActiveOrganization updates the active organization for a session.
func (r *SessionRepository) SetActiveOrganization(ctx context.Context, sessionID, orgID string) error {
	query := `UPDATE sessions SET active_organization_id = $1, last_seen_at = NOW() WHERE id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, orgID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update active organization for session: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// RevokeSession marks a session as revoked.
func (r *SessionRepository) RevokeSession(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, sessionID)
	return err
}

// RevokeAllUserSessions invalidates all active sessions for a user (e.g. on password change).
func (r *SessionRepository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// ListUserSessions returns recent sessions for a user.
func (r *SessionRepository) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	query := `
		SELECT id, user_id, token_hash, active_organization_id, ip_address, user_agent, expires_at, revoked_at, created_at, last_seen_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY last_seen_at DESC
		LIMIT 20
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.TokenHash,
			&s.ActiveOrganizationID,
			&s.IPAddress,
			&s.UserAgent,
			&s.ExpiresAt,
			&s.RevokedAt,
			&s.CreatedAt,
			&s.LastSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

