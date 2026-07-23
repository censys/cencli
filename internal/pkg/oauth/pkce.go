package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateVerifier returns a new PKCE code verifier (RFC 7636).
// 32 random bytes base64url-encoded yields a 43-character verifier,
// which satisfies the 43-128 character requirement.
func GenerateVerifier() (string, error) {
	b, err := randomBytes(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ChallengeS256 derives the S256 code challenge for a verifier (RFC 7636).
func ChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// GenerateState returns a new opaque state parameter for the authorization request.
func GenerateState() (string, error) {
	b, err := randomBytes(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
