package credential

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/config"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/store"
)

func TestActive(t *testing.T) {
	ctx := context.Background()

	t.Run("oauth wins when more recently used", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ds := mocks.NewMockStore(ctrl)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.OAuthSessionName).
			Return(&store.ValueForAuth{Name: config.OAuthSessionName, LastUsedAt: time.Now()}, nil)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).
			Return(&store.ValueForAuth{Name: config.AuthName, LastUsedAt: time.Now().Add(-time.Hour)}, nil)

		rec, kind, err := Active(ctx, ds)
		require.NoError(t, err)
		assert.Equal(t, KindOAuth, kind)
		assert.Equal(t, config.OAuthSessionName, rec.Name)
	})

	t.Run("pat wins when more recently used", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ds := mocks.NewMockStore(ctrl)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.OAuthSessionName).
			Return(&store.ValueForAuth{Name: config.OAuthSessionName, LastUsedAt: time.Now().Add(-time.Hour)}, nil)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).
			Return(&store.ValueForAuth{Name: config.AuthName, LastUsedAt: time.Now()}, nil)

		_, kind, err := Active(ctx, ds)
		require.NoError(t, err)
		assert.Equal(t, KindPersonalAccessToken, kind)
	})

	t.Run("none configured returns ErrAuthNotFound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ds := mocks.NewMockStore(ctrl)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.OAuthSessionName).Return(nil, authdom.ErrAuthNotFound)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(nil, authdom.ErrAuthNotFound)

		_, kind, err := Active(ctx, ds)
		require.ErrorIs(t, err, authdom.ErrAuthNotFound)
		assert.Equal(t, KindNone, kind)
	})
}

func TestResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("oauth session yields account and org binding", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ds := mocks.NewMockStore(ctrl)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.OAuthSessionName).
			Return(&store.ValueForAuth{
				Name:       config.OAuthSessionName,
				Value:      `{"access_token":"at","email":"u@censys.com","org_id":"org-1","org_name":"Censys"}`,
				LastUsedAt: time.Now(),
			}, nil)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(nil, authdom.ErrAuthNotFound)

		info, err := Resolve(ctx, ds)
		require.NoError(t, err)
		assert.Equal(t, KindOAuth, info.Kind)
		assert.Equal(t, "u@censys.com", info.Account)
		assert.Equal(t, "org-1", info.OrgID)
		assert.Equal(t, "Censys", info.OrgName)
		assert.True(t, info.IsOrgBoundOAuth())
		assert.False(t, info.IsFreeAccountOAuth())
	})

	t.Run("nothing configured yields KindNone without error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ds := mocks.NewMockStore(ctrl)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.OAuthSessionName).Return(nil, authdom.ErrAuthNotFound)
		ds.EXPECT().GetLastUsedAuthByName(ctx, config.AuthName).Return(nil, authdom.ErrAuthNotFound)

		info, err := Resolve(ctx, ds)
		require.NoError(t, err)
		assert.Equal(t, KindNone, info.Kind)
	})
}
