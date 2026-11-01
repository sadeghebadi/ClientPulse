package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
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

