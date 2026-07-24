// Package oauth runs the OAuth2 authorization-code + PKCE flow used by
// `censys auth login`. The standard protocol mechanics (authorization URL,
// PKCE, token exchange, and refresh) are handled by golang.org/x/oauth2; this
// package adds the loopback-browser login flow, session persistence, and token
// revocation (RFC 7009), none of which x/oauth2 covers.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	// ClientID is the OAuth2 client provisioned for cencli. It is a public
	// (PKCE) client, so there is no client secret.
	ClientID = "censys-cencli"

	// DefaultIssuer is the Censys OAuth2 authorization server URL. It is a
	// fixed production value, hardcoded and not user-configurable.
	DefaultIssuer = "https://oauth2.censys.io"

	// DefaultAudience is the Censys Platform API audience requested for OAuth2
	// access tokens (RFC 8707). It is a fixed production value, hardcoded and
	// not user-configurable.
	DefaultAudience = "https://api.platform.censys.io"

	// RedirectPort is the loopback port registered in the client's redirect
	// URI (http://127.0.0.1:5555/callback).
	RedirectPort = 5555

	// Scopes requested at login: identity, refresh tokens, and Platform API access.
	Scopes = "openid offline_access censys.api email"

	authorizePath = "/oauth2/auth"
	tokenPath     = "/oauth2/token"
	revokePath    = "/oauth2/revoke"
)

// Config describes the client parameters for a flow. The authorization server
// endpoints (issuer, audience) are not configurable; they are the hardcoded
// DefaultIssuer / DefaultAudience constants.
type Config struct {
	// ClientID overrides the default cencli client ID (used in tests).
	ClientID string
	// Scopes overrides the default requested scopes (used in tests).
	Scopes string
	// RedirectPort overrides the default loopback port. 0 in tests selects a
	// random free port.
	RedirectPort *int
	// LoginTimeout bounds how long Login waits for the browser callback.
	LoginTimeout time.Duration
}

// Client performs OAuth2 flows against the Censys authorization server.
type Client struct {
	cfg  Config
	http *http.Client
	// issuer and audience are the hardcoded authorization-server endpoints.
	// They default to DefaultIssuer / DefaultAudience and are only overridden
	// by tests (same-package) to target a local server.
	issuer   string
	audience string
}

// NewClient builds a Client, applying defaults for unset config fields. The
// HTTP client is passed to x/oauth2 (via the request context) and used for
// revocation, so our transport (user agent, timeouts) applies throughout.
func NewClient(cfg Config, httpClient *http.Client) *Client {
	if cfg.ClientID == "" {
		cfg.ClientID = ClientID
	}
	if cfg.Scopes == "" {
		cfg.Scopes = Scopes
	}
	if cfg.RedirectPort == nil {
		port := RedirectPort
		cfg.RedirectPort = &port
	}
	if cfg.LoginTimeout <= 0 {
		cfg.LoginTimeout = 5 * time.Minute
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		cfg:      cfg,
		http:     httpClient,
		issuer:   strings.TrimRight(DefaultIssuer, "/"),
		audience: DefaultAudience,
	}
}

// oauthConfig builds the x/oauth2 config for a login using the given loopback
// redirect URI (empty when only the token endpoint is needed, e.g. refresh).
func (c *Client) oauthConfig(redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    c.cfg.ClientID,
		RedirectURL: redirectURI,
		Scopes:      strings.Fields(c.cfg.Scopes),
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.issuer + authorizePath,
			TokenURL: c.issuer + tokenPath,
			// Public PKCE client: send client_id in the request body, no secret.
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

// httpContext injects our HTTP client so x/oauth2 uses it for token requests.
func (c *Client) httpContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, c.http)
}

// authCodeOptions carries the audience (RFC 8707) into the authorization URL.
func (c *Client) authCodeOptions() []oauth2.AuthCodeOption {
	if c.audience == "" {
		return nil
	}
	return []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("audience", c.audience)}
}

// Exchange trades an authorization code (plus the PKCE verifier) for tokens.
func (c *Client) Exchange(ctx context.Context, code, verifier, redirectURI string) (*Session, error) {
	tok, err := c.oauthConfig(redirectURI).Exchange(c.httpContext(ctx), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, wrapTokenError(err)
	}
	return sessionFromToken(tok, c.issuer), nil
}

// Refresh trades a refresh token for a new token set. The authorization server
// rotates refresh tokens: the returned session's refresh token replaces the
// one passed in.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	tok, err := c.oauthConfig("").TokenSource(c.httpContext(ctx), &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return nil, wrapTokenError(err)
	}
	return sessionFromToken(tok, c.issuer), nil
}

// Revoke invalidates a token (and, for refresh tokens, the whole grant).
// x/oauth2 has no revocation support, so this posts the RFC 7009 request directly.
func (c *Client) Revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", c.cfg.ClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.issuer+revokePath, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request to authorization server failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return &ServerError{StatusCode: resp.StatusCode}
	}
	return nil
}

// sessionFromToken converts an x/oauth2 token into a persisted Session,
// pulling the id_token/scope from the response extras and deriving claims.
func sessionFromToken(tok *oauth2.Token, issuer string) *Session {
	sess := &Session{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    tok.Expiry,
		Issuer:       issuer,
	}
	if idToken, ok := tok.Extra("id_token").(string); ok {
		sess.IDToken = idToken
	}
	if scope, ok := tok.Extra("scope").(string); ok {
		sess.Scope = scope
	}
	sess.populateClaims()
	sess.populateOrgID()
	return sess
}

// generateState returns an opaque state parameter for the authorization request.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ServerError is an OAuth2 error response (RFC 6749 section 5.2) from the
// authorization server.
type ServerError struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *ServerError) Error() string {
	msg := fmt.Sprintf("authorization server returned %d", e.StatusCode)
	if e.Code != "" {
		msg += ": " + e.Code
	}
	if e.Description != "" {
		msg += ": " + e.Description
	}
	return msg
}

// wrapTokenError maps an x/oauth2 token error into a ServerError when it
// carries an RFC 6749 error response, so callers get a consistent type.
func wrapTokenError(err error) error {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		return &ServerError{
			StatusCode:  re.Response.StatusCode,
			Code:        re.ErrorCode,
			Description: re.ErrorDescription,
		}
	}
	return err
}
