package censys

import (
	"context"
	"io"
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

// newTestCollectionsSDK allows three attempts, so a wrapper that retries is visible.
func newTestCollectionsSDK(t *testing.T, handler http.HandlerFunc) *collectionsSDK {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return newCollectionsSDK(&censysSDK{
		client: censys.New(censys.WithServerURL(srv.URL), censys.WithSecurity("test-token")),
		retryStrategy: config.RetryStrategy{
			MaxAttempts: 3,
			BaseDelay:   time.Millisecond,
			Backoff:     config.BackoffFixed,
		},
	})
}

func TestCollectionsSDK_CreateCollection(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		expectErr   bool
		expectCalls int32
	}{
		{
			name:        "success",
			status:      http.StatusOK,
			expectCalls: 1,
		},
		{
			name:        "5xx is not retried",
			status:      http.StatusBadGateway,
			expectErr:   true,
			expectCalls: 1,
		},
		{
			name:        "429 is not retried",
			status:      http.StatusTooManyRequests,
			expectErr:   true,
			expectCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			sdk := newTestCollectionsSDK(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v3/collections", r.URL.Path)
				assert.Equal(t, testOrgID, r.URL.Query().Get("organization_id"))
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, `{"name":"web","query":"host.services.port: 443"}`, string(body))

				if tc.status != http.StatusOK {
					writeProblem(w, tc.status)
					return
				}
				writeJSON(w, "application/json", http.StatusOK,
					`{"result":{"id":"c1","name":"web","query":"host.services.port: 443"}}`)
			})

			res, err := sdk.CreateCollection(context.Background(), CreateCollectionRequest{
				OrgID: mo.Some(testOrgID),
				Name:  "web",
				Query: "host.services.port: 443",
			})

			assert.Equal(t, tc.expectCalls, calls.Load())
			if tc.expectErr {
				require.Error(t, err)
				assert.Equal(t, int64(tc.status), err.StatusCode().OrEmpty())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res.Data)
			assert.Equal(t, "web", res.Data.Name)
			assert.Equal(t, uint64(1), res.Metadata.Attempts)
		})
	}
}
