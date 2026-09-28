package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/censys-sdk-go/models/components"

	dnsmocks "github.com/censys/cencli/gen/app/dns/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	dnsapp "github.com/censys/cencli/internal/app/dns"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

var (
	testFrom   = time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	testTo     = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	windowArgs = []string{"--start", "2026-09-21T00:00:00Z", "--end", "2026-09-28T00:00:00Z"}
)

func strPtr(s string) *string { return &s }

func testMeta() *responsemeta.ResponseMeta {
	return &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200}
}

// defaultParams is what the command passes with windowArgs and no other flags.
// An unset --record-type gives an empty, non-nil slice (flags/slice.go:57).
func defaultParams() dnsapp.Params {
	return dnsapp.Params{
		OrgID:       mo.None[identifiers.OrganizationID](),
		FromTime:    testFrom,
		ToTime:      testTo,
		RecordTypes: []string{},
		PageSize:    mo.Some[uint64](100),
		MaxPages:    mo.Some[uint64](10),
	}
}

func domainName(t *testing.T, raw string) assets.DomainName {
	t.Helper()
	d, err := assets.NewDomainName(raw)
	require.NoError(t, err)
	return d
}

func hostID(t *testing.T, raw string) assets.HostID {
	t.Helper()
	h, err := assets.NewHostID(raw)
	require.NoError(t, err)
	return h
}

func nameResult(ips ...string) dnsapp.NameResolutionsResult {
	records := make([]*components.DNSResolutionRecord, 0, len(ips))
	for _, ip := range ips {
		records = append(records, &components.DNSResolutionRecord{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr(ip), FirstSeen: testFrom, LastSeen: testTo})
	}
	return dnsapp.NameResolutionsResult{Meta: testMeta(), Records: records, TotalRecords: int64(len(ips))}
}

func ipResult(domains ...string) dnsapp.IPResolutionsResult {
	records := make([]*components.DNSIPResolutionRecord, 0, len(domains))
	for _, d := range domains {
		records = append(records, &components.DNSIPResolutionRecord{Domain: d, RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: testTo})
	}
	return dnsapp.IPResolutionsResult{Meta: testMeta(), Records: records, TotalRecords: int64(len(domains))}
}

// withWindow appends the fixed test window to the given arguments.
func withWindow(extra ...string) []string {
	return append(append([]string{}, extra...), windowArgs...)
}

type dnsTestCase struct {
	name   string
	dnsSvc func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service
	// setup runs after config.New. Global flags (--streaming, --quiet) live on the
	// real root command only, so tests set them through viper.
	setup  func()
	args   []string
	assert func(t *testing.T, stdout, stderr string, err error)
}

func runDNSTestCases(t *testing.T, testCases []dnsTestCase) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)
			if tc.setup != nil {
				tc.setup()
			}

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cmdContext := command.NewCommandContext(cfg, newOrgLookupStore(ctrl), command.WithDNSService(tc.dnsSvc(t, ctrl)))
			rootCmd, err := command.RootCommandToCobra(NewDNSCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func noCalls(_ *testing.T, ctrl *gomock.Controller) dnsapp.Service {
	return dnsmocks.NewMockDNSService(ctrl)
}

func TestDNSCommand(t *testing.T) {
	runDNSTestCases(t, []dnsTestCase{
		{
			name: "success - name uses NameResolutions",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("104.18.10.84"), nil)
				return ms
			},
			args: withWindow("censys.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				var got []map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				require.Len(t, got, 1)
				require.Equal(t, "104.18.10.84", got[0]["ip"])
			},
		},
		{
			name: "success - ip uses IPResolutions",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).Return(ipResult("censys.com"), nil)
				return ms
			},
			args: withWindow("104.18.10.84", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"censys.com"`)
			},
		},
		{
			name: "success - name with --timeline uses NameResolutionRanges",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutionRanges(gomock.Any(), domainName(t, "censys.com"), defaultParams()).
					Return(dnsapp.NameResolutionRangesResult{Meta: testMeta(), Records: []*components.DNSResolutionRangeRecord{}}, nil)
				return ms
			},
			args:   withWindow("censys.com", "--timeline", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - ip with -t uses IPResolutionRanges",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutionRanges(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).
					Return(dnsapp.IPResolutionRangesResult{Meta: testMeta(), Records: []*components.DNSIPResolutionRangeRecord{}}, nil)
				return ms
			},
			args:   withWindow("104.18.10.84", "-t", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - defanged ip is an ip",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult(), nil)
				return ms
			},
			args:   withWindow("8.8.8[.]8", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - pasted url is a name",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult(), nil)
				return ms
			},
			args:   withWindow("https://Censys.com/some/path", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - record types and pagination flags pass through",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				p := defaultParams()
				p.RecordTypes = []string{"a", "mx"}
				p.PageSize = mo.Some[uint64](50)
				p.MaxPages = mo.Some[uint64](2)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), p).Return(nameResult(), nil)
				return ms
			},
			args:   withWindow("censys.com", "-r", "a,mx", "-n", "50", "-p", "2", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - max pages -1 fetches all and warns",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				p := defaultParams()
				p.MaxPages = mo.None[uint64]()
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), p).Return(nameResult(), nil)
				return ms
			},
			args: withWindow("censys.com", "-p", "-1", "-O", "json"),
			assert: func(t *testing.T, _, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "fetching all pages")
			},
		},
		{
			name: "success - no data prints an empty JSON list",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*components.DNSResolutionRecord{}}, nil)
				return ms
			},
			args: withWindow("censys.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, "[]", strings.TrimSpace(stdout))
			},
		},
		{
			name: "success - streaming emits one line per record",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ assets.DomainName, _ dnsapp.Params) (dnsapp.NameResolutionsResult, cenclierrors.CencliError) {
						for _, r := range nameResult("1.1.1.1", "2.2.2.2").Records {
							require.NoError(t, streaming.Emit(ctx, r))
						}
						return dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*components.DNSResolutionRecord{}, TotalRecords: 2}, nil
					})
				return ms
			},
			setup: func() { viper.Set("streaming", true) },
			args:  withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				require.Len(t, lines, 2)
				require.Contains(t, lines[0], `"1.1.1.1"`)
			},
		},
		{
			name:   "error - host:port is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("censys.com:443"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid asset ID: censys.com:443")
				require.Contains(t, err.Error(), "remove the port")
			},
		},
		{
			name:   "error - bracketed ipv6 is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("[2001:db8::1]"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "remove the brackets")
			},
		},
		{
			name:   "error - a list is too many assets",
			dnsSvc: noCalls,
			args:   withWindow("a.com,b.com"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "2 assets provided, only 1 are supported")
			},
		},
		{
			name:   "error - end before start is rejected",
			dnsSvc: noCalls,
			args:   []string{"censys.com", "--start", "2026-09-28T00:00:00Z", "--end", "2026-09-21T00:00:00Z"},
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid time window")
			},
		},
		{
			name:   "error - max pages 0 is rejected",
			dnsSvc: noCalls,
			args:   withWindow("censys.com", "-p", "0"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be -1 or >= 1")
			},
		},
		{
			name:   "error - no argument is rejected",
			dnsSvc: noCalls,
			args:   []string{},
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
			},
		},
		{
			name: "error - 403 error is returned",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, dnsapp.NewAccessDeniedError())
				return ms
			},
			args: withWindow("censys.com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Search and Core plans")
			},
		},
		{
			name:   "success - help",
			dnsSvc: noCalls,
			args:   []string{"--help"},
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"censys view <ip>"`)
				require.Contains(t, stdout, "--timeline")
			},
		},
	})
}

// newOrgLookupStore returns a store mock that reports no stored org-id, which is
// what ResolveOrgID consults for a personal-access-token credential.
func newOrgLookupStore(ctrl *gomock.Controller) store.Store {
	ms := storemocks.NewMockStore(ctrl)
	ms.EXPECT().GetLastUsedGlobalByName(gomock.Any(), gomock.Any()).
		Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound).AnyTimes()
	return ms
}

// fields splits the output into lines of whitespace-separated fields, so the
// assertions do not depend on the table's column padding or on rawtable's " | "
// separator between data cells (header cells are separated by plain padding
// instead, so no filtering is needed there).
func fields(out string) [][]string {
	var lines [][]string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var f []string
		for _, tok := range strings.Fields(line) {
			if tok == "|" {
				continue
			}
			f = append(f, tok)
		}
		if len(f) > 0 {
			lines = append(lines, f)
		}
	}
	return lines
}

func TestDNSCommand_Short(t *testing.T) {
	mx := int64(10)
	runDNSTestCases(t, []dnsTestCase{
		{
			name: "success - name table is sorted by type and shows the window",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("aspmx.l.google.com"), Priority: &mx, FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr("104.18.10.84"), FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeSoa, Mname: strPtr("ns1.example.net"), Rname: strPtr("dns.example.net"), FirstSeen: testFrom, LastSeen: testTo},
					},
					TotalRecords: 3,
				}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Equal(t, []string{"DNS", "records", "for", "censys.com", "(3)"}, lines[0])
				require.Equal(t, []string{"Active", "2026-09-21", "00:00", "→", "2026-09-28", "00:00", "UTC"}, lines[1])
				require.Equal(t, []string{"Type", "Value", "First", "Seen", "Last", "Seen"}, lines[2])
				require.Equal(t, []string{"A", "104.18.10.84", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[3])
				require.Equal(t, []string{"MX", "10", "aspmx.l.google.com", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[4])
				require.Equal(t, []string{"SOA", "ns1.example.net", "dns.example.net", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[5])
				require.NotContains(t, stderr, "Showing")
			},
		},
		{
			name: "success - MX rows sort by numeric priority, and an unrecognized record type sorts last",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ten, five := int64(10), int64(5)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("b.example"), Priority: &ten, FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("a.example"), Priority: &five, FirstSeen: testFrom, LastSeen: testTo},
						// A record type the SDK does not define yet; the table must still
						// place it after every known type instead of first (slices.Index's -1).
						{RecordType: components.DNSResolutionRecordRecordType("CAA"), FirstSeen: testFrom, LastSeen: testTo},
					},
					TotalRecords: 3,
				}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Equal(t, []string{"MX", "5", "a.example", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[3], "priority 5 sorts before priority 10")
				require.Equal(t, []string{"MX", "10", "b.example", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[4])
				require.Equal(t, "CAA", lines[5][0], "an unrecognized type sorts after every known type")
			},
		},
		{
			name: "success - ip table is sorted by last seen, newest first",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				older := testTo.Add(-48 * time.Hour)
				ms.EXPECT().IPResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.IPResolutionsResult{
					Meta: testMeta(),
					Records: []*components.DNSIPResolutionRecord{
						{Domain: "old.example.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: older},
						{Domain: "new.example.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: testTo},
					},
					TotalRecords: 2,
				}, nil)
				return ms
			},
			args: withWindow("104.18.10.84"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Equal(t, []string{"Domains", "resolving", "to", "104.18.10.84", "(2)"}, lines[0])
				require.Equal(t, []string{"Domain", "Type", "First", "Seen", "Last", "Seen"}, lines[2])
				require.Equal(t, "new.example.com", lines[3][0])
				require.Equal(t, "old.example.com", lines[4][0])
			},
		},
		{
			name: "success - timeline title and columns",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutionRanges(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionRangesResult{
					Meta: testMeta(),
					Records: []*components.DNSResolutionRangeRecord{
						{RecordType: components.DNSResolutionRangeRecordRecordTypeA, IP: strPtr("2.2.2.2"), FirstObserved: testFrom.Add(time.Hour), LastObserved: testTo},
						{RecordType: components.DNSResolutionRangeRecordRecordTypeA, IP: strPtr("1.1.1.1"), FirstObserved: testFrom, LastObserved: testTo},
					},
					TotalRecords: 2,
				}, nil)
				return ms
			},
			args: withWindow("censys.com", "--timeline"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Equal(t, []string{"DNS", "timeline", "for", "censys.com", "(2)"}, lines[0])
				require.Equal(t, []string{"Type", "Value", "First", "Observed", "Last", "Observed"}, lines[2])
				require.Equal(t, "1.1.1.1", lines[3][1], "oldest first")
			},
		},
		{
			name: "success - truncated result shows the count and a note",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				r := nameResult("1.1.1.1", "2.2.2.2")
				r.TotalRecords = 5321
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(r, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "DNS records for censys.com (2 of 5321)")
				require.Contains(t, stderr, "Showing 2 of 5321 records. Use --max-pages -1 to fetch all.")
			},
		},
		{
			name: "success - truncation note is suppressed by --quiet",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				r := nameResult("1.1.1.1")
				r.TotalRecords = 10
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(r, nil)
				return ms
			},
			setup: func() { viper.Set("quiet", true) },
			args:  withWindow("censys.com"),
			assert: func(t *testing.T, _, stderr string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stderr, "Showing")
			},
		},
		{
			name: "success - empty result explains how to widen the window",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*components.DNSResolutionRecord{}}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Active 2026-09-21 00:00 → 2026-09-28 00:00 UTC")
				require.Contains(t, stdout, "No DNS records found in this window. Widen it with --duration (e.g. -d 90d).")
			},
		},
	})
}
