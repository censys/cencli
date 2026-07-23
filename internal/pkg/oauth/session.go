package oauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// expiryLeeway is subtracted from the access token expiry when deciding
// whether a refresh is needed, so tokens are refreshed slightly early
// rather than rejected by the API mid-flight.
const expiryLeeway = 60 * time.Second

// Session is the persisted result of an OAuth2 login. It is stored as JSON
// in the credential store and refreshed transparently when the access token
// expires.
type Session struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Issuer       string    `json:"issuer,omitempty"`
	Subject      string    `json:"subject,omitempty"`
	Email        string    `json:"email,omitempty"`
}

// ParseSession decodes a Session previously serialized with Marshal.
func ParseSession(value string) (*Session, error) {
	var s Session
	if err := json.Unmarshal([]byte(value), &s); err != nil {
		return nil, fmt.Errorf("failed to parse stored oauth session: %w", err)
	}
	if s.AccessToken == "" {
		return nil, fmt.Errorf("stored oauth session has no access token")
	}
	return &s, nil
}

// Marshal serializes the session for storage.
func (s *Session) Marshal() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("failed to serialize oauth session: %w", err)
	}
	return string(b), nil
}

// Expired reports whether the access token is expired (or will be within
// the refresh leeway) at the given time.
func (s *Session) Expired(now time.Time) bool {
	if s.ExpiresAt.IsZero() {
		return false
	}
	return !now.Before(s.ExpiresAt.Add(-expiryLeeway))
}

// Account returns the best human-readable identifier for the logged-in user.
func (s *Session) Account() string {
	if s.Email != "" {
		return s.Email
	}
	return s.Subject
}

// populateClaims extracts identity claims (sub, email) from the ID token
// payload. The token is received directly from the issuer over TLS during
// login/refresh, so the signature is not re-verified here; the claims are
// used for display purposes only.
func (s *Session) populateClaims() {
	if s.IDToken == "" {
		return
	}
	parts := strings.Split(s.IDToken, ".")
	if len(parts) != 3 {
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return
	}
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return
	}
	s.Subject = claims.Subject
	s.Email = claims.Email
}
