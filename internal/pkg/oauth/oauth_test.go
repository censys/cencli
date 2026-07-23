package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientAt builds a Client whose authorization-server endpoints target baseURL.
// The real issuer/audience are hardcoded constants (not part of Config); this
// same-package helper is the test-only seam to point the client at a local
// httptest server.
func clientAt(baseURL string, doer HTTPDoer, cfg Config) *Client {
	c := NewClient(cfg, doer)
	if baseURL != "" {
		c.issuer = strings.TrimRight(baseURL, "/")
	}
	return c
}

func TestGenerateVerifier(t *testing.T) {
	v1, err := GenerateVerifier()
	require.NoError(t, err)
	v2, err := GenerateVerifier()
	require.NoError(t, err)

	assert.Len(t, v1, 43) // 32 bytes base64url without padding
	assert.NotEqual(t, v1, v2)
	assert.NotContains(t, v1, "=")
	assert.NotContains(t, v1, "+")
	assert.NotContains(t, v1, "/")
}

func TestChallengeS256(t *testing.T) {
	// Known vector from RFC 7636 appendix B.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", ChallengeS256(verifier))
}

func TestGenerateState(t *testing.T) {
	s1, err := GenerateState()
	require.NoError(t, err)
	s2, err := GenerateState()
	require.NoError(t, err)
	assert.Len(t, s1, 32)
	assert.NotEqual(t, s1, s2)
}

func TestAuthCodeURL(t *testing.T) {
	c := NewClient(Config{}, nil)

	rawURL := c.AuthCodeURL("test-state", "test-challenge", "http://127.0.0.1:5555/callback")
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	issuer, err := url.Parse(DefaultIssuer)
	require.NoError(t, err)

	assert.Equal(t, issuer.Scheme, parsed.Scheme)
	assert.Equal(t, issuer.Host, parsed.Host)
	assert.Equal(t, "/oauth2/auth", parsed.Path)

	q := parsed.Query()
	assert.Equal(t, ClientID, q.Get("client_id"))
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "http://127.0.0.1:5555/callback", q.Get("redirect_uri"))
	assert.Equal(t, Scopes, q.Get("scope"))
	assert.Equal(t, "test-state", q.Get("state"))
	assert.Equal(t, "test-challenge", q.Get("code_challenge"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.Equal(t, DefaultAudience, q.Get("audience"))
}

// newTokenServer returns an httptest server acting as the authorization
// server's token/revocation endpoints, plus a pointer to the last form it saw.
func newTokenServer(t *testing.T, tokenStatus int, tokenBody any) (*httptest.Server, *url.Values) {
	t.Helper()
	var lastForm url.Values
	mux := http.NewServeMux()
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		lastForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(tokenStatus)
		require.NoError(t, json.NewEncoder(w).Encode(tokenBody))
	})
	mux.HandleFunc(revokePath, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		lastForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &lastForm
}

func TestExchange(t *testing.T) {
	srv, lastForm := newTokenServer(t, http.StatusOK, map[string]any{
		"access_token":  "ory_at_abc",
		"token_type":    "bearer",
		"refresh_token": "ory_rt_def",
		"scope":         "openid offline_access censys.api",
		"expires_in":    3600,
	})

	c := clientAt(srv.URL, srv.Client(), Config{})
	sess, err := c.Exchange(context.Background(), "auth-code", "verifier123", "http://127.0.0.1:5555/callback")
	require.NoError(t, err)

	assert.Equal(t, "authorization_code", lastForm.Get("grant_type"))
	assert.Equal(t, "auth-code", lastForm.Get("code"))
	assert.Equal(t, "verifier123", lastForm.Get("code_verifier"))
	assert.Equal(t, "http://127.0.0.1:5555/callback", lastForm.Get("redirect_uri"))
	assert.Equal(t, ClientID, lastForm.Get("client_id"))

	assert.Equal(t, "ory_at_abc", sess.AccessToken)
	assert.Equal(t, "ory_rt_def", sess.RefreshToken)
	assert.Equal(t, srv.URL, sess.Issuer)
	assert.False(t, sess.Expired(time.Now()))
	assert.True(t, sess.Expired(time.Now().Add(2*time.Hour)))
}

func TestExchangeParsesIDTokenClaims(t *testing.T) {
	// Payload: {"sub":"user-123","email":"user@censys.com"}
	idToken := "eyJhbGciOiJSUzI1NiJ9." +
		"eyJzdWIiOiJ1c2VyLTEyMyIsImVtYWlsIjoidXNlckBjZW5zeXMuY29tIn0." +
		"c2ln"
	srv, _ := newTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "ory_at_abc",
		"id_token":     idToken,
		"expires_in":   3600,
	})

	c := clientAt(srv.URL, srv.Client(), Config{})
	sess, err := c.Exchange(context.Background(), "code", "verifier", "http://127.0.0.1:5555/callback")
	require.NoError(t, err)
	assert.Equal(t, "user@censys.com", sess.Email)
	assert.Equal(t, "user-123", sess.Subject)
	assert.Equal(t, "user@censys.com", sess.Account())
}

func TestRefresh(t *testing.T) {
	srv, lastForm := newTokenServer(t, http.StatusOK, map[string]any{
		"access_token":  "ory_at_new",
		"refresh_token": "ory_rt_new",
		"expires_in":    3600,
	})

	c := clientAt(srv.URL, srv.Client(), Config{})
	sess, err := c.Refresh(context.Background(), "ory_rt_old")
	require.NoError(t, err)

	assert.Equal(t, "refresh_token", lastForm.Get("grant_type"))
	assert.Equal(t, "ory_rt_old", lastForm.Get("refresh_token"))
	assert.Equal(t, ClientID, lastForm.Get("client_id"))
	assert.Equal(t, "ory_at_new", sess.AccessToken)
	assert.Equal(t, "ory_rt_new", sess.RefreshToken)
}

func TestTokenEndpointError(t *testing.T) {
	srv, _ := newTokenServer(t, http.StatusBadRequest, map[string]any{
		"error":             "invalid_grant",
		"error_description": "The provided authorization grant is invalid",
	})

	c := clientAt(srv.URL, srv.Client(), Config{})
	_, err := c.Refresh(context.Background(), "ory_rt_revoked")
	require.Error(t, err)

	var serverErr *ServerError
	require.ErrorAs(t, err, &serverErr)
	assert.Equal(t, http.StatusBadRequest, serverErr.StatusCode)
	assert.Equal(t, "invalid_grant", serverErr.Code)
	assert.Contains(t, err.Error(), "invalid_grant")
}

func TestRevoke(t *testing.T) {
	srv, lastForm := newTokenServer(t, http.StatusOK, nil)

	c := clientAt(srv.URL, srv.Client(), Config{})
	require.NoError(t, c.Revoke(context.Background(), "ory_rt_abc"))
	assert.Equal(t, "ory_rt_abc", lastForm.Get("token"))
	assert.Equal(t, ClientID, lastForm.Get("client_id"))
}

func TestSessionRoundTrip(t *testing.T) {
	sess := &Session{
		AccessToken:  "ory_at_abc",
		RefreshToken: "ory_rt_def",
		ExpiresAt:    time.Now().Add(time.Hour).UTC(),
		Email:        "user@censys.com",
	}
	value, err := sess.Marshal()
	require.NoError(t, err)

	parsed, err := ParseSession(value)
	require.NoError(t, err)
	assert.Equal(t, sess.AccessToken, parsed.AccessToken)
	assert.Equal(t, sess.RefreshToken, parsed.RefreshToken)
	assert.Equal(t, sess.Email, parsed.Email)
	assert.WithinDuration(t, sess.ExpiresAt, parsed.ExpiresAt, time.Second)
}

func TestParseSessionInvalid(t *testing.T) {
	_, err := ParseSession("not-json")
	assert.Error(t, err)

	_, err = ParseSession(`{"refresh_token":"only"}`)
	assert.Error(t, err)
}

func TestSessionExpired(t *testing.T) {
	now := time.Now()

	// no expiry recorded -> never considered expired
	assert.False(t, (&Session{AccessToken: "a"}).Expired(now))

	sess := &Session{AccessToken: "a", ExpiresAt: now.Add(10 * time.Minute)}
	assert.False(t, sess.Expired(now))
	// within the leeway window counts as expired
	assert.True(t, sess.Expired(now.Add(10*time.Minute-30*time.Second)))
	assert.True(t, sess.Expired(now.Add(time.Hour)))
}

func TestLogin(t *testing.T) {
	port := 0 // random free port so tests don't collide with a real login
	srv, lastForm := newTokenServer(t, http.StatusOK, map[string]any{
		"access_token":  "ory_at_abc",
		"refresh_token": "ory_rt_def",
		"expires_in":    3600,
	})

	c := clientAt(srv.URL, srv.Client(), Config{
		RedirectPort: &port,
		LoginTimeout: 10 * time.Second,
	})

	// The fake "browser": parse the authorize URL and immediately hit the
	// loopback callback with a code and the expected state.
	sess, err := c.Login(context.Background(), func(authorizeURL string) error {
		parsed, parseErr := url.Parse(authorizeURL)
		require.NoError(t, parseErr)
		q := parsed.Query()
		assert.Equal(t, "S256", q.Get("code_challenge_method"))
		assert.NotEmpty(t, q.Get("code_challenge"))

		callback := fmt.Sprintf("%s?state=%s&code=%s",
			q.Get("redirect_uri"), url.QueryEscape(q.Get("state")), "test-code")
		go func() {
			resp, getErr := http.Get(callback)
			if getErr == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, "ory_at_abc", sess.AccessToken)
	assert.Equal(t, "test-code", lastForm.Get("code"))
	assert.NotEmpty(t, lastForm.Get("code_verifier"))
}

func TestLoginAuthorizationError(t *testing.T) {
	port := 0
	srv, _ := newTokenServer(t, http.StatusOK, nil)

	c := clientAt(srv.URL, srv.Client(), Config{
		RedirectPort: &port,
		LoginTimeout: 10 * time.Second,
	})

	_, err := c.Login(context.Background(), func(authorizeURL string) error {
		parsed, parseErr := url.Parse(authorizeURL)
		require.NoError(t, parseErr)
		q := parsed.Query()
		callback := fmt.Sprintf("%s?state=%s&error=access_denied&error_description=user+denied",
			q.Get("redirect_uri"), url.QueryEscape(q.Get("state")))
		go func() {
			resp, getErr := http.Get(callback)
			if getErr == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")
}

func TestLoginRejectsStateMismatch(t *testing.T) {
	port := 0
	srv, _ := newTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "ory_at_abc",
		"expires_in":   3600,
	})

	c := clientAt(srv.URL, srv.Client(), Config{
		RedirectPort: &port,
		LoginTimeout: 10 * time.Second,
	})

	sess, err := c.Login(context.Background(), func(authorizeURL string) error {
		parsed, parseErr := url.Parse(authorizeURL)
		require.NoError(t, parseErr)
		q := parsed.Query()
		redirectURI := q.Get("redirect_uri")
		state := q.Get("state")
		go func() {
			// A forged/stale request with the wrong state must be rejected...
			resp, getErr := http.Get(redirectURI + "?state=wrong&code=evil-code")
			if getErr == nil {
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
				_ = resp.Body.Close()
			}
			// ...while the legitimate redirect still completes the flow.
			resp, getErr = http.Get(fmt.Sprintf("%s?state=%s&code=good-code", redirectURI, url.QueryEscape(state)))
			if getErr == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ory_at_abc", sess.AccessToken)
}

func TestLoginContextCancelled(t *testing.T) {
	port := 0
	c := clientAt("http://127.0.0.1:1", nil, Config{ // issuer never reached
		RedirectPort: &port,
		LoginTimeout: time.Minute,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := c.Login(ctx, func(string) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoginTimeout(t *testing.T) {
	port := 0
	c := clientAt("http://127.0.0.1:1", nil, Config{
		RedirectPort: &port,
		LoginTimeout: 100 * time.Millisecond,
	})

	_, err := c.Login(context.Background(), func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestLoginPortInUse(t *testing.T) {
	// Occupy a port, then ask Login to bind it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port

	c := clientAt("http://127.0.0.1:1", nil, Config{RedirectPort: &port})
	_, err = c.Login(context.Background(), func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to listen")
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(Config{}, nil)
	assert.Equal(t, ClientID, c.cfg.ClientID)
	assert.Equal(t, Scopes, c.cfg.Scopes)
	assert.Equal(t, RedirectPort, *c.cfg.RedirectPort)
	assert.Equal(t, 5*time.Minute, c.cfg.LoginTimeout)
	assert.Equal(t, DefaultIssuer, c.issuer)
	assert.Equal(t, DefaultAudience, c.audience)
	assert.NotNil(t, c.http)
}
