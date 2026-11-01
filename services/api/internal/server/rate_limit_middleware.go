package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"clientpulse/services/api/internal/rate"
)

// RateLimit creates middleware to rate limit requests based on client IP.
func RateLimit(limiter *rate.Limiter, endpointName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			key := endpointName + ":" + ip

			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "Too many requests. Please wait a moment before trying again.",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Trusted private subnets for Docker / local reverse proxies (Caddy, Traefik, Docker bridge)
var trustedNetworks = []*net.IPNet{
	mustParseCIDR("127.0.0.0/8"),
	mustParseCIDR("::1/128"),
	mustParseCIDR("10.0.0.0/8"),
	mustParseCIDR("172.16.0.0/12"),
	mustParseCIDR("192.168.0.0/16"),
}

func mustParseCIDR(s string) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipnet
}

func isTrustedRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	for _, network := range trustedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// getClientIP securely resolves client IP:
// Only trusts X-Forwarded-For if the immediate connection (r.RemoteAddr) originates from a trusted proxy.
// If RemoteAddr is not a trusted proxy, X-Forwarded-For is ignored to prevent client IP spoofing.
func getClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if isTrustedRemoteAddr(remoteHost) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			clientCandidate := strings.TrimSpace(parts[0])
			if net.ParseIP(clientCandidate) != nil {
				return clientCandidate
			}
		}
		xRealIP := r.Header.Get("X-Real-IP")
		if xRealIP != "" && net.ParseIP(strings.TrimSpace(xRealIP)) != nil {
			return strings.TrimSpace(xRealIP)
		}
	}

	return remoteHost
}
