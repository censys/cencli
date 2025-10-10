package responsemeta

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizedHeaders_MasksSensitive(t *testing.T) {
	reqHeaders := http.Header{}
	resHeaders := http.Header{}

	reqHeaders.Set("Authorization", "Bearer abc")
	reqHeaders.Set("X-Api-Key", "secret")
	resHeaders.Set("Set-Cookie", "sid=123")
	resHeaders.Set("Content-Type", "application/json")
	resHeaders.Set("Proxy-Authorization", "Basic abc")

	got := sanitizedHeaders(reqHeaders, resHeaders)

	require.Equal(t, "********", got["req-Authorization"])        // case preserved in key
	require.Equal(t, "********", got["req-X-Api-Key"])            // custom header
	require.Equal(t, "********", got["res-Set-Cookie"])           // cookie masked
	require.Equal(t, "application/json", got["res-Content-Type"]) // non-sensitive allowed
	require.Equal(t, "********", got["res-Proxy-Authorization"])  // masked
}

func TestSanitizedURL_StripsUserinfo(t *testing.T) {
	u, _ := url.Parse("https://user:pass@example.com/path?q=1")
	s := sanitizedURL(u)
	require.Equal(t, "https://example.com/path?q=1", s)
}
