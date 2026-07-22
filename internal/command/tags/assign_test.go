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

func runAssignCommand(t *testing.T, svc apptags.Service, args []string, stdin io.Reader) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewAssignCommand(cmdContext))
	require.NoError(t, buildErr)

	if stdin != nil {
		rootCmd.SetIn(stdin)
	}
	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

// assignResult builds a service result with the given assigned assets and
// optional failures, echoing the tag identifier.
func assignResult(tagID string, assigned []string, failures map[string]string) apptags.AssignResult {
	res := apptags.AssignResult{Meta: okMeta(), TagID: tagID}
	for _, a := range assigned {
		res.Assignments = append(res.Assignments, apptags.Assignment{
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
			cenclierrors.NewCencliError(errors.New("some assets failed")))
	}
	return res
}

// assignNoCallService returns a service that must not be called (validation is
// expected to fail before the service is reached).
func assignNoCallService(ctrl *gomock.Controller) apptags.Service {
	return tagsmocks.NewMockTagsService(ctrl)
}

func TestTagsAssignCommand(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		stdin   io.Reader
		service func(t *testing.T, ctrl *gomock.Controller) apptags.Service
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "positional assets assigned - short output with index-lag note",
			args: []string{"alpha", "8.8.8.8", "1.1.1.1"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return assignResult("alpha", []string{"8.8.8.8", "1.1.1.1"}, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
				require.Contains(t, stderr, "few minutes")
			},
		},
		{
			name: "comma-separated positional assets are split",
			args: []string{"alpha", "8.8.8.8,1.1.1.1"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return assignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "mixed asset types are accepted in one call",
			args: []string{"alpha", "8.8.8.8", "platform.censys.io:443"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
						require.ElementsMatch(t, []string{"8.8.8.8", "platform.censys.io:443"}, p.AssetIDs)
						return assignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "stdin input via --input-file -",
			args:  []string{"alpha", "--input-file", "-"},
			stdin: bytes.NewBufferString("8.8.8.8\n1.1.1.1\n"),
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
						return assignResult("alpha", p.AssetIDs, nil), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "unknown asset is rejected before the service is called",
			args: []string{"alpha", "8.8.8.8", "not-an-asset"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return assignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not-an-asset")
			},
		},
		{
			name: "no assets is an error",
			args: []string{"alpha"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return assignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "empty tag id is rejected",
			args: []string{"   ", "8.8.8.8"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return assignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
			},
		},
		{
			name: "partial failure is surfaced to stderr but data still renders",
			args: []string{"alpha", "8.8.8.8", "1.1.1.1"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).Return(
					assignResult("alpha", []string{"8.8.8.8"}, map[string]string{"1.1.1.1": "Forbidden"}),
					nil,
				)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
				require.Contains(t, stderr, "some assets failed")
			},
		},
		{
			name: "json output renders per-asset payload",
			args: []string{"alpha", "8.8.8.8", "--output-format", "json"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).Return(
					assignResult("alpha", []string{"8.8.8.8"}, nil), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"asset": "8.8.8.8"`)
				require.Contains(t, stdout, `"assigned": true`)
			},
		},
		{
			name: "org id flag is threaded to the service",
			args: []string{"alpha", "8.8.8.8", "--org-id", "11111111-1111-1111-1111-111111111111"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
						require.True(t, p.OrgID.IsPresent())
						return assignResult("alpha", p.AssetIDs, nil), nil
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

			stdout, stderr, err := runAssignCommand(t, tc.service(t, ctrl), tc.args, tc.stdin)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

func TestTagsAssignCommand_InputFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dir := t.TempDir()
	file := filepath.Join(dir, "assets.txt")
	require.NoError(t, os.WriteFile(file, []byte("8.8.8.8\n1.1.1.1\n"), 0o600))

	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().Assign(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p apptags.AssignParams) (apptags.AssignResult, cenclierrors.CencliError) {
			require.Equal(t, []string{"8.8.8.8", "1.1.1.1"}, p.AssetIDs)
			return assignResult("alpha", p.AssetIDs, nil), nil
		})

	_, _, err := runAssignCommand(t, m, []string{"alpha", "--input-file", file}, nil)
	require.NoError(t, err)
}
