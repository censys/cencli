package http

import (
	"net"
	"net/http"
	"time"
)

type Client struct {
	http.Client
}

func New(userAgent string) *Client {
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
			},
			// Rely on per-request contexts for cancellation. Keep transport-level safety timeouts above.
			Timeout: 0,
		},
	}
}

type roundTripper struct {
	http.RoundTripper
	userAgent string
}

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	existingUserAgent := req.Header.Get("User-Agent")
	if existingUserAgent == "" {
		req.Header.Set("User-Agent", r.userAgent)
	} else {
		req.Header.Set("User-Agent", existingUserAgent+" "+r.userAgent)
	}
	return r.RoundTripper.RoundTrip(req)
}
