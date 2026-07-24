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
	oauthOrgName  string
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

	var oauthOrgID, oauthOrgName string
	if isOAuth {
		oauthClient := oauth.NewClient(oauth.Config{}, &httpClient.Client)
		sdkOpts = append(sdkOpts, censys.WithSecuritySource(oauthSecuritySource(ds, oauthClient)))
		// Read the org the session is locked to (empty for a free account).
		if sess, perr := oauth.ParseSession(cred.Value); perr == nil {
			oauthOrgID = sess.OrgID
			oauthOrgName = sess.OrgName
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
		oauthOrgName:  oauthOrgName,
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
			return c.annotateAuthError(err), attempt
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

	return c.annotateAuthError(lastErr), maxAttempts
}

// oauthScopeError augments an underlying client error with guidance about the
// active OAuth session's scope. It embeds the original error so status code,
// title, and exit-code behavior are unchanged.
type oauthScopeError struct {
	ClientError
	hint string
}

func (e *oauthScopeError) Error() string { return e.ClientError.Error() + "\n\n" + e.hint }

func (e *oauthScopeError) Unwrap() error { return e.ClientError }

// annotateAuthError appends OAuth-scope guidance to a 403 so a request that
// targets an organization (or the free account) outside the session's scope
// yields an actionable message instead of a bare "forbidden". The API is the
// source of truth; this only adds a hint for the OAuth case.
func (c *censysSDK) annotateAuthError(err ClientError) ClientError {
	if err == nil || !c.isOAuth {
		return err
	}
	if status := err.StatusCode(); !status.IsPresent() || status.MustGet() != 403 {
		return err
	}
	var hint string
	if c.oauthOrgID != "" {
		org := c.oauthOrgName
		if org == "" {
			org = c.oauthOrgID
		}
		hint = fmt.Sprintf("This login is scoped to organization %s; it cannot act on a different organization or your free account. Run `censys auth logout` then `censys auth login` to switch.", org)
	} else {
		hint = "This login is scoped to your free account; it cannot access organizations. Run `censys auth logout` then `censys auth login` and select the organization to access it."
	}
	return &oauthScopeError{ClientError: err, hint: hint}
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
