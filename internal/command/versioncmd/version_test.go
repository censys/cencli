package versioncmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func TestVersion_PrintsJSONByDefault(t *testing.T) {
	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := command.NewCommandContext(cfg, storemocks.NewMockStore(ctrl))

	root, cerr := command.RootCommandToCobra(NewVersionCommand(ctx))
	require.NoError(t, cerr)
	root.SetArgs([]string{})

	require.NoError(t, root.Execute())
	// Output is JSON object; just assert it contains version key string token
	require.Contains(t, stdout.String(), "version")
}
