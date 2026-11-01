package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/ssrf"
)

type CheckConfig struct {
	TargetURL         string
	Method            string
	TimeoutSeconds    int
	ExpectedStatusMin int
	ExpectedStatusMax int
	FollowRedirects   bool
	MaxRedirects      int
	ExpectedText      *string
	JSONPointer       *string
	JSONExpectedValue *string
	ScheduledAt       time.Time
}

type CheckExecutionResult struct {
	StartedAt        time.Time
	CompletedAt      time.Time
	LatencyMs        int64
	Result           string // "success", "failure", "timeout", "blocked", "error"
	HTTPStatusCode   *int
	ErrorCategory    *string
	ErrorMessage     *string
	TLSVersion       *string
	TLSCipher        *string
	TLSCertIssuer    *string
	TLSCertSubject   *string
	TLSCertNotBefore *time.Time
	TLSCertNotAfter  *time.Time
	TLSDaysRemaining *int
	TLSIsValid       *bool
	ResolvedIP       *string
	DNSDurationMs    int64
}

type Checker struct {
	guard         *ssrf.Guard
	workerID      string
	testTransport http.RoundTripper
}

func NewChecker(guard *ssrf.Guard, workerID string) *Checker {
	if guard == nil {
		guard = ssrf.NewGuard(nil)
	}
	return &Checker{
		guard:    guard,
		workerID: workerID,
	}
}

// SetTestTransport allows injecting a controlled transport for unit tests
// while preserving production security checks.
func (c *Checker) SetTestTransport(rt http.RoundTripper) {
	c.testTransport = rt
}

// Execute performs DNS resolution, SSL inspection, and HTTP request safely with SSRF protection.
func (c *Checker) Execute(ctx context.Context, cfg CheckConfig) *CheckExecutionResult {
	startedAt := time.Now().UTC()
	result := &CheckExecutionResult{
		StartedAt: startedAt,
	}

	defer func() {
		result.CompletedAt = time.Now().UTC()
		result.LatencyMs = result.CompletedAt.Sub(startedAt).Milliseconds()
	}()

	// 1. SSRF URL validation
	parsedURL, err := c.guard.ValidateURL(cfg.TargetURL)
	if err != nil {
		result.Result = "blocked"
		cat := "ssrf_blocked"
		msg := err.Error()
		result.ErrorCategory = &cat
		result.ErrorMessage = &msg
		return result
	}

	// 2. DNS Resolution Step
	hostname := parsedURL.Hostname()
	dnsStart := time.Now()
	var resolvedIPs []net.IP

	if directIP := net.ParseIP(hostname); directIP != nil {
		if ssrf.IsProhibitedIP(directIP) {
			result.Result = "blocked"
			cat := "ssrf_prohibited_ip"
			msg := fmt.Sprintf("prohibited IP %s", directIP.String())
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}
		resolvedIPs = []net.IP{directIP}
	} else {
		dnsCtx, dnsCancel := context.WithTimeout(ctx, 5*time.Second)
		ips, err := c.guard.Resolver().LookupIP(dnsCtx, "ip", hostname)
		dnsCancel()
		result.DNSDurationMs = time.Since(dnsStart).Milliseconds()

		if err != nil {
			result.Result = "failure"
			cat := "dns_error"
			msg := fmt.Sprintf("DNS resolution failed: %v", err)
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}
		if len(ips) == 0 {
			result.Result = "failure"
			cat := "dns_no_records"
			msg := "no A or AAAA records found"
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}

		// Verify every returned IP against SSRF guard
		for _, ip := range ips {
			if ssrf.IsProhibitedIP(ip) {
				result.Result = "blocked"
				cat := "ssrf_dns_prohibited_ip"
				msg := fmt.Sprintf("DNS returned prohibited IP %s", ip.String())
				result.ErrorCategory = &cat
				result.ErrorMessage = &msg
				return result
			}
		}
		resolvedIPs = ips
	}

	if len(resolvedIPs) > 0 {
		ipStr := resolvedIPs[0].String()
		result.ResolvedIP = &ipStr
	}

	// 3. Configure HTTP Client with SSRF Guard and timeouts
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := c.guard.NewSafeClient(timeout, cfg.MaxRedirects)
	if c.testTransport != nil {
		client.Transport = c.testTransport
	}
	if !cfg.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method != "GET" && method != "HEAD" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), nil)
	if err != nil {
		result.Result = "error"
		cat := "request_build_error"
		msg := err.Error()
		result.ErrorCategory = &cat
		result.ErrorMessage = &msg
		return result
	}

	req.Header.Set("User-Agent", "ClientPulse-Monitor/1.0 (+https://clientpulse.io)")
	req.Header.Set("Accept", "*/*")

	// 4. Outbound HTTP Execution
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			result.Result = "timeout"
			cat := "timeout"
			msg := fmt.Sprintf("request timed out after %s", timeout)
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}
		if errors.Is(err, ssrf.ErrProhibitedIP) || errors.Is(err, ssrf.ErrProhibitedHost) || strings.Contains(err.Error(), "prohibited") {
			result.Result = "blocked"
			cat := "ssrf_blocked"
			msg := err.Error()
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}

		result.Result = "failure"
		cat := "connection_error"
		msg := err.Error()
		result.ErrorCategory = &cat
		result.ErrorMessage = &msg
		return result
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result.HTTPStatusCode = &statusCode

	// 5. SSL / TLS Metadata Extraction (for HTTPS)
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		tlsVer := tlsVersionString(resp.TLS.Version)
		tlsCiph := tls.CipherSuiteName(resp.TLS.CipherSuite)
		issuer := cert.Issuer.CommonName
		if issuer == "" && len(cert.Issuer.Organization) > 0 {
			issuer = cert.Issuer.Organization[0]
		}
		subject := cert.Subject.CommonName
		notBefore := cert.NotBefore.UTC()
		notAfter := cert.NotAfter.UTC()
		daysRemaining := int(time.Until(notAfter).Hours() / 24)
		isValid := time.Now().UTC().After(notBefore) && time.Now().UTC().Before(notAfter)

		result.TLSVersion = &tlsVer
		result.TLSCipher = &tlsCiph
		result.TLSCertIssuer = &issuer
		result.TLSCertSubject = &subject
		result.TLSCertNotBefore = &notBefore
		result.TLSCertNotAfter = &notAfter
		result.TLSDaysRemaining = &daysRemaining
		result.TLSIsValid = &isValid

		if !isValid {
			result.Result = "failure"
			cat := "ssl_certificate_invalid"
			msg := "SSL certificate is expired or not yet valid"
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}
	}

	// 6. Expected HTTP Status Validation
	minStatus := cfg.ExpectedStatusMin
	if minStatus <= 0 {
		minStatus = 200
	}
	maxStatus := cfg.ExpectedStatusMax
	if maxStatus <= 0 {
		maxStatus = 399
	}

	if statusCode < minStatus || statusCode > maxStatus {
		result.Result = "failure"
		cat := "http_status_error"
		msg := fmt.Sprintf("HTTP status %d outside expected range [%d-%d]", statusCode, minStatus, maxStatus)
		result.ErrorCategory = &cat
		result.ErrorMessage = &msg
		return result
	}

	// 7. Response Body Inspection (Max 256 KB)
	if method != "HEAD" && (cfg.ExpectedText != nil || cfg.JSONPointer != nil) {
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		const maxBodyBytes = 256 * 1024 // 256 KB
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		if err != nil {
			result.Result = "failure"
			cat := "body_read_error"
			msg := fmt.Sprintf("failed to read response body: %v", err)
			result.ErrorCategory = &cat
			result.ErrorMessage = &msg
			return result
		}

		// Expected Text matching
		if cfg.ExpectedText != nil && *cfg.ExpectedText != "" {
			isText := strings.Contains(contentType, "text") ||
				strings.Contains(contentType, "json") ||
				strings.Contains(contentType, "xml") ||
				strings.Contains(contentType, "html") ||
				strings.Contains(contentType, "javascript")
			if !isText && len(contentType) > 0 {
				result.Result = "failure"
				cat := "invalid_content_type"
				msg := fmt.Sprintf("non-text content type %q cannot match expected text", contentType)
				result.ErrorCategory = &cat
				result.ErrorMessage = &msg
				return result
			}

			if !strings.Contains(string(bodyBytes), *cfg.ExpectedText) {
				result.Result = "failure"
				cat := "body_text_mismatch"
				msg := fmt.Sprintf("expected text %q was not found in response body", *cfg.ExpectedText)
				result.ErrorCategory = &cat
				result.ErrorMessage = &msg
				return result
			}
		}

		// JSON Pointer Assertion
		if cfg.JSONPointer != nil && *cfg.JSONPointer != "" {
			var jsonData interface{}
			if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
				result.Result = "failure"
				cat := "json_parse_error"
				msg := fmt.Sprintf("failed to parse response as JSON: %v", err)
				result.ErrorCategory = &cat
				result.ErrorMessage = &msg
				return result
			}

			val, err := evaluateJSONPointer(jsonData, *cfg.JSONPointer)
			if err != nil {
				result.Result = "failure"
				cat := "json_pointer_not_found"
				msg := fmt.Sprintf("JSON pointer %q evaluation error: %v", *cfg.JSONPointer, err)
				result.ErrorCategory = &cat
				result.ErrorMessage = &msg
				return result
			}

			if cfg.JSONExpectedValue != nil && *cfg.JSONExpectedValue != "" {
				expectedVal := *cfg.JSONExpectedValue
				actualValStr := fmt.Sprintf("%v", val)
				if actualValStr != expectedVal {
					result.Result = "failure"
					cat := "json_assertion_failed"
					msg := fmt.Sprintf("JSON pointer %q value %q did not match expected %q", *cfg.JSONPointer, actualValStr, expectedVal)
					result.ErrorCategory = &cat
					result.ErrorMessage = &msg
					return result
				}
			}
		}
	}

	result.Result = "success"
	return result
}

func tlsVersionString(ver uint16) string {
	switch ver {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("TLS 0x%04x", ver)
	}
}

// evaluateJSONPointer implements RFC 6901 JSON pointer evaluation.
func evaluateJSONPointer(doc interface{}, pointer string) (interface{}, error) {
	if pointer == "" {
		return doc, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, errors.New("JSON pointer must start with '/'")
	}

	tokens := strings.Split(pointer[1:], "/")
	current := doc

	for _, token := range tokens {
		// RFC 6901 unescaping
		token = strings.ReplaceAll(token, "~1", "/")
		token = strings.ReplaceAll(token, "~0", "~")

		switch val := current.(type) {
		case map[string]interface{}:
			sub, ok := val[token]
			if !ok {
				return nil, fmt.Errorf("property %q not found in object", token)
			}
			current = sub
		case []interface{}:
			idx, err := strconv.Atoi(token)
			if err != nil || idx < 0 || idx >= len(val) {
				return nil, fmt.Errorf("array index %q out of bounds", token)
			}
			current = val[idx]
		default:
			return nil, fmt.Errorf("cannot descend into non-container type %T", current)
		}
	}

	return current, nil
}
