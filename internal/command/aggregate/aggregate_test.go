package aggregate

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	aggregatemocks "github.com/censys/cencli/gen/app/aggregate/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/aggregate"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestAggregateCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) aggregate.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases - basic functionality
		{
			name: "success - basic query and field - no flags",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 1000},
						{Key: "443", Count: 800},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "443")
				require.Contains(t, stdout, "1000")
				require.Contains(t, stdout, "800")
			},
		},
		{
			name: "success - with org ID flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 50*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "US", Count: 500},
						{Key: "CA", Count: 200},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--org-id", "12345678-1234-1234-1234-123456789abc", "ip:1.1.1.1", "location.country"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "US")
				require.Contains(t, stdout, "CA")
				require.Contains(t, stdout, "500")
				require.Contains(t, stdout, "200")
			},
		},
		{
			name: "success - with collection ID flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 75*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "22", Count: 300},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--collection-id", "87654321-4321-4321-4321-cba987654321", "host.services.protocol:SSH", "host.services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "22")
				require.Contains(t, stdout, "300")
			},
		},
		{
			name: "success - with num-buckets flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 120*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 100},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--num-buckets", "50", "services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "100")
			},
		},
		{
			name: "success - with count-by-level flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 90*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 150},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--count-by-level", "service", "services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "150")
			},
		},
		{
			name: "success - with filter-by-query flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 110*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 250},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--filter-by-query", "services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "250")
			},
		},
		{
			name: "success - all flags combined",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 200*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "result1", Count: 400},
						{Key: "result2", Count: 300},
						{Key: "result3", Count: 200},
					},
				}, nil)
				return mockSvc
			},
			args: []string{
				"--org-id", "22222222-2222-2222-2222-222222222222",
				"--collection-id", "11111111-1111-1111-1111-111111111111",
				"--num-buckets", "100",
				"--count-by-level", "protocol",
				"--filter-by-query",
				"complex query",
				"complex.field",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "result1")
				require.Contains(t, stdout, "result2")
				require.Contains(t, stdout, "result3")
				require.Contains(t, stdout, "400")
				require.Contains(t, stdout, "300")
				require.Contains(t, stdout, "200")
			},
		},
		{
			name: "success - short flags",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 60*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "short", Count: 50},
					},
				}, nil)
				return mockSvc
			},
			args: []string{
				"-c", "33333333-3333-3333-3333-333333333333",
				"-n", "10",
				"-l", "host",
				"-f",
				"test query",
				"test.field",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "short")
				require.Contains(t, stdout, "50")
			},
		},

		// Error cases - argument validation
		{
			name: "error - no arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 2 arg(s), received 0")
			},
		},
		{
			name: "error - only one argument",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"query"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 2 arg(s), received 1")
			},
		},
		{
			name: "error - too many arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"query", "field", "extra"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 2 arg(s), received 3")
			},
		},

		// Error cases - flag validation
		{
			name: "error - invalid org ID format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--org-id", "invalid-uuid", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - invalid collection ID format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--collection-id", "not-a-uuid", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - num-buckets too small",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--num-buckets", "0", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be >= 1")
			},
		},
		{
			name: "error - num-buckets too large",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--num-buckets", "10001", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be <= 10000")
			},
		},
		{
			name: "error - num-buckets invalid format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--num-buckets", "not-a-number", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid syntax")
			},
		},

		// Error cases - service errors
		{
			name: "error - service returns error",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{}, cenclierrors.NewCencliError(errors.New("invalid query syntax")))
				return mockSvc
			},
			args: []string{"invalid query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid query syntax")
			},
		},

		// Edge cases - boundary values
		{
			name: "success - minimum num-buckets",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 40*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "single", Count: 1},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--num-buckets", "1", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "single")
				require.Contains(t, stdout, "1")
			},
		},
		{
			name: "success - maximum num-buckets",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 500*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "max", Count: 10000},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--num-buckets", "10000", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "max")
				require.Contains(t, stdout, "10000")
			},
		},

		// Special characters and edge cases in query/field
		{
			name: "success - query with special characters",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 80*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 100},
					},
				}, nil)
				return mockSvc
			},
			args: []string{`services.service_name:"HTTP/1.1" AND location.country_code:US`, "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "100")
			},
		},
		{
			name: "success - empty results",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta:    responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 30*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{},
				}, nil)
				return mockSvc
			},
			args: []string{"nonexistent:query", "nonexistent.field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should not contain any bucket data, but should complete successfully
				require.NotContains(t, stdout, "error")
			},
		},

		// Output format tests
		{
			name: "success - raw flag outputs JSON",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 1000},
						{Key: "443", Count: 800},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--raw", "services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain JSON output
				require.Contains(t, stdout, `"key"`)
				require.Contains(t, stdout, `"count"`)
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "443")
			},
		},
		{
			name: "success - raw flag short form outputs JSON",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "22", Count: 500},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"-r", "protocol:SSH", "port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain JSON output
				require.Contains(t, stdout, `"key"`)
				require.Contains(t, stdout, `"count"`)
				require.Contains(t, stdout, "22")
			},
		},
		{
			name: "success - default outputs raw table",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 1000},
						{Key: "443", Count: 800},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain table output (not JSON)
				require.Contains(t, stdout, "Aggregation Results")
				require.Contains(t, stdout, "query:")
				require.Contains(t, stdout, "80")
				require.Contains(t, stdout, "443")
				require.Contains(t, stdout, "1000")
				require.Contains(t, stdout, "800")
				// Should NOT be JSON format
				require.NotContains(t, stdout, `"key"`)
			},
		},

		// Header format tests
		{
			name: "success - header shows default values when flags not provided",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 1000},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "query: services.service_name:HTTP")
				require.Contains(t, stdout, `count by: ""`)
				require.Contains(t, stdout, "filtered: false")
			},
		},
		{
			name: "success - header shows values when count-by-level and filter-by-query flags provided",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "80", Count: 1000},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--count-by-level", "host", "--filter-by-query", "services.service_name:HTTP", "services.port"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "query: services.service_name:HTTP")
				require.Contains(t, stdout, "count by: host")
				require.Contains(t, stdout, "filtered: true")
			},
		},
		{
			name: "success - header shows count-by-level only when provided",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				mockSvc := aggregatemocks.NewMockAggregateService(ctrl)
				mockSvc.EXPECT().Aggregate(
					gomock.Any(),
					gomock.AssignableToTypeOf(aggregate.Params{}),
				).Return(aggregate.Result{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Buckets: []aggregate.Bucket{
						{Key: "SSH", Count: 500},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"-l", "service", "host.services.port=22", "host.services.protocol"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "query: host.services.port=22")
				require.Contains(t, stdout, "count by: service")
				require.Contains(t, stdout, "filtered: false")
			},
		},

		// Flag conflict tests
		{
			name: "error - raw and interactive flags together",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"--raw", "--interactive", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Equal(t, flags.NewConflictingFlagsError("raw", "interactive"), err)
			},
		},
		{
			name: "error - raw and interactive flags together (short form)",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) aggregate.Service {
				return aggregatemocks.NewMockAggregateService(ctrl)
			},
			args: []string{"-r", "-i", "query", "field"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot use --raw and --interactive flags together")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			aggregateSvc := tc.service(ctrl)
			cmdContext := command.NewCommandContext(cfg, tc.store(ctrl), command.WithAggregateService(aggregateSvc))
			rootCmd, err := command.RootCommandToCobra(NewAggregateCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			execErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cenclierrors.NewCencliError(execErr))
		})
	}
}
