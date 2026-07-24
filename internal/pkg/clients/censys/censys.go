package censys

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	censys "github.com/censys/censys-sdk-go"
	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	clienthttp "github.com/censys/cencli/internal/pkg/clients/http"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	applog "github.com/censys/cencli/internal/pkg/log"
	"github.com/censys/cencli/internal/pkg/oauth"
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
	isOAuth       bool
	oauthOrgID    string
	logger        *slog.Logger
}

func (c *censysSDK) HasOrgID() bool {
	return c.hasOrgID
}

// OAuthSession reports whether an OAuth login is active and the org it is locked
// to (empty for a free-account session).
func (c *censysSDK) OAuthSession() (isOAuth bool, orgID string) {
	return c.isOAuth, c.oauthOrgID
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
	cfg *config.Config,
) (Client, error) {
	// Create logger for HTTP and retry debugging (only logs when debug=true)
	var logger *slog.Logger
	if cfg.Debug {
		logger = applog.New(cfg.Debug, nil)
	}

	httpClient := clienthttp.New(cfg.Timeouts.HTTP, buildUserAgent(), logger)
	sdkOpts := []censys.SDKOption{
		censys.WithClient(httpClient),
	}

	cred, isOAuth, err := ActiveCredential(ctx, ds)
	if err != nil {
		return nil, err
	}

	var oauthOrgID string
	if isOAuth {
		oauthClient := oauth.NewClient(oauth.Config{}, &httpClient.Client)
		sdkOpts = append(sdkOpts, censys.WithSecuritySource(oauthSecuritySource(ds, oauthClient)))
		// Read the org the session is locked to (empty for a free account).
		if sess, perr := oauth.ParseSession(cred.Value); perr == nil {
			oauthOrgID = sess.OrgID
		}
	} else {
		sdkOpts = append(sdkOpts, censys.WithSecurity(cred.Value))
	}

	// An OAuth login is self-scoped: the session dictates the org, so the stored
	// org-id global is ignored. PATs are not org-scoped and fall back to it.
	hasOrgID := false
	switch {
	case isOAuth:
		if oauthOrgID != "" {
			hasOrgID = true
			sdkOpts = append(sdkOpts, censys.WithOrganizationID(oauthOrgID))
		}
	default:
		storedOrgID, orgErr := ds.GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName)
		if orgErr == nil {
			hasOrgID = true
			sdkOpts = append(sdkOpts, censys.WithOrganizationID(storedOrgID.Value))
		} else if !errors.Is(orgErr, store.ErrGlobalNotFound) {
			return nil, fmt.Errorf("failed to get last used orgID: %w", orgErr)
		}
	}

	censysSDK := &censysSDK{
		client:        censys.New(sdkOpts...),
		retryStrategy: cfg.RetryStrategy,
		hasOrgID:      hasOrgID,
		isOAuth:       isOAuth,
		oauthOrgID:    oauthOrgID,
		logger:        logger,
	}

	return &censysSDKImpl{
		censysSDK:               censysSDK,
		GlobalDataClient:        newGlobalDataSDK(censysSDK),
		CollectionsClient:       newCollectionsSDK(censysSDK),
		ThreatHuntingClient:     newThreatHuntingSDK(censysSDK),
		AccountManagementClient: newAccountManagementSDK(censysSDK),
	}, nil
}

// ActiveCredential resolves the credential API requests should authenticate
// with: the OAuth session from `censys auth login` or a stored personal
// access token, whichever was most recently used/activated. Returns
// auth.ErrAuthNotFound when neither is configured.
func ActiveCredential(ctx context.Context, ds store.Store) (*store.ValueForAuth, bool, error) {
	oauthRec, oauthErr := ds.GetLastUsedAuthByName(ctx, config.OAuthSessionName)
	if oauthErr != nil && !errors.Is(oauthErr, authdom.ErrAuthNotFound) {
		return nil, false, fmt.Errorf("failed to get oauth session: %w", oauthErr)
	}
	patRec, patErr := ds.GetLastUsedAuthByName(ctx, config.AuthName)
	if patErr != nil && !errors.Is(patErr, authdom.ErrAuthNotFound) {
		return nil, false, fmt.Errorf("failed to get last used auth: %w", patErr)
	}

	switch {
	case oauthErr == nil && patErr == nil:
		if oauthRec.LastUsedAt.Before(patRec.LastUsedAt) {
			return patRec, false, nil
		}
		return oauthRec, true, nil
	case oauthErr == nil:
		return oauthRec, true, nil
	case patErr == nil:
		return patRec, false, nil
	default:
		return nil, false, authdom.ErrAuthNotFound
	}
}

// oauthSecuritySource returns a per-request token source for the SDK. It
// loads the stored OAuth session, refreshing (and persisting) it when the
// access token is expired. The access token is sent as a bearer token, same
// as a personal access token.
func oauthSecuritySource(ds store.Store, oauthClient *oauth.Client) func(context.Context) (components.Security, error) {
	var mu sync.Mutex
	return func(ctx context.Context) (components.Security, error) {
		mu.Lock()
		defer mu.Unlock()

		rec, err := ds.GetLastUsedAuthByName(ctx, config.OAuthSessionName)
		if err != nil {
			return components.Security{}, fmt.Errorf("failed to load oauth session (run `censys auth login`): %w", err)
		}
		sess, err := oauth.ParseSession(rec.Value)
		if err != nil {
			return components.Security{}, fmt.Errorf("%w (run `censys auth login`)", err)
		}

		if !sess.Expired(time.Now()) {
			return components.Security{PersonalAccessToken: sess.AccessToken}, nil
		}

		if sess.RefreshToken == "" {
			return components.Security{}, errors.New("your session has expired; run `censys auth login` to log in again")
		}
		newSess, err := oauthClient.Refresh(ctx, sess.RefreshToken)
		if err != nil {
			return components.Security{}, fmt.Errorf("failed to refresh your session (run `censys auth login` to log in again): %w", err)
		}
		// Refresh responses may omit identity claims; carry them over for display.
		if newSess.Email == "" {
			newSess.Email = sess.Email
		}
		if newSess.Subject == "" {
			newSess.Subject = sess.Subject
		}
		// Org binding is fixed per grant; carry it over if the refresh omits it.
		if newSess.OrgID == "" {
			newSess.OrgID = sess.OrgID
		}

		value, err := newSess.Marshal()
		if err != nil {
			return components.Security{}, err
		}
		if _, err := ds.AddValueForAuth(ctx, config.OAuthSessionName, rec.Description, value); err != nil {
			return components.Security{}, fmt.Errorf("failed to persist refreshed oauth session: %w", err)
		}
		// Best effort: the new row is newer, so a leftover old row is ignored
		// by GetLastUsedAuthByName anyway.
		_, _ = ds.DeleteValueForAuth(ctx, rec.ID)

		return components.Security{PersonalAccessToken: newSess.AccessToken}, nil
	}
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
		if c.logger != nil {
			var statusCode int64
			if lastErr.StatusCode().IsPresent() {
				statusCode = lastErr.StatusCode().MustGet()
			}
			c.logger.Debug("retrying request", "attempt", attempt, "max_attempts", maxAttempts, "status", statusCode, "delay", delay)
		}
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
