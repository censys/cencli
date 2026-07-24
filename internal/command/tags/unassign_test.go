package tags

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	tagsmocks "github.com/censys/cencli/gen/app/tags/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	apptags "github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

type unassignSeams struct {
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

func runUnassignCommand(t *testing.T, svc apptags.Service, seams unassignSeams, args []string, stdin io.Reader) (stdout, stderr string, err error) {
	t.Helper()

	tempDir := t.TempDir()
	viper.Reset()
	cfg, cfgErr := config.New(tempDir)
	require.NoError(t, cfgErr)

	var outBuf, errBuf bytes.Buffer
	formatter.Stdout = &outBuf
	formatter.Stderr = &errBuf

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := storemocks.NewMockStore(ctrl)
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithTagsService(svc))
	cmd := NewUnassignCommand(cmdContext)
	if seams.confirm != nil {
		cmd.confirm = seams.confirm
	}
	if seams.stdinIsTTY != nil {
		cmd.stdinIsTTY = seams.stdinIsTTY
	}
	rootCmd, buildErr := command.RootCommandToCobra(cmd)
	require.NoError(t, buildErr)

	if stdin != nil {
		rootCmd.SetIn(stdin)
	}
	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

// unassignResult builds a service result with the given removed assets and
// optional failures, echoing the tag identifier.
func unassignResult(tagID string, unassigned []string, failures map[string]string) apptags.UnassignResult {
	res := apptags.UnassignResult{Meta: okMeta(), TagID: tagID}
	for _, a := range unassigned {
		res.Unassigned = append(res.Unassigned, apptags.Assignment{
			ID: a + "-id", AssetID: a, AssetType: "host", TagID: tagID,
			PlatformRef: "https://platform.censys.io/hosts/" + a,
		})
	}
	for asset, msg := range failures {
		res.Failures = append(res.Failures, apptags.AssignmentFailure{
			AssetID: asset, Err: cenclierrors.NewCencliError(errors.New(msg)),
		})
	}
	if len(res.Failures) > 0 {
		res.PartialError = cenclierrors.ToPartialError(
			cenclierrors.NewCencliError(errors.New("some assets failed to unassign")))
	}
	return res
}

// unassignNoCallService returns a service that must not be called (validation or
// an aborted confirmation is expected before the service is reached).
func unassignNoCallService(ctrl *gomock.Controller) apptags.Service {
	return tagsmocks.NewMockTagsService(ctrl)
}

// ttySeams simulates an interactive terminal with a confirm that returns the
// given answer, recording whether it was invoked.
func ttySeams(answer bool, called *bool) unassignSeams {
	return unassignSeams{
		confirm: func(_ context.Context, _ string) (bool, error) {
			if called != nil {
				*called = true
			}
			return answer, nil
		},
		stdinIsTTY: func() bool { return true },
	}
}

// noPromptSeams fails the test if the confirm prompt is ever shown.
func noPromptSeams(t *testing.T) unassignSeams {
	t.Helper()
	return unassignSeams{
		confirm: func(_ context.Context, _ string) (bool, error) {
			t.Fatal("confirm prompt should not be shown")
			return false, nil
		},
		stdinIsTTY: func() bool { return true },
	}
}

func TestTagsUnassignCommand(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		stdin   io.Reader
		seams   func(t *testing.T) unassignSeams
		service func(t *testing.T, ctrl *gomock.Controller) apptags.Service
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name:  "single positional asset does not prompt",
			args:  []string{"alpha", "8.8.8.8"},
			seams: noPromptSeams,
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8"}, p.AssetIDs)
						return unassignResult("alpha", []string{"8.8.8.8"}, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stderr, "few minutes")
			},
		},
		{
			name:  "multiple assets prompt and proceed when accepted",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1"},
			seams: func(t *testing.T) unassignSeams { called := false; return ttySeams(true, &called) },
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return unassignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			name:  "multiple assets aborted when declined - service not called",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1"},
			seams: func(t *testing.T) unassignSeams { called := false; return ttySeams(false, &called) },
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return unassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "Unassign aborted.")
			},
		},
		{
			name:    "multiple assets non-TTY without --yes requires confirmation",
			args:    []string{"alpha", "8.8.8.8", "1.1.1.1"},
			seams:   func(_ *testing.T) unassignSeams { return unassignSeams{stdinIsTTY: func() bool { return false }} },
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service { return unassignNoCallService(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "confirmation required")
			},
		},
		{
			name:  "--yes skips the prompt for multiple assets",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1", "--yes"},
			seams: noPromptSeams,
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return unassignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "comma-separated positional assets are split",
			args:  []string{"alpha", "8.8.8.8,1.1.1.1", "--yes"},
			seams: noPromptSeams,
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return unassignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "stdin input via --input-file - (prompts, skipped with --yes)",
			args:  []string{"alpha", "--input-file", "-", "--yes"},
			stdin: bytes.NewBufferString("8.8.8.8\n1.1.1.1\n"),
			seams: noPromptSeams,
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return unassignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "unknown asset is rejected before the service is called",
			args:    []string{"alpha", "8.8.8.8", "not-an-asset"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service { return unassignNoCallService(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not-an-asset")
			},
		},
		{
			name:    "no assets is an error",
			args:    []string{"alpha"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service { return unassignNoCallService(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name:    "empty tag id is rejected",
			args:    []string{"   ", "8.8.8.8"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service { return unassignNoCallService(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
			},
		},
		{
			name:  "partial failure is surfaced to stderr but data still renders",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1", "--yes"},
			seams: noPromptSeams,
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).Return(
					unassignResult("alpha", []string{"8.8.8.8"}, map[string]string{"1.1.1.1": "not assigned"}),
					nil,
				)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
				require.Contains(t, stderr, "some assets failed to unassign")
			},
		},
		{
			name: "json output renders per-asset payload",
			args: []string{"alpha", "8.8.8.8", "--output-format", "json"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).Return(
					unassignResult("alpha", []string{"8.8.8.8"}, nil), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"asset": "8.8.8.8"`)
				require.Contains(t, stdout, `"unassigned": true`)
			},
		},
		{
			name: "org id flag is threaded to the service",
			args: []string{"alpha", "8.8.8.8", "--org-id", "11111111-1111-1111-1111-111111111111"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
						require.True(t, p.OrgID.IsPresent())
						return unassignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var seams unassignSeams
			if tc.seams != nil {
				seams = tc.seams(t)
			}
			stdout, stderr, err := runUnassignCommand(t, tc.service(t, ctrl), seams, tc.args, tc.stdin)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

func TestTagsUnassignCommand_InputFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dir := t.TempDir()
	file := filepath.Join(dir, "assets.txt")
	require.NoError(t, os.WriteFile(file, []byte("8.8.8.8\n1.1.1.1\n"), 0o600))

	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().Unassign(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p apptags.UnassignParams) (apptags.UnassignResult, cenclierrors.CencliError) {
			require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
			return unassignResult("alpha", p.AssetIDs, nil), nil
		})

	_, _, err := runUnassignCommand(t, m, noPromptSeams(t), []string{"alpha", "--input-file", file, "--yes"}, nil)
	require.NoError(t, err)
}
