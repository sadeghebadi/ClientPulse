package ssrf

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrProhibitedScheme   = errors.New("prohibited URL scheme: only http and https are allowed")
	ErrUserInfoNotAllowed = errors.New("userinfo in URL is not allowed")
	ErrProhibitedPort     = errors.New("prohibited port: only ports 80 and 443 are allowed")
	ErrProhibitedHost     = errors.New("prohibited host: private, internal, or metadata hostnames are blocked")
	ErrProhibitedIP       = errors.New("prohibited IP: private, loopback, link-local, multicast, or metadata address")
	ErrNoIPAddresses      = errors.New("no IP addresses returned for hostname")
	ErrTooManyRedirects   = errors.New("maximum redirects exceeded")
)

// Prohibited IPv4 CIDRs
var prohibitedIPv4CIDRs = []string{
	"0.0.0.0/8",          // Current network (RFC 1122)
	"10.0.0.0/8",         // Private network (RFC 1918)
	"100.64.0.0/10",      // Shared address / CGNAT (RFC 6598)
	"127.0.0.0/8",        // Loopback (RFC 1122)
	"169.254.0.0/16",     // Link-local / Cloud metadata (RFC 3927)
	"172.16.0.0/12",      // Private network (RFC 1918)
	"192.0.0.0/24",       // IETF protocol assignments (RFC 6890)
	"192.0.2.0/24",       // TEST-NET-1 (RFC 5737)
	"192.88.99.0/24",     // 6to4 relay anycast (RFC 7526)
	"192.168.0.0/16",     // Private network (RFC 1918)
	"198.18.0.0/15",      // Network interconnect benchmark (RFC 2544)
	"198.51.100.0/24",    // TEST-NET-2 (RFC 5737)
	"203.0.113.0/24",     // TEST-NET-3 (RFC 5737)
	"224.0.0.0/4",        // Multicast (RFC 5771)
	"240.0.0.0/4",        // Reserved for future use (RFC 1112)
	"255.255.255.255/32", // Limited broadcast (RFC 919)
}

// Prohibited IPv6 CIDRs
var prohibitedIPv6CIDRs = []string{
	"::/128",         // Unspecified address (RFC 4291)
	"::1/128",        // Loopback (RFC 4291)
	"64:ff9b::/96",   // IPv4/IPv6 translation (RFC 6052)
	"100::/64",       // Discard prefix (RFC 6666)
	"2001::/23",      // IETF protocol assignments (RFC 2928)
	"2001:db8::/32",  // Documentation (RFC 3849)
	"fc00::/7",       // Unique Local Address / ULA (RFC 4193)
	"fe80::/10",      // Link-local unicast (RFC 4291)
	"ff00::/8",       // Multicast (RFC 4291)
}

var parsedProhibitedNets []*net.IPNet

func init() {
	for _, cidr := range prohibitedIPv4CIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedProhibitedNets = append(parsedProhibitedNets, ipNet)
		}
	}
	for _, cidr := range prohibitedIPv6CIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedProhibitedNets = append(parsedProhibitedNets, ipNet)
		}
	}
}

// IPResolver resolves hostnames to IP addresses.
type IPResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

type defaultResolver struct{}

func (d *defaultResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

// DefaultResolver is the standard DNS resolver.
var DefaultResolver IPResolver = &defaultResolver{}

// Guard encapsulates URL validation, DNS resolution, and safe HTTP transport creation.
type Guard struct {
	resolver IPResolver
}

// NewGuard creates a new SSRF Guard.
func NewGuard(resolver IPResolver) *Guard {
	if resolver == nil {
		resolver = DefaultResolver
	}
	return &Guard{resolver: resolver}
}

// Resolver returns the configured IP resolver.
func (g *Guard) Resolver() IPResolver {
	if g.resolver == nil {
		return DefaultResolver
	}
	return g.resolver
}

// ValidateURL validates that a raw URL is safe from SSRF attack vectors.
func (g *Guard) ValidateURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// 1. Enforce Scheme
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, ErrProhibitedScheme
	}

	// 2. Reject Userinfo
	if parsedURL.User != nil {
		return nil, ErrUserInfoNotAllowed
	}

	// 3. Extract Host and Port
	host := parsedURL.Host
	if host == "" {
		return nil, errors.New("empty host in URL")
	}

	hostname := parsedURL.Hostname()
	portStr := parsedURL.Port()

	// 4. Validate Port: only 80 and 443 permitted in MVP
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil || (port != 80 && port != 443) {
			return nil, ErrProhibitedPort
		}
	}

	// 5. Check Prohibited Hostnames
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") {
		return nil, ErrProhibitedHost
	}
	if strings.HasSuffix(lowerHost, ".local") ||
		strings.HasSuffix(lowerHost, ".internal") ||
		strings.HasSuffix(lowerHost, ".lan") ||
		strings.HasSuffix(lowerHost, ".home.arpa") {
		return nil, ErrProhibitedHost
	}

	// Block cloud metadata hostnames
	if lowerHost == "metadata.google.internal" ||
		lowerHost == "instance-data" ||
		lowerHost == "metadata" ||
		lowerHost == "169.254.169.254" {
		return nil, ErrProhibitedHost
	}

	// 6. Direct IP validation
	if ip := net.ParseIP(hostname); ip != nil {
		if IsProhibitedIP(ip) {
			return nil, fmt.Errorf("%w: %s", ErrProhibitedIP, ip.String())
		}
	} else {
		// Non-IP hostname: must be a valid DNS name with at least one dot
		if !strings.Contains(lowerHost, ".") {
			return nil, fmt.Errorf("%w: single-label hostnames are not allowed", ErrProhibitedHost)
		}
	}

	return parsedURL, nil
}

// IsProhibitedIP checks if an IP is in any private, loopback, link-local, multicast, or metadata range.
func IsProhibitedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Handle IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1)
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	if ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}

	// Check against parsed prohibited CIDR ranges
	for _, network := range parsedProhibitedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// DialContext creates a safe TCP dialer pinned to the pre-validated public IP address.
func (g *Guard) SafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("unsupported network %s", network)
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address format %s: %w", addr, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || (port != 80 && port != 443) {
		return nil, ErrProhibitedPort
	}

	var targetIP net.IP

	if parsedIP := net.ParseIP(host); parsedIP != nil {
		if IsProhibitedIP(parsedIP) {
			return nil, fmt.Errorf("%w: %s", ErrProhibitedIP, parsedIP.String())
		}
		targetIP = parsedIP
	} else {
		// Resolve DNS A and AAAA records at execution time
		ips, err := g.resolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("DNS resolution failed for %s: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("%w for %s", ErrNoIPAddresses, host)
		}

		// Inspect all returned addresses; if ANY address is prohibited, reject the target!
		for _, ip := range ips {
			if IsProhibitedIP(ip) {
				return nil, fmt.Errorf("%w: DNS returned prohibited IP %s for %s", ErrProhibitedIP, ip.String(), host)
			}
		}

		// Pin the first validated public IP to the outbound connection
		targetIP = ips[0]
	}

	// Dial the validated pinned IP
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	pinnedAddr := net.JoinHostPort(targetIP.String(), portStr)
	return dialer.DialContext(ctx, network, pinnedAddr)
}

// NewSafeTransport constructs an http.Transport configured with SSRF protection,
// DNS pinning, SNI preservation, and proxy bypass prevention.
func (g *Guard) NewSafeTransport(serverName string) *http.Transport {
	return &http.Transport{
		Proxy:                 nil, // Prevent proxy environment variables from bypassing SSRF guard
		DialContext:           g.SafeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: false,
		},
	}
}

// NewSafeClient creates a fully configured http.Client that enforces SSRF guard on initial request and every redirect.
func (g *Guard) NewSafeClient(timeout time.Duration, maxRedirects int) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxRedirects < 0 {
		maxRedirects = 5
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil, // Strictly no proxy
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return g.SafeDialContext(ctx, network, addr)
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return ErrTooManyRedirects
			}
			// Independently re-validate each redirect target
			if _, err := g.ValidateURL(req.URL.String()); err != nil {
				return fmt.Errorf("prohibited redirect to %s: %w", req.URL.String(), err)
			}
			return nil
		},
	}

	return client
}

