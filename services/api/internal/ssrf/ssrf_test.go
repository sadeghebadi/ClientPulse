package ssrf

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type mockResolver struct {
	hosts map[string][]net.IP
	err   error
}

func (m *mockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ips, ok := m.hosts[host]; ok {
		return ips, nil
	}
	return nil, errors.New("host not found in mock resolver")
}

func TestSSRFURLValidation(t *testing.T) {
	guard := NewGuard(nil)

	negativeCases := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost"},
		{"localhost with port 80", "http://localhost:80"},
		{"localhost subdomain", "http://sub.localhost"},
		{"IPv4 loopback", "http://127.0.0.1"},
		{"IPv4 loopback port 8080", "http://127.0.0.1:8080"},
		{"IPv4 private 10.x", "http://10.0.0.1"},
		{"IPv4 private 172.16.x", "http://172.16.0.1"},
		{"IPv4 private 192.168.x", "http://192.168.1.1"},
		{"Cloud metadata IP", "http://169.254.169.254"},
		{"Cloud metadata hostname google", "http://metadata.google.internal"},
		{"Cloud metadata hostname instance-data", "http://instance-data"},
		{"IPv6 loopback", "http://[::1]"},
		{"IPv6 ULA", "http://[fc00::1]"},
		{"IPv6 Link Local", "http://[fe80::1]"},
		{"IPv4-mapped IPv6 loopback", "http://[::ffff:127.0.0.1]"},
		{"IPv4-mapped IPv6 private", "http://[::ffff:192.168.1.1]"},
		{"IPv4-mapped IPv6 metadata", "http://[::ffff:169.254.169.254]"},
		{"Non-standard port 22", "http://example.com:22"},
		{"Non-standard port 3000", "http://example.com:3000"},
		{"Prohibited scheme ftp", "ftp://example.com"},
		{"Prohibited scheme file", "file:///etc/passwd"},
		{"Prohibited scheme gopher", "gopher://example.com"},
		{"Userinfo in URL", "http://user:password@example.com"},
		{"Internal hostname .local", "http://printer.local"},
		{"Internal hostname .internal", "http://service.internal"},
		{"Internal hostname .lan", "http://nas.lan"},
		{"Single label hostname", "http://intranet"},
	}

	for _, tc := range negativeCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := guard.ValidateURL(tc.url)
			if err == nil {
				t.Fatalf("expected URL %q to be rejected, but it was accepted", tc.url)
			}
		})
	}

	positiveCases := []struct {
		name string
		url  string
	}{
		{"Standard HTTP", "http://example.com"},
		{"Standard HTTPS", "https://example.com"},
		{"HTTP explicit port 80", "http://example.com:80"},
		{"HTTPS explicit port 443", "https://example.com:443"},
		{"Path and query parameters", "https://example.com/api/v1/health?check=1"},
		{"Subdomain public", "https://status.github.com"},
	}

	for _, tc := range positiveCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := guard.ValidateURL(tc.url)
			if err != nil {
				t.Fatalf("expected URL %q to be valid, got error: %v", tc.url, err)
			}
			if parsed == nil {
				t.Fatalf("expected parsed URL to be non-nil")
			}
		})
	}
}

func TestSSRFIsProhibitedIP(t *testing.T) {
	blockedIPs := []string{
		"127.0.0.1",
		"127.0.0.2",
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.0.1",
		"192.168.255.255",
		"169.254.169.254",
		"169.254.1.1",
		"100.64.0.1", // CGNAT
		"0.0.0.0",
		"224.0.0.1",       // Multicast
		"240.0.0.1",       // Reserved
		"255.255.255.255", // Broadcast
		"::1",             // IPv6 Loopback
		"::",              // IPv6 Unspecified
		"fe80::1",         // IPv6 Link-local
		"fc00::1",         // IPv6 ULA
		"fd12:3456::1",    // IPv6 ULA
		"ff02::1",         // IPv6 Multicast
		"::ffff:127.0.0.1",
		"::ffff:10.0.0.1",
		"::ffff:169.254.169.254",
	}

	for _, ipStr := range blockedIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("failed to parse test IP %q", ipStr)
		}
		if !IsProhibitedIP(ip) {
			t.Errorf("expected IP %s to be PROHIBITED, but it was allowed", ipStr)
		}
	}

	allowedIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34", // example.com
		"2606:2800:220:1:248:1893:25c8:1946",
	}

	for _, ipStr := range allowedIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("failed to parse test IP %q", ipStr)
		}
		if IsProhibitedIP(ip) {
			t.Errorf("expected IP %s to be ALLOWED, but it was prohibited", ipStr)
		}
	}
}

func TestSSRFDNSResolutionInspection(t *testing.T) {
	// Case 1: Hostname resolves to both public and private IP
	resolverWithMixedIPs := &mockResolver{
		hosts: map[string][]net.IP{
			"mixed.example.com": {
				net.ParseIP("93.184.216.34"),
				net.ParseIP("10.0.0.1"), // Private!
			},
		},
	}
	guardMixed := NewGuard(resolverWithMixedIPs)
	_, err := guardMixed.SafeDialContext(context.Background(), "tcp", "mixed.example.com:80")
	if err == nil {
		t.Fatal("expected connection to fail when DNS returns mixed public and private IPs")
	}

	// Case 2: Hostname resolves to loopback
	resolverLoopback := &mockResolver{
		hosts: map[string][]net.IP{
			"rebinding.example.com": {
				net.ParseIP("127.0.0.1"),
			},
		},
	}
	guardLoopback := NewGuard(resolverLoopback)
	_, err = guardLoopback.SafeDialContext(context.Background(), "tcp", "rebinding.example.com:80")
	if err == nil {
		t.Fatal("expected connection to fail when DNS returns loopback IP")
	}

	// Case 3: Hostname resolves to metadata IP
	resolverMetadata := &mockResolver{
		hosts: map[string][]net.IP{
			"fake-metadata.example.com": {
				net.ParseIP("169.254.169.254"),
			},
		},
	}
	guardMetadata := NewGuard(resolverMetadata)
	_, err = guardMetadata.SafeDialContext(context.Background(), "tcp", "fake-metadata.example.com:80")
	if err == nil {
		t.Fatal("expected connection to fail when DNS returns cloud metadata IP")
	}
}

func TestSSRFRedirectProtection(t *testing.T) {
	// Public server redirects to private IP or metadata
	privateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("sensitive internal data"))
	}))
	defer privateServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attempt redirect to private server or metadata
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer redirectServer.Close()

	guard := NewGuard(nil)
	client := guard.NewSafeClient(5*time.Second, 5)

	_, err := client.Get(redirectServer.URL)
	if err == nil {
		t.Fatal("expected redirect to metadata endpoint to be blocked, but succeeded")
	}
}

func TestSSRFProxyBypassPrevention(t *testing.T) {
	// Ensure that even if HTTP_PROXY or HTTPS_PROXY environment variables are set,
	// the SafeClient does NOT route through the proxy!
	os.Setenv("HTTP_PROXY", "http://malicious-proxy.local:8080")
	os.Setenv("HTTPS_PROXY", "http://malicious-proxy.local:8080")
	defer func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("HTTPS_PROXY")
	}()

	guard := NewGuard(nil)
	client := guard.NewSafeClient(2*time.Second, 5)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected client transport to be *http.Transport")
	}

	if transport.Proxy != nil {
		t.Fatal("transport.Proxy MUST be nil to prevent proxy environment bypass")
	}
}
