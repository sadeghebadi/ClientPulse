package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	TLSMode   string // "starttls", "tls", "none", or "auto"
}

type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
}

type Client struct {
	cfg SMTPConfig
}

func NewClient(cfg SMTPConfig) *Client {
	if cfg.FromName == "" {
		cfg.FromName = "ClientPulse Alerts"
	}
	if cfg.TLSMode == "" || cfg.TLSMode == "auto" {
		if cfg.Port == 465 {
			cfg.TLSMode = "tls"
		} else if cfg.Port == 1025 {
			cfg.TLSMode = "none"
		} else {
			cfg.TLSMode = "starttls"
		}
	}
	return &Client{cfg: cfg}
}

// SendEmail delivers an email via SMTP respecting configured TLS mode.
func (c *Client) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	if c.cfg.Host == "" {
		return errors.New("SMTP host is not configured")
	}
	if to == "" {
		return errors.New("recipient email address cannot be empty")
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	fromHeader := fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", c.cfg.FromName), c.cfg.FromEmail)
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)

	boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msgBuilder.WriteString("\r\n")

	// Plain text part
	if textBody != "" {
		msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msgBuilder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		msgBuilder.WriteString(textBody)
		msgBuilder.WriteString("\r\n\r\n")
	}

	// HTML part
	if htmlBody != "" {
		msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msgBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		msgBuilder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		msgBuilder.WriteString(htmlBody)
		msgBuilder.WriteString("\r\n\r\n")
	}

	msgBuilder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	msgData := []byte(msgBuilder.String())

	var auth smtp.Auth
	if c.cfg.Username != "" || c.cfg.Password != "" {
		auth = smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
	}

	switch strings.ToLower(c.cfg.TLSMode) {
	case "tls", "ssl":
		return c.sendDirectTLS(addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	case "none":
		return c.sendPlain(addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	default: // "starttls"
		return c.sendSTARTTLS(addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	}
}

func (c *Client) sendPlain(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP plain dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP new client failed: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write data failed: %w", err)
	}
	return w.Close()
}

func (c *Client) sendSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP STARTTLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP new client failed: %w", err)
	}
	defer client.Quit()

	// Check for STARTTLS support
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         c.cfg.Host,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP StartTLS failed: %w", err)
		}
	} else {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write data failed: %w", err)
	}
	return w.Close()
}

func (c *Client) sendDirectTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName:         c.cfg.Host,
		InsecureSkipVerify: false,
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("SMTP TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP TLS client failed: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP TLS auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	return w.Close()
}

