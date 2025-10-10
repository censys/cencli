package formatter

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/styles"
)

func TestPrintAppResponseMeta_NoHeaders(t *testing.T) {
	var buf bytes.Buffer
	Stderr = &buf
	req := &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io", Path: "/v1"}}
	res := &http.Response{StatusCode: 200, Header: http.Header{}}
	meta := responsemeta.NewResponseMeta(req, res, 0, 1)
	PrintAppResponseMeta(styles.GlobalStyles, meta, false, true)
	out := buf.String()
	if !strings.Contains(out, "200 (OK)") {
		t.Fatalf("expected status line, got: %s", out)
	}
}

func TestPrintAppResponseMeta_SanitizesHeaders(t *testing.T) {
	var buf bytes.Buffer
	Stderr = &buf
	req := &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io", Path: "/v1"}, Header: http.Header{
		"Authorization": []string{"Bearer SECRET"},
	}}
	res := &http.Response{StatusCode: 429, Header: http.Header{
		"Set-Cookie":   []string{"k=v"},
		"X-Request-Id": []string{"abc"},
	}}
	meta := responsemeta.NewResponseMeta(req, res, 0, 2)
	PrintAppResponseMeta(styles.GlobalStyles, meta, true, true)
	out := buf.String()
	if strings.Contains(out, "Bearer SECRET") || strings.Contains(out, "k=v") {
		t.Fatalf("expected sanitized headers, got: %s", out)
	}
	if !strings.Contains(out, "req-Authorization: ********") {
		t.Fatalf("expected redacted authorization, got: %s", out)
	}
	if !strings.Contains(out, "res-Set-Cookie: ********") {
		t.Fatalf("expected redacted set-cookie, got: %s", out)
	}
	if !strings.Contains(out, "res-X-Request-Id: abc") {
		t.Fatalf("expected non-sensitive header present, got: %s", out)
	}
}
