package auth

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/oauth"
	"github.com/censys/cencli/internal/store"
)

// stubTransport answers every request locally with 200 OK, so tests exercise
// the auth commands without reaching the real authorization server. The wire
// format of revoke/token requests is covered by the oauth package's own tests
// against an httptest server.
type stubTransport struct{}

func (stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// useStubOAuthClient points the auth commands at a stub-backed OAuth client for
// the duration of a test, restoring the real factory afterwards.
func useStubOAuthClient(t *testing.T) {
	t.Helper()
	orig := newOAuthClient
	newOAuthClient = func() *oauth.Client {
		return oauth.NewClient(oauth.Config{}, &http.Client{Transport: stubTransport{}})
	}
	t.Cleanup(func() { newOAuthClient = orig })
}

// newTestCommand builds the auth command tree against a mock store and
// captures stdout/stderr.
func newTestCommand(t *testing.T) (*storemocks.MockStore, *bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockStore := storemocks.NewMockStore(ctrl)
	ctx := command.NewCommandContext(cfg, mockStore)

	root, cerr := command.RootCommandToCobra(NewAuthCommand(ctx))
	require.NoError(t, cerr)

	return mockStore, &stdout, &stderr, func(args ...string) error {
		root.SetArgs(args)
		return root.Execute()
	}
}

func TestAuth_HelpShows(t *testing.T) {
	_, stdout, _, execute := newTestCommand(t)
	require.NoError(t, execute("--help"))
	out := stdout.String()
	assert.Contains(t, out, "login")
	assert.Contains(t, out, "logout")
	assert.Contains(t, out, "status")
}

func TestAuthStatus_NoCredentials(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.OAuthSessionName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)
	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.AuthName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)

	require.NoError(t, execute("status"))
	assert.Contains(t, stdout.String(), "No credentials configured")
	assert.Contains(t, stdout.String(), "censys auth login")
}

func TestAuthStatus_PAT(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.OAuthSessionName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)
	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.AuthName).
		Return(&store.ValueForAuth{
			Name:        config.AuthName,
			Description: "my-token",
			Value:       "pat-value",
			LastUsedAt:  time.Now(),
		}, nil)

	require.NoError(t, execute("status"))
	assert.Contains(t, stdout.String(), "personal access token [my-token]")
}

func TestAuthStatus_OAuthSession(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.OAuthSessionName).
		Return(&store.ValueForAuth{
			Name:        config.OAuthSessionName,
			Description: "zackaryia@zackaryia.com",
			Value:       `{"access_token":"ory_at_abc","refresh_token":"ory_rt_def","scope":"openid offline_access censys.api","email":"zackaryia@zackaryia.com","expires_at":"2100-01-01T00:00:00Z"}`,
			LastUsedAt:  time.Now(),
		}, nil)
	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.AuthName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)

	require.NoError(t, execute("status"))
	out := stdout.String()
	assert.Contains(t, out, "Logged in as [zackaryia@zackaryia.com]")
	// Only the "Logged in as" line is shown; no scopes or expiry details.
	assert.NotContains(t, out, "openid offline_access censys.api")
	assert.NotContains(t, out, "valid until")
}

func TestAuthStatus_JSONOutput(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.OAuthSessionName).
		Return(&store.ValueForAuth{
			Name:        config.OAuthSessionName,
			Description: "user@censys.com",
			Value:       `{"access_token":"ory_at_abc","email":"user@censys.com","org_id":"8c96558b-e12b-450b-91df-098960153f13","org_name":"Censys","expires_at":"2100-01-01T00:00:00Z"}`,
			LastUsedAt:  time.Now(),
		}, nil)
	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.AuthName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)

	require.NoError(t, execute("status", "--output-format", "json"))
	out := stdout.String()
	assert.Contains(t, out, `"method": "oauth"`)
	assert.Contains(t, out, `"scope": "organization"`)
	assert.Contains(t, out, `"organization_name": "Censys"`)
	assert.Contains(t, out, `"account": "user@censys.com"`)
}

func TestAuthStatus_OAuthSessionNoEmail(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.OAuthSessionName).
		Return(&store.ValueForAuth{
			Name:       config.OAuthSessionName,
			Value:      `{"access_token":"ory_at_abc","refresh_token":"ory_rt_def","expires_at":"2020-01-01T00:00:00Z"}`,
			LastUsedAt: time.Now(),
		}, nil)
	mockStore.EXPECT().GetLastUsedAuthByName(gomock.Any(), config.AuthName).
		Return((*store.ValueForAuth)(nil), authdom.ErrAuthNotFound)

	require.NoError(t, execute("status"))
	assert.Contains(t, stdout.String(), "Logged in via `censys auth login`")
}

func TestAuthLogout_NotLoggedIn(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)

	mockStore.EXPECT().GetValuesForAuth(gomock.Any(), config.OAuthSessionName).
		Return(nil, authdom.ErrAuthNotFound)

	require.NoError(t, execute("logout"))
	assert.Contains(t, stdout.String(), "not logged in")
}

func TestAuthLogout_RemovesSession(t *testing.T) {
	mockStore, stdout, _, execute := newTestCommand(t)
	useStubOAuthClient(t)

	mockStore.EXPECT().GetValuesForAuth(gomock.Any(), config.OAuthSessionName).
		Return([]*store.ValueForAuth{
			{
				ID:    7,
				Name:  config.OAuthSessionName,
				Value: `{"access_token":"ory_at_abc","refresh_token":"ory_rt_def"}`,
			},
		}, nil)
	mockStore.EXPECT().DeleteValueForAuth(gomock.Any(), int64(7)).
		Return(&store.ValueForAuth{ID: 7}, nil)

	// Revocation is best effort (and stubbed here); the contract we assert is
	// that the session is removed locally and the user is told they logged out.
	require.NoError(t, execute("logout"))
	assert.Contains(t, stdout.String(), "Logged out")
}

func TestAuthLogin_HelpShowsNoBrowserFlag(t *testing.T) {
	_, stdout, _, execute := newTestCommand(t)
	require.NoError(t, execute("login", "--help"))
	assert.Contains(t, stdout.String(), "no-browser")
}
