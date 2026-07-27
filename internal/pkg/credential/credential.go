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

// IsBoundToOrg reports whether the credential is locked to one organization, so
// it can act only on that organization — not on another, and not on the user's
// free account.
func (i Info) IsBoundToOrg() bool { return !i.AllowsManualOrg() && i.OrgID != "" }

// IsBoundToFreeAccount reports whether the credential is locked to the user's
// free account, so it cannot act on any organization.
func (i Info) IsBoundToFreeAccount() bool { return !i.AllowsManualOrg() && i.OrgID == "" }

// AllowsManualOrg reports whether the organization may be chosen per request,
// via the --org-id flag or the stored org-id global.
//
// Only a personal access token works that way, because it is the one credential
// that is not organization-scoped. KindNone is included because no credential is
// configured at all, so the command fails on authentication rather than here.
//
// Every other kind carries its own organization binding, so this is deliberately
// an allowlist: a credential kind added later is restrictive by default instead
// of silently inheriting personal-access-token behavior.
func (i Info) AllowsManualOrg() bool {
	return i.Kind == KindPersonalAccessToken || i.Kind == KindNone
}

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
