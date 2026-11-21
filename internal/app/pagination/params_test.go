package pagination

import (
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name     string
		pageSize mo.Option[uint64]
		maxPages mo.Option[uint64]
		wantErr  string
	}{
		{
			name:     "absent values are valid",
			pageSize: mo.None[uint64](),
			maxPages: mo.None[uint64](),
		},
		{
			name:     "positive values are valid",
			pageSize: mo.Some[uint64](10),
			maxPages: mo.Some[uint64](1),
		},
		{
			name:     "zero page size is rejected",
			pageSize: mo.Some[uint64](0),
			maxPages: mo.None[uint64](),
			wantErr:  "page size must be greater than 0",
		},
		{
			name:     "zero max pages is rejected",
			pageSize: mo.None[uint64](),
			maxPages: mo.Some[uint64](0),
			wantErr:  "max pages must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateParams(tc.pageSize, tc.maxPages)
			if tc.wantErr == "" {
				require.Nil(t, err)
				return
			}
			require.NotNil(t, err)
			require.Equal(t, tc.wantErr, err.Error())
			require.Equal(t, "Invalid Pagination Parameters", err.Title())
			require.True(t, err.ShouldPrintUsage())
		})
	}
}

func TestOptionalInt64(t *testing.T) {
	require.Equal(t, mo.None[int64](), OptionalInt64(mo.None[uint64]()))
	require.Equal(t, mo.Some[int64](25), OptionalInt64(mo.Some[uint64](25)))
}
