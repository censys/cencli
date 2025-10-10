package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type globalsSuite struct {
	suite.Suite
	sctx, tctx       context.Context
	scancel, tcancel context.CancelFunc
	globalsStore     GlobalsStore
}

func (s *globalsSuite) SetupSuite() {
	s.sctx, s.scancel = context.WithCancel(context.Background())
	if deadline, ok := s.T().Deadline(); ok {
		s.sctx, s.scancel = context.WithDeadline(s.sctx, deadline)
	}
}

func (s *globalsSuite) TearDownSuite() {
	s.scancel()
}

func (s *globalsSuite) SetupTest() {
	s.tctx, s.tcancel = context.WithCancel(context.Background())
	if deadline, ok := s.T().Deadline(); ok {
		s.tctx, s.tcancel = context.WithDeadline(s.tctx, deadline)
	}
	dir := s.T().TempDir()
	var err error
	s.globalsStore, err = New(dir)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), s.globalsStore)
}

func (s *globalsSuite) TearDownTest() {
	s.tcancel()
}

func TestGlobalsSuite(t *testing.T) {
	suite.Run(t, new(globalsSuite))
}

func (s *globalsSuite) TestGlobals_CRUD() {
	now := time.Now()
	name := "test-name"
	description := "test-description"
	value := "test-value"

	global, err := s.globalsStore.AddValueForGlobal(s.tctx, name, description, value)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), global)

	assert.Equal(s.T(), description, global.Description)
	assert.Equal(s.T(), name, global.Name)
	assert.Equal(s.T(), value, global.Value)
	assert.WithinDuration(s.T(), now, global.CreatedAt, time.Second)
	assert.WithinDuration(s.T(), now, global.LastUsedAt, time.Second)

	values, err := s.globalsStore.GetValuesForGlobal(s.tctx, name)
	require.NoError(s.T(), err)
	require.Len(s.T(), values, 1)
	assert.Equal(s.T(), global.ID, values[0].ID)
	assert.Equal(s.T(), global.Description, values[0].Description)
	assert.Equal(s.T(), global.Name, values[0].Name)
	assert.Equal(s.T(), global.Value, values[0].Value)
	assert.WithinDuration(s.T(), global.CreatedAt, values[0].CreatedAt, time.Second)
	assert.WithinDuration(s.T(), global.LastUsedAt, values[0].LastUsedAt, time.Second)

	deleted, err := s.globalsStore.DeleteValueForGlobal(s.tctx, global.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), deleted)
	assert.Equal(s.T(), global.ID, deleted.ID)
	assert.Equal(s.T(), global.Description, deleted.Description)
	assert.Equal(s.T(), global.Name, deleted.Name)
	assert.Equal(s.T(), global.Value, deleted.Value)
	assert.WithinDuration(s.T(), global.CreatedAt, deleted.CreatedAt, time.Second)
	assert.WithinDuration(s.T(), global.LastUsedAt, deleted.LastUsedAt, time.Second)
}

func (s *globalsSuite) TestGlobals_UpdateLastUsedAtToNow() {
	globalName := "test-global"

	global1, err := s.globalsStore.AddValueForGlobal(s.tctx, globalName, "first-global", "value1")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), global1)

	time.Sleep(1 * time.Second)

	global2, err := s.globalsStore.AddValueForGlobal(s.tctx, globalName, "second-global", "value2")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), global2)

	time.Sleep(1 * time.Second)

	global3, err := s.globalsStore.AddValueForGlobal(s.tctx, globalName, "third-global", "value3")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), global3)

	lastUsed, err := s.globalsStore.GetLastUsedGlobalByName(s.tctx, globalName)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastUsed)
	assert.Equal(s.T(), global3.ID, lastUsed.ID)
	assert.Equal(s.T(), "third-global", lastUsed.Description)
	assert.Equal(s.T(), "value3", lastUsed.Value)

	updateTime := time.Now()
	err = s.globalsStore.UpdateGlobalLastUsedAtToNow(s.tctx, global1.ID)
	require.NoError(s.T(), err)

	updatedLastUsed, err := s.globalsStore.GetLastUsedGlobalByName(s.tctx, globalName)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), updatedLastUsed)
	assert.Equal(s.T(), global1.ID, updatedLastUsed.ID)
	assert.Equal(s.T(), "first-global", updatedLastUsed.Description)
	assert.Equal(s.T(), "value1", updatedLastUsed.Value)

	assert.WithinDuration(s.T(), updateTime, updatedLastUsed.LastUsedAt, time.Second)

	assert.WithinDuration(s.T(), global1.CreatedAt, updatedLastUsed.CreatedAt, time.Second)
}
