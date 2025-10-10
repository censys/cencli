package fixtures

import (
	"bufio"
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertHas200(t *testing.T, stderr []byte) {
	assert.True(t, bytes.Contains(stderr, []byte("200 (OK)")), "expected 200 (OK) in stderr, got: %s", string(stderr))
}

// unmarshalJSONAny unmarshals the JSON data into the given type.
// It will require that the unmarshal operation was successful,
// to prevent panics.
func unmarshalJSONAny[T any](t *testing.T, data []byte) T {
	var expected T
	var raw any

	err := json.Unmarshal(data, &raw)
	require.NoError(t, err, "failed to unmarshal JSON")

	err = json.Unmarshal(data, &expected)
	require.NoError(t, err,
		"failed to unmarshal JSON: expected type %T, got %T",
		expected, raw)

	return expected
}

var (
	// Match any sequence of non-whitespace characters and template variables together
	// This will match patterns like "/{{yellow transport_protocol}}" or "{{blue port}}/{{yellow transport_protocol}}"
	// by consuming all non-whitespace and template vars in one go
	templatedVars = regexp.MustCompile(`[^\s\/]*\{\{.*?\}\}[^\s\/{]*`)

	// templateMatchThreshold is the proportion of words that must be found in a rendered template
	// compared to the template itself.
	templateMatchThreshold = 0.5
)

func assertRenderedTemplate(t *testing.T, template []byte, stdout []byte) {
	strippedTemplate := templatedVars.ReplaceAllString(string(template), "")
	strippedTemplate = strings.TrimSpace(strippedTemplate)
	expectedWords := strings.Fields(strippedTemplate)

	found := 0
	for _, word := range expectedWords {
		if strings.Contains(string(stdout), word) {
			found++
		}
	}

	required := int(float64(len(expectedWords)) * templateMatchThreshold)
	assert.GreaterOrEqual(t, found, required,
		"only %d/%d (%.2f%%) expected words found in stdout, need at least %.0f%%",
		found, len(expectedWords),
		float64(found)/float64(len(expectedWords))*100,
		templateMatchThreshold*100)
}

// removeFirstNLines removes the first n lines from the given data.
func removeFirstNLines(data []byte, n int) []byte {
	if n <= 0 {
		return data
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var buf bytes.Buffer
	count := 0
	for scanner.Scan() {
		count++
		if count > n {
			buf.Write(scanner.Bytes())
			buf.WriteByte('\n')
		}
	}
	out := buf.Bytes()
	// Trim leading newlines or spaces that might cause a visible dot in diffs
	out = bytes.TrimLeft(out, "\r\n")
	return out
}

// assertGoldenFile asserts that the given stdout matches the contents of the golden file.
// It normalizes both inputs by trimming trailing whitespace to handle platform differences.
// leadingLinesRemoved is the number of lines to remove from the beginning of the stdout.
func assertGoldenFile(t *testing.T, golden, stdout []byte, leadingLinesRemoved int) {
	golden = removeFirstNLines(golden, leadingLinesRemoved)
	normalizedGolden := bytes.TrimRight(golden, "\n\r\t ")
	normalizedStdout := bytes.TrimRight(stdout, "\n\r\t ")
	assert.Equal(t, string(normalizedGolden), string(normalizedStdout))
}
