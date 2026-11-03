package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrUserSuspended          = errors.New("user account has been suspended")
	ErrEmailNotVerified       = errors.New("email address must be verified")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrPasswordSameAsCurrent  = errors.New("new password cannot be identical to current password")
)

type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	orgRepo     *repository.OrganizationRepository
	tokenRepo   *repository.TokenRepository
	auditRepo   *repository.AuditRepository
	adminRepo   *repository.PlatformAdminRepository
	mailer      mail.EmailSender
	cfg         *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	orgRepo *repository.OrganizationRepository,
	tokenRepo *repository.TokenRepository,
	auditRepo *repository.AuditRepository,
	mailer mail.EmailSender,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		orgRepo:     orgRepo,
		tokenRepo:   tokenRepo,
		auditRepo:   auditRepo,
		mailer:      mailer,
		cfg:         cfg,
	}
}

// WithPlatformAdminRepo injects platform admin repo for global privilege resolution.
func (s *AuthService) WithPlatformAdminRepo(adminRepo *repository.PlatformAdminRepository) *AuthService {
	s.adminRepo = adminRepo
	return s
}

type SignupRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	DisplayName      string `json:"display_name"`
	OrganizationName string `json:"organization_name,omitempty"`
	IPAddress        string `json:"-"`
	UserAgent        string `json:"-"`
}

type AuthResult struct {
	User            *repository.User         `json:"user"`
	Organization    *repository.Organization `json:"organization,omitempty"`
	Role            string                   `json:"role,omitempty"`
	IsPlatformAdmin bool                     `json:"is_platform_admin"`
	RawSession      string                   `json:"-"`
	RawCSRF         string                   `json:"csrf_token"`
}

type MeResult struct {
	User                *repository.User        `json:"user"`
	ActiveOrganization  *repository.Organization `json:"active_organization,omitempty"`
	Role                string                  `json:"role,omitempty"`
	IsPlatformAdmin     bool                    `json:"is_platform_admin"`
	Memberships         []repository.Membership `json:"memberships"`
}

// Signup creates a user account, hashes password with Argon2id, provisions initial org, and issues a session.
func (s *AuthService) Signup(ctx context.Context, req SignupRequest) (*AuthResult, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return nil, errors.New("valid email address is required")
	}
	if req.DisplayName == "" {
		return nil, errors.New("display name is required")
	}
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	// Hash password using Argon2id
	passHash, err := auth.HashPassword(req.Password, auth.DefaultArgon2Params)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.userRepo.CreateUser(ctx, req.Email, passHash, req.DisplayName)
	if err != nil {
		return nil, err
	}

	// Automatically provision initial organization per FR-AUTH-04
	orgName := strings.TrimSpace(req.OrganizationName)
	if orgName == "" {
		orgName = fmt.Sprintf("%s's Agency", user.DisplayName)
	}

	slug := slugify(orgName)
	if slug == "" {
		slug = "agency-" + user.ID[:8]
	}

	// Guarantee unique slug
	uniqueSlug := slug
	for i := 1; i <= 5; i++ {
		_, err := s.orgRepo.GetOrganizationBySlug(ctx, uniqueSlug)
		if errors.Is(err, repository.ErrOrgNotFound) {
			break
		}
		uniqueSlug = fmt.Sprintf("%s-%d", slug, i)
	}

	org, err := s.orgRepo.CreateOrganization(ctx, orgName, uniqueSlug, s.cfg.DefaultLocale, s.cfg.DefaultTimezone)
	if err != nil {
		// Fallback slug with random suffix if collision
		randSuffix, _ := auth.GenerateSecureToken(4)
		org, err = s.orgRepo.CreateOrganization(ctx, orgName, fmt.Sprintf("%s-%s", slug, strings.ToLower(randSuffix)), s.cfg.DefaultLocale, s.cfg.DefaultTimezone)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial organization: %w", err)
		}
	}

	// Register creator as Owner
	_, err = s.orgRepo.AddMember(ctx, org.ID, user.ID, "owner")
	if err != nil {
		return nil, fmt.Errorf("failed to attach organization owner membership: %w", err)
	}

	// Create session
	rawSessionToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	hashedSessionToken := auth.HashToken(rawSessionToken, s.cfg.TokenHashPepper)

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	sess, err := s.sessionRepo.CreateSession(ctx, user.ID, hashedSessionToken, req.IPAddress, req.UserAgent, expiresAt, &org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate Session-bound Signed CSRF token
	rawCSRFToken, err := auth.GenerateSessionCSRFToken(sess.ID, s.cfg.SessionSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate csrf token: %w", err)
	}

	// Send verification email in background
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = s.SendVerificationEmail(bgCtx, user.ID, user.Email, user.DisplayName)
	}()

	// Record audit event
	ipHash := hashIP(req.IPAddress)
	_ = s.auditRepo.RecordEvent(ctx, &org.ID, &user.ID, "auth.signup", "user", user.ID, nil, ipHash)

	return &AuthResult{
		User:         user,
		Organization: org,
		Role:         "owner",
		RawSession:   rawSessionToken,
		RawCSRF:      rawCSRFToken,
	}, nil
}

// Login verifies credentials and creates a session.
func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	valid, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	if user.Status == "suspended" || user.Status == "deactivated" {
		return nil, ErrUserSuspended
	}

	// Resolve user's organizations
	memberships, err := s.orgRepo.ListUserMemberships(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user memberships: %w", err)
	}

	var activeOrg *repository.Organization
	activeRole := ""
	var activeOrgID *string

	if len(memberships) > 0 {
		activeOrgID = &memberships[0].OrganizationID
		activeRole = memberships[0].Role
		org, err := s.orgRepo.GetOrganizationByID(ctx, memberships[0].OrganizationID)
		if err == nil {
			activeOrg = org
		}
	}

	// Issue session
	rawSessionToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	hashedSessionToken := auth.HashToken(rawSessionToken, s.cfg.TokenHashPepper)

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	sess, err := s.sessionRepo.CreateSession(ctx, user.ID, hashedSessionToken, ip, userAgent, expiresAt, activeOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	rawCSRFToken, err := auth.GenerateSessionCSRFToken(sess.ID, s.cfg.SessionSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate csrf token: %w", err)
	}

	ipHash := hashIP(ip)
	_ = s.auditRepo.RecordEvent(ctx, activeOrgID, &user.ID, "auth.login", "user", user.ID, nil, ipHash)

	isAdmin := false
	if s.adminRepo != nil {
		isAdmin, _ = s.adminRepo.IsUserPlatformAdmin(ctx, user.ID)
	}

	return &AuthResult{
		User:            user,
		Organization:    activeOrg,
		Role:            activeRole,
		IsPlatformAdmin: isAdmin,
		RawSession:      rawSessionToken,
		RawCSRF:         rawCSRFToken,
	}, nil
}

// Logout revokes the given session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.RevokeSession(ctx, sessionID)
}

// GetMe retrieves identity, memberships, and active organization.
func (s *AuthService) GetMe(ctx context.Context, userID, sessionID string) (*MeResult, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	memberships, err := s.orgRepo.ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}

	var activeOrg *repository.Organization
	activeRole := ""

	// Check if session has active organization
	if sessionID != "" {
		sessions, err := s.sessionRepo.ListUserSessions(ctx, userID)
		if err == nil {
			for _, sess := range sessions {
				if sess.ID == sessionID && sess.ActiveOrganizationID != nil {
					// Verify user is still a member of this active organization
					for _, m := range memberships {
						if m.OrganizationID == *sess.ActiveOrganizationID {
							activeRole = m.Role
							org, err := s.orgRepo.GetOrganizationByID(ctx, m.OrganizationID)
							if err == nil {
								activeOrg = org
							}
							break
						}
					}
					break
				}
			}
		}
	}

	// Fallback to first membership if active org not resolved
	if activeOrg == nil && len(memberships) > 0 {
		activeRole = memberships[0].Role
		org, err := s.orgRepo.GetOrganizationByID(ctx, memberships[0].OrganizationID)
		if err == nil {
			activeOrg = org
		}
	}

	isAdmin := false
	if s.adminRepo != nil {
		isAdmin, _ = s.adminRepo.IsUserPlatformAdmin(ctx, userID)
	}

	return &MeResult{
		User:               user,
		ActiveOrganization: activeOrg,
		Role:               activeRole,
		IsPlatformAdmin:    isAdmin,
		Memberships:        memberships,
	}, nil
}

// SendVerificationEmail issues a verification token and sends it via email.
func (s *AuthService) SendVerificationEmail(ctx context.Context, userID, email, displayName string) error {
	rawToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return err
	}
	tokenHash := auth.HashToken(rawToken, s.cfg.TokenHashPepper)

	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = s.tokenRepo.CreateEmailVerificationToken(ctx, userID, tokenHash, expiresAt)
	if err != nil {
		return err
	}

	baseURL := s.cfg.PublicAppURL
	if baseURL == "" {
		baseURL = s.cfg.AppBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, url.QueryEscape(rawToken))
	return s.mailer.SendVerificationEmail(ctx, email, displayName, verifyURL)
}

// VerifyEmail processes an email verification token atomically.
func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	tokenHash := auth.HashToken(rawToken, s.cfg.TokenHashPepper)

	rec, err := s.tokenRepo.ConsumeEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return errors.New("invalid or expired verification link")
	}

	if err := s.userRepo.SetEmailVerified(ctx, rec.UserID); err != nil {
		return err
	}

	return nil
}

// ForgotPassword handles password reset requests with anti-enumeration protection.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Prevent user enumeration: always return success
		return nil
	}

	rawToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return err
	}
	tokenHash := auth.HashToken(rawToken, s.cfg.TokenHashPepper)

	// Expiry strictly 30 minutes per PRD FR-AUTH-03
	expiresAt := time.Now().Add(30 * time.Minute)
	_, err = s.tokenRepo.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt)
	if err != nil {
		return err
	}

	baseURL := s.cfg.PublicAppURL
	if baseURL == "" {
		baseURL = s.cfg.AppBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, url.QueryEscape(rawToken))
	return s.mailer.SendPasswordResetEmail(ctx, user.Email, user.DisplayName, resetURL)
}

// ResetPassword sets a new password using a valid reset token and revokes all active sessions.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if err := auth.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	tokenHash := auth.HashToken(rawToken, s.cfg.TokenHashPepper)
	// Atomically consume token in single SQL statement to prevent concurrent replay
	rec, err := s.tokenRepo.ConsumePasswordResetToken(ctx, tokenHash)
	if err != nil {
		return errors.New("invalid or expired password reset link")
	}

	passHash, err := auth.HashPassword(newPassword, auth.DefaultArgon2Params)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, rec.UserID, passHash); err != nil {
		return err
	}

	// Invalidate all active sessions per PRD FR-AUTH-03
	_ = s.sessionRepo.RevokeAllUserSessions(ctx, rec.UserID)

	return nil
}

// ChangePassword updates password for an authenticated user and revokes prior sessions.
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	valid, err := auth.VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil || !valid {
		return errors.New("current password is incorrect")
	}

	if currentPassword == newPassword {
		return ErrPasswordSameAsCurrent
	}

	if err := auth.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	passHash, err := auth.HashPassword(newPassword, auth.DefaultArgon2Params)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, passHash); err != nil {
		return err
	}

	// Revoke existing sessions to secure account
	_ = s.sessionRepo.RevokeAllUserSessions(ctx, userID)

	return nil
}

// Helper functions
func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func hashIP(ip string) string {
	if ip == "" {
		return ""
	}
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:8])
}

