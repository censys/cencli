package identifiers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTagID(t *testing.T) {
	t.Run("name has no UID", func(t *testing.T) {
		id := NewTagID("my-tag")
		require.Equal(t, "my-tag", id.String())
		require.True(t, id.UID().IsAbsent())
	})

	t.Run("uuid parses into UID", func(t *testing.T) {
		u := uuid.New()
		id := NewTagID(u.String())
		require.Equal(t, u.String(), id.String())
		require.True(t, id.UID().IsPresent())
		require.Equal(t, u, id.UID().MustGet())
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		id := NewTagID("  spaced-tag  ")
		require.Equal(t, "spaced-tag", id.String())
		require.True(t, id.UID().IsAbsent())
	})

	t.Run("uppercase uuid still parses", func(t *testing.T) {
		id := NewTagID("6BA7B810-9DAD-11D1-80B4-00C04FD430C8")
		require.True(t, id.UID().IsPresent())
	})
}
