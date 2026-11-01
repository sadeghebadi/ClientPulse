package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
)

// EmailSender defines the interface for delivering emails.
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
	SendVerificationEmail(ctx context.Context, to, displayName, verifyURL string) error
	SendPasswordResetEmail(ctx context.Context, to, displayName, resetURL string) error
}

// SMTPConfig holds SMTP connection parameters.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPMailer delivers emails over standard SMTP (e.g. Mailpit, AWS SES, SendGrid).
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer creates an SMTP-backed email sender.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// SendEmail transmits an email via SMTP.
func (m *SMTPMailer) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	var auth smtp.Auth
	if m.cfg.Username != "" || m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	headers := make(map[string]string)
	headers["From"] = m.cfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = `text/html; charset="UTF-8"`

	var msgBuilder strings.Builder
	for k, v := range headers {
		msgBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	msg := []byte(msgBuilder.String())

	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg)
}

// SendVerificationEmail builds and sends an email verification notification.
func (m *SMTPMailer) SendVerificationEmail(ctx context.Context, to, displayName, verifyURL string) error {
	subject := "Verify your ClientPulse account"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #1e293b; padding: 24px;">
  <div style="max-width: 560px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 32px;">
    <h2 style="color: #0f172a; margin-top: 0;">Welcome to ClientPulse</h2>
    <p>Hi %s,</p>
    <p>Thank you for signing up for ClientPulse. Please verify your email address to complete your registration and secure your account.</p>
    <div style="margin: 28px 0;">
      <a href="%s" style="background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 500; display: inline-block;">Verify Email Address</a>
    </div>
    <p style="color: #64748b; font-size: 14px;">If you did not create a ClientPulse account, you can safely ignore this email.</p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
    <p style="color: #94a3b8; font-size: 12px; margin-bottom: 0;">ClientPulse • Multi-tenant Agency Reliability & Maintenance Platform</p>
  </div>
</body>
</html>`, displayName, verifyURL)

	text := fmt.Sprintf("Hi %s,\n\nPlease verify your email by visiting: %s\n\nIf you did not sign up, please ignore this email.", displayName, verifyURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

// SendPasswordResetEmail builds and sends a password reset instructions notification.
func (m *SMTPMailer) SendPasswordResetEmail(ctx context.Context, to, displayName, resetURL string) error {
	subject := "Reset your ClientPulse password"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #1e293b; padding: 24px;">
  <div style="max-width: 560px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 32px;">
    <h2 style="color: #0f172a; margin-top: 0;">Password Reset Request</h2>
    <p>Hi %s,</p>
    <p>We received a request to reset your ClientPulse account password. Click the button below to choose a new password. This link is valid for 30 minutes.</p>
    <div style="margin: 28px 0;">
      <a href="%s" style="background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 500; display: inline-block;">Reset Password</a>
    </div>
    <p style="color: #64748b; font-size: 14px;">If you did not request a password reset, you can safely ignore this message. Your password will remain unchanged.</p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
    <p style="color: #94a3b8; font-size: 12px; margin-bottom: 0;">ClientPulse • Multi-tenant Agency Reliability & Maintenance Platform</p>
  </div>
</body>
</html>`, displayName, resetURL)

	text := fmt.Sprintf("Hi %s,\n\nReset your ClientPulse password: %s\n\nValid for 30 minutes. If you did not request this, ignore this email.", displayName, resetURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

// MockMailer records sent emails in memory for unit and integration testing.
type MockMailer struct {
	mu     sync.Mutex
	Emails []SentEmail
}

type SentEmail struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

func NewMockMailer() *MockMailer {
	return &MockMailer{}
}

func (m *MockMailer) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Emails = append(m.Emails, SentEmail{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	return nil
}

func (m *MockMailer) SendVerificationEmail(ctx context.Context, to, displayName, verifyURL string) error {
	return m.SendEmail(ctx, to, "Verify your ClientPulse account", verifyURL, verifyURL)
}

func (m *MockMailer) SendPasswordResetEmail(ctx context.Context, to, displayName, resetURL string) error {
	return m.SendEmail(ctx, to, "Reset your ClientPulse password", resetURL, resetURL)
}

// GetLastEmail retrieves the most recently sent email in the mock.
func (m *MockMailer) GetLastEmail() *SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Emails) == 0 {
		return nil
	}
	last := m.Emails[len(m.Emails)-1]
	return &last
}

