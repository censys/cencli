package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

type Client struct {
	http.Client
}

// Options configures the HTTP client.
type Options struct {
	RequestTimeout time.Duration
	UserAgent      string
	Logger         *slog.Logger

	// Proxy settings — empty falls back to HTTP_PROXY/HTTPS_PROXY env vars.
	ProxyURL string

	// TLS settings
	CABundlePath       string
	InsecureSkipVerify bool
	ClientCertPath     string
	ClientKeyPath      string
	// DisableHTTP2 forces HTTP/1.1. Useful when the proxy or network
	// infrastructure doesn't support HTTP/2.
	DisableHTTP2 bool
}

// New creates an HTTP client configured for CLI usage.
func New(opts Options) (*Client, error) {
	netDialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	base := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           netDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if err := applyProxy(base, netDialer, opts.ProxyURL); err != nil {
		return nil, err
	}

	if opts.DisableHTTP2 {
		base.ForceAttemptHTTP2 = false
		// Empty TLSNextProto map disables h2 ALPN negotiation on TLS connections.
		base.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}

	tlsCfg, err := buildTLSConfig(opts.CABundlePath, opts.ClientCertPath, opts.ClientKeyPath, opts.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}
	base.TLSClientConfig = tlsCfg

	return &Client{
		Client: http.Client{
			Transport: &roundTripper{
				RoundTripper: base,
				userAgent:    opts.UserAgent,
				logger:       opts.Logger,
			},
			Timeout: opts.RequestTimeout,
		},
	}, nil
}

func applyProxy(base *http.Transport, netDialer *net.Dialer, proxyURL string) error {
	if proxyURL == "" {
		return nil
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
	}
	switch parsed.Scheme {
	case "http", "https":
		base.Proxy = http.ProxyURL(parsed)
	case "socks5", "socks5h":
		socksDialer, err := proxy.FromURL(parsed, netDialer)
		if err != nil {
			return fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}
		base.Proxy = nil
		if cd, ok := socksDialer.(proxy.ContextDialer); ok {
			base.DialContext = cd.DialContext
		} else {
			d := socksDialer
			base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.Dial(network, addr)
			}
		}
	default:
		return fmt.Errorf("unsupported proxy scheme %q: use http, https, socks5, or socks5h", parsed.Scheme)
	}
	return nil
}

func buildTLSConfig(caBundlePath, clientCertPath, clientKeyPath string, insecureSkipVerify bool) (*tls.Config, error) {
	cfg := &tls.Config{}
	hasConfig := false

	if insecureSkipVerify {
		cfg.InsecureSkipVerify = true //nolint:gosec
		hasConfig = true
	}

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
		cfg.RootCAs = pool
		hasConfig = true
	}

	switch {
	case clientCertPath != "" && clientKeyPath != "":
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
		hasConfig = true
	case clientCertPath != "" || clientKeyPath != "":
		return nil, fmt.Errorf("tls.client-cert and tls.client-key must both be set")
	}

	if !hasConfig {
		return nil, nil
	}
	return cfg, nil
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
