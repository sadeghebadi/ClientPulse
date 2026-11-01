package rate

import (
	"testing"
	"time"
)

func TestLimiterAllowAndLimit(t *testing.T) {
	limiter := NewLimiter(3, 100*time.Millisecond)

	key := "test-ip-127.0.0.1"

	if !limiter.Allow(key) {
		t.Fatal("1st request should be allowed")
	}
	if !limiter.Allow(key) {
		t.Fatal("2nd request should be allowed")
	}
	if !limiter.Allow(key) {
		t.Fatal("3rd request should be allowed")
	}

	// 4th request must be rejected
	if limiter.Allow(key) {
		t.Fatal("4th request should exceed rate limit")
	}

	// Reset should allow again
	limiter.Reset(key)
	if !limiter.Allow(key) {
		t.Fatal("request after reset should be allowed")
	}
}

