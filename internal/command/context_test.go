package command

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/mo"
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

	t.Run("org-bound oauth with a matching flag is allowed", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String()}, nil)
		got, err := c.ResolveOrgID(ctx, orgFlag(sessionUUID))
		require.Nil(t, err)
		assert.Equal(t, sessionUUID.String(), got.MustGet().String())
	})

	// A flag that disagrees with the session is NOT rejected locally: it is sent
	// and the API decides (and its 403 carries a scope hint).
	t.Run("org-bound oauth passes a differing flag through", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth, OrgID: sessionUUID.String(), OrgName: "Censys"}, nil)
		got, err := c.ResolveOrgID(ctx, orgFlag(otherUUID))
		require.Nil(t, err)
		assert.Equal(t, otherUUID.String(), got.MustGet().String())
	})

	t.Run("free-account oauth passes an org flag through", func(t *testing.T) {
		c := newCtx(t, credential.Info{Kind: credential.KindOAuth}, nil)
		got, err := c.ResolveOrgID(ctx, orgFlag(otherUUID))
		require.Nil(t, err)
		assert.Equal(t, otherUUID.String(), got.MustGet().String())
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
