package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUserAgentInjection_NoExisting(t *testing.T) {
	serverUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := New(0, "cencli-test/0.1", nil)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	if serverUA != "cencli-test/0.1" {
		t.Fatalf("expected UA 'cencli-test/0.1', got %q", serverUA)
	}
}

func TestUserAgentInjection_AppendsExisting(t *testing.T) {
	serverUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := New(0, "cencli-test/0.1", nil)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "existing-UA")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	expected := "existing-UA cencli-test/0.1"
	if serverUA != expected {
		t.Fatalf("expected UA %q, got %q", expected, serverUA)
	}
}

func TestUserAgentRoundTripper_AppendsOrSets(t *testing.T) {
	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Fatalf("expected user-agent to be set")
		}
		return &http.Response{StatusCode: 200, Body: http.NoBody, Request: r}, nil
	})

	rt := roundTripper{RoundTripper: base, userAgent: "cencli/test"}

	// No existing UA: should set
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("User-Agent"); got != "cencli/test" {
		t.Fatalf("expected UA set, got %q", got)
	}

	// Existing UA: should append
	req2, _ := http.NewRequest("GET", "https://example.com", nil)
	req2.Header.Set("User-Agent", "curl/8.0")
	if _, err := rt.RoundTrip(req2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req2.Header.Get("User-Agent"); got != "curl/8.0 cencli/test" {
		t.Fatalf("expected UA appended, got %q", got)
	}
}

func TestNew_SetsUserAgent_AndNoDefaultTimeout(t *testing.T) {
	c := New(0, "cencli/ua", nil)
	if c.Timeout != 0 {
		t.Fatalf("expected timeout 0 (disabled), got %v", c.Timeout)
	}
	// Intercept the inner transport while keeping UA injector
	rt, ok := c.Transport.(*roundTripper)
	if !ok {
		t.Fatalf("expected *roundTripper transport")
	}
	rt.RoundTripper = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("User-Agent"); got == "" || got != "cencli/ua" {
			t.Fatalf("expected UA 'cencli/ua', got %q", got)
		}
		return &http.Response{StatusCode: 200, Body: http.NoBody, Request: r}, nil
	})
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if _, err := c.Do(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
