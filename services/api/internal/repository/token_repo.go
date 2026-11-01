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
	ErrTokenNotFound = errors.New("token not found, expired, or already used")
)

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

// CreatePasswordResetToken stores a hashed password reset token with 30-minute expiration.
func (r *TokenRepository) CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*TokenRecord, error) {
	query := `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, expiresAt).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert password reset token: %w", err)
	}
	return rec, nil
}

// GetValidPasswordResetToken retrieves an unused, unexpired password reset token.
func (r *TokenRepository) GetValidPasswordResetToken(ctx context.Context, tokenHash string) (*TokenRecord, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to query password reset token: %w", err)
	}
	return rec, nil
}

// ConsumePasswordResetToken atomically verifies and marks a password reset token as used in a single statement.
// This prevents concurrent double-consumption race conditions.
func (r *TokenRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string) (*TokenRecord, error) {
	query := `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to consume password reset token: %w", err)
	}
	return rec, nil
}

// MarkPasswordResetTokenUsed invalidates a password reset token.
func (r *TokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenID string) error {
	query := `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token used: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// CreateEmailVerificationToken stores a hashed email verification token.
func (r *TokenRepository) CreateEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*TokenRecord, error) {
	query := `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, expiresAt).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert email verification token: %w", err)
	}
	return rec, nil
}

// GetValidEmailVerificationToken retrieves an unused, unexpired email verification token.
func (r *TokenRepository) GetValidEmailVerificationToken(ctx context.Context, tokenHash string) (*TokenRecord, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to query email verification token: %w", err)
	}
	return rec, nil
}

// ConsumeEmailVerificationToken atomically verifies and marks an email verification token as used in a single statement.
// This prevents concurrent double-consumption race conditions.
func (r *TokenRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*TokenRecord, error) {
	query := `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`
	rec := &TokenRecord{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.UsedAt,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to consume email verification token: %w", err)
	}
	return rec, nil
}

// MarkEmailVerificationTokenUsed invalidates an email verification token.
func (r *TokenRepository) MarkEmailVerificationTokenUsed(ctx context.Context, tokenID string) error {
	query := `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark email verification token used: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

