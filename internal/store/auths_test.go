package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type authsSuite struct {
	suite.Suite
	sctx, tctx       context.Context
	scancel, tcancel context.CancelFunc
	authsStore       AuthsStore
}

func (s *authsSuite) SetupSuite() {
	s.sctx, s.scancel = context.WithCancel(context.Background())
	if deadline, ok := s.T().Deadline(); ok {
		s.sctx, s.scancel = context.WithDeadline(s.sctx, deadline)
	}
}

func (s *authsSuite) TearDownSuite() {
	s.scancel()
}

func (s *authsSuite) SetupTest() {
	s.tctx, s.tcancel = context.WithCancel(context.Background())
	if deadline, ok := s.T().Deadline(); ok {
		s.tctx, s.tcancel = context.WithDeadline(s.tctx, deadline)
	}
	dir := s.T().TempDir()
	var err error
	s.authsStore, err = New(dir)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), s.authsStore)
}

func (s *authsSuite) TearDownTest() {
	s.tcancel()
}

func TestAuthsSuite(t *testing.T) {
	suite.Run(t, new(authsSuite))
}

func (s *authsSuite) TestAuths_CRUD() {
	now := time.Now()
	name := "test-name"
	description := "test-description"
	value := "test-value"

	auth, err := s.authsStore.AddValueForAuth(s.tctx, name, description, value)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth)

	assert.Equal(s.T(), description, auth.Description)
	assert.Equal(s.T(), name, auth.Name)
	assert.Equal(s.T(), value, auth.Value)
	assert.WithinDuration(s.T(), now, auth.CreatedAt, time.Second)
	assert.WithinDuration(s.T(), now, auth.LastUsedAt, time.Second)

	values, err := s.authsStore.GetValuesForAuth(s.tctx, name)
	require.NoError(s.T(), err)
	require.Len(s.T(), values, 1)
	assert.Equal(s.T(), auth.ID, values[0].ID)
	assert.Equal(s.T(), auth.Description, values[0].Description)
	assert.Equal(s.T(), auth.Name, values[0].Name)
	assert.Equal(s.T(), auth.Value, values[0].Value)
	assert.WithinDuration(s.T(), auth.CreatedAt, values[0].CreatedAt, time.Second)
	assert.WithinDuration(s.T(), auth.LastUsedAt, values[0].LastUsedAt, time.Second)

	deleted, err := s.authsStore.DeleteValueForAuth(s.tctx, auth.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), deleted)
	assert.Equal(s.T(), auth.ID, deleted.ID)
	assert.Equal(s.T(), auth.Description, deleted.Description)
	assert.Equal(s.T(), auth.Name, deleted.Name)
	assert.Equal(s.T(), auth.Value, deleted.Value)
	assert.WithinDuration(s.T(), auth.CreatedAt, deleted.CreatedAt, time.Second)
	assert.WithinDuration(s.T(), auth.LastUsedAt, deleted.LastUsedAt, time.Second)
}

func (s *authsSuite) TestAuths_UpdateLastUsedAtToNow() {
	authName := "test-auth"

	auth1, err := s.authsStore.AddValueForAuth(s.tctx, authName, "first-auth", "value1")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth1)

	time.Sleep(1 * time.Second)

	auth2, err := s.authsStore.AddValueForAuth(s.tctx, authName, "second-auth", "value2")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth2)

	time.Sleep(1 * time.Second)

	auth3, err := s.authsStore.AddValueForAuth(s.tctx, authName, "third-auth", "value3")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth3)

	lastUsed, err := s.authsStore.GetLastUsedAuthByName(s.tctx, authName)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastUsed)
	assert.Equal(s.T(), auth3.ID, lastUsed.ID)
	assert.Equal(s.T(), "third-auth", lastUsed.Description)
	assert.Equal(s.T(), "value3", lastUsed.Value)

	updateTime := time.Now()
	err = s.authsStore.UpdateAuthLastUsedAtToNow(s.tctx, auth1.ID)
	require.NoError(s.T(), err)

	updatedLastUsed, err := s.authsStore.GetLastUsedAuthByName(s.tctx, authName)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), updatedLastUsed)
	assert.Equal(s.T(), auth1.ID, updatedLastUsed.ID)
	assert.Equal(s.T(), "first-auth", updatedLastUsed.Description)
	assert.Equal(s.T(), "value1", updatedLastUsed.Value)

	assert.WithinDuration(s.T(), updateTime, updatedLastUsed.LastUsedAt, time.Second)

	assert.WithinDuration(s.T(), auth1.CreatedAt, updatedLastUsed.CreatedAt, time.Second)
}
