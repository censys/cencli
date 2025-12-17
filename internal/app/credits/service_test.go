package credits

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

func TestCreditsService_GetOrganizationCreditDetails(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		orgID  uuid.UUID
		ctx    func() context.Context
		assert func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError)
	}{
		{
			name:  "success - basic organization credits",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationCredits{
						Balance:           1000,
						CreditExpirations: []components.CreditExpiration{},
						AutoReplenishConfig: components.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 100*time.Millisecond, res.Meta.Latency)
				require.Equal(t, int64(1000), res.Data.Balance)
				require.Len(t, res.Data.CreditExpirations, 0)
				require.False(t, res.Data.AutoReplenishConfig.Enabled)
				require.False(t, res.Data.AutoReplenishConfig.Threshold.IsPresent())
				require.False(t, res.Data.AutoReplenishConfig.Amount.IsPresent())
			},
		},
		{
			name:  "success - with credit expirations",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				expiresAt := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  150 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationCredits{
						Balance: 5000,
						CreditExpirations: []components.CreditExpiration{
							{
								Balance:   2000,
								CreatedAt: &createdAt,
								ExpiresAt: &expiresAt,
							},
							{
								Balance:   3000,
								CreatedAt: &createdAt,
								ExpiresAt: &expiresAt,
							},
						},
						AutoReplenishConfig: components.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 150*time.Millisecond, res.Meta.Latency)
				require.Equal(t, int64(5000), res.Data.Balance)
				require.Len(t, res.Data.CreditExpirations, 2)
				require.Equal(t, int64(2000), res.Data.CreditExpirations[0].Balance)
				require.True(t, res.Data.CreditExpirations[0].CreationDate.IsPresent())
				require.True(t, res.Data.CreditExpirations[0].ExpirationDate.IsPresent())
			},
		},
		{
			name:  "success - with auto replenish config enabled",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				threshold := int64(100)
				amount := int64(500)
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationCredits{
						Balance:           2500,
						CreditExpirations: []components.CreditExpiration{},
						AutoReplenishConfig: components.AutoReplenishConfig{
							Enabled:   true,
							Threshold: &threshold,
							Amount:    &amount,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, int64(2500), res.Data.Balance)
				require.True(t, res.Data.AutoReplenishConfig.Enabled)
				require.True(t, res.Data.AutoReplenishConfig.Threshold.IsPresent())
				require.Equal(t, int64(100), res.Data.AutoReplenishConfig.Threshold.MustGet())
				require.True(t, res.Data.AutoReplenishConfig.Amount.IsPresent())
				require.Equal(t, int64(500), res.Data.AutoReplenishConfig.Amount.MustGet())
			},
		},
		{
			name:  "error - structured client error",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Organization not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{}, structuredErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationCreditDetailsResult{}, res)

				var structuredErr client.ClientStructuredError
				require.True(t, errors.As(err, &structuredErr))
				require.True(t, structuredErr.StatusCode().IsPresent())
				require.Equal(t, int64(404), structuredErr.StatusCode().MustGet())
			},
		},
		{
			name:  "error - generic client error",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Internal server error",
					StatusCode: 500,
					Body:       "Server temporarily unavailable",
				})
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{}, genericErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationCreditDetailsResult{}, res)

				var genericErr client.ClientGenericError
				require.True(t, errors.As(err, &genericErr))
				require.Equal(t, int64(500), genericErr.StatusCode().MustGet())
			},
		},
		{
			name:  "error - unknown client error",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(client.Result[components.OrganizationCredits]{}, unknownErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationCreditDetailsResult{}, res)

				var unknownErr client.ClientError
				require.True(t, errors.As(err, &unknownErr))
				require.Contains(t, err.Error(), "network timeout")
			},
		},
		{
			name:  "context cancellation - cancelled context",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).DoAndReturn(func(ctx context.Context, orgID uuid.UUID) (client.Result[components.OrganizationCredits], client.ClientError) {
					select {
					case <-ctx.Done():
						return client.Result[components.OrganizationCredits]{}, client.NewClientError(ctx.Err())
					default:
						t.Error("Expected context to be cancelled")
						return client.Result[components.OrganizationCredits]{}, client.NewClientError(errors.New("context should have been cancelled"))
					}
				})
				return mockClient
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			assert: func(t *testing.T, res OrganizationCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationCreditDetailsResult{}, res)
				require.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := tc.client(ctrl)
			svc := New(c)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.GetOrganizationCreditDetails(ctx, tc.orgID)
			tc.assert(t, res, err)
		})
	}
}

func TestCreditsService_GetUserCreditDetails(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		ctx    func() context.Context
		assert func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - basic user credits",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  80 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.UserCredits{
						Balance: 500,
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 80*time.Millisecond, res.Meta.Latency)
				require.Equal(t, int64(500), res.Data.Balance)
				require.False(t, res.Data.ResetsAt.IsPresent())
			},
		},
		{
			name: "success - with resets_at",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				resetsAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  90 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.UserCredits{
						Balance:  1000,
						ResetsAt: &resetsAt,
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 90*time.Millisecond, res.Meta.Latency)
				require.Equal(t, int64(1000), res.Data.Balance)
				require.True(t, res.Data.ResetsAt.IsPresent())
				require.Equal(t, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), res.Data.ResetsAt.MustGet())
			},
		},
		{
			name: "success - zero balance",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.UserCredits{
						Balance: 0,
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, int64(0), res.Data.Balance)
			},
		},
		{
			name: "error - structured client error (unauthorized)",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Unauthorized access"
				status := int64(401)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{}, structuredErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, UserCreditDetailsResult{}, res)

				var structuredErr client.ClientStructuredError
				require.True(t, errors.As(err, &structuredErr))
				require.True(t, structuredErr.StatusCode().IsPresent())
				require.Equal(t, int64(401), structuredErr.StatusCode().MustGet())
			},
		},
		{
			name: "error - generic client error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Service unavailable",
					StatusCode: 503,
					Body:       "Try again later",
				})
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{}, genericErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, UserCreditDetailsResult{}, res)

				var genericErr client.ClientGenericError
				require.True(t, errors.As(err, &genericErr))
				require.Equal(t, int64(503), genericErr.StatusCode().MustGet())
			},
		},
		{
			name: "error - unknown client error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("connection refused"))
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(client.Result[components.UserCredits]{}, unknownErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, UserCreditDetailsResult{}, res)

				var unknownErr client.ClientError
				require.True(t, errors.As(err, &unknownErr))
				require.Contains(t, err.Error(), "connection refused")
			},
		},
		{
			name: "context cancellation - cancelled context",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).DoAndReturn(func(ctx context.Context) (client.Result[components.UserCredits], client.ClientError) {
					select {
					case <-ctx.Done():
						return client.Result[components.UserCredits]{}, client.NewClientError(ctx.Err())
					default:
						t.Error("Expected context to be cancelled")
						return client.Result[components.UserCredits]{}, client.NewClientError(errors.New("context should have been cancelled"))
					}
				})
				return mockClient
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			assert: func(t *testing.T, res UserCreditDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, UserCreditDetailsResult{}, res)
				require.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := tc.client(ctrl)
			svc := New(c)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.GetUserCreditDetails(ctx)
			tc.assert(t, res, err)
		})
	}
}

// TestParseOrganizationCreditDetails tests the organization credit details parsing functionality
func TestParseOrganizationCreditDetails(t *testing.T) {
	testCases := []struct {
		name     string
		input    *components.OrganizationCredits
		expected OrganizationCreditDetails
	}{
		{
			name: "empty credit expirations and disabled auto replenish",
			input: &components.OrganizationCredits{
				Balance:           1000,
				CreditExpirations: []components.CreditExpiration{},
				AutoReplenishConfig: components.AutoReplenishConfig{
					Enabled: false,
				},
			},
			expected: OrganizationCreditDetails{
				Balance:           1000,
				CreditExpirations: nil,
				AutoReplenishConfig: AutoReplenishConfig{
					Enabled: false,
				},
			},
		},
		{
			name: "with credit expirations and dates",
			input: func() *components.OrganizationCredits {
				createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				expiresAt := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
				return &components.OrganizationCredits{
					Balance: 5000,
					CreditExpirations: []components.CreditExpiration{
						{
							Balance:   2500,
							CreatedAt: &createdAt,
							ExpiresAt: &expiresAt,
						},
					},
					AutoReplenishConfig: components.AutoReplenishConfig{
						Enabled: false,
					},
				}
			}(),
			expected: func() OrganizationCreditDetails {
				return OrganizationCreditDetails{
					Balance: 5000,
					CreditExpirations: []CreditExpiration{
						{
							Balance:        2500,
							CreationDate:   mo.Some(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
							ExpirationDate: mo.Some(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
						},
					},
					AutoReplenishConfig: AutoReplenishConfig{
						Enabled: false,
					},
				}
			}(),
		},
		{
			name: "with auto replenish enabled",
			input: func() *components.OrganizationCredits {
				threshold := int64(100)
				amount := int64(1000)
				return &components.OrganizationCredits{
					Balance:           3000,
					CreditExpirations: []components.CreditExpiration{},
					AutoReplenishConfig: components.AutoReplenishConfig{
						Enabled:   true,
						Threshold: &threshold,
						Amount:    &amount,
					},
				}
			}(),
			expected: func() OrganizationCreditDetails {
				return OrganizationCreditDetails{
					Balance:           3000,
					CreditExpirations: nil,
					AutoReplenishConfig: AutoReplenishConfig{
						Enabled:   true,
						Threshold: mo.Some(int64(100)),
						Amount:    mo.Some(int64(1000)),
					},
				}
			}(),
		},
		{
			name: "credit expiration without dates",
			input: &components.OrganizationCredits{
				Balance: 1500,
				CreditExpirations: []components.CreditExpiration{
					{
						Balance: 1500,
					},
				},
				AutoReplenishConfig: components.AutoReplenishConfig{
					Enabled: false,
				},
			},
			expected: OrganizationCreditDetails{
				Balance: 1500,
				CreditExpirations: []CreditExpiration{
					{
						Balance: 1500,
					},
				},
				AutoReplenishConfig: AutoReplenishConfig{
					Enabled: false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseOrganizationCreditDetails(tc.input)
			require.Equal(t, tc.expected.Balance, result.Balance)
			require.Equal(t, tc.expected.AutoReplenishConfig.Enabled, result.AutoReplenishConfig.Enabled)
			require.Equal(t, tc.expected.AutoReplenishConfig.Threshold.IsPresent(), result.AutoReplenishConfig.Threshold.IsPresent())
			require.Equal(t, tc.expected.AutoReplenishConfig.Amount.IsPresent(), result.AutoReplenishConfig.Amount.IsPresent())
			if tc.expected.AutoReplenishConfig.Threshold.IsPresent() {
				require.Equal(t, tc.expected.AutoReplenishConfig.Threshold.MustGet(), result.AutoReplenishConfig.Threshold.MustGet())
			}
			if tc.expected.AutoReplenishConfig.Amount.IsPresent() {
				require.Equal(t, tc.expected.AutoReplenishConfig.Amount.MustGet(), result.AutoReplenishConfig.Amount.MustGet())
			}
			require.Len(t, result.CreditExpirations, len(tc.expected.CreditExpirations))
			for i, ce := range tc.expected.CreditExpirations {
				require.Equal(t, ce.Balance, result.CreditExpirations[i].Balance)
				require.Equal(t, ce.CreationDate.IsPresent(), result.CreditExpirations[i].CreationDate.IsPresent())
				require.Equal(t, ce.ExpirationDate.IsPresent(), result.CreditExpirations[i].ExpirationDate.IsPresent())
			}
		})
	}
}

// TestParseUserCreditDetails tests the user credit details parsing functionality
func TestParseUserCreditDetails(t *testing.T) {
	testCases := []struct {
		name     string
		input    *components.UserCredits
		expected UserCreditDetails
	}{
		{
			name: "basic balance without resets_at",
			input: &components.UserCredits{
				Balance: 500,
			},
			expected: UserCreditDetails{
				Balance: 500,
			},
		},
		{
			name: "with resets_at",
			input: func() *components.UserCredits {
				resetsAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
				return &components.UserCredits{
					Balance:  1000,
					ResetsAt: &resetsAt,
				}
			}(),
			expected: func() UserCreditDetails {
				return UserCreditDetails{
					Balance:  1000,
					ResetsAt: mo.Some(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
				}
			}(),
		},
		{
			name: "zero balance",
			input: &components.UserCredits{
				Balance: 0,
			},
			expected: UserCreditDetails{
				Balance: 0,
			},
		},
		{
			name: "large balance",
			input: &components.UserCredits{
				Balance: 9223372036854775807,
			},
			expected: UserCreditDetails{
				Balance: 9223372036854775807,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseUserCreditDetails(tc.input)
			require.Equal(t, tc.expected.Balance, result.Balance)
			require.Equal(t, tc.expected.ResetsAt.IsPresent(), result.ResetsAt.IsPresent())
			if tc.expected.ResetsAt.IsPresent() {
				require.Equal(t, tc.expected.ResetsAt.MustGet(), result.ResetsAt.MustGet())
			}
		})
	}
}
