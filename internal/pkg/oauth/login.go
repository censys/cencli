package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// callbackResult carries the outcome of the browser redirect back to Login.
type callbackResult struct {
	code string
	err  error
}

// Login runs the full authorization-code + PKCE flow:
//
//  1. start a loopback HTTP listener for the registered redirect URI
//  2. hand the authorization URL to openURL (which typically prints it and
//     opens the user's browser)
//  3. wait for the redirect carrying the authorization code
//  4. exchange the code (with the PKCE verifier) for tokens
//
// It blocks until the flow completes, ctx is cancelled, or the login timeout
// elapses.
func (c *Client) Login(ctx context.Context, openURL func(authorizeURL string) error) (*Session, error) {
	verifier := oauth2.GenerateVerifier()
	state, err := generateState()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *c.cfg.RedirectPort))
	if err != nil {
		return nil, fmt.Errorf(
			"unable to listen on 127.0.0.1:%d for the login redirect (is another `censys auth login` or an application using that port running?): %w",
			*c.cfg.RedirectPort, err,
		)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port)

	results := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			// Not our redirect (stale tab, stray request) — reject it but keep waiting.
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		var result callbackResult
		if errCode := q.Get("error"); errCode != "" {
			result.err = fmt.Errorf("authorization failed: %s: %s", errCode, q.Get("error_description"))
			writeCallbackPage(w, "Login failed", "You can close this window and return to the terminal.")
		} else if code := q.Get("code"); code != "" {
			result.code = code
			writeCallbackPage(w, "Login successful", "You are now authenticated with the Censys CLI. You can close this window.")
		} else {
			result.err = fmt.Errorf("authorization redirect contained no code")
			writeCallbackPage(w, "Login failed", "You can close this window and return to the terminal.")
		}
		select {
		case results <- result:
		default:
		}
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	authCodeURL := c.oauthConfig(redirectURI).AuthCodeURL(
		state,
		append(c.authCodeOptions(), oauth2.S256ChallengeOption(verifier))...,
	)
	if err := openURL(authCodeURL); err != nil {
		return nil, err
	}

	select {
	case result := <-results:
		if result.err != nil {
			return nil, result.err
		}
		return c.Exchange(ctx, result.code, verifier, redirectURI)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.cfg.LoginTimeout):
		return nil, fmt.Errorf("timed out after %s waiting for the browser login to complete", c.cfg.LoginTimeout)
	}
}

func writeCallbackPage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>%[1]s - Censys CLI</title></head>
<body style="font-family: -apple-system, system-ui, sans-serif; display: flex; justify-content: center; margin-top: 15vh;">
<div style="text-align: center;">
<h1>%[1]s</h1>
<p>%[2]s</p>
</div>
</body>
</html>`, title, body)
}
