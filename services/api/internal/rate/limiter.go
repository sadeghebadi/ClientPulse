package rate

import (
	"sync"
	"time"
)

// Limiter implements an in-memory thread-safe rate limiter.
type Limiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string][]time.Time
}

// NewLimiter creates a new rate limiter with the specified limit and time window.
func NewLimiter(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		window:  window,
		limit:   limit,
		entries: make(map[string][]time.Time),
	}

	// Periodically purge stale entries to prevent memory leaks
	go l.cleanupLoop(window * 2)

	return l
}

// Allow checks if the given key is allowed to perform an action.
// If allowed, it records the request timestamp and returns true.
// If rate limit exceeded, it returns false.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	timestamps := l.entries[key]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.limit {
		l.entries[key] = valid
		return false
	}

	valid = append(valid, now)
	l.entries[key] = valid
	return true
}

// Reset clears recorded entries for a given key (e.g. after successful login).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *Limiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-l.window)
		for key, timestamps := range l.entries {
			var valid []time.Time
			for _, t := range timestamps {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(l.entries, key)
			} else {
				l.entries[key] = valid
			}
		}
		l.mu.Unlock()
	}
}

