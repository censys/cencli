// Package credential resolves which stored credential API requests should
// authenticate with (an OAuth login or a personal access token) and describes
// its consent scope. This is store/domain policy, deliberately kept out of the
// SDK-client layer.
package credential

import (
	"context"
	"errors"
	"fmt"

	"github.com/censys/cencli/internal/config"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/oauth"
	"github.com/censys/cencli/internal/store"
)

// Kind identifies the type of credential authenticating requests. Modeling it
// as an enum (rather than an "isOAuth" bool) leaves room for future kinds such
// as service accounts.
type Kind int

const (
	// KindNone means no credential is configured.
	KindNone Kind = iota
	// KindPersonalAccessToken is a stored personal access token.
	KindPersonalAccessToken
	// KindOAuth is a browser OAuth login (`censys auth login`).
	KindOAuth
)

func (k Kind) String() string {
	switch k {
	case KindNone:
		return "none"
	case KindPersonalAccessToken:
		return "personal access token"
	case KindOAuth:
		return "oauth"
	default:
		return fmt.Sprintf("unknown(%d)", int(k))
	}
}

// Info describes the active credential for scoping and display decisions. For a
// personal access token only Kind is set; OAuth logins also carry the consent
// scope (OrgID empty means the free account) and the logged-in account.
type Info struct {
	Kind    Kind
	Account string
	OrgID   string
	OrgName string
}

// IsOrgBoundOAuth reports an OAuth login locked to a specific organization.
func (i Info) IsOrgBoundOAuth() bool { return i.Kind == KindOAuth && i.OrgID != "" }

// IsFreeAccountOAuth reports an OAuth login scoped to the user's free account.
func (i Info) IsFreeAccountOAuth() bool { return i.Kind == KindOAuth && i.OrgID == "" }

// Active resolves the credential requests should authenticate with: the OAuth
// session from `censys auth login` or a stored personal access token, whichever
// was most recently used/activated. It returns the stored record alongside its
// Kind, or authdom.ErrAuthNotFound when neither is configured.
func Active(ctx context.Context, ds store.Store) (*store.ValueForAuth, Kind, error) {
	oauthRec, oauthErr := ds.GetLastUsedAuthByName(ctx, config.OAuthSessionName)
	if oauthErr != nil && !errors.Is(oauthErr, authdom.ErrAuthNotFound) {
		return nil, KindNone, fmt.Errorf("failed to get oauth session: %w", oauthErr)
	}
	patRec, patErr := ds.GetLastUsedAuthByName(ctx, config.AuthName)
	if patErr != nil && !errors.Is(patErr, authdom.ErrAuthNotFound) {
		return nil, KindNone, fmt.Errorf("failed to get last used auth: %w", patErr)
	}

	switch {
	case oauthErr == nil && patErr == nil:
		if oauthRec.LastUsedAt.Before(patRec.LastUsedAt) {
			return patRec, KindPersonalAccessToken, nil
		}
		return oauthRec, KindOAuth, nil
	case oauthErr == nil:
		return oauthRec, KindOAuth, nil
	case patErr == nil:
		return patRec, KindPersonalAccessToken, nil
	default:
		return nil, KindNone, authdom.ErrAuthNotFound
	}
}

// Resolve returns the Info describing the active credential, parsing the OAuth
// session for its account and org binding. The Kind is KindNone (no error) when
// nothing is configured.
func Resolve(ctx context.Context, ds store.Store) (Info, error) {
	rec, kind, err := Active(ctx, ds)
	if err != nil {
		if errors.Is(err, authdom.ErrAuthNotFound) {
			return Info{Kind: KindNone}, nil
		}
		return Info{Kind: KindNone}, err
	}

	info := Info{Kind: kind}
	if kind == KindOAuth {
		if sess, perr := oauth.ParseSession(rec.Value); perr == nil {
			info.Account = sess.Account()
			info.OrgID = sess.OrgID
			info.OrgName = sess.OrgName
		}
	}
	return info, nil
}
