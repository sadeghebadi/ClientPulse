package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/testhelper"
)

func setupTestAuthService(t *testing.T) (*AuthService, *mail.MockMailer, func()) {
	t.Helper()
	db := testhelper.GetTestDB(t)
	cfg := testhelper.GetTestConfig()

	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)
	tokenRepo := repository.NewTokenRepository(db.Pool)
	auditRepo := repository.NewAuditRepository(db.Pool)
	mockMailer := mail.NewMockMailer()

	authService := NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)

	cleanup := func() {
		db.Close()
	}

	return authService, mockMailer, cleanup
}

func TestSignupAndLoginFlow(t *testing.T) {
	svc, _, cleanup := setupTestAuthService(t)
	defer cleanup()

	ctx := context.Background()
	randID, _ := auth.GenerateSecureToken(6)
	email := fmt.Sprintf("user-%s@example.com", randID)
	password := "Secure-Password-12345!"
	displayName := "Jane Developer"

	// 1. Successful Registration
	signupRes, err := svc.Signup(ctx, SignupRequest{
		Email:            email,
		Password:         password,
		DisplayName:      displayName,
		OrganizationName: "Jane Tech Agency",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	if signupRes.User.Email != strings.ToLower(email) {
		t.Errorf("expected email %s, got %s", email, signupRes.User.Email)
	}
	if signupRes.Organization == nil {
		t.Fatal("expected auto-created initial organization")
	}
	if signupRes.Role != "owner" {
		t.Errorf("expected creator to be owner, got %s", signupRes.Role)
	}
	if signupRes.RawSession == "" {
		t.Fatal("expected raw session token to be returned for cookie setting")
	}

	// 2. Password hash never exposed in JSON serialization
	userJSON, err := json.Marshal(signupRes.User)
	if err != nil {
		t.Fatalf("failed to marshal user: %v", err)
	}
	if strings.Contains(string(userJSON), "password_hash") || strings.Contains(string(userJSON), "$argon2id") {
		t.Fatal("CRITICAL: password hash exposed in JSON output!")
	}

	// 3. Duplicate email rejection
	_, err = svc.Signup(ctx, SignupRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Duplicate Jane",
	})
	if err != repository.ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists on duplicate email, got: %v", err)
	}

	// 4. Successful login
	loginRes, err := svc.Login(ctx, email, password, "127.0.0.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginRes.User.ID != signupRes.User.ID {
		t.Errorf("expected user id %s, got %s", signupRes.User.ID, loginRes.User.ID)
	}

	// 5. Incorrect password rejection
	_, err = svc.Login(ctx, email, "Incorrect-Password-9999!", "127.0.0.1", "TestAgent/1.0")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for bad password, got: %v", err)
	}

	// 6. Non-existent user login rejection (anti-enumeration check)
	_, err = svc.Login(ctx, "nonexistent@example.com", "Some-Password-12345!", "127.0.0.1", "TestAgent/1.0")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for unknown email, got: %v", err)
	}
}

func TestSessionExpirationAndRevocation(t *testing.T) {
	svc, _, cleanup := setupTestAuthService(t)
	defer cleanup()

	ctx := context.Background()
	randID, _ := auth.GenerateSecureToken(6)
	email := fmt.Sprintf("session-%s@example.com", randID)
	password := "Secure-Password-12345!"

	_, err := svc.Signup(ctx, SignupRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Session Tester",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	// Login to get a session
	loginRes, err := svc.Login(ctx, email, password, "127.0.0.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	tokenHash := auth.HashToken(loginRes.RawSession, svc.cfg.TokenHashPepper)
	sess, err := svc.sessionRepo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("failed to retrieve session: %v", err)
	}

	// Revoke session
	if err := svc.Logout(ctx, sess.ID); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Verify revoked session is rejected
	_, err = svc.sessionRepo.GetSessionByTokenHash(ctx, tokenHash)
	if err != repository.ErrSessionRevoked {
		t.Fatalf("expected ErrSessionRevoked, got: %v", err)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	svc, mockMailer, cleanup := setupTestAuthService(t)
	defer cleanup()

	ctx := context.Background()
	randID, _ := auth.GenerateSecureToken(6)
	email := fmt.Sprintf("reset-%s@example.com", randID)
	oldPassword := "Old-Password-123456!"
	newPassword := "Brand-New-Password-123456!"

	signupRes, err := svc.Signup(ctx, SignupRequest{
		Email:       email,
		Password:    oldPassword,
		DisplayName: "Reset Tester",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	// Anti-enumeration test: non-existent email returns nil error
	if err := svc.ForgotPassword(ctx, "does-not-exist@example.com"); err != nil {
		t.Fatalf("forgot password should not error on unknown email: %v", err)
	}

	// Request password reset for existing user
	if err := svc.ForgotPassword(ctx, email); err != nil {
		t.Fatalf("forgot password failed: %v", err)
	}

	// Find the reset token sent via mock mailer
	lastEmail := mockMailer.GetLastEmail()
	if lastEmail == nil {
		t.Fatal("expected password reset email to be sent")
	}

	// Extract token from reset URL
	var rawToken string
	parts := strings.Split(lastEmail.HTMLBody, "token=")
	if len(parts) > 1 {
		rawToken = strings.Split(parts[1], `"`)[0]
	}
	if rawToken == "" {
		t.Fatal("could not extract token from reset email body")
	}

	// Reset password with token
	if err := svc.ResetPassword(ctx, rawToken, newPassword); err != nil {
		t.Fatalf("reset password failed: %v", err)
	}

	// Single-use enforcement: re-using the token must fail!
	err = svc.ResetPassword(ctx, rawToken, "Another-New-Password-123456!")
	if err == nil {
		t.Fatal("expected reused token to be rejected, but it succeeded!")
	}

	// Verify old password no longer works
	_, err = svc.Login(ctx, email, oldPassword, "127.0.0.1", "TestAgent/1.0")
	if err != ErrInvalidCredentials {
		t.Fatalf("old password should be rejected, got: %v", err)
	}

	// Verify new password works
	_, err = svc.Login(ctx, email, newPassword, "127.0.0.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}

	// Verify all prior sessions were revoked on password reset
	priorHash := auth.HashToken(signupRes.RawSession, svc.cfg.TokenHashPepper)
	_, err = svc.sessionRepo.GetSessionByTokenHash(ctx, priorHash)
	if err != repository.ErrSessionRevoked {
		t.Fatalf("expected prior session to be revoked, got: %v", err)
	}
}

func TestEmailVerificationFlow(t *testing.T) {
	svc, mockMailer, cleanup := setupTestAuthService(t)
	defer cleanup()

	ctx := context.Background()
	randID, _ := auth.GenerateSecureToken(6)
	email := fmt.Sprintf("verify-%s@example.com", randID)
	password := "Secure-Password-12345!"

	signupRes, err := svc.Signup(ctx, SignupRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Verify Tester",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	// Send verification email directly to test
	if err := svc.SendVerificationEmail(ctx, signupRes.User.ID, signupRes.User.Email, signupRes.User.DisplayName); err != nil {
		t.Fatalf("failed to send verification email: %v", err)
	}

	lastEmail := mockMailer.GetLastEmail()
	if lastEmail == nil {
		t.Fatal("expected verification email to be sent")
	}

	var rawToken string
	parts := strings.Split(lastEmail.HTMLBody, "token=")
	if len(parts) > 1 {
		rawToken = strings.Split(parts[1], `"`)[0]
	}
	if rawToken == "" {
		t.Fatal("could not extract verification token")
	}

	// Verify email with token
	if err := svc.VerifyEmail(ctx, rawToken); err != nil {
		t.Fatalf("verify email failed: %v", err)
	}

	// Single-use: verifying again must fail
	err = svc.VerifyEmail(ctx, rawToken)
	if err == nil {
		t.Fatal("expected reused verification token to fail")
	}

	// Verify user record now has email_verified_at set
	user, err := svc.userRepo.GetUserByID(ctx, signupRes.User.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user.EmailVerifiedAt == nil {
		t.Fatal("expected EmailVerifiedAt to be populated")
	}
}
