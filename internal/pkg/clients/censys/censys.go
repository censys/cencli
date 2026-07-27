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
	"github.com/censys/cencli/internal/pkg/credential"
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
	// CredentialInfo describes the credential authenticating requests.
	CredentialInfo() credential.Info
}

type censysSDK struct {
	client        *censys.SDK
	retryStrategy config.RetryStrategy
	hasOrgID      bool
	cred          credential.Info
	logger        *slog.Logger
}

func (c *censysSDK) HasOrgID() bool {
	return c.hasOrgID
}

// CredentialInfo describes the credential authenticating requests (kind and, for
// OAuth logins, the consent scope).
func (c *censysSDK) CredentialInfo() credential.Info {
	return c.cred
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

	rec, kind, err := credential.Active(ctx, ds)
	if err != nil {
		return nil, err
	}

	var cred credential.Info
	cred.Kind = kind
	if kind == credential.KindOAuth {
		oauthClient := oauth.NewClient(oauth.Config{}, &httpClient.Client)
		sdkOpts = append(sdkOpts, censys.WithSecuritySource(oauthSecuritySource(ds, oauthClient)))
		// Read the account and org the session is locked to (empty org = free account).
		if sess, perr := oauth.ParseSession(rec.Value); perr == nil {
			cred.Account = sess.Account()
			cred.OrgID = sess.OrgID
			cred.OrgName = sess.OrgName
		}
	} else {
		sdkOpts = append(sdkOpts, censys.WithSecurity(rec.Value))
	}

	// An OAuth login is self-scoped: the session dictates the org, so the stored
	// org-id global is ignored. PATs are not org-scoped and fall back to it.
	hasOrgID := false
	switch {
	case kind == credential.KindOAuth:
		if cred.OrgID != "" {
			hasOrgID = true
			sdkOpts = append(sdkOpts, censys.WithOrganizationID(cred.OrgID))
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
		cred:          cred,
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

// sessionRefresher exchanges a refresh token for a new session. It is a seam
// over oauth.Client.Refresh so the security source can be tested without the
// hardcoded authorization server.
type sessionRefresher func(ctx context.Context, refreshToken string) (*oauth.Session, error)

// oauthSecuritySource returns a per-request token source for the SDK. It
// loads the stored OAuth session, refreshing (and persisting) it when the
// access token is expired. The access token is sent as a bearer token, same
// as a personal access token.
func oauthSecuritySource(ds store.Store, oauthClient *oauth.Client) func(context.Context) (components.Security, error) {
	return newOAuthSecuritySource(ds, oauthClient.Refresh)
}

func newOAuthSecuritySource(ds store.Store, refresh sessionRefresher) func(context.Context) (components.Security, error) {
	var mu sync.Mutex
	return func(ctx context.Context) (components.Security, error) {
		// The mutex serializes refreshes within this process; the re-read and
		// fallback below handle a concurrent cencli process racing on the same
		// grant (the authorization server rotates refresh tokens, so only one
		// refresh of a given token can win).
		mu.Lock()
		defer mu.Unlock()

		rec, sess, err := loadOAuthSession(ctx, ds)
		if err != nil {
			return components.Security{}, err
		}
		if !sess.Expired(time.Now()) {
			return components.Security{PersonalAccessToken: sess.AccessToken}, nil
		}
		if sess.RefreshToken == "" {
			return components.Security{}, errors.New("your session has expired; run `censys auth login` to log in again")
		}

		newSess, err := refresh(ctx, sess.RefreshToken)
		if err != nil {
			// A concurrent invocation may have refreshed and rotated the token
			// out from under us. Re-read the stored session; if it is now valid,
			// use it rather than forcing the user to log in again.
			if _, fresh, rerr := loadOAuthSession(ctx, ds); rerr == nil && !fresh.Expired(time.Now()) {
				return components.Security{PersonalAccessToken: fresh.AccessToken}, nil
			}
			return components.Security{}, fmt.Errorf("failed to refresh your session (run `censys auth login` to log in again): %w", err)
		}
		carryOverSessionClaims(newSess, sess)

		value, err := newSess.Marshal()
		if err != nil {
			return components.Security{}, err
		}
		// Update the row in place (atomic) rather than add-then-delete.
		if _, err := ds.UpdateValueForAuth(ctx, rec.ID, rec.Description, value); err != nil {
			return components.Security{}, fmt.Errorf("failed to persist refreshed oauth session: %w", err)
		}

		return components.Security{PersonalAccessToken: newSess.AccessToken}, nil
	}
}

// loadOAuthSession reads and parses the active OAuth session from the store.
func loadOAuthSession(ctx context.Context, ds store.Store) (*store.ValueForAuth, *oauth.Session, error) {
	rec, err := ds.GetLastUsedAuthByName(ctx, config.OAuthSessionName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load oauth session (run `censys auth login`): %w", err)
	}
	sess, err := oauth.ParseSession(rec.Value)
	if err != nil {
		return nil, nil, fmt.Errorf("%w (run `censys auth login`)", err)
	}
	return rec, sess, nil
}

// carryOverSessionClaims copies identity/binding fields a refresh response may
// omit from the previous session, since they are stable for the grant's life.
func carryOverSessionClaims(newSess, prev *oauth.Session) {
	if newSess.Email == "" {
		newSess.Email = prev.Email
	}
	if newSess.Subject == "" {
		newSess.Subject = prev.Subject
	}
	if newSess.OrgID == "" {
		newSess.OrgID = prev.OrgID
	}
	if newSess.OrgName == "" {
		newSess.OrgName = prev.OrgName
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
