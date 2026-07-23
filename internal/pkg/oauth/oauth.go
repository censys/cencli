// Package oauth implements the OAuth2 authorization-code + PKCE flow used by
// `censys auth login`, plus token refresh and revocation, against the Censys
// authorization server (Ory Hydra).
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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

// HTTPDoer is the minimal HTTP client interface required by Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

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
	http HTTPDoer
	// issuer and audience are the hardcoded authorization-server endpoints.
	// They default to DefaultIssuer / DefaultAudience and are only overridden
	// by tests (same-package) to target a local server.
	issuer   string
	audience string
}

// NewClient builds a Client, applying defaults for unset config fields.
func NewClient(cfg Config, doer HTTPDoer) *Client {
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
	if doer == nil {
		doer = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		cfg:      cfg,
		http:     doer,
		issuer:   strings.TrimRight(DefaultIssuer, "/"),
		audience: DefaultAudience,
	}
}

// AuthCodeURL builds the authorization request URL the user's browser visits.
func (c *Client) AuthCodeURL(state, challenge, redirectURI string) string {
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", c.cfg.Scopes)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	if c.audience != "" {
		q.Set("audience", c.audience)
	}
	return c.issuer + authorizePath + "?" + q.Encode()
}

// Exchange trades an authorization code (plus the PKCE verifier) for tokens.
func (c *Client) Exchange(ctx context.Context, code, verifier, redirectURI string) (*Session, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("code_verifier", verifier)
	return c.token(ctx, form)
}

// Refresh trades a refresh token for a new token set. Note the authorization
// server rotates refresh tokens: the returned session's refresh token
// replaces the one passed in.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.cfg.ClientID)
	return c.token(ctx, form)
}

// Revoke invalidates a token (and, for refresh tokens, the whole grant).
func (c *Client) Revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", c.cfg.ClientID)

	resp, err := c.postForm(ctx, c.issuer+revokePath, form)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return newServerError(resp)
	}
	return nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (c *Client) token(ctx context.Context, form url.Values) (*Session, error) {
	resp, err := c.postForm(ctx, c.issuer+tokenPath, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, newServerError(resp)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token response contained no access token")
	}

	sess := &Session{
		AccessToken:  tr.AccessToken,
		TokenType:    tr.TokenType,
		RefreshToken: tr.RefreshToken,
		IDToken:      tr.IDToken,
		Scope:        tr.Scope,
		Issuer:       c.issuer,
	}
	if tr.ExpiresIn > 0 {
		sess.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	sess.populateClaims()
	sess.populateOrgID()
	return sess, nil
}

func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to authorization server failed: %w", err)
	}
	return resp, nil
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

func newServerError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	serverErr := &ServerError{StatusCode: resp.StatusCode}
	var oauthErr struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &oauthErr); err == nil {
		serverErr.Code = oauthErr.Code
		serverErr.Description = oauthErr.Description
	}
	return serverErr
}
