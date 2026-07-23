package enrich

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/app/streaming"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func strPtr(s string) *string { return &s }

func okResult(ip string) client.Result[components.HostEnrichment] {
	return client.Result[components.HostEnrichment]{
		Data: &components.HostEnrichment{IP: strPtr(ip)},
		Metadata: client.Metadata{
			Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
			Response: &http.Response{StatusCode: 200},
			Latency:  10 * time.Millisecond,
		},
	}
}

func hostIDs(t *testing.T, ips ...string) []assets.HostID {
	t.Helper()
	ids := make([]assets.HostID, 0, len(ips))
	for _, ip := range ips {
		id, err := assets.NewHostID(ip)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return ids
}

func TestEnrichService_EnrichHosts(t *testing.T) {
	t.Run("success preserves input order", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		ips := []string{"8.8.8.8", "1.1.1.1", "9.9.9.9"}
		for _, ip := range ips {
			mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), ip).Return(okResult(ip), nil)
		}

		svc := New(mockClient)
		res, err := svc.EnrichHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs(t, ips...))

		require.Nil(t, err)
		require.Nil(t, res.PartialError)
		require.Len(t, res.Hosts, 3)
		// Despite concurrent completion, output is in input order.
		require.Equal(t, "8.8.8.8", *res.Hosts[0].IP)
		require.Equal(t, "1.1.1.1", *res.Hosts[1].IP)
		require.Equal(t, "9.9.9.9", *res.Hosts[2].IP)
		assert.Equal(t, uint64(3), res.Meta.PageCount)
	})

	t.Run("partial failure surfaces failures and a partial error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), "8.8.8.8").Return(okResult("8.8.8.8"), nil)

		status := int64(404)
		detail := "host not found"
		mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), "1.1.1.1").Return(
			client.Result[components.HostEnrichment]{},
			client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status}),
		)

		svc := New(mockClient)
		res, err := svc.EnrichHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs(t, "8.8.8.8", "1.1.1.1"))

		require.Nil(t, err)
		require.Len(t, res.Hosts, 1)
		require.Equal(t, "8.8.8.8", *res.Hosts[0].IP)
		require.Len(t, res.Failures, 1)
		require.Equal(t, "1.1.1.1", res.Failures[0].HostID.String())
		require.NotNil(t, res.PartialError)
		require.Contains(t, res.PartialError.Error(), "1 of 2")
	})

	t.Run("all failures returns a hard error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), gomock.Any()).
			Return(client.Result[components.HostEnrichment]{}, client.NewClientError(errors.New("boom"))).
			Times(2)

		svc := New(mockClient)
		res, err := svc.EnrichHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs(t, "8.8.8.8", "1.1.1.1"))

		require.NotNil(t, err)
		require.Contains(t, err.Error(), "boom")
		require.Empty(t, res.Hosts)
	})

	t.Run("daily limit (429) returns a daily-limit error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		err429 := client.NewCensysClientGenericError(&sdkerrors.SDKError{
			Message:    "too many requests",
			StatusCode: http.StatusTooManyRequests,
			Body:       "daily limit reached",
		})
		mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), "8.8.8.8").
			Return(client.Result[components.HostEnrichment]{}, err429)

		svc := New(mockClient)
		_, err := svc.EnrichHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs(t, "8.8.8.8"))

		require.NotNil(t, err)
		assert.Equal(t, "Daily Enrichment Limit Reached", err.Title())
	})

	t.Run("empty input returns an empty result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl) // no expectations

		svc := New(mockClient)
		res, err := svc.EnrichHosts(context.Background(), mo.None[identifiers.OrganizationID](), nil)

		require.Nil(t, err)
		require.Empty(t, res.Hosts)
		require.Empty(t, res.Failures)
	})

	t.Run("streaming emits results and leaves Hosts empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		ips := []string{"8.8.8.8", "1.1.1.1"}
		for _, ip := range ips {
			mockClient.EXPECT().EnrichHost(gomock.Any(), mo.None[string](), ip).Return(okResult(ip), nil)
		}

		emitter, items := streaming.NewChannelEmitter(10)
		ctx := streaming.WithEmitter(context.Background(), emitter)

		var got []string
		done := make(chan struct{})
		go func() {
			for it := range items {
				if it.Done {
					break
				}
				if eh, ok := it.Data.(*assets.EnrichedHost); ok {
					got = append(got, *eh.IP)
				}
			}
			close(done)
		}()

		svc := New(mockClient)
		res, err := svc.EnrichHosts(ctx, mo.None[identifiers.OrganizationID](), hostIDs(t, ips...))
		emitter.Close(nil)
		<-done // happens-before: safe to read `got` after this

		require.Nil(t, err)
		require.Empty(t, res.Hosts) // streaming emits items rather than collecting them
		require.ElementsMatch(t, ips, got)
	})
}
