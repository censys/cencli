package censys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/config"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/store"
)

func TestNewCensysSDK(t *testing.T) {
	ctx := context.Background()

	t.Run("success with PAT and OrgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mocks.NewMockStore(ctrl)

		mockStore.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(&store.ValueForAuth{
			Name:       "auth",
			Value:      "test-pat-token",
			LastUsedAt: time.Now(),
		}, nil)

		mockStore.EXPECT().GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName).Return(&store.ValueForGlobal{
			Name:       "orgid",
			Value:      "test-org-id",
			LastUsedAt: time.Now(),
		}, nil)

		client, err := NewCensysSDK(ctx, mockStore, config.RetryStrategy{})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, client.HasOrgID())
	})

	t.Run("success with PAT only (no OrgID)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mocks.NewMockStore(ctrl)

		mockStore.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(&store.ValueForAuth{
			Name:       "auth",
			Value:      "test-pat-token",
			LastUsedAt: time.Now(),
		}, nil)

		mockStore.EXPECT().GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName).Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound)

		client, err := NewCensysSDK(ctx, mockStore, config.RetryStrategy{})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.False(t, client.HasOrgID())
	})

	t.Run("error when PAT not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mocks.NewMockStore(ctrl)

		mockStore.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)

		client, err := NewCensysSDK(ctx, mockStore, config.RetryStrategy{})
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.True(t, errors.Is(err, authdom.ErrAuthNotFound))
	})

	t.Run("error when PAT retrieval fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mocks.NewMockStore(ctrl)

		mockStore.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return((*store.ValueForAuth)(nil), errors.New("db error"))

		client, err := NewCensysSDK(ctx, mockStore, config.RetryStrategy{})
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to get last used auth")
	})

	t.Run("error when OrgID retrieval fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mocks.NewMockStore(ctrl)

		mockStore.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(&store.ValueForAuth{
			Name:       "auth",
			Value:      "test-pat-token",
			LastUsedAt: time.Now(),
		}, nil)

		mockStore.EXPECT().GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName).Return((*store.ValueForGlobal)(nil), errors.New("db error"))

		client, err := NewCensysSDK(ctx, mockStore, config.RetryStrategy{})
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to get last used orgID")
	})
}

func TestParseSDKError(t *testing.T) {
	t.Run("SDK error", func(t *testing.T) {
		sdkErr := &sdkerrors.SDKError{
			Message:    "SDK error occurred",
			StatusCode: 400,
		}

		result := NewClientError(sdkErr)
		assert.NotNil(t, result)

		var censysErr ClientError
		assert.True(t, errors.As(result, &censysErr))
	})

	t.Run("Model error", func(t *testing.T) {
		modelErr := &sdkerrors.ErrorModel{
			Detail:   strPtr("Invalid request"),
			Instance: strPtr("/api/v1/error"),
		}

		result := NewClientError(modelErr)
		assert.NotNil(t, result)

		var censysErr ClientError
		assert.True(t, errors.As(result, &censysErr))
	})

	t.Run("Generic error", func(t *testing.T) {
		genericErr := errors.New("generic error")

		result := NewClientError(genericErr)
		assert.NotNil(t, result)

		var censysErr ClientError
		assert.True(t, errors.As(result, &censysErr))
	})
}

func TestCensysSDK_HasOrgID(t *testing.T) {
	t.Run("with OrgID", func(t *testing.T) {
		sdk := &censysSDK{
			hasOrgID: true,
		}
		assert.True(t, sdk.HasOrgID())
	})

	t.Run("without OrgID", func(t *testing.T) {
		sdk := &censysSDK{
			hasOrgID: false,
		}
		assert.False(t, sdk.HasOrgID())
	})
}

func TestCensysSDK_ExecuteWithRetry(t *testing.T) {
	testCases := []struct {
		name        string
		strategy    config.RetryStrategy
		responses   []ClientError
		mutate      func(attempt int, cancel context.CancelFunc)
		expectErr   func(t *testing.T, err ClientError)
		expectCalls uint64
	}{
		{
			name: "success without retry",
			strategy: config.RetryStrategy{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffFixed,
			},
			responses: []ClientError{nil},
			expectErr: func(t *testing.T, err ClientError) {
				require.NoError(t, err)
			},
			expectCalls: uint64(1),
		},
		{
			name: "retry once with fixed backoff on 429",
			strategy: config.RetryStrategy{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffFixed,
			},
			responses: []ClientError{
				newGenericCensysError(429),
				nil,
			},
			expectErr: func(t *testing.T, err ClientError) {
				require.NoError(t, err)
			},
			expectCalls: uint64(2),
		},
		{
			name: "linear backoff retries for 500 then succeeds",
			strategy: config.RetryStrategy{
				MaxAttempts: 4,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffLinear,
			},
			responses: []ClientError{
				newGenericCensysError(500),
				nil,
			},
			expectErr: func(t *testing.T, err ClientError) {
				require.NoError(t, err)
			},
			expectCalls: uint64(2),
		},
		{
			name: "exponential backoff exhausts retries",
			strategy: config.RetryStrategy{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffExponential,
			},
			responses: func() []ClientError {
				retryableErr := newGenericCensysError(429)
				return []ClientError{retryableErr, retryableErr, retryableErr}
			}(),
			expectErr: func(t *testing.T, err ClientError) {
				require.Error(t, err)
				assert.Equal(t, newGenericCensysError(429).Error(), err.Error())
			},
			expectCalls: uint64(3),
		},
		{
			name: "non-retryable error returns immediately",
			strategy: config.RetryStrategy{
				MaxAttempts: 4,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffFixed,
			},
			responses: []ClientError{newGenericCensysError(400)},
			expectErr: func(t *testing.T, err ClientError) {
				require.Error(t, err)
				var genericErr ClientGenericError
				assert.ErrorAs(t, err, &genericErr)
				assert.Equal(t, int64(400), genericErr.StatusCode().MustGet())
			},
			expectCalls: uint64(1),
		},
		{
			name: "context cancellation stops retries",
			strategy: config.RetryStrategy{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffFixed,
			},
			responses: []ClientError{newGenericCensysError(429)},
			mutate: func(attempt int, cancel context.CancelFunc) {
				if attempt == 1 {
					cancel()
				}
			},
			expectErr: func(t *testing.T, err ClientError) {
				require.Error(t, err)
				var unknownErr ClientError
				assert.ErrorAs(t, err, &unknownErr)
				assert.Contains(t, err.Error(), "cancelled")
			},
			expectCalls: uint64(1),
		},
		{
			name: "zero max attempts defaults to single try",
			strategy: config.RetryStrategy{
				MaxAttempts: 0,
				BaseDelay:   1 * time.Millisecond,
				Backoff:     config.BackoffExponential,
			},
			responses: []ClientError{newGenericCensysError(500)},
			expectErr: func(t *testing.T, err ClientError) {
				require.Error(t, err)
				var genericErr ClientGenericError
				assert.ErrorAs(t, err, &genericErr)
				assert.Equal(t, int64(500), genericErr.StatusCode().MustGet())
			},
			expectCalls: uint64(1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdk := &censysSDK{retryStrategy: tc.strategy}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			callCount := 0
			op := func() ClientError {
				callCount++
				if tc.mutate != nil {
					tc.mutate(callCount, cancel)
				}
				if callCount-1 >= len(tc.responses) {
					t.Fatalf("unexpected additional call: %d", callCount)
				}
				return tc.responses[callCount-1]
			}

			err, attempts := sdk.executeWithRetry(ctx, op)
			tc.expectErr(t, err)
			assert.Equal(t, tc.expectCalls, uint64(callCount))
			assert.Equal(t, tc.expectCalls, attempts)
		})
	}
}

func TestCensysSDK_ExecuteWithRetryNilOperation(t *testing.T) {
	sdk := &censysSDK{retryStrategy: config.RetryStrategy{MaxAttempts: 2}}
	err, attempts := sdk.executeWithRetry(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, uint64(1), attempts)
	var unknownErr ClientError
	assert.ErrorAs(t, err, &unknownErr)
	assert.Contains(t, err.Error(), "operationFn cannot be nil")
}

// helper for retry tests
func newGenericCensysError(code int) ClientError {
	return NewCensysClientGenericError(&sdkerrors.SDKError{Message: "retryable", StatusCode: code})
}

func TestCalculateRetryDelay(t *testing.T) {
	base := 100 * time.Millisecond
	max := 500 * time.Millisecond

	// Fixed
	if got := calculateRetryDelay(base, max, config.BackoffFixed, 1); got != base {
		t.Fatalf("fixed backoff: got %v", got)
	}
	// Linear
	if got := calculateRetryDelay(base, max, config.BackoffLinear, 3); got != 300*time.Millisecond {
		t.Fatalf("linear backoff: got %v", got)
	}
	// Exponential capped
	if got := calculateRetryDelay(base, max, config.BackoffExponential, 5); got != max {
		t.Fatalf("exp capped: got %v", got)
	}
}

type fakeErr struct{ code int64 }

func (e fakeErr) Title() string                { return "" }
func (e fakeErr) Error() string                { return "" }
func (e fakeErr) ShouldPrintUsage() bool       { return false }
func (e fakeErr) Status() string               { return "" }
func (e fakeErr) StatusCode() mo.Option[int64] { return mo.Some(e.code) }

func TestShouldRetryCensysError(t *testing.T) {
	if !shouldRetryCensysError(fakeErr{code: 429}) {
		t.Fatalf("expected retry on 429")
	}
	if !shouldRetryCensysError(fakeErr{code: 500}) {
		t.Fatalf("expected retry on 500")
	}
	if shouldRetryCensysError(fakeErr{code: 400}) {
		t.Fatalf("did not expect retry on 400")
	}
}
