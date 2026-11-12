package refreshtoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const rawBytes = 32

// NewOpaque generates a URL-safe opaque refresh token (plaintext for the client).
func NewOpaque() (string, error) {
	b := make([]byte, rawBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns SHA-256 hex of the plaintext token for DB storage.
func Hash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Fingerprint returns a short prefix of Hash for debug logs (never log raw tokens).
func Fingerprint(plaintext string) string {
	h := Hash(plaintext)
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
