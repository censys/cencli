package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// nameResult is a name lookup result for input with one A record per IP.
func nameResult(input string, ips ...string) dnsapp.NameResolutionsResult {
	records := make([]*components.DNSResolutionRecord, 0, len(ips))
	for _, ip := range ips {
		records = append(records, &components.DNSResolutionRecord{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr(ip), FirstSeen: testFrom, LastSeen: testTo})
	}
	return dnsapp.NameResolutionsResult{Meta: testMeta(), Records: wrapName(input, records), TotalRecords: int64(len(ips))}
}

// ipResult is an IP lookup result for input with one A record per domain.
func ipResult(input string, domains ...string) dnsapp.IPResolutionsResult {
	records := make([]*components.DNSIPResolutionRecord, 0, len(domains))
	for _, d := range domains {
		records = append(records, &components.DNSIPResolutionRecord{Domain: d, RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: testTo})
	}
	return dnsapp.IPResolutionsResult{Meta: testMeta(), Records: wrapIP(input, records), TotalRecords: int64(len(domains))}
}

// wrapName, wrapNameRanges, and wrapIP wrap SDK records with their input, as
// the service does.
func wrapName(input string, records []*components.DNSResolutionRecord) []*dnsapp.NameRecord {
	out := make([]*dnsapp.NameRecord, 0, len(records))
	for _, r := range records {
		out = append(out, &dnsapp.NameRecord{Input: input, DNSResolutionRecord: r})
	}
	return out
}

func wrapNameRanges(input string, records []*components.DNSResolutionRangeRecord) []*dnsapp.NameRangeRecord {
	out := make([]*dnsapp.NameRangeRecord, 0, len(records))
	for _, r := range records {
		out = append(out, &dnsapp.NameRangeRecord{Input: input, DNSResolutionRangeRecord: r})
	}
	return out
}

func wrapIP(input string, records []*components.DNSIPResolutionRecord) []*dnsapp.IPRecord {
	out := make([]*dnsapp.IPRecord, 0, len(records))
	for _, r := range records {
		out = append(out, &dnsapp.IPRecord{Input: input, DNSIPResolutionRecord: r})
	}
	return out
}

// writeInputFile writes content to a temporary file and returns its path.
func writeInputFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inputs.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
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
	setup func()
	args  []string
	// stdin feeds the command's input, for --input-file -.
	stdin  string
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
			if tc.stdin != "" {
				rootCmd.SetIn(bytes.NewBufferString(tc.stdin))
			}
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
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com", "104.18.10.84"), nil)
				return ms
			},
			args: withWindow("censys.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				var got []map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				require.Len(t, got, 1)
				require.Equal(t, "104.18.10.84", got[0]["ip"])
				require.Equal(t, "censys.com", got[0]["input"])
			},
		},
		{
			name: "success - ip uses IPResolutions",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).Return(ipResult("104.18.10.84", "censys.com"), nil)
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
					Return(dnsapp.NameResolutionRangesResult{Meta: testMeta(), Records: []*dnsapp.NameRangeRecord{}}, nil)
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
					Return(dnsapp.IPResolutionRangesResult{Meta: testMeta(), Records: []*dnsapp.IPRangeRecord{}}, nil)
				return ms
			},
			args:   withWindow("104.18.10.84", "-t", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - defanged ip is an ip",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult("8.8.8.8"), nil)
				return ms
			},
			args:   withWindow("8.8.8[.]8", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - pasted url is a name",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("https://Censys.com/some/path", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - ip in a url is an ip",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult("8.8.8.8"), nil)
				return ms
			},
			args:   withWindow("https://8.8.8.8/", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - defanged url with an ip is an ip",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "1.2.3.4"), defaultParams()).Return(ipResult("1.2.3.4"), nil)
				return ms
			},
			args:   withWindow("hxxp://1.2.3[.]4/x", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - defanged name is a name",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("censys[.]com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - a url with a numeric path is not mistaken for a cidr",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult("8.8.8.8"), nil)
				return ms
			},
			args:   withWindow("https://8.8.8.8/32", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name:   "error - a cidr range is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("8.8.8.8/32"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "a CIDR range is not supported; give one IP address")
			},
		},
		{
			name:   "error - a cidr range with a smaller prefix is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("10.0.0.0/24"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "a CIDR range is not supported; give one IP address")
			},
		},
		{
			name:   "error - a defanged cidr range is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("8[.]8[.]8[.]8/32"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "a CIDR range is not supported; give one IP address")
			},
		},
		{
			name:   "error - a partially defanged cidr range is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("10.0.0[.]0/24"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "a CIDR range is not supported; give one IP address")
			},
		},
		{
			name: "success - record types and pagination flags pass through",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				p := defaultParams()
				p.RecordTypes = []string{"a", "mx"}
				p.PageSize = mo.Some[uint64](50)
				p.MaxPages = mo.Some[uint64](2)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), p).Return(nameResult("censys.com"), nil)
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
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), p).Return(nameResult("censys.com"), nil)
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
					Return(dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*dnsapp.NameRecord{}}, nil)
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
						for _, r := range nameResult("censys.com", "1.1.1.1", "2.2.2.2").Records {
							require.NoError(t, streaming.Emit(ctx, r))
						}
						return dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*dnsapp.NameRecord{}, TotalRecords: 2}, nil
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
				for _, line := range lines {
					require.Contains(t, line, `"input":"censys.com"`)
				}
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
			name: "success - a url with a comma in its path is one input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("https://censys.com/a,b", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - a defanged url with a comma is one input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("hxxps://censys[.]com/a,b", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - a bracketed-scheme defanged name is a name",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("hxxps[://]censys[.]com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - a bracketed-scheme url with a comma is one input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com"), nil)
				return ms
			},
			args:   withWindow("hxxps[://]censys.com/a,b", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - a bracketed-scheme defanged url with an ip is an ip",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult("8.8.8.8"), nil)
				return ms
			},
			args:   withWindow("hxxp[://]8.8.8[.]8/x", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
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
				require.Contains(t, err.Error(), "you must provide at least one asset")
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
			name: "success - --domain narrows an ip timeline",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				p := defaultParams()
				p.Domain = mo.Some(domainName(t, "censys.com"))
				ms.EXPECT().IPResolutionRanges(gomock.Any(), hostID(t, "104.18.10.84"), p).
					Return(dnsapp.IPResolutionRangesResult{Meta: testMeta(), Records: []*dnsapp.IPRangeRecord{}}, nil)
				return ms
			},
			args:   withWindow("104.18.10.84", "--timeline", "--domain", "censys.com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - --domain normalizes a defanged name",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				p := defaultParams()
				p.Domain = mo.Some(domainName(t, "censys.com"))
				ms.EXPECT().IPResolutionRanges(gomock.Any(), hostID(t, "104.18.10.84"), p).
					Return(dnsapp.IPResolutionRangesResult{Meta: testMeta(), Records: []*dnsapp.IPRangeRecord{}}, nil)
				return ms
			},
			args:   withWindow("104.18.10.84", "--timeline", "--domain", "censys[.]com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name:   "error - --domain with a name input is rejected",
			dnsSvc: noCalls,
			args:   withWindow("censys.com", "--timeline", "--domain", "x.com"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--domain applies only to an IP lookup with --timeline")
			},
		},
		{
			name:   "error - --domain without --timeline is rejected",
			dnsSvc: noCalls,
			args:   withWindow("104.18.10.84", "--domain", "censys.com"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--domain applies only to an IP lookup with --timeline")
			},
		},
		{
			name:   "error - --domain with an invalid name is rejected",
			dnsSvc: noCalls,
			args:   withWindow("104.18.10.84", "--timeline", "--domain", "notadomain"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid asset ID: notadomain")
				require.Contains(t, err.Error(), "a domain name must contain a dot")
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
				require.Contains(t, stdout, "141.193.213.10 --timeline --domain censys.com")
				require.Contains(t, stdout, "dns <name|ip>...")
				require.Contains(t, stdout, "censys.com,104.18.10.84")
				require.Contains(t, stdout, "--input-file iocs.txt")
				require.Contains(t, stdout, "--input-file - # read inputs from STDIN")
			},
		},
	})
}

// TestDNSCommand_RecordTypeValidatedBeforeClient covers an invalid --record-type
// with neither a DNS service nor a client configured: PreRun must reject the
// record type before it ever needs a service, so this fails with the
// record-type error instead of "Censys Client Not Configured".
func TestDNSCommand_RecordTypeValidatedBeforeClient(t *testing.T) {
	tempDir := t.TempDir()
	viper.Reset()
	cfg, err := config.New(tempDir)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// No WithDNSService and no client: DNSService() would fail with "not
	// configured" if it were ever called.
	cmdContext := command.NewCommandContext(cfg, newOrgLookupStore(ctrl))
	rootCmd, err := command.RootCommandToCobra(NewDNSCommand(cmdContext))
	require.NoError(t, err)

	rootCmd.SetArgs(withWindow("104.18.10.84", "-r", "MX"))
	cmdErr := rootCmd.Execute()
	require.Error(t, cmdErr)
	require.Contains(t, cmdErr.Error(), "invalid record type 'MX'")
	require.NotContains(t, cmdErr.Error(), "not configured")
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
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("aspmx.l.google.com"), Priority: &mx, FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr("104.18.10.84"), FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeSoa, Mname: strPtr("ns1.example.net"), Rname: strPtr("dns.example.net"), FirstSeen: testFrom, LastSeen: testTo},
					}),
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
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("b.example"), Priority: &ten, FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("a.example"), Priority: &five, FirstSeen: testFrom, LastSeen: testTo},
						// A record type the SDK does not define yet; the table must still
						// place it after every known type instead of first (slices.Index's -1).
						{RecordType: components.DNSResolutionRecordRecordType("CAA"), FirstSeen: testFrom, LastSeen: testTo},
					}),
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
					Records: wrapIP("104.18.10.84", []*components.DNSIPResolutionRecord{
						{Domain: "old.example.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: older},
						{Domain: "new.example.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: testTo},
					}),
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
					Records: wrapNameRanges("censys.com", []*components.DNSResolutionRangeRecord{
						{RecordType: components.DNSResolutionRangeRecordRecordTypeA, IP: strPtr("2.2.2.2"), FirstObserved: testFrom.Add(time.Hour), LastObserved: testTo},
						{RecordType: components.DNSResolutionRangeRecordRecordTypeA, IP: strPtr("1.1.1.1"), FirstObserved: testFrom, LastObserved: testTo},
					}),
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
				r := nameResult("censys.com", "1.1.1.1", "2.2.2.2")
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
			name: "success - no note when --max-pages -1 already fetched everything",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				r := nameResult("censys.com", "1.1.1.1", "2.2.2.2")
				r.TotalRecords = 5321
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(r, nil)
				return ms
			},
			args: withWindow("censys.com", "-p", "-1"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "DNS records for censys.com (2 of 5321)")
				require.NotContains(t, stderr, "Showing")
			},
		},
		{
			name: "success - no note when a partial error already explains the truncation",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				r := nameResult("censys.com", "1.1.1.1", "2.2.2.2")
				r.TotalRecords = 5321
				r.PartialError = cenclierrors.ToPartialError(cenclierrors.NewCencliError(errors.New("page 2 failed")))
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(r, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "DNS records for censys.com (2 of 5321)")
				require.NotContains(t, stderr, "Showing")
				require.Contains(t, stderr, "page 2 failed")
			},
		},
		{
			name: "success - a TXT value with control characters does not break the table row",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeTxt, Value: strPtr("a\nb\tc\rd"), FirstSeen: testFrom, LastSeen: testTo},
					}),
					TotalRecords: 1,
				}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Len(t, lines, 4, "the TXT value must not introduce extra lines")
				require.Equal(t, []string{"TXT", "a", "b", "c", "d", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[3])
			},
		},
		{
			name: "success - a TXT value with an ESC byte renders without it",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeTxt, Value: strPtr("\x1b[31mred\x1b[0m"), FirstSeen: testFrom, LastSeen: testTo},
					}),
					TotalRecords: 1,
				}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stdout, "\x1b")
				require.Contains(t, stdout, "red")
			},
		},
		{
			name: "success - a TXT value padded with control characters still shows the visible text after truncation",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeTxt, Value: strPtr(strings.Repeat("\x1b", 60) + "hello"), FirstSeen: testFrom, LastSeen: testTo},
					}),
					TotalRecords: 1,
				}, nil)
				return ms
			},
			args: withWindow("censys.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stdout, "\x1b")
				require.Contains(t, stdout, "hello")
			},
		},
		{
			name: "success - a domain with ESC and BEL bytes renders without them",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.IPResolutionsResult{
					Meta: testMeta(),
					Records: wrapIP("104.18.10.84", []*components.DNSIPResolutionRecord{
						{Domain: "\x1b]8;;http://x\x07", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: testFrom, LastSeen: testTo},
					}),
					TotalRecords: 1,
				}, nil)
				return ms
			},
			args: withWindow("104.18.10.84"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stdout, "\x1b")
				require.NotContains(t, stdout, "\x07")
			},
		},
		{
			name: "success - truncation note is suppressed by --quiet",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				r := nameResult("censys.com", "1.1.1.1")
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
					Return(dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*dnsapp.NameRecord{}}, nil)
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

func TestDNSCommand_Template(t *testing.T) {
	mx := int64(10)
	runDNSTestCases(t, []dnsTestCase{
		{
			name: "success - name records render input, type, value, and seen dates",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionsResult{
					Meta: testMeta(),
					Records: wrapName("censys.com", []*components.DNSResolutionRecord{
						{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr("104.18.10.84"), FirstSeen: testFrom, LastSeen: testTo},
						{RecordType: components.DNSResolutionRecordRecordTypeMx, MailServer: strPtr("aspmx.l.google.com"), Priority: &mx, FirstSeen: testFrom, LastSeen: testTo},
					}),
					TotalRecords: 2,
				}, nil)
				return ms
			},
			args: withWindow("censys.com", "-O", "template"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "censys.com")
				require.Contains(t, stdout, "[A]")
				require.Contains(t, stdout, "Value: 104.18.10.84")
				require.Contains(t, stdout, "[MX]")
				require.Contains(t, stdout, "Value: 10 aspmx.l.google.com")
				require.Contains(t, stdout, "First Seen: "+testFrom.Format(time.RFC3339))
				require.Contains(t, stdout, "Last Seen: "+testTo.Format(time.RFC3339))
			},
		},
		{
			name: "success - ip records render input, type, domain, and seen dates",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().IPResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(ipResult("104.18.10.84", "censys.com"), nil)
				return ms
			},
			args: withWindow("104.18.10.84", "-O", "template"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "104.18.10.84")
				require.Contains(t, stdout, "[A]")
				require.Contains(t, stdout, "Value: censys.com")
				require.Contains(t, stdout, "First Seen: "+testFrom.Format(time.RFC3339))
				require.Contains(t, stdout, "Last Seen: "+testTo.Format(time.RFC3339))
			},
		},
		{
			name: "success - name timeline records use first/last observed",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutionRanges(gomock.Any(), gomock.Any(), gomock.Any()).Return(dnsapp.NameResolutionRangesResult{
					Meta: testMeta(),
					Records: wrapNameRanges("censys.com", []*components.DNSResolutionRangeRecord{
						{RecordType: components.DNSResolutionRangeRecordRecordTypeA, IP: strPtr("1.1.1.1"), FirstObserved: testFrom, LastObserved: testTo},
					}),
					TotalRecords: 1,
				}, nil)
				return ms
			},
			args: withWindow("censys.com", "--timeline", "-O", "template"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "First Observed: "+testFrom.Format(time.RFC3339))
				require.Contains(t, stdout, "Last Observed: "+testTo.Format(time.RFC3339))
				require.NotContains(t, stdout, "First Seen:")
			},
		},
		{
			name: "success - no records renders nothing",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*dnsapp.NameRecord{}}, nil)
				return ms
			},
			args: withWindow("censys.com", "-O", "template"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Empty(t, strings.TrimSpace(stdout))
			},
		},
	})
}

// jsonInputs decodes a JSON array of records and returns each record's input.
func jsonInputs(t *testing.T, stdout string) []string {
	t.Helper()
	var got []map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	inputs := make([]string, 0, len(got))
	for _, r := range got {
		input, ok := r["input"].(string)
		require.True(t, ok, "every record has an input")
		inputs = append(inputs, input)
	}
	return inputs
}

// apiError is a hard (first-page) lookup error that is not a 403.
func apiError(msg string) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(errors.New(msg))
}

func TestDNSCommand_MultipleInputs(t *testing.T) {
	twoLines := writeInputFile(t, "a.com\nb.com,104.18.10.84\n")
	tooMany := make([]string, 0, maxInputs+1)
	for i := 0; i <= maxInputs; i++ {
		tooMany = append(tooMany, fmt.Sprintf("host%d.example.com", i))
	}

	runDNSTestCases(t, []dnsTestCase{
		{
			name: "success - a comma list looks up each name in order",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), defaultParams()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com", "b.com"}, jsonInputs(t, stdout))
			},
		},
		{
			name: "success - a list mixes name and ip lookups",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), defaultParams()).Return(nameResult("censys.com", "104.18.10.84"), nil),
					ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).Return(ipResult("104.18.10.84", "censys.com"), nil),
				)
				return ms
			},
			args: withWindow("censys.com,104.18.10.84", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"censys.com", "104.18.10.84"}, jsonInputs(t, stdout))
			},
		},
		{
			name: "success - several positional arguments",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), defaultParams()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com", "b.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com", "b.com"}, jsonInputs(t, stdout))
			},
		},
		{
			name: "success - --input-file reads one input or a comma list per line",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), defaultParams()).Return(nameResult("b.com", "2.2.2.2"), nil),
					ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).Return(ipResult("104.18.10.84", "censys.com"), nil),
				)
				return ms
			},
			args: withWindow("--input-file", twoLines, "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com", "b.com", "104.18.10.84"}, jsonInputs(t, stdout))
			},
		},
		{
			name: "success - --input-file overrides the positional arguments",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), defaultParams()).Return(nameResult("b.com"), nil),
					ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), defaultParams()).Return(ipResult("104.18.10.84"), nil),
				)
				return ms
			},
			args:   withWindow("ignored.com", "-i", twoLines, "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) { require.NoError(t, err) },
		},
		{
			name: "success - --input-file - reads inputs from stdin",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "8.8.8.8"), defaultParams()).Return(ipResult("8.8.8.8", "dns.google"), nil),
				)
				return ms
			},
			stdin: "a.com\n8.8.8[.]8\n",
			args:  withWindow("-i", "-", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com", "8.8.8.8"}, jsonInputs(t, stdout))
			},
		},
		{
			name: "success - duplicate inputs are looked up once",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), defaultParams()).Return(nameResult("a.com", "1.1.1.1"), nil)
				return ms
			},
			args: withWindow("a.com,a.com", "a[.]com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com"}, jsonInputs(t, stdout))
			},
		},
		{
			name:   "error - more than the maximum inputs is too many assets",
			dnsSvc: noCalls,
			args:   withWindow(strings.Join(tooMany, ",")),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "101 assets provided, only 100 are supported")
			},
		},
		{
			name:   "error - only blank inputs is no assets",
			dnsSvc: noCalls,
			args:   withWindow(" , "),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "you must provide at least one asset")
			},
		},
		{
			name:   "error - an invalid input among valid ones is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("censys.com,a..b.com"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid asset ID: a..b.com")
				require.Contains(t, err.Error(), "empty label")
			},
		},
		{
			name:   "error - an MX filter with an ip in the list is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("censys.com,104.18.10.84", "-r", "MX"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid record type 'MX'")
			},
		},
		{
			name:   "error - --domain with a name in the list is rejected before any API call",
			dnsSvc: noCalls,
			args:   withWindow("104.18.10.84,censys.com", "--timeline", "--domain", "x.com"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--domain applies only to an IP lookup with --timeline")
			},
		},
		{
			name: "success - one input's error does not stop the others",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, apiError("upstream failed")),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"b.com"}, jsonInputs(t, stdout))
				require.Contains(t, stderr, "a.com: upstream failed")
			},
		},
		{
			name: "error - every input failing returns the first error",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, apiError("first failed")),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, apiError("second failed")),
				)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, _, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "a.com: first failed")
				require.Contains(t, stderr, "b.com: second failed")
			},
		},
		{
			name: "error - a 403 stops the remaining lookups",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, dnsapp.NewAccessDeniedError())
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Search and Core plans")
			},
		},
		{
			name: "success - a 403 after a success still stops the remaining lookups",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, dnsapp.NewAccessDeniedError()),
				)
				return ms
			},
			args: withWindow("a.com,b.com,c.com", "-O", "json"),
			assert: func(t *testing.T, _, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Search and Core plans")
			},
		},
		{
			name: "success - a partial error for one input is printed with the others' data",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				partial := nameResult("a.com", "1.1.1.1")
				partial.PartialError = cenclierrors.ToPartialError(apiError("page 2 failed"))
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(partial, nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"a.com", "b.com"}, jsonInputs(t, stdout))
				require.Contains(t, stderr, "page 2 failed")
			},
		},
		{
			name: "success - no data for any input prints an empty JSON list",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nameResult("a.com"), nil).Times(2)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "json"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, "[]", strings.TrimSpace(stdout))
			},
		},
		{
			name: "success - yaml records carry their input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(nameResult("a.com", "1.1.1.1"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com", "-O", "yaml"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "input: a.com")
				require.Contains(t, stdout, "input: b.com")
			},
		},
		{
			name: "success - streaming emits every input's records with their input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				emit := func(input string) func(context.Context, assets.DomainName, dnsapp.Params) (dnsapp.NameResolutionsResult, cenclierrors.CencliError) {
					return func(ctx context.Context, _ assets.DomainName, _ dnsapp.Params) (dnsapp.NameResolutionsResult, cenclierrors.CencliError) {
						for _, r := range nameResult(input, "1.1.1.1").Records {
							require.NoError(t, streaming.Emit(ctx, r))
						}
						return dnsapp.NameResolutionsResult{Meta: testMeta(), Records: []*dnsapp.NameRecord{}, TotalRecords: 1}, nil
					}
				}
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).DoAndReturn(emit("a.com")),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).DoAndReturn(emit("b.com")),
				)
				return ms
			},
			setup: func() { viper.Set("streaming", true) },
			args:  withWindow("a.com,b.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				require.Len(t, lines, 2)
				require.Contains(t, lines[0], `"input":"a.com"`)
				require.Contains(t, lines[1], `"input":"b.com"`)
			},
		},
		{
			name: "success - max pages -1 warning names the input count",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				ms.EXPECT().NameResolutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nameResult("a.com"), nil).Times(2)
				return ms
			},
			args: withWindow("a.com,b.com", "-p", "-1", "-O", "json"),
			assert: func(t *testing.T, _, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "fetching all pages (--max-pages=-1) for each of 2 inputs")
			},
		},
	})
}

func TestDNSCommand_ShortMultipleInputs(t *testing.T) {
	runDNSTestCases(t, []dnsTestCase{
		{
			name: "success - one section per input, window once",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "censys.com"), gomock.Any()).Return(nameResult("censys.com", "104.18.10.84"), nil),
					ms.EXPECT().IPResolutions(gomock.Any(), hostID(t, "104.18.10.84"), gomock.Any()).Return(ipResult("104.18.10.84", "censys.com"), nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "empty.example.com"), gomock.Any()).Return(nameResult("empty.example.com"), nil),
				)
				return ms
			},
			args: withWindow("censys.com,104.18.10.84,empty.example.com"),
			assert: func(t *testing.T, stdout, _ string, err error) {
				require.NoError(t, err)
				lines := fields(stdout)
				require.Equal(t, []string{"Active", "2026-09-21", "00:00", "→", "2026-09-28", "00:00", "UTC"}, lines[0])
				require.Equal(t, []string{"DNS", "records", "for", "censys.com", "(1)"}, lines[1])
				require.Equal(t, []string{"Type", "Value", "First", "Seen", "Last", "Seen"}, lines[2])
				require.Equal(t, []string{"A", "104.18.10.84", "2026-09-21", "00:00", "2026-09-28", "00:00"}, lines[3])
				require.Equal(t, []string{"Domains", "resolving", "to", "104.18.10.84", "(1)"}, lines[4])
				require.Equal(t, []string{"Domain", "Type", "First", "Seen", "Last", "Seen"}, lines[5])
				require.Equal(t, "censys.com", lines[6][0])
				require.Equal(t, []string{"DNS", "records", "for", "empty.example.com", "(0)"}, lines[7])
				require.Equal(t, "No", lines[8][0])
				require.Len(t, lines, 9)
				require.Equal(t, 1, strings.Count(stdout, "Active 2026-09-21"), "the window is printed once")
				require.Contains(t, stdout, "No DNS records found in this window. Widen it with --duration (e.g. -d 90d).")
				require.Contains(t, stdout, "\n\nDomains resolving to 104.18.10.84 (1)", "sections are separated by a blank line")
			},
		},
		{
			name: "success - the truncation note names the input",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				truncated := nameResult("a.com", "1.1.1.1")
				truncated.TotalRecords = 50
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(truncated, nil),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "DNS records for a.com (1 of 50)")
				require.Contains(t, stderr, "Showing 1 of 50 records for a.com. Use --max-pages -1 to fetch all.")
				require.Equal(t, 1, strings.Count(stderr, "Showing"))
			},
		},
		{
			name: "success - a failed input has no section",
			dnsSvc: func(t *testing.T, ctrl *gomock.Controller) dnsapp.Service {
				ms := dnsmocks.NewMockDNSService(ctrl)
				gomock.InOrder(
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "a.com"), gomock.Any()).Return(dnsapp.NameResolutionsResult{}, apiError("upstream failed")),
					ms.EXPECT().NameResolutions(gomock.Any(), domainName(t, "b.com"), gomock.Any()).Return(nameResult("b.com", "2.2.2.2"), nil),
				)
				return ms
			},
			args: withWindow("a.com,b.com"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stdout, "a.com")
				require.Contains(t, stdout, "DNS records for b.com (1)")
				require.Contains(t, stderr, "a.com: upstream failed")
			},
		},
	})
}
