package command

import (
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/flags"
)

func TestParseWaitFlags(t *testing.T) {
	const defaultTimeout = 30 * time.Minute

	testCases := []struct {
		name          string
		args          []string
		expectWait    bool
		expectTimeout mo.Option[time.Duration]
		expectErr     string
	}{
		{name: "no flags", expectTimeout: mo.Some(defaultTimeout)},
		{name: "wait uses the default timeout", args: []string{"--wait"}, expectWait: true, expectTimeout: mo.Some(defaultTimeout)},
		{name: "wait with a timeout", args: []string{"--wait", "--timeout", "5m"}, expectWait: true, expectTimeout: mo.Some(5 * time.Minute)},
		{name: "zero timeout means no limit", args: []string{"-w", "--timeout", "0"}, expectWait: true, expectTimeout: mo.None[time.Duration]()},
		{name: "timeout without wait", args: []string{"--timeout", "5m"}, expectErr: "--timeout only applies while polling"},
		{name: "negative timeout", args: []string{"--wait", "--timeout", "-1m"}, expectErr: "must not be negative"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			waitFlag := flags.NewBoolFlag(cmd.Flags(), "wait", "w", false, "")
			timeoutFlag := flags.NewHumanDurationFlag(cmd.Flags(), false, "timeout", "", mo.Some(defaultTimeout), "")
			require.NoError(t, cmd.Flags().Parse(tc.args))

			wait, timeout, err := ParseWaitFlags(cmd, waitFlag, timeoutFlag)

			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				assert.True(t, err.ShouldPrintUsage())
				return
			}
			require.Nil(t, err)
			assert.Equal(t, tc.expectWait, wait)
			assert.Equal(t, tc.expectTimeout, timeout)
		})
	}
}
