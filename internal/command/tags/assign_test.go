package tags

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samber/mo"
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

// assignSeams overrides the interactive dependencies a bulk assignment uses. The
// explicit-asset mode never touches them, so they can be left unset there.
type assignSeams struct {
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
	// quiet stands in for the global --quiet flag, which lives on the real root
	// command and so is not registered when a subcommand is mounted alone.
	quiet bool
}

func runAssignCommand(t *testing.T, svc apptags.Service, args []string, stdin io.Reader) (stdout, stderr string, err error) {
	t.Helper()
	return runAssignCommandWithSeams(t, svc, assignSeams{}, args, stdin)
}

func runAssignCommandWithSeams(
	t *testing.T,
	svc apptags.Service,
	seams assignSeams,
	args []string,
	stdin io.Reader,
) (stdout, stderr string, err error) {
	t.Helper()

	tempDir := t.TempDir()
	viper.Reset()
	cfg, cfgErr := config.New(tempDir)
	require.NoError(t, cfgErr)
	if seams.quiet {
		// PreRun re-reads the config from viper, so setting the struct field would
		// be overwritten; viper is also where the real --quiet flag lands.
		viper.Set("quiet", true)
	}

	var outBuf, errBuf bytes.Buffer
	formatter.Stdout = &outBuf
	formatter.Stderr = &errBuf

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := storemocks.NewMockStore(ctrl)
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithTagsService(svc))
	cmd := NewAssignCommand(cmdContext)
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
			AssetID: asset,
			Err:     cenclierrors.NewCencliError(errors.New(msg)),
			// The service reduces every failure to a one-line Detail; the views
			// read that, not Err, so the fixture has to carry it too.
			Detail: msg,
			Status: mo.Some(int64(409)),
		})
	}
	// Mirrors the service: a partial error only when something also succeeded.
	// A run where every asset failed is not partial, and the command turns it
	// into a non-zero exit itself.
	if len(res.Failures) > 0 && len(res.Assignments) > 0 {
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
		seams   assignSeams
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
			// Explicit assignment never prompts, so --yes has nothing to skip.
			// It is rejected rather than ignored, like every other bulk-only flag.
			name: "--yes is rejected in explicit mode",
			args: []string{"alpha", "8.8.8.8", "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return assignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--yes only applies to a bulk assignment")
			},
		},
		{
			name:  "--quiet suppresses the index-lag note in explicit mode",
			args:  []string{"alpha", "8.8.8.8"},
			seams: assignSeams{quiet: true},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Assign(gomock.Any(), gomock.Any()).Return(
					assignResult("alpha", []string{"8.8.8.8"}, nil), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Empty(t, stderr)
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

			stdout, stderr, err := runAssignCommandWithSeams(t, tc.service(t, ctrl), tc.seams, tc.args, tc.stdin)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

// bulkSubmitted is what the service returns for an accepted bulk job.
func bulkSubmitted(status string) apptags.BulkAssignResult {
	return apptags.BulkAssignResult{Meta: okMeta(), Operation: operation(testOperationID, status)}
}

// bulkNoCallService asserts no bulk submit happens, proving the input was
// rejected at the command boundary.
func bulkNoCallService(ctrl *gomock.Controller) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().Assign(gomock.Any(), gomock.Any()).Times(0)
	return m
}

// bulkSubmitOnly expects a submit and no polling.
func bulkSubmitOnly(ctrl *gomock.Controller, status string) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted(status), nil)
	return m
}

// bulkSubmitAndWait expects a submit followed by polling that ends on the given
// status.
func bulkSubmitAndWait(ctrl *gomock.Controller, finalStatus string) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted("pending"), nil)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
		apptags.GetOperationResult{Meta: okMeta(), Operation: finishedOperation(finalStatus)}, nil)
	return m
}

// alwaysTTY makes the command believe it can prompt.
func alwaysTTY() func() bool { return func() bool { return true } }

// TestTagsAssignCommand_AllAssetsFail pins the contract for a run where no asset
// succeeded: the per-asset results still render in every output mode, and the
// exit code is non-zero. Re-assigning already-tagged assets makes this the
// common failure, not an edge case - the API returns 409 for every one.
func TestTagsAssignCommand_AllAssetsFail(t *testing.T) {
	allFailed := func(ctrl *gomock.Controller) apptags.Service {
		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().Assign(gomock.Any(), gomock.Any()).Return(
			assignResult("alpha", nil, map[string]string{
				"8.8.8.8": "assignment already exists",
				"1.1.1.1": "assignment already exists",
			}), nil)
		return m
	}

	t.Run("short output lists every failed asset and exits non-zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		stdout, stderr, err := runAssignCommand(t, allFailed(ctrl),
			[]string{"alpha", "8.8.8.8", "1.1.1.1"}, nil)

		require.Error(t, err)
		require.Equal(t, 1, formatter.ExitCode(err))
		// Both assets named, not just whichever failed first.
		require.Contains(t, stdout, "8.8.8.8")
		require.Contains(t, stdout, "1.1.1.1")
		require.Contains(t, stdout, "already exists")
		require.Contains(t, err.Error(), "2 of 2 failed")
		// Nothing was tagged, so the index-lag note would be nonsense.
		require.NotContains(t, stderr, "few minutes")
	})

	t.Run("json output still emits the full array", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		stdout, _, err := runAssignCommand(t, allFailed(ctrl),
			[]string{"alpha", "8.8.8.8", "1.1.1.1", "--output-format", "json"}, nil)

		require.Error(t, err)
		// The whole point: a script gets parseable results alongside the failure,
		// where it previously got an empty stdout.
		require.Contains(t, stdout, `"asset": "8.8.8.8"`)
		require.Contains(t, stdout, `"asset": "1.1.1.1"`)
		require.Contains(t, stdout, `"assigned": false`)
		// A one-line reason plus the status as a number, not the API's whole
		// problem document escaped into a string.
		require.Contains(t, stdout, `"error": "assignment already exists"`)
		require.Contains(t, stdout, `"error_status": 409`)
		require.NotContains(t, stdout, `\"title\"`)
	})

	t.Run("a cut-short run does not claim every asset failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// One failure recorded out of three assets: the loop stopped early, so
		// the message must count what was attempted, not what was asked for.
		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().Assign(gomock.Any(), gomock.Any()).Return(
			assignResult("alpha", nil, map[string]string{"8.8.8.8": "boom"}), nil)

		_, _, err := runAssignCommand(t, m,
			[]string{"alpha", "8.8.8.8", "1.1.1.1", "9.9.9.9"}, nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "1 of 3 failed")
	})
}

func TestTagsAssignCommand_Bulk(t *testing.T) {
	const query = "host.services.port: 22"

	testCases := []struct {
		name    string
		args    []string
		seams   assignSeams
		service func(t *testing.T, ctrl *gomock.Controller) apptags.Service
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "--query with positional assets is a mode conflict",
			args: []string{"alpha", "8.8.8.8", "--query", query},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be combined with explicit assets")
			},
		},
		{
			name: "--query with --input-file is a mode conflict",
			args: []string{"alpha", "--input-file", "-", "--query", query},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be combined with explicit assets")
			},
		},
		{
			name: "blank --query is rejected",
			args: []string{"alpha", "--query", "   "},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--query must not be empty")
			},
		},
		{
			name: "--max-assets without --query is rejected",
			args: []string{"alpha", "8.8.8.8", "--max-assets", "10"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--max-assets only applies to a bulk assignment")
			},
		},
		{
			name: "--wait without --query is rejected",
			args: []string{"alpha", "8.8.8.8", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--wait only applies to a bulk assignment")
			},
		},
		{
			name: "--timeout without --wait is rejected",
			args: []string{"alpha", "--query", query, "--timeout", "5m"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout only applies while polling")
			},
		},
		{
			name:  "non-interactive without --yes refuses to submit",
			args:  []string{"alpha", "--query", query},
			seams: assignSeams{stdinIsTTY: func() bool { return false }},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "confirmation required")
			},
		},
		{
			name: "declining the prompt aborts without submitting",
			args: []string{"alpha", "--query", query},
			seams: assignSeams{
				stdinIsTTY: alwaysTTY(),
				confirm:    func(context.Context, string) (bool, error) { return false, nil },
			},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "Assignment aborted.")
			},
		},
		{
			name: "--yes submits and reports the operation with a track hint",
			args: []string{"alpha", "--query", query, "--yes"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkAssignParams) (apptags.BulkAssignResult, cenclierrors.CencliError) {
						require.Equal(t, query, p.Query)
						require.Equal(t, "alpha", p.TagID.String())
						require.False(t, p.MaxAssets.IsPresent())
						return bulkSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Tag Operation")
				require.Contains(t, stdout, testOperationID)
				require.Contains(t, stderr, "Track with: censys tags operations get alpha "+testOperationID)
				require.Contains(t, stderr, "few minutes")
			},
		},
		{
			name: "--max-assets is threaded to the service",
			args: []string{"alpha", "--query", query, "--max-assets", "250", "--yes"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkAssignParams) (apptags.BulkAssignResult, cenclierrors.CencliError) {
						require.True(t, p.MaxAssets.IsPresent())
						require.Equal(t, int64(250), p.MaxAssets.MustGet())
						return bulkSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "org id is threaded to the service",
			args: []string{"alpha", "--query", query, "--yes", "--org-id", "11111111-1111-1111-1111-111111111111"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkAssignParams) (apptags.BulkAssignResult, cenclierrors.CencliError) {
						require.True(t, p.OrgID.IsPresent())
						return bulkSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "json output renders the operation payload",
			args: []string{"alpha", "--query", query, "--yes", "--output-format", "json"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitOnly(ctrl, "pending")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"status": "pending"`)
				require.Contains(t, stdout, `"type": "bulk_create"`)
			},
		},
		{
			name: "--wait polls the submitted operation and renders the final status",
			args: []string{"alpha", "--query", query, "--yes", "--wait"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						// The wait must follow the operation the submit just created.
						require.Equal(t, testOperationID, p.OperationID)
						require.True(t, p.Timeout.IsPresent())
						return apptags.GetOperationResult{
							Meta: okMeta(), Operation: finishedOperation("succeeded"),
						}, nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "succeeded")
				// Waiting to the end replaces the hint with the outcome.
				require.NotContains(t, stderr, "Track with")
			},
		},
		{
			name: "--wait ending at the asset limit warns but succeeds",
			args: []string{"alpha", "--query", query, "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitAndWait(ctrl, "limit_reached")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "asset limit")
				require.Contains(t, stderr, "few minutes")
			},
		},
		{
			name: "--wait ending failed exits non-zero",
			args: []string{"alpha", "--query", query, "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitAndWait(ctrl, "failed")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				// The payload still renders; only the exit code reports the outcome.
				require.Contains(t, stdout, "failed")
			},
		},
		{
			name: "--wait ending cancelled exits non-zero",
			args: []string{"alpha", "--query", query, "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitAndWait(ctrl, "cancelled")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, stdout, "cancelled")
			},
		},
		{
			name: "interrupting the wait keeps the job and prints how to follow it",
			args: []string{"alpha", "--query", query, "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
					apptags.GetOperationResult{}, cenclierrors.NewInterruptedError())
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, stderr, "continues server-side")
				require.Contains(t, stderr, "Track with: censys tags operations get alpha "+testOperationID)
			},
		},
		{
			name: "a wait that times out still points at the running job",
			args: []string{"alpha", "--query", query, "--yes", "--wait", "--timeout", "5s"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
					apptags.GetOperationResult{},
					apptags.NewOperationWaitTimeoutError(testOperationID, "running", 5*time.Second))
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, stderr, "Track with: censys tags operations get alpha "+testOperationID)
			},
		},
		{
			name: "--timeout 0 waits without a limit",
			args: []string{"alpha", "--query", query, "--yes", "--wait", "--timeout", "0"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(bulkSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						// Zero means unbounded, not "give up before the first poll".
						require.False(t, p.Timeout.IsPresent())
						return apptags.GetOperationResult{
							Meta: okMeta(), Operation: finishedOperation("succeeded"),
						}, nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a negative --timeout is rejected",
			args: []string{"alpha", "--query", query, "--yes", "--wait", "--timeout", "-5m"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout must not be negative")
			},
		},
		{
			name:  "--quiet suppresses the hint and the index-lag note",
			args:  []string{"alpha", "--query", query, "--yes"},
			seams: assignSeams{quiet: true},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitOnly(ctrl, "pending")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Empty(t, stderr)
				// The operation itself is the result, so it still renders.
				require.Contains(t, stdout, testOperationID)
			},
		},
		{
			name: "a tag name needing quoting is safe to paste back",
			args: []string{"my tag", "--query", query, "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkSubmitOnly(ctrl, "pending")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, `operations get "my tag" `+testOperationID)
			},
		},
		{
			name: "--max-assets 0 is passed through as no explicit cap",
			args: []string{"alpha", "--query", query, "--max-assets", "0", "--yes"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkAssignParams) (apptags.BulkAssignResult, cenclierrors.CencliError) {
						require.True(t, p.MaxAssets.IsPresent())
						require.Equal(t, int64(0), p.MaxAssets.MustGet())
						return bulkSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a failed submit reports the error and nothing to track",
			args: []string{"alpha", "--query", query, "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkAssign(gomock.Any(), gomock.Any()).Return(
					apptags.BulkAssignResult{}, cenclierrors.NewCencliError(errors.New("Permission denied")))
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.NotContains(t, stderr, "Track with")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Bulk always confirms, so default to an accepted prompt on a TTY.
			seams := tc.seams
			if seams.stdinIsTTY == nil {
				seams.stdinIsTTY = alwaysTTY()
			}
			if seams.confirm == nil {
				seams.confirm = func(context.Context, string) (bool, error) { return true, nil }
			}

			stdout, stderr, err := runAssignCommandWithSeams(t, tc.service(t, ctrl), seams, tc.args, nil)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

// TestTagsAssignCommand_BulkConfirmationMessage pins what the prompt tells the
// user before they approve a job that could tag a very large number of assets.
func TestTagsAssignCommand_BulkConfirmationMessage(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "without --max-assets the plan limit applies",
			args:     []string{"alpha", "--query", "host.services.port: 22"},
			contains: []string{`"alpha"`, "host.services.port: 22", "your plan's tag asset limit"},
		},
		{
			name:     "with --max-assets the cap is spelled out",
			args:     []string{"alpha", "--query", "host.services.port: 22", "--max-assets", "250"},
			contains: []string{"at most 250 asset(s)"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var prompt string
			seams := assignSeams{
				stdinIsTTY: alwaysTTY(),
				confirm: func(_ context.Context, message string) (bool, error) {
					prompt = message
					// Declining keeps the test off the submit path.
					return false, nil
				},
			}

			_, _, err := runAssignCommandWithSeams(t, bulkNoCallService(ctrl), seams, tc.args, nil)
			require.NoError(t, err)
			for _, want := range tc.contains {
				require.Contains(t, prompt, want)
			}
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
