package formatter

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/styles"
)

// PrintAppResponseMeta renders application-level response metadata without leaking http.Request/Response.
// When verbose is true, sanitized headers are printed for debugging purposes, as well as the request URL.
func PrintAppResponseMeta(st *styles.Styles, meta *responsemeta.ResponseMeta, verbose bool, colored bool) {
	if !colored {
		restore := styles.TemporarilyDisableStyles()
		defer restore()
	}
	var output strings.Builder

	// Request line
	if verbose {
		requestLine := st.Signature.Render(meta.Method) +
			" " + st.Info.Render(meta.URL)
		output.WriteString(requestLine)
		output.WriteString("\n\n")
	}

	// Status line
	statusCodeStyle := st.Signature
	if meta.Status < 200 || meta.Status >= 300 {
		statusCodeStyle = st.Danger
	}
	status := http.StatusText(meta.Status)
	statusLine := statusCodeStyle.Render(fmt.Sprintf("%d", meta.Status)) +
		" (" + st.Info.Render(status) + ") - " +
		st.Tertiary.Render(meta.Latency.String())

	if meta.PageCount > 0 {
		statusLine += " - " + st.Secondary.Render(fmt.Sprintf("pages: %d", meta.PageCount))
	}

	if meta.RetryCount > 0 {
		statusLine += " - " + st.Secondary.Render(fmt.Sprintf("retries: %d", meta.RetryCount))
	}
	output.WriteString(statusLine)
	output.WriteString("\n")

	if verbose && len(meta.Headers) > 0 {
		// Stable ordering for tests and readability
		keys := make([]string, 0, len(meta.Headers))
		for k := range meta.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := meta.Headers[k]
			output.WriteString(st.Indent4.Render(st.Tertiary.Render(k) + ": " + st.Primary.Render(v)))
			output.WriteString("\n")
		}
	}

	fmt.Fprint(Stderr, output.String())
}
