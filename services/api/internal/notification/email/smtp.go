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
	TLSMode   string        // "starttls", "tls", "none", or "auto"
	Timeout   time.Duration // Delivery operation timeout; defaults to 15s if <= 0
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
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
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

// SendEmail delivers an email via SMTP respecting configured TLS mode and bounded deadlines.
func (c *Client) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	if c.cfg.Host == "" {
		return errors.New("SMTP host is not configured")
	}
	if to == "" {
		return errors.New("recipient email address cannot be empty")
	}

	timeout := c.cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	// Apply bounded execution: ensure context has an active bounded deadline.
	var cancel context.CancelFunc
	if dl, ok := ctx.Deadline(); !ok || time.Until(dl) > timeout {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("SMTP context aborted before transmission: %w", err)
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
		return c.sendDirectTLS(ctx, addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	case "none":
		return c.sendPlain(ctx, addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	default: // "starttls"
		return c.sendSTARTTLS(ctx, addr, auth, c.cfg.FromEmail, []string{to}, msgData)
	}
}

func applyConnDeadlines(ctx context.Context, conn net.Conn) func() {
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	stop := context.AfterFunc(ctx, func() {
		_ = conn.SetDeadline(time.Now())
	})
	return func() {
		stop()
	}
}

func executeSMTPTransaction(ctx context.Context, client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				if ctx.Err() != nil {
					return fmt.Errorf("SMTP auth aborted: %w", ctx.Err())
				}
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		} else {
			if err := client.Auth(auth); err != nil {
				if ctx.Err() != nil {
					return fmt.Errorf("SMTP auth aborted: %w", ctx.Err())
				}
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP MAIL FROM aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("SMTP RCPT TO aborted: %w", ctx.Err())
			}
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP DATA command aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP write data aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP write data failed: %w", err)
	}
	if err := w.Close(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP data close aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP data close failed: %w", err)
	}
	return nil
}

func (c *Client) sendPlain(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP plain dial aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP plain dial failed: %w", err)
	}
	defer conn.Close()

	cleanup := applyConnDeadlines(ctx, conn)
	defer cleanup()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP new client failed: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP new client failed: %w", err)
	}
	defer func() { _ = client.Quit() }()

	return executeSMTPTransaction(ctx, client, auth, from, to, msg)
}

func (c *Client) sendSTARTTLS(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP STARTTLS dial aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP STARTTLS dial failed: %w", err)
	}
	defer conn.Close()

	cleanup := applyConnDeadlines(ctx, conn)
	defer cleanup()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP new client failed: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP new client failed: %w", err)
	}
	defer func() { _ = client.Quit() }()

	// Check for STARTTLS support
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         c.cfg.Host,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("SMTP StartTLS aborted: %w", ctx.Err())
			}
			return fmt.Errorf("SMTP StartTLS failed: %w", err)
		}
	} else {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	return executeSMTPTransaction(ctx, client, auth, from, to, msg)
}

func (c *Client) sendDirectTLS(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName:         c.cfg.Host,
		InsecureSkipVerify: false,
	}

	var dialer net.Dialer
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP TLS dial aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP TLS dial failed: %w", err)
	}
	defer rawConn.Close()

	cleanup := applyConnDeadlines(ctx, rawConn)
	defer cleanup()

	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP TLS handshake aborted: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP TLS handshake failed: %w", err)
	}

	client, err := smtp.NewClient(tlsConn, c.cfg.Host)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("SMTP TLS client failed: %w", ctx.Err())
		}
		return fmt.Errorf("SMTP TLS client failed: %w", err)
	}
	defer func() { _ = client.Quit() }()

	return executeSMTPTransaction(ctx, client, auth, from, to, msg)
}

