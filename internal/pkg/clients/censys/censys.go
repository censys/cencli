package censys

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	censys "github.com/censys/censys-sdk-go"

	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	clienthttp "github.com/censys/cencli/internal/pkg/clients/http"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/store"
	"github.com/censys/cencli/internal/version"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/censys_client_mock.go -package=mocks -imports components=github.com/censys/censys-sdk-go/models/components github.com/censys/cencli/internal/pkg/clients/censys Client
type Client interface {
	GlobalDataClient
	CollectionsClient
	ThreatHuntingClient
	AccountManagementClient
	HasOrgID() bool
}

type censysSDK struct {
	client        *censys.SDK
	retryStrategy config.RetryStrategy
	hasOrgID      bool
}

func (c *censysSDK) HasOrgID() bool {
	return c.hasOrgID
}

type censysSDKImpl struct {
	*censysSDK
	GlobalDataClient
	CollectionsClient
	ThreatHuntingClient
	AccountManagementClient
}

var _ Client = &censysSDKImpl{}

func NewCensysSDK(
	ctx context.Context,
	ds store.Store,
	httpRequestTimeout time.Duration,
	retryStrategy config.RetryStrategy,
) (Client, error) {
	sdkOpts := []censys.SDKOption{
		censys.WithClient(clienthttp.New(httpRequestTimeout, buildUserAgent())),
	}

	storedPAT, err := ds.GetLastUsedAuthByName(ctx, config.AuthName)
	if err != nil {
		if errors.Is(err, authdom.ErrAuthNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get last used auth: %w", err)
	}
	sdkOpts = append(sdkOpts, censys.WithSecurity(storedPAT.Value))

	hasOrgID := false
	storedOrgID, err := ds.GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName)
	if err == nil {
		hasOrgID = true
		sdkOpts = append(sdkOpts, censys.WithOrganizationID(storedOrgID.Value))
	} else if !errors.Is(err, store.ErrGlobalNotFound) {
		return nil, fmt.Errorf("failed to get last used orgID: %w", err)
	}

	censysSDK := &censysSDK{
		client:        censys.New(sdkOpts...),
		retryStrategy: retryStrategy,
		hasOrgID:      hasOrgID,
	}

	return &censysSDKImpl{
		censysSDK:               censysSDK,
		GlobalDataClient:        newGlobalDataSDK(censysSDK),
		CollectionsClient:       newCollectionsSDK(censysSDK),
		ThreatHuntingClient:     newThreatHuntingSDK(censysSDK),
		AccountManagementClient: newAccountManagementSDK(censysSDK),
	}, nil
}

func buildUserAgent() string {
	return fmt.Sprintf("cencli/%s (%s; %s %s)", version.Version, version.Date, runtime.GOOS, runtime.GOARCH)
}

func (c *censysSDK) executeWithRetry(ctx context.Context, operationFn func() ClientError) (ClientError, uint64) {
	if operationFn == nil {
		return wrapCencliError(cenclierrors.NewCencliError(errors.New("operationFn cannot be nil"))), 1
	}

	maxAttempts := c.retryStrategy.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	baseDelay := c.retryStrategy.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	var lastErr ClientError
	for attempt := uint64(1); attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return wrapCencliError(cenclierrors.ParseContextError(err)), attempt
		}

		err := operationFn()
		if err == nil {
			return nil, attempt
		}

		lastErr = err
		if attempt == maxAttempts || !shouldRetryCensysError(err) {
			return err, attempt
		}

		delay := calculateRetryDelay(baseDelay, c.retryStrategy.MaxDelay, c.retryStrategy.Backoff, attempt)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return wrapCencliError(cenclierrors.ParseContextError(ctx.Err())), attempt
		}
	}

	return lastErr, maxAttempts
}

func shouldRetryCensysError(err ClientError) bool {
	statusOpt := err.StatusCode()
	if statusOpt.IsPresent() {
		status := statusOpt.MustGet()
		return status == 429 || status >= 500
	}
	return false
}

func calculateRetryDelay(baseDelay, maxDelay time.Duration, backoff config.BackoffType, attempt uint64) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	var delay time.Duration

	switch backoff {
	case config.BackoffLinear:
		delay = time.Duration(attempt) * baseDelay
	case config.BackoffExponential:
		delay = time.Duration(1<<(attempt-1)) * baseDelay
	default:
		delay = baseDelay
	}

	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}

	return delay
}
