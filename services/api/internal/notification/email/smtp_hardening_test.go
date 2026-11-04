package email

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestSMTP_TCPConnectionTimeout verifies bounded TCP connection behavior.
func TestSMTP_TCPConnectionTimeout(t *testing.T) {
	// 198.51.100.1 is RFC 5737 TEST-NET-2 (non-routable blackhole IP)
	client := NewClient(SMTPConfig{
		Host:      "198.51.100.1",
		Port:      9876,
		Timeout:   100 * time.Millisecond,
		FromEmail: "test@clientpulse.local",
	})

	start := time.Now()
	err := client.SendEmail(context.Background(), "user@example.com", "Test", "<p>hi</p>", "hi")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected TCP dial timeout error, got nil")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected TCP dial to fail within bounded timeout, but took %v", elapsed)
	}
}

// TestSMTP_ServerAcceptsConnectionButNeverResponds verifies that if an SMTP server
// accepts the TCP handshake but never transmits the initial 220 banner, execution remains bounded.
func TestSMTP_ServerAcceptsConnectionButNeverResponds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Accept TCP connection but write nothing and keep connection open until test finishes
	done := make(chan struct{})
	defer close(done)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		<-done
	}()

	client := NewClient(SMTPConfig{
		Host:      "127.0.0.1",
		Port:      port,
		TLSMode:   "none",
		Timeout:   150 * time.Millisecond,
		FromEmail: "test@clientpulse.local",
	})

	start := time.Now()
	err = client.SendEmail(context.Background(), "user@example.com", "Subject", "Body", "Body")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error due to missing SMTP banner, got nil")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected bounded failure within ~150ms, took %v", elapsed)
	}
}

// TestSMTP_ServerStallsDuringMessageTransmission verifies that if an SMTP server
// hangs while accepting message data, execution unblocks and aborts within the deadline.
func TestSMTP_ServerStallsDuringMessageTransmission(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	defer close(done)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)

		// 1. Initial 220 banner
		writer.WriteString("220 mock.clientpulse.local ESMTP Mock\r\n")
		writer.Flush()

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			if strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO") {
				writer.WriteString("250-mock.clientpulse.local\r\n250 HELP\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "MAIL FROM:") {
				writer.WriteString("250 2.1.0 Sender OK\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "RCPT TO:") {
				writer.WriteString("250 2.1.5 Recipient OK\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "DATA") {
				writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
				writer.Flush()
				// Stall: do not respond to final message body
				<-done
				return
			}
		}
	}()

	client := NewClient(SMTPConfig{
		Host:      "127.0.0.1",
		Port:      port,
		TLSMode:   "none",
		Timeout:   150 * time.Millisecond,
		FromEmail: "test@clientpulse.local",
	})

	start := time.Now()
	err = client.SendEmail(context.Background(), "user@example.com", "Subject", "Body", "Body")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error due to stalled message transmission, got nil")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected bounded failure within ~150ms, took %v", elapsed)
	}
}

// TestSMTP_ContextCancellation verifies that cancelling the caller's context
// immediately aborts the in-flight SMTP transmission.
func TestSMTP_ContextCancellation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	defer close(done)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Sleep and hold connection
		<-done
	}()

	client := NewClient(SMTPConfig{
		Host:      "127.0.0.1",
		Port:      port,
		TLSMode:   "none",
		Timeout:   5 * time.Second,
		FromEmail: "test@clientpulse.local",
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel quickly
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err = client.SendEmail(ctx, "user@example.com", "Subject", "Body", "Body")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected error to indicate context canceled, got: %v", err)
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected cancellation to abort immediately, took %v", elapsed)
	}
}

// TestSMTP_STARTTLSNegotiationFailure verifies error handling when STARTTLS negotiation fails.
func TestSMTP_STARTTLSNegotiationFailure(t *testing.T) {
	t.Run("ServerRejectsSTARTTLSCommand", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to start mock listener: %v", err)
		}
		defer ln.Close()

		port := ln.Addr().(*net.TCPAddr).Port

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)

			writer.WriteString("220 mock.clientpulse.local ESMTP\r\n")
			writer.Flush()

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				cmd := strings.ToUpper(strings.TrimSpace(line))
				if strings.HasPrefix(cmd, "EHLO") {
					writer.WriteString("250-mock.clientpulse.local\r\n250-STARTTLS\r\n250 HELP\r\n")
					writer.Flush()
				} else if strings.HasPrefix(cmd, "STARTTLS") {
					// Reject STARTTLS
					writer.WriteString("454 4.7.0 TLS not available due to temporary reason\r\n")
					writer.Flush()
				} else if strings.HasPrefix(cmd, "QUIT") {
					writer.WriteString("221 2.0.0 Bye\r\n")
					writer.Flush()
					return
				}
			}
		}()

		client := NewClient(SMTPConfig{
			Host:      "127.0.0.1",
			Port:      port,
			TLSMode:   "starttls",
			Timeout:   2 * time.Second,
			FromEmail: "test@clientpulse.local",
		})

		err = client.SendEmail(context.Background(), "user@example.com", "Subject", "Body", "Body")
		if err == nil {
			t.Fatal("expected STARTTLS failure error, got nil")
		}
		if !strings.Contains(err.Error(), "StartTLS") {
			t.Fatalf("expected error mentioning StartTLS, got: %v", err)
		}
	})

	t.Run("ServerDoesNotSupportSTARTTLS", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to start mock listener: %v", err)
		}
		defer ln.Close()

		port := ln.Addr().(*net.TCPAddr).Port

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)

			writer.WriteString("220 mock.clientpulse.local ESMTP\r\n")
			writer.Flush()

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				cmd := strings.ToUpper(strings.TrimSpace(line))
				if strings.HasPrefix(cmd, "EHLO") {
					// No STARTTLS advertised
					writer.WriteString("250-mock.clientpulse.local\r\n250 8BITMIME\r\n")
					writer.Flush()
				} else if strings.HasPrefix(cmd, "QUIT") {
					writer.WriteString("221 2.0.0 Bye\r\n")
					writer.Flush()
					return
				}
			}
		}()

		client := NewClient(SMTPConfig{
			Host:      "127.0.0.1",
			Port:      port,
			TLSMode:   "starttls",
			Timeout:   2 * time.Second,
			FromEmail: "test@clientpulse.local",
		})

		err = client.SendEmail(context.Background(), "user@example.com", "Subject", "Body", "Body")
		if err == nil {
			t.Fatal("expected STARTTLS not supported error, got nil")
		}
		if !strings.Contains(err.Error(), "does not support STARTTLS") {
			t.Fatalf("expected error about unsupported STARTTLS, got: %v", err)
		}
	})
}

// TestSMTP_PortAutoDetection verifies that port auto-detection correctly selects TLSMode.
func TestSMTP_PortAutoDetection(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{465, "tls"},
		{1025, "none"},
		{587, "starttls"},
		{25, "starttls"},
	}

	for _, tt := range tests {
		c := NewClient(SMTPConfig{
			Host: "smtp.example.com",
			Port: tt.port,
		})
		if c.cfg.TLSMode != tt.expected {
			t.Errorf("port %d: expected TLSMode %s, got %s", tt.port, tt.expected, c.cfg.TLSMode)
		}
	}
}

// Helper to make port string
func intToString(p int) string {
	return strconv.Itoa(p)
}
