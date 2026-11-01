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
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// CreateUser creates a new user record.
func (r *UserRepository) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	emailNormalized := strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	query := `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, display_name, email_verified_at, status, created_at, updated_at
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, emailNormalized, passwordHash, displayName).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return user, nil
}

// GetUserByEmail finds a user by normalized email address.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	emailNormalized := strings.ToLower(strings.TrimSpace(email))
	query := `
		SELECT id, email, password_hash, display_name, email_verified_at, status, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND status != 'deactivated'
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, emailNormalized).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	return user, nil
}

// GetUserByID finds a user by UUID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, display_name, email_verified_at, status, created_at, updated_at
		FROM users
		WHERE id = $1 AND status != 'deactivated'
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	return user, nil
}

// UpdatePassword updates the password hash for a user.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'active'
	`
	cmdTag, err := r.pool.Exec(ctx, query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetEmailVerified marks a user's email as verified.
func (r *UserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET email_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	cmdTag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark email verified: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateProfile updates the display name of a user.
func (r *UserRepository) UpdateProfile(ctx context.Context, userID, displayName string) (*User, error) {
	query := `
		UPDATE users
		SET display_name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, password_hash, display_name, email_verified_at, status, created_at, updated_at
	`
	user := &User{}
	err := r.pool.QueryRow(ctx, query, displayName, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	return user, nil
}
