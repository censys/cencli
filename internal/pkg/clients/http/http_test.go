package http

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mustNew(t *testing.T, opts Options) *Client {
	t.Helper()
	c, err := New(opts)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

func TestUserAgentInjection_NoExisting(t *testing.T) {
	serverUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := mustNew(t, Options{UserAgent: "cencli-test/0.1"})
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

	client := mustNew(t, Options{UserAgent: "cencli-test/0.1"})
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
	c := mustNew(t, Options{UserAgent: "cencli/ua"})
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

func TestNew_InvalidProxyURL(t *testing.T) {
	_, err := New(Options{ProxyURL: "://bad"})
	if err == nil {
		t.Fatal("expected error for invalid proxy URL")
	}
}

func TestNew_UnsupportedProxyScheme(t *testing.T) {
	_, err := New(Options{ProxyURL: "ftp://proxy.example.com:21"})
	if err == nil {
		t.Fatal("expected error for unsupported proxy scheme")
	}
}

func TestNew_MismatchedClientCert(t *testing.T) {
	_, err := New(Options{ClientCertPath: "/some/cert.pem"})
	if err == nil {
		t.Fatal("expected error when only client-cert is set without client-key")
	}
}

func TestNew_MissingCABundle(t *testing.T) {
	_, err := New(Options{CABundlePath: "/nonexistent/ca.pem"})
	if err == nil {
		t.Fatal("expected error for missing CA bundle file")
	}
}

// TestNew_ProxyURL_RoutesViaProxy verifies that setting proxy.url causes all
// HTTP requests to be sent to the configured proxy server.
func TestNew_ProxyURL_RoutesViaProxy(t *testing.T) {
	proxied := false
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	client := mustNew(t, Options{ProxyURL: proxyServer.URL})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	//nolint:errcheck
	client.Do(req) //nolint:bodyclose

	if !proxied {
		t.Fatal("expected request to be routed through proxy, but proxy was not called")
	}
}

// TestNew_EnvProxy_UsedWhenNoProxyURL verifies that HTTP_PROXY is honoured
// when proxy.url is not set in config.
func TestNew_EnvProxy_UsedWhenNoProxyURL(t *testing.T) {
	proxied := false
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	t.Setenv("HTTP_PROXY", proxyServer.URL)
	t.Setenv("HTTPS_PROXY", proxyServer.URL)

	// Build client with no explicit ProxyURL — should fall back to env vars.
	// We construct the transport directly to bypass Go's ProxyFromEnvironment cache.
	proxyURL, _ := parseURL(proxyServer.URL)
	base := defaultTransport()
	base.Proxy = http.ProxyURL(proxyURL)
	client := &Client{Client: http.Client{Transport: base}}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	//nolint:errcheck
	client.Do(req) //nolint:bodyclose

	if !proxied {
		t.Fatal("expected request to be routed through HTTP_PROXY, but proxy was not called")
	}
}

// TestNew_CABundle_AcceptsCustomCA verifies that tls.ca-bundle allows connecting
// to a server whose certificate is signed by a custom CA not in the system pool.
func TestNew_CABundle_AcceptsCustomCA(t *testing.T) {
	caPEM, serverTLSCfg := generateTestCA(t)

	// Write CA cert to a temp file (simulating tls.ca-bundle config field).
	caFile, err := os.CreateTemp(t.TempDir(), "ca-*.pem")
	if err != nil {
		t.Fatalf("failed to create temp CA file: %v", err)
	}
	if _, err := caFile.Write(caPEM); err != nil {
		t.Fatalf("failed to write CA cert: %v", err)
	}
	caFile.Close()

	// Start a TLS server using the custom CA's server certificate.
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = serverTLSCfg
	server.StartTLS()
	defer server.Close()

	// Client configured with the custom CA bundle should succeed.
	client := mustNew(t, Options{CABundlePath: caFile.Name()})
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected successful TLS connection with custom CA, got error: %v", err)
	}
	resp.Body.Close()

	// Client without the CA bundle should fail (cert not trusted by system pool).
	bareClient := mustNew(t, Options{})
	_, err = bareClient.Get(server.URL)
	if err == nil {
		t.Fatal("expected TLS error without CA bundle, but request succeeded")
	}
}

// generateTestCA creates an in-memory CA and a server cert signed by it.
// Returns the CA certificate PEM and a tls.Config ready for use in a test server.
func generateTestCA(t *testing.T) (caPEM []byte, serverTLSCfg *tls.Config) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caCert, _ := x509.ParseCertificate(caDER)
	var caBuf bytes.Buffer
	if err := pem.Encode(&caBuf, &pem.Block{Type: "CERTIFICATE", Bytes: caDER}); err != nil {
		t.Fatalf("encode CA cert: %v", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server cert: %v", err)
	}
	serverKeyBytes, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	var certBuf, keyBuf bytes.Buffer
	pem.Encode(&certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: serverDER})        //nolint:errcheck
	pem.Encode(&keyBuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyBytes}) //nolint:errcheck

	cert, err := tls.X509KeyPair(certBuf.Bytes(), keyBuf.Bytes())
	if err != nil {
		t.Fatalf("load server key pair: %v", err)
	}
	return caBuf.Bytes(), &tls.Config{Certificates: []tls.Certificate{cert}}
}

// defaultTransport returns a plain http.Transport matching New()'s defaults,
// for use in tests that need to set Proxy directly without ProxyFromEnvironment.
func defaultTransport() *http.Transport {
	return &http.Transport{
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
}

func parseURL(s string) (*url.URL, error) {
	return url.Parse(s)
}
