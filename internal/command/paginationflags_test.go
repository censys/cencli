package command

import (
	"testing"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/flags"
)

func TestParsePaginationFlags(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		expectPageSize mo.Option[uint64]
		expectMaxPages mo.Option[uint64]
		expectErr      string
	}{
		{name: "defaults", expectPageSize: mo.Some[uint64](100), expectMaxPages: mo.Some[uint64](1)},
		{name: "explicit values", args: []string{"--page-size", "5", "--max-pages", "3"}, expectPageSize: mo.Some[uint64](5), expectMaxPages: mo.Some[uint64](3)},
		{name: "-1 means all pages", args: []string{"--max-pages", "-1"}, expectPageSize: mo.Some[uint64](100), expectMaxPages: mo.None[uint64]()},
		{name: "zero max pages", args: []string{"--max-pages", "0"}, expectErr: "must be -1 or >= 1"},
		{name: "other negative max pages", args: []string{"--max-pages", "-2"}, expectErr: "must be -1 or >= 1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			pageSizeFlag := flags.NewIntegerFlag(cmd.Flags(), false, "page-size", "", mo.Some[int64](100), "", mo.Some[int64](1), mo.None[int64]())
			maxPagesFlag := flags.NewIntegerFlag(cmd.Flags(), false, "max-pages", "", mo.Some[int64](1), "", mo.None[int64](), mo.None[int64]())
			require.NoError(t, cmd.Flags().Parse(tc.args))

			pageSize, maxPages, err := ParsePaginationFlags(pageSizeFlag, maxPagesFlag)
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, tc.expectPageSize, pageSize)
			assert.Equal(t, tc.expectMaxPages, maxPages)
		})
	}
}
