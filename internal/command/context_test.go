package command

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	clientmocks "github.com/censys/cencli/gen/client/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/credential"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/store"
)

func TestResolveOrgID(t *testing.T) {
	ctx := context.Background()
	sessionUUID := uuid.New()
	otherUUID := uuid.New()
	orgFlag := func(u uuid.UUID) mo.Option[identifiers.OrganizationID] {
		return mo.Some(identifiers.NewOrganizationID(u))
	}
	none := mo.None[identifiers.OrganizationID]()

	newCtx := func(t *testing.T, info credential.Info, setup func(*storemocks.MockStore)) *Context {
		t.Helper()
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		viper.Reset()
		cfg, err := config.New(t.TempDir())
		require.NoError(t, err)
		ds := storemocks.NewMockStore(ctrl)
		if setup != nil {
			setup(ds)
		}
		c := NewCommandContext(cfg, ds)
		cli := clientmocks.NewMockClient(ctrl)
		cli.EXPECT().CredentialInfo().Return(info).AnyTimes()
		c.SetCensysClient(cli)
		return c
	}

	t.Run("org-bound oauth without a flag uses the session org", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String(), OrgName: "Censys"}, nil)
		got, err := c.ResolveOrgID(ctx, none)
		require.Nil(t, err)
		require.True(t, got.IsPresent())
		assert.Equal(t, sessionUUID.String(), got.MustGet().String())
	})

	// --org-id is a personal-access-token concept, so it is rejected under OAuth
	// even when it names the very org the session is bound to.
	t.Run("org-bound oauth rejects --org-id naming the same org", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String(), OrgName: "Censys"}, nil)
		_, err := c.ResolveOrgID(ctx, orgFlag(sessionUUID))
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "only applies to personal access tokens")
		assert.True(t, err.ShouldPrintUsage())
	})

	t.Run("org-bound oauth rejects --org-id naming a different org", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String(), OrgName: "Censys"}, nil)
		_, err := c.ResolveOrgID(ctx, orgFlag(otherUUID))
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "the organization [Censys]")
	})

	t.Run("free-account oauth rejects --org-id", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth}, nil)
		_, err := c.ResolveOrgID(ctx, orgFlag(otherUUID))
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "scoped to your free account")
	})

	t.Run("oauth never consults the stored org-id global", func(t *testing.T) {
		// No GetLastUsedGlobalByName expectation: the mock controller fails the
		// test if the OAuth path reads the global.
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String()}, nil)
		got, err := c.ResolveOrgID(ctx, none)
		require.Nil(t, err)
		assert.Equal(t, sessionUUID.String(), got.MustGet().String())
	})

	t.Run("free-account oauth without a flag resolves to none", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth}, nil)
		got, err := c.ResolveOrgID(ctx, none)
		require.Nil(t, err)
		assert.False(t, got.IsPresent())
	})

	t.Run("pat uses the flag when provided", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindPersonalAccessToken}, nil)
		got, err := c.ResolveOrgID(ctx, orgFlag(otherUUID))
		require.Nil(t, err)
		assert.Equal(t, otherUUID.String(), got.MustGet().String())
	})

	t.Run("pat falls back to the stored org-id global", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindPersonalAccessToken}, func(ds *storemocks.MockStore) {
			ds.EXPECT().GetLastUsedGlobalByName(gomock.Any(), config.OrgIDGlobalName).
				Return(&store.ValueForGlobal{Value: otherUUID.String()}, nil)
		})
		got, err := c.ResolveOrgID(ctx, none)
		require.Nil(t, err)
		assert.Equal(t, otherUUID.String(), got.MustGet().String())
	})

	t.Run("free-account credential: required-org commands say why", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth}, nil)
		_, err := c.ResolveRequiredOrgID(&cobra.Command{Use: "enrich"}, none)
		require.NotNil(t, err)
		assert.Equal(t, "Organization Required", err.Title())
		assert.Contains(t, err.Error(), "uses an organization API")
		assert.Contains(t, err.Error(), "Your current log in is scoped to your free account")
		// Must NOT suggest --org-id or config org-id, which cannot help here.
		assert.NotContains(t, err.Error(), "config org-id add")
	})

	t.Run("pat with no org configured still gets the org-id guidance", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindPersonalAccessToken}, func(ds *storemocks.MockStore) {
			ds.EXPECT().GetLastUsedGlobalByName(gomock.Any(), config.OrgIDGlobalName).
				Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound)
		})
		_, err := c.ResolveRequiredOrgID(&cobra.Command{Use: "enrich"}, none)
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "config org-id add")
	})

	t.Run("org-bound credential cannot read the free account", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String(), OrgName: "Censys"}, nil)
		err := c.EnsureFreeAccountAccess(&cobra.Command{Use: "credits"}, "To view your organization's credits run 'censys org credits'")
		require.NotNil(t, err)
		assert.Equal(t, "Free User Account Required", err.Title())
		assert.Contains(t, err.Error(), "the organization [Censys]")
		assert.Contains(t, err.Error(), "censys org credits")
	})

	t.Run("free-account and pat credentials may read the free account", func(t *testing.T) {
		free := newCtx(t, credential.Info{Kind: credential.KindOAuth}, nil)
		require.Nil(t, free.EnsureFreeAccountAccess(&cobra.Command{Use: "credits"}, ""))
		pat := newCtx(t, credential.Info{Kind: credential.KindPersonalAccessToken}, nil)
		require.Nil(t, pat.EnsureFreeAccountAccess(&cobra.Command{Use: "credits"}, ""))
	})

	t.Run("pat with no stored global resolves to none", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindPersonalAccessToken}, func(ds *storemocks.MockStore) {
			ds.EXPECT().GetLastUsedGlobalByName(gomock.Any(), config.OrgIDGlobalName).
				Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound)
		})
		got, err := c.ResolveOrgID(ctx, none)
		require.Nil(t, err)
		assert.False(t, got.IsPresent())
	})
}

// Every command that accepts --org-id must reject it the same way under a
// credential that defines its own organization — including the org-required
// commands, which resolve through ResolveRequiredOrgID.
func TestOrgIDFlagRejectedConsistently(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	viper.Reset()
	cfg, cErr := config.New(t.TempDir())
	require.NoError(t, cErr)

	c := NewCommandContext(cfg, storemocks.NewMockStore(ctrl))
	cli := clientmocks.NewMockClient(ctrl)
	cli.EXPECT().CredentialInfo().
		Return(credential.Info{Kind: credential.KindOAuth}).AnyTimes()
	c.SetCensysClient(cli)

	flag := mo.Some(identifiers.NewOrganizationID(uuid.New()))

	// Optional-org commands (search, view, history, censeye, aggregate).
	_, err := c.ResolveOrgID(context.Background(), flag)
	require.NotNil(t, err)
	assert.Equal(t, "Organization ID Not Applicable", err.Title())

	// Required-org commands (enrich, org *): same error, not "Organization Required".
	_, rErr := c.ResolveRequiredOrgID(&cobra.Command{Use: "enrich"}, flag)
	require.NotNil(t, rErr)
	assert.Equal(t, "Organization ID Not Applicable", rErr.Title())
}
