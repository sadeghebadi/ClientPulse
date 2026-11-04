package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockEmailSender struct {
	called   bool
	to       string
	subject  string
	htmlBody string
	textBody string
	errToRet error
}

func (m *mockEmailSender) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	m.called = true
	m.to = to
	m.subject = subject
	m.htmlBody = htmlBody
	m.textBody = textBody
	return m.errToRet
}

func TestSMTPMailer_SendEmailDelegation(t *testing.T) {
	mockSender := &mockEmailSender{}
	mailer := NewSMTPMailerWithSender(mockSender)

	ctx := context.Background()
	err := mailer.SendEmail(ctx, "user@example.com", "Test Subject", "<p>Hello</p>", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mockSender.called {
		t.Fatal("expected SendEmail to be called on underlying client")
	}
	if mockSender.to != "user@example.com" {
		t.Errorf("expected to user@example.com, got %s", mockSender.to)
	}
	if mockSender.subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %s", mockSender.subject)
	}
	if mockSender.htmlBody != "<p>Hello</p>" {
		t.Errorf("expected htmlBody '<p>Hello</p>', got %s", mockSender.htmlBody)
	}
	if mockSender.textBody != "Hello" {
		t.Errorf("expected textBody 'Hello', got %s", mockSender.textBody)
	}
}

func TestSMTPMailer_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("underlying network failure")
	mockSender := &mockEmailSender{errToRet: expectedErr}
	mailer := NewSMTPMailerWithSender(mockSender)

	err := mailer.SendEmail(context.Background(), "user@example.com", "Subject", "HTML", "Text")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestSMTPMailer_SendVerificationEmail(t *testing.T) {
	mockSender := &mockEmailSender{}
	mailer := NewSMTPMailerWithSender(mockSender)

	verifyURL := "https://app.clientpulse.io/verify-email?token=abcdef123456"
	err := mailer.SendVerificationEmail(context.Background(), "newuser@example.com", "Alice Smith", verifyURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mockSender.called {
		t.Fatal("expected SendEmail to be invoked")
	}
	if mockSender.to != "newuser@example.com" {
		t.Errorf("expected recipient newuser@example.com, got %s", mockSender.to)
	}
	if !strings.Contains(mockSender.subject, "Verify your ClientPulse account") {
		t.Errorf("unexpected subject: %s", mockSender.subject)
	}
	if !strings.Contains(mockSender.htmlBody, verifyURL) {
		t.Errorf("expected verification URL in HTML body")
	}
	if !strings.Contains(mockSender.htmlBody, "Alice Smith") {
		t.Errorf("expected displayName Alice Smith in HTML body")
	}
	if !strings.Contains(mockSender.textBody, verifyURL) {
		t.Errorf("expected verification URL in text body")
	}
}

func TestSMTPMailer_SendPasswordResetEmail(t *testing.T) {
	mockSender := &mockEmailSender{}
	mailer := NewSMTPMailerWithSender(mockSender)

	resetURL := "https://app.clientpulse.io/reset-password?token=secrettoken999"
	err := mailer.SendPasswordResetEmail(context.Background(), "resetuser@example.com", "Bob Jones", resetURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mockSender.called {
		t.Fatal("expected SendEmail to be invoked")
	}
	if mockSender.to != "resetuser@example.com" {
		t.Errorf("expected recipient resetuser@example.com, got %s", mockSender.to)
	}
	if !strings.Contains(mockSender.subject, "Reset your ClientPulse password") {
		t.Errorf("unexpected subject: %s", mockSender.subject)
	}
	if !strings.Contains(mockSender.htmlBody, resetURL) {
		t.Errorf("expected reset URL in HTML body")
	}
	if !strings.Contains(mockSender.htmlBody, "Bob Jones") {
		t.Errorf("expected displayName Bob Jones in HTML body")
	}
	if !strings.Contains(mockSender.textBody, resetURL) {
		t.Errorf("expected reset URL in text body")
	}
}

func TestSMTPMailer_NilClientError(t *testing.T) {
	mailer := &SMTPMailer{client: nil}
	err := mailer.SendEmail(context.Background(), "a@b.com", "s", "h", "t")
	if err == nil {
		t.Fatal("expected error when client is nil, got nil")
	}
}

func TestNewSMTPMailer_ConstructsClient(t *testing.T) {
	mailer := NewSMTPMailer(SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "alerts@example.com",
		FromName: "Custom Alerts",
		TLSMode:  "starttls",
	})
	if mailer.client == nil {
		t.Fatal("expected non-nil client")
	}
}
