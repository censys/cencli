package http

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Client struct {
	http.Client
}

// New creates an HTTP client configured for CLI usage.
// If logger is non-nil, requests and responses will be logged at Debug level.
func New(requestTimeout time.Duration, userAgent string, logger *slog.Logger) *Client {
	// Custom base transport tuned for CLI usage
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		Client: http.Client{
			Transport: &roundTripper{
				RoundTripper: base,
				userAgent:    userAgent,
				logger:       logger,
			},
			Timeout: requestTimeout,
		},
	}
}

type roundTripper struct {
	http.RoundTripper
	userAgent string
	logger    *slog.Logger
}

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	existingUserAgent := req.Header.Get("User-Agent")
	if existingUserAgent == "" {
		req.Header.Set("User-Agent", r.userAgent)
	} else {
		req.Header.Set("User-Agent", existingUserAgent+" "+r.userAgent)
	}

	if r.logger != nil {
		r.logger.Debug("http request", "method", req.Method, "url", req.URL.String())
	}

	start := time.Now()
	resp, err := r.RoundTripper.RoundTrip(req)
	duration := time.Since(start)

	if r.logger != nil {
		if err != nil {
			r.logger.Debug("http error", "method", req.Method, "url", req.URL.String(), "error", err, "duration", duration)
		} else {
			r.logger.Debug("http response", "method", req.Method, "url", req.URL.String(), "status", resp.StatusCode, "duration", duration)
		}
	}

	return resp, err
}
