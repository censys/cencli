package censys

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	censys "github.com/censys/censys-sdk-go"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/config"
)

// newTestGlobalDataSDK allows three attempts, so a wrapper that retries is visible.
func newTestGlobalDataSDK(t *testing.T, handler http.HandlerFunc) *globalDataSDK {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return newGlobalDataSDK(&censysSDK{
		client: censys.New(censys.WithServerURL(srv.URL), censys.WithSecurity("test-token")),
		retryStrategy: config.RetryStrategy{
			MaxAttempts: 3,
			BaseDelay:   time.Millisecond,
			Backoff:     config.BackoffFixed,
		},
	})
}

func writeJSON(w http.ResponseWriter, contentType string, status int, body string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func writeProblem(w http.ResponseWriter, status int) {
	writeJSON(w, "application/problem+json", status,
		fmt.Sprintf(`{"title":%q,"status":%d,"detail":"boom"}`, http.StatusText(status), status))
}

func TestGlobalDataSDK_WebPropertyTimeline(t *testing.T) {
	fromTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		orgID        mo.Option[string]
		statuses     []int
		expectCalls  int32
		expectEvents int
	}{
		{
			name:         "success sends inverted bounds",
			orgID:        mo.Some("11111111-1111-1111-1111-111111111111"),
			statuses:     []int{http.StatusOK},
			expectCalls:  1,
			expectEvents: 1,
		},
		{
			name:         "retries a 5xx",
			orgID:        mo.None[string](),
			statuses:     []int{http.StatusInternalServerError, http.StatusOK},
			expectCalls:  2,
			expectEvents: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			sdk := newTestGlobalDataSDK(t, func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v3/global/asset/webproperty/example.com:443/timeline", r.URL.EscapedPath())
				q := r.URL.Query()
				// start_time is the bound closest to now; end_time the one furthest back.
				assert.Equal(t, toTime.Format(time.RFC3339), q.Get("start_time"))
				assert.Equal(t, fromTime.Format(time.RFC3339), q.Get("end_time"))
				assert.Equal(t, tc.orgID.OrEmpty(), q.Get("organization_id"))

				status := tc.statuses[n-1]
				if status != http.StatusOK {
					writeProblem(w, status)
					return
				}
				writeJSON(w, "application/vnd.censys.api.v3.web_timeline_event.v1+json", http.StatusOK,
					`{"result":{"events":[{"resource":{"event_time":"2026-09-27T10:00:00Z"}}],"scanned_to":"2026-09-01T00:00:00Z"}}`)
			})

			res, err := sdk.WebPropertyTimeline(context.Background(), tc.orgID, "example.com:443", fromTime, toTime)

			assert.Equal(t, tc.expectCalls, calls.Load())
			require.NoError(t, err)
			require.NotNil(t, res.Data)
			require.Len(t, res.Data.Events, tc.expectEvents)
			assert.Equal(t, "2026-09-27T10:00:00Z", *res.Data.Events[0].Resource.EventTime)
			assert.True(t, res.Data.ScannedTo.Equal(fromTime))
			assert.Equal(t, uint64(tc.expectCalls), res.Metadata.Attempts)
		})
	}
}
