package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateSecureToken generates a cryptographically secure random token of the given byte length,
// encoded as standard URL-safe base64 without padding.
func GenerateSecureToken(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 32
	}
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken hashes a raw token with SHA-256 and an optional server secret pepper.
// This ensures that even if database records are exposed, tokens cannot be extracted.
func HashToken(rawToken, pepper string) string {
	h := hmac.New(sha256.New, []byte(pepper))
	h.Write([]byte(rawToken))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSessionCSRFToken generates a cryptographically signed CSRF token bound to a session ID.
// Format: <entropy>.<hmac-sha256(secret, sessionID + "." + entropy)>
func GenerateSessionCSRFToken(sessionID, secret string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate csrf entropy: %w", err)
	}
	entropy := base64.RawURLEncoding.EncodeToString(b)
	payload := sessionID + "." + entropy

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	sig := hex.EncodeToString(h.Sum(nil))

	return entropy + "." + sig, nil
}

// VerifySessionCSRFToken validates that a CSRF token has a valid signature bound to the specific session ID.
func VerifySessionCSRFToken(token, sessionID, secret string) bool {
	if token == "" || sessionID == "" || secret == "" {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	entropy := parts[0]
	sig := parts[1]

	payload := sessionID + "." + entropy
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) == 1
}


