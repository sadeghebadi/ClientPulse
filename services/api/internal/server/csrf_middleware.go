package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// CSRFProtection validates the Double-Submit CSRF cookie against X-CSRF-Token header
// for state-mutating requests authenticated via cookies.
func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods do not mutate state
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// If authenticated exclusively via Bearer token in header, CSRF is not applicable
		// because cross-origin browser requests cannot inject arbitrary headers without CORS preflight.
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		// Check if request carries session cookie
		sessionCookie, err := r.Cookie(SessionCookieName)
		if err != nil || sessionCookie.Value == "" {
			// Unauthenticated request (e.g. signup, login) does not require CSRF
			next.ServeHTTP(w, r)
			return
		}

		// Cookie-authenticated mutating request: enforce CSRF token match
		csrfCookie, err := r.Cookie(CSRFCookieName)
		if err != nil || csrfCookie.Value == "" {
			rejectCSRF(w)
			return
		}

		csrfHeader := r.Header.Get("X-CSRF-Token")
		if csrfHeader == "" {
			rejectCSRF(w)
			return
		}

		if subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) != 1 {
			rejectCSRF(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func rejectCSRF(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "CSRF_VALIDATION_FAILED",
			"message": "Invalid or missing CSRF token.",
		},
	})
}

