package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"clientpulse/services/api/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture the HTTP status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestID middleware ensures every request has a unique correlation ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			var b [16]byte
			_, _ = rand.Read(b[:])
			reqID = hex.EncodeToString(b[:])
		}

		w.Header().Set("X-Request-ID", reqID)
		ctx := logger.WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SanitizeRequestPath redacts secret tokens from request paths (heartbeat pings, public report share tokens).
func SanitizeRequestPath(path string) string {
	if strings.HasPrefix(path, "/api/v1/heartbeat-pings/") {
		return "/api/v1/heartbeat-pings/[REDACTED]"
	}
	if strings.HasPrefix(path, "/public/reports/") {
		parts := strings.Split(strings.TrimPrefix(path, "/public/reports/"), "/")
		if len(parts) > 1 && parts[1] != "" {
			return "/public/reports/[REDACTED]/" + strings.Join(parts[1:], "/")
		}
		return "/public/reports/[REDACTED]"
	}
	return path
}

// RequestLogger logs incoming HTTP requests with structured metadata and token redaction.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			reqID := logger.GetRequestID(r.Context())

			log.InfoContext(r.Context(), "http_request",
				"method", r.Method,
				"path", SanitizeRequestPath(r.URL.Path),
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", reqID,
			)
		})
	}
}

// Recovery handles panics cleanly by returning a standardized JSON error response.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					reqID := logger.GetRequestID(r.Context())
					stack := string(debug.Stack())

					log.ErrorContext(r.Context(), "unhandled_panic",
						"error", fmt.Sprintf("%v", rec),
						"stack", stack,
						"request_id", reqID,
					)

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"code":       "INTERNAL_ERROR",
							"message":    "An unexpected internal error occurred.",
							"request_id": reqID,
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders applies protective HTTP headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// CORS handles Cross-Origin Resource Sharing based on allowed origins.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				for _, o := range allowedOrigins {
					if o == "*" || strings.EqualFold(o, origin) {
						allowed = true
						break
					}
				}

				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-CSRF-Token, X-Organization-ID")
					w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

