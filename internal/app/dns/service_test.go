package dns

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/app/streaming"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

var (
	from = time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	to   = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
)

func strPtr(s string) *string { return &s }

// okMeta is the same response metadata the tags service tests use.
func okMeta() client.Metadata {
	return client.Metadata{
		Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
		Response: &http.Response{StatusCode: 200},
		Latency:  100 * time.Millisecond,
	}
}

func apiError(status int64) client.ClientError {
	detail := "error"
	return client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
}

func mustDomain(t *testing.T, raw string) assets.DomainName {
	t.Helper()
	d, err := assets.NewDomainName(raw)
	require.NoError(t, err)
	return d
}

func mustHost(t *testing.T, raw string) assets.HostID {
	t.Helper()
	h, err := assets.NewHostID(raw)
	require.NoError(t, err)
	return h
}

func baseParams() Params {
	return Params{FromTime: from, ToTime: to, PageSize: mo.Some[uint64](100), MaxPages: mo.Some[uint64](10)}
}

func nameBoundsPage(next string, total int64, ips ...string) client.Result[components.DNSNameResolutionBoundResponse] {
	records := make([]components.DNSResolutionRecord, 0, len(ips))
	for _, ip := range ips {
		records = append(records, components.DNSResolutionRecord{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr(ip), FirstSeen: from, LastSeen: to})
	}
	return client.Result[components.DNSNameResolutionBoundResponse]{
		Metadata: okMeta(),
		Data:     &components.DNSNameResolutionBoundResponse{Name: "censys.com", NextPageToken: next, Records: records, TotalRecords: total},
	}
}

func TestNameResolutions(t *testing.T) {
	testCases := []struct {
		name   string
		params func() Params
		client func(ctrl *gomock.Controller) client.Client
		assert func(t *testing.T, res NameResolutionsResult, err error)
	}{
		{
			name:   "success - single page, record types normalized",
			params: func() Params { p := baseParams(); p.RecordTypes = []string{"a", " mx ", "A"}; return p },
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), mo.None[string](), "censys.com", from, to,
					[]string{"A", "MX"}, mo.Some[int64](100), mo.None[string]()).
					Return(nameBoundsPage("", 1, "104.18.10.84"), nil)
				return mc
			},
			assert: func(t *testing.T, res NameResolutionsResult, err error) {
				require.NoError(t, err)
				require.Len(t, res.Records, 1)
				require.Equal(t, "104.18.10.84", *res.Records[0].IP)
				require.Equal(t, int64(1), res.TotalRecords)
				require.Nil(t, res.PartialError)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name:   "success - two pages",
			params: baseParams,
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), "censys.com", from, to, []string(nil), gomock.Any(), mo.None[string]()).
						Return(nameBoundsPage("p2", 2, "1.1.1.1"), nil),
					mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), "censys.com", from, to, []string(nil), gomock.Any(), mo.Some("p2")).
						Return(nameBoundsPage("", 2, "2.2.2.2"), nil),
				)
				return mc
			},
			assert: func(t *testing.T, res NameResolutionsResult, err error) {
				require.NoError(t, err)
				require.Len(t, res.Records, 2)
				require.Equal(t, uint64(2), res.Meta.PageCount)
			},
		},
		{
			name:   "success - no data gives an empty, non-nil slice",
			params: baseParams,
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nameBoundsPage("", 0), nil)
				return mc
			},
			assert: func(t *testing.T, res NameResolutionsResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, res.Records)
				require.Empty(t, res.Records)
			},
		},
		{
			name:   "success - max pages 1 stops after the first page",
			params: func() Params { p := baseParams(); p.MaxPages = mo.Some[uint64](1); return p },
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), mo.None[string]()).
					Return(nameBoundsPage("p2", 50, "1.1.1.1"), nil)
				return mc
			},
			assert: func(t *testing.T, res NameResolutionsResult, err error) {
				require.NoError(t, err)
				require.Len(t, res.Records, 1)
				require.Equal(t, int64(50), res.TotalRecords)
			},
		},
		{
			name:   "error - invalid record type makes no API call",
			params: func() Params { p := baseParams(); p.RecordTypes = []string{"CNAME"}; return p },
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, _ NameResolutionsResult, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid record type 'CNAME'; supported values: A, AAAA, MX, NS, SOA, TXT")
			},
		},
		{
			name:   "error - 403 maps to the plan message",
			params: baseParams,
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSNameResolutionBoundResponse]{}, apiError(403))
				return mc
			},
			assert: func(t *testing.T, _ NameResolutionsResult, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Active DNS is available only on the Censys Search and Core plans")
			},
		},
		{
			name:   "error - 500 is returned unchanged",
			params: baseParams,
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSNameResolutionBoundResponse]{}, apiError(500))
				return mc
			},
			assert: func(t *testing.T, _ NameResolutionsResult, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error")
				require.NotContains(t, err.Error(), "Search and Core plans")
			},
		},
		{
			name:   "success - partial result when page 2 fails",
			params: baseParams,
			client: func(ctrl *gomock.Controller) client.Client {
				mc := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), mo.None[string]()).
						Return(nameBoundsPage("p2", 2, "1.1.1.1"), nil),
					mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), mo.Some("p2")).
						Return(client.Result[components.DNSNameResolutionBoundResponse]{}, apiError(500)),
				)
				return mc
			},
			assert: func(t *testing.T, res NameResolutionsResult, err error) {
				require.NoError(t, err)
				require.Len(t, res.Records, 1)
				require.Error(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "error")
			},
		},
		{
			name:   "error - zero max pages is rejected",
			params: func() Params { p := baseParams(); p.MaxPages = mo.Some[uint64](0); return p },
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, _ NameResolutionsResult, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "max pages must be greater than 0")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.NameResolutions(context.Background(), mustDomain(t, "censys.com"), tc.params())
			tc.assert(t, res, err)
		})
	}
}

func TestNameResolutionRanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mocks.NewMockClient(ctrl)
	mc.EXPECT().ListDNSNameResolutionRanges(gomock.Any(), mo.None[string](), "censys.com", from, to, []string{"TXT"}, mo.Some[int64](100), mo.None[string]()).
		Return(client.Result[components.DNSNameResolutionRangeResponse]{
			Metadata: okMeta(),
			Data: &components.DNSNameResolutionRangeResponse{
				Name:         "censys.com",
				Records:      []components.DNSResolutionRangeRecord{{RecordType: components.DNSResolutionRangeRecordRecordTypeTxt, Value: strPtr("v=spf1"), FirstObserved: from, LastObserved: to}},
				TotalRecords: 1,
			},
		}, nil)

	p := baseParams()
	p.RecordTypes = []string{"txt"}
	res, err := New(mc).NameResolutionRanges(context.Background(), mustDomain(t, "censys.com"), p)
	require.NoError(t, err)
	require.Len(t, res.Records, 1)
	require.Equal(t, "v=spf1", *res.Records[0].Value)
}

func TestIPResolutions(t *testing.T) {
	t.Run("success - domains for an ip", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mc := mocks.NewMockClient(ctrl)
		mc.EXPECT().ListDNSIPResolutionBounds(gomock.Any(), mo.None[string](), "104.18.10.84", from, to, []string(nil), mo.Some[int64](100), mo.None[string]()).
			Return(client.Result[components.DNSIPResolutionBoundResponse]{
				Metadata: okMeta(),
				Data: &components.DNSIPResolutionBoundResponse{
					IP:           "104.18.10.84",
					Records:      []components.DNSIPResolutionRecord{{Domain: "censys.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: from, LastSeen: to}},
					TotalRecords: 1,
				},
			}, nil)

		res, err := New(mc).IPResolutions(context.Background(), mustHost(t, "104.18.10.84"), baseParams())
		require.NoError(t, err)
		require.Len(t, res.Records, 1)
		require.Equal(t, "censys.com", res.Records[0].Domain)
	})

	t.Run("error - MX is not valid for an ip", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		p := baseParams()
		p.RecordTypes = []string{"MX"}
		_, err := New(mocks.NewMockClient(ctrl)).IPResolutions(context.Background(), mustHost(t, "104.18.10.84"), p)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid record type 'MX'; supported values: A, AAAA")
	})
}

func TestIPResolutionRanges(t *testing.T) {
	t.Run("success - domains for time ranges", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mc := mocks.NewMockClient(ctrl)
		mc.EXPECT().ListDNSIPResolutionRanges(gomock.Any(), mo.None[string](), "104.18.10.84", from, to, []string{"AAAA"}, mo.None[string](), mo.Some[int64](100), mo.None[string]()).
			Return(client.Result[components.DNSIPResolutionRangeResponse]{
				Metadata: okMeta(),
				Data: &components.DNSIPResolutionRangeResponse{
					IP:           "104.18.10.84",
					Records:      []components.DNSIPResolutionRangeRecord{{Domain: "censys.com", RecordType: components.DNSIPResolutionRangeRecordRecordTypeAaaa, FirstSeen: from, LastSeen: to}},
					TotalRecords: 1,
				},
			}, nil)

		p := baseParams()
		p.RecordTypes = []string{"aaaa"}
		res, err := New(mc).IPResolutionRanges(context.Background(), mustHost(t, "104.18.10.84"), p)
		require.NoError(t, err)
		require.Len(t, res.Records, 1)
	})

	t.Run("success - domain filter passes through to the client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mc := mocks.NewMockClient(ctrl)
		mc.EXPECT().ListDNSIPResolutionRanges(gomock.Any(), mo.None[string](), "104.18.10.84", from, to, []string(nil), mo.Some("censys.com"), mo.Some[int64](100), mo.None[string]()).
			Return(client.Result[components.DNSIPResolutionRangeResponse]{
				Metadata: okMeta(),
				Data: &components.DNSIPResolutionRangeResponse{
					IP:           "104.18.10.84",
					Records:      []components.DNSIPResolutionRangeRecord{{Domain: "censys.com", RecordType: components.DNSIPResolutionRangeRecordRecordTypeA, FirstSeen: from, LastSeen: to}},
					TotalRecords: 1,
				},
			}, nil)

		p := baseParams()
		p.Domain = mo.Some(mustDomain(t, "censys.com"))
		res, err := New(mc).IPResolutionRanges(context.Background(), mustHost(t, "104.18.10.84"), p)
		require.NoError(t, err)
		require.Len(t, res.Records, 1)
	})
}

func TestNameResolutions_Streaming(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mocks.NewMockClient(ctrl)
	mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nameBoundsPage("", 2, "1.1.1.1", "2.2.2.2"), nil)

	emitter, items := streaming.NewChannelEmitter(64)
	ctx := streaming.WithEmitter(context.Background(), emitter)
	res, err := New(mc).NameResolutions(ctx, mustDomain(t, "censys.com"), baseParams())
	require.NoError(t, err)
	require.Empty(t, res.Records)

	emitter.Close(nil)
	count := 0
	for item := range items {
		// Close sends a final Done item; it is not a record.
		if item.Done {
			break
		}
		count++
	}
	require.Equal(t, 2, count)
}

// TestLookups_403MapsToPlanMessage checks each of the four service methods
// maps a 403 from its client method to the plan-requirement message. Per
// controller ruling C6.
func TestLookups_403MapsToPlanMessage(t *testing.T) {
	testCases := []struct {
		name string
		run  func(ctrl *gomock.Controller) error
	}{
		{
			name: "NameResolutions",
			run: func(ctrl *gomock.Controller) error {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSNameResolutionBoundResponse]{}, apiError(403))
				_, err := New(mc).NameResolutions(context.Background(), mustDomain(t, "censys.com"), baseParams())
				return err
			},
		},
		{
			name: "NameResolutionRanges",
			run: func(ctrl *gomock.Controller) error {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSNameResolutionRanges(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSNameResolutionRangeResponse]{}, apiError(403))
				_, err := New(mc).NameResolutionRanges(context.Background(), mustDomain(t, "censys.com"), baseParams())
				return err
			},
		},
		{
			name: "IPResolutions",
			run: func(ctrl *gomock.Controller) error {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSIPResolutionBounds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSIPResolutionBoundResponse]{}, apiError(403))
				_, err := New(mc).IPResolutions(context.Background(), mustHost(t, "104.18.10.84"), baseParams())
				return err
			},
		},
		{
			name: "IPResolutionRanges",
			run: func(ctrl *gomock.Controller) error {
				mc := mocks.NewMockClient(ctrl)
				mc.EXPECT().ListDNSIPResolutionRanges(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.DNSIPResolutionRangeResponse]{}, apiError(403))
				_, err := New(mc).IPResolutionRanges(context.Background(), mustHost(t, "104.18.10.84"), baseParams())
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			err := tc.run(ctrl)
			require.Error(t, err)
			require.Contains(t, err.Error(), "Active DNS is available only on the Censys Search and Core plans")
		})
	}
}
