package checker

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type testRoundTripper struct {
	targetHost string
	inner      http.RoundTripper
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = t.targetHost
	if t.inner != nil {
		return t.inner.RoundTrip(cloned)
	}
	return http.DefaultTransport.RoundTrip(cloned)
}

type mockCheckerResolver struct {
	hosts map[string][]net.IP
	err   error
}

func (m *mockCheckerResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ips, ok := m.hosts[host]; ok {
		return ips, nil
	}
	return nil, fmt.Errorf("mock resolver: host %s not found", host)
}

func TestCheckerHTTPMethodsAndStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/200":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ClientPulse Status OK"))
		case "/404":
			w.WriteHeader(http.StatusNotFound)
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
		case "/head":
			w.WriteHeader(http.StatusOK)
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"up","metrics":{"latency":42}}`))
		case "/binary":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{0x00, 0xFF, 0x12, 0x34})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	tsURL, _ := url.Parse(ts.URL)
	testRT := &testRoundTripper{
		targetHost: tsURL.Host,
		inner:      ts.Client().Transport,
	}

	checker := NewChecker(nil, "worker-test")
	// Inject controlled test transport
	checker.SetTestTransport(testRT)

	// 1. Success 200 with Expected Text
	textMatch := "ClientPulse Status OK"
	res := checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/200",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		ExpectedText:      &textMatch,
	})
	if res.Result != "success" {
		errMsg := ""
		if res.ErrorMessage != nil {
			errMsg = *res.ErrorMessage
		}
		t.Fatalf("expected success, got %s (err: %s)", res.Result, errMsg)
	}
	if res.HTTPStatusCode == nil || *res.HTTPStatusCode != 200 {
		t.Fatalf("expected status 200, got %v", res.HTTPStatusCode)
	}

	// 2. Expected Text Mismatch
	textMismatch := "Missing Expected Keyword"
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/200",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		ExpectedText:      &textMismatch,
	})
	if res.Result != "failure" {
		t.Fatalf("expected failure on text mismatch, got %s", res.Result)
	}
	if res.ErrorCategory == nil || *res.ErrorCategory != "body_text_mismatch" {
		t.Fatalf("expected error category 'body_text_mismatch', got %v", res.ErrorCategory)
	}

	// 3. Expected Text on Non-Text binary content
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/binary",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		ExpectedText:      &textMatch,
	})
	if res.Result != "failure" {
		t.Fatalf("expected failure on binary content-type, got %s", res.Result)
	}
	if res.ErrorCategory == nil || *res.ErrorCategory != "invalid_content_type" {
		t.Fatalf("expected error category 'invalid_content_type', got %v", res.ErrorCategory)
	}

	// 4. JSON Pointer Assertion Match
	jsonPtr := "/metrics/latency"
	expectedVal := "42"
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/json",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		JSONPointer:       &jsonPtr,
		JSONExpectedValue: &expectedVal,
	})
	if res.Result != "success" {
		t.Fatalf("expected JSON assertion success, got %s (err: %v)", res.Result, res.ErrorMessage)
	}

	// 5. JSON Pointer Assertion Mismatch
	wrongVal := "999"
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/json",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		JSONPointer:       &jsonPtr,
		JSONExpectedValue: &wrongVal,
	})
	if res.Result != "failure" {
		t.Fatalf("expected JSON assertion failure, got %s", res.Result)
	}
	if res.ErrorCategory == nil || *res.ErrorCategory != "json_assertion_failed" {
		t.Fatalf("expected error category 'json_assertion_failed', got %v", res.ErrorCategory)
	}

	// 6. HTTP 404
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/404",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 399,
	})
	if res.Result != "failure" {
		t.Fatalf("expected failure on 404, got %s", res.Result)
	}
	if res.HTTPStatusCode == nil || *res.HTTPStatusCode != 404 {
		t.Fatalf("expected status 404, got %v", res.HTTPStatusCode)
	}

	// 7. HTTP 500
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/500",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 399,
	})
	if res.Result != "failure" {
		t.Fatalf("expected failure on 500, got %s", res.Result)
	}

	// 8. HEAD Method
	res = checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "https://example.com/head",
		Method:            "HEAD",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
	})
	if res.Result != "success" {
		t.Fatalf("expected HEAD success, got %s", res.Result)
	}
}

func TestJSONPointerEvaluation(t *testing.T) {
	docJSON := `{
		"status": "healthy",
		"version": "2.4.1",
		"data": {
			"users": [
				{"id": "usr-1", "active": true},
				{"id": "usr-2", "active": false}
			],
			"nested/path": "slash-value"
		}
	}`

	var doc interface{}
	if err := json.Unmarshal([]byte(docJSON), &doc); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}

	tests := []struct {
		pointer  string
		expected string
		wantErr  bool
	}{
		{"/status", "healthy", false},
		{"/version", "2.4.1", false},
		{"/data/users/0/id", "usr-1", false},
		{"/data/users/1/active", "false", false},
		{"/data/nested~1path", "slash-value", false},
		{"/nonexistent", "", true},
		{"/data/users/5/id", "", true},
		{"invalid-no-slash", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.pointer, func(t *testing.T) {
			val, err := evaluateJSONPointer(doc, tc.pointer)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for pointer %s, got nil", tc.pointer)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for pointer %s: %v", tc.pointer, err)
				}
				actual := fmt.Sprintf("%v", val)
				if actual != tc.expected {
					t.Fatalf("expected %q, got %q", tc.expected, actual)
				}
			}
		})
	}
}

func TestSSLMetadataExtractionAndDaysRemaining(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	notBefore := time.Now().Add(-24 * time.Hour)
	notAfter := time.Now().Add(15 * 24 * time.Hour) // 15 days remaining (triggers < 30 days warning)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "clientpulse-test.io",
			Organization: []string{"ClientPulse Test Org"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse cert: %v", err)
	}

	daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)
	if daysRemaining < 14 || daysRemaining > 15 {
		t.Fatalf("expected 14-15 days remaining, got %d", daysRemaining)
	}

	isValid := time.Now().UTC().After(cert.NotBefore) && time.Now().UTC().Before(cert.NotAfter)
	if !isValid {
		t.Fatal("expected certificate to be valid")
	}

	// Verify thresholds (30, 14, 7 days)
	if daysRemaining > 30 {
		t.Fatal("expected certificate to trigger <30 days warning threshold")
	}
}

func TestSSRFBlockedDuringExecution(t *testing.T) {
	checker := NewChecker(nil, "worker-test")

	res := checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "http://127.0.0.1/sensitive",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 399,
	})

	if res.Result != "blocked" {
		t.Fatalf("expected result 'blocked', got %s", res.Result)
	}
	if res.ErrorCategory == nil || *res.ErrorCategory != "ssrf_blocked" {
		t.Fatalf("expected error category 'ssrf_blocked', got %v", res.ErrorCategory)
	}
}

func TestDNSFailureCategorization(t *testing.T) {
	checker := NewChecker(nil, "worker-test")

	res := checker.Execute(context.Background(), CheckConfig{
		TargetURL:         "http://this-non-existent-domain-123456789.com",
		Method:            "GET",
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 399,
	})

	if res.Result != "failure" {
		t.Fatalf("expected result 'failure', got %s", res.Result)
	}
	if res.ErrorCategory == nil || *res.ErrorCategory != "dns_error" {
		t.Fatalf("expected error category 'dns_error', got %v", res.ErrorCategory)
	}
}
