package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/tenant"
)

// CSRFProtection enforces defense-in-depth CSRF defenses:
// 1. Session-bound signed token validation (HMAC-SHA256).
// 2. Double-submit cookie check (cookie matches X-CSRF-Token header).
// 3. Strict Origin and Referer validation against whitelist.
// 4. Sec-Fetch-Site header inspection (blocks cross-site mutations).
func CSRFProtection(cfg *config.Config, allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			// Check if request is authenticated with a user session
			u := tenant.GetUserContext(r.Context())
			if u == nil || u.SessionID == "" {
				// Unauthenticated request (e.g. signup, login) does not require CSRF token
				next.ServeHTTP(w, r)
				return
			}

			// 1. Defense-in-depth: Inspect Sec-Fetch-Site header
			fetchSite := r.Header.Get("Sec-Fetch-Site")
			if strings.EqualFold(fetchSite, "cross-site") {
				rejectCSRF(w, "CSRF_CROSS_SITE_BLOCKED", "Cross-site request blocked by Sec-Fetch-Site policy.")
				return
			}

			// 2. Strict Origin / Referer Validation
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !isAllowedOrigin(origin, allowedOrigins) {
					rejectCSRF(w, "CSRF_ORIGIN_INVALID", "Request origin is not permitted.")
					return
				}
			} else {
				// Fallback to Referer header if Origin is omitted
				referer := r.Header.Get("Referer")
				if referer != "" {
					refURL, err := url.Parse(referer)
					if err != nil || !isAllowedOrigin(refURL.Scheme+"://"+refURL.Host, allowedOrigins) {
						rejectCSRF(w, "CSRF_REFERER_INVALID", "Request referer is not permitted.")
						return
					}
				}
			}

			// 3. Cookie and Header presence check
			csrfCookie, err := r.Cookie(CSRFCookieName)
			if err != nil || csrfCookie.Value == "" {
				rejectCSRF(w, "CSRF_TOKEN_MISSING", "CSRF cookie is missing.")
				return
			}

			csrfHeader := r.Header.Get("X-CSRF-Token")
			if csrfHeader == "" {
				rejectCSRF(w, "CSRF_TOKEN_MISSING", "X-CSRF-Token header is missing.")
				return
			}

			// 4. Double-submit constant-time match
			if subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) != 1 {
				rejectCSRF(w, "CSRF_TOKEN_MISMATCH", "CSRF token does not match.")
				return
			}

			// 5. Session-bound HMAC verification
			// Verifies token signature was generated with server secret and bound to THIS exact session ID
			if !auth.VerifySessionCSRFToken(csrfHeader, u.SessionID, cfg.SessionSecret) {
				rejectCSRF(w, "CSRF_TOKEN_INVALID", "Invalid or forged CSRF token.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(origin string, allowed []string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(origin, strings.ToLower(strings.TrimSpace(a))) {
			return true
		}
	}
	return false
}

func rejectCSRF(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
