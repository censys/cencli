package scan

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/google/uuid"

	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

const (
	testOrgUUID = "00000000-0000-0000-0000-000000000001"
	testScanID  = "3f2b9c1e-0000-4000-8000-000000000001"
)

var testOrgID = identifiers.NewOrganizationID(uuid.MustParse(testOrgUUID))

func strPtr(s string) *string { return &s }

func task(status components.TrackedScanTaskStatus) components.TrackedScanTask {
	return components.TrackedScanTask{Description: strPtr("HTTP endpoint scan"), Status: &status}
}

func trackedScanResult(completed bool, tasks ...components.TrackedScanTask) client.Result[components.TrackedScan] {
	return client.Result[components.TrackedScan]{
		Data: &components.TrackedScan{
			TrackedScanID: strPtr(testScanID),
			Completed:     &completed,
			CreateTime:    strPtr("2026-09-29T14:02:11Z"),
			Tasks:         tasks,
		},
		Metadata: client.Metadata{
			Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
			Response: &http.Response{StatusCode: 200},
			Latency:  10 * time.Millisecond,
			Attempts: 1,
		},
	}
}

// newTestService builds a service whose poll loop records the delays it was
// asked to sleep for instead of actually waiting.
func newTestService(c client.Client, delays *[]time.Duration) *scanService {
	return &scanService{
		client: c,
		sleep: func(ctx context.Context, d time.Duration) error {
			*delays = append(*delays, d)
			return ctx.Err()
		},
	}
}
