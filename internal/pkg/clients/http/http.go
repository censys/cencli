package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	http.Client
}

// New creates an HTTP client configured for CLI usage.
// If logger is non-nil, requests and responses will be logged at Debug level.
// If proxyURL is non-empty, it overrides environment-based proxy detection.
// If caBundlePath is non-empty, the PEM file at that path is appended to the system CA pool.
func New(requestTimeout time.Duration, userAgent string, logger *slog.Logger, proxyURL string, caBundlePath string) (*Client, error) {
	proxyFunc := http.ProxyFromEnvironment
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
		}
		proxyFunc = http.ProxyURL(parsed)
	}

	var tlsConfig *tls.Config
	if caBundlePath != "" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		pem, err := os.ReadFile(caBundlePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA bundle %q: %w", caBundlePath, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in CA bundle: %s", caBundlePath)
		}
		tlsConfig = &tls.Config{RootCAs: pool}
	}

	base := &http.Transport{
		Proxy: proxyFunc,
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
		TLSClientConfig:       tlsConfig,
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
	}, nil
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
