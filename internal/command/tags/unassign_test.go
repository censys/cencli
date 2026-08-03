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
	// quiet stands in for the global --quiet flag, which lives on the real root
	// command and so is not registered when a subcommand is mounted alone.
	quiet bool
}

func runUnassignCommand(t *testing.T, svc apptags.Service, seams unassignSeams, args []string, stdin io.Reader) (stdout, stderr string, err error) {
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
			AssetID: asset,
			Err:     cenclierrors.NewCencliError(errors.New(msg)),
			// The service reduces every failure to a one-line Detail; the views
			// read that, not Err, so the fixture has to carry it too.
			Detail: msg,
		})
	}
	// Mirrors the service: a partial error only when something also succeeded.
	// A run where every asset failed is not partial, and the command turns it
	// into a non-zero exit itself.
	if len(res.Failures) > 0 && len(res.Unassigned) > 0 {
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
			// Explicit unassignment does not prompt, however many assets are named:
			// they were all typed by the caller. Only a bulk removal confirms.
			name:  "multiple assets do not prompt and are threaded in order",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1"},
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
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			// Nothing on stdin used to be fatal here; with no prompt to answer it
			// is now just a normal run.
			name:  "multiple assets run without a terminal",
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1"},
			seams: func(_ *testing.T) unassignSeams { return unassignSeams{stdinIsTTY: func() bool { return false }} },
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().Unassign(gomock.Any(), gomock.Any()).Return(
					unassignResult("alpha", []string{"8.8.8.8", "1.1.1.1"}, nil), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			// --yes has nothing to skip outside bulk mode, so it is rejected
			// rather than ignored, like --wait and --timeout.
			name:    "--yes is rejected in explicit mode",
			args:    []string{"alpha", "8.8.8.8", "1.1.1.1", "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service { return unassignNoCallService(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--yes only applies to a bulk unassignment")
			},
		},
		{
			name:  "comma-separated positional assets are split",
			args:  []string{"alpha", "8.8.8.8,1.1.1.1"},
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
			name:  "stdin input via --input-file -",
			args:  []string{"alpha", "--input-file", "-"},
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
			args:  []string{"alpha", "8.8.8.8", "1.1.1.1"},
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

// deleteOperation is operation()'s bulk_delete twin. TotalCount stays zero
// because the API only sets it once a bulk delete completes.
func deleteOperation(status string) apptags.TagOperation {
	op := operation(testOperationID, status)
	op.Type = "bulk_delete"
	op.TotalCount = 0
	op.Query = nil
	return op
}

// finishedDeleteOperation is a bulk_delete job that has reached a terminal status.
func finishedDeleteOperation(status string) apptags.TagOperation {
	ended := time.Unix(60, 0).UTC()
	op := deleteOperation(status)
	op.EndedAt = &ended
	op.TotalCount = 100
	op.ProcessedCount = 100
	op.SuccessfulCount = 90
	return op
}

// bulkUnassignSubmitted is what the service returns for an accepted bulk removal.
func bulkUnassignSubmitted(status string) apptags.BulkUnassignResult {
	return apptags.BulkUnassignResult{Meta: okMeta(), Operation: deleteOperation(status)}
}

// bulkUnassignNoCallService asserts neither removal path runs, proving the input
// was rejected at the command boundary.
func bulkUnassignNoCallService(ctrl *gomock.Controller) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().Unassign(gomock.Any(), gomock.Any()).Times(0)
	return m
}

// bulkUnassignSubmitOnly expects a submit and no polling.
func bulkUnassignSubmitOnly(ctrl *gomock.Controller, status string) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted(status), nil)
	return m
}

// bulkUnassignSubmitAndWait expects a submit followed by polling that ends on the
// given status.
func bulkUnassignSubmitAndWait(ctrl *gomock.Controller, finalStatus string) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted("pending"), nil)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
		apptags.GetOperationResult{Meta: okMeta(), Operation: finishedDeleteOperation(finalStatus)}, nil)
	return m
}

// TestTagsUnassignCommand_AllAssetsFail mirrors the assign contract: when no
// asset succeeded the per-asset results still render and the exit is non-zero.
// Unassign reaches this easily, since an asset the tag was never on is a
// failure by design rather than a silent no-op.
func TestTagsUnassignCommand_AllAssetsFail(t *testing.T) {
	allFailed := func(ctrl *gomock.Controller) apptags.Service {
		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().Unassign(gomock.Any(), gomock.Any()).Return(
			unassignResult("alpha", nil, map[string]string{
				"8.8.8.8": "asset is not assigned to this tag",
				"1.1.1.1": "asset is not assigned to this tag",
			}), nil)
		return m
	}

	t.Run("short output lists every failed asset and exits non-zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		stdout, stderr, err := runUnassignCommand(t, allFailed(ctrl), noPromptSeams(t),
			[]string{"alpha", "8.8.8.8", "1.1.1.1"}, nil)

		require.Error(t, err)
		require.Equal(t, 1, formatter.ExitCode(err))
		require.Contains(t, stdout, "8.8.8.8")
		require.Contains(t, stdout, "1.1.1.1")
		require.Contains(t, stdout, "not assigned")
		require.Contains(t, err.Error(), "2 of 2 failed")
		require.NotContains(t, stderr, "few minutes")
	})

	t.Run("json output still emits the full array", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		stdout, _, err := runUnassignCommand(t, allFailed(ctrl), noPromptSeams(t),
			[]string{"alpha", "8.8.8.8", "1.1.1.1", "--output-format", "json"}, nil)

		require.Error(t, err)
		require.Contains(t, stdout, `"asset": "8.8.8.8"`)
		require.Contains(t, stdout, `"asset": "1.1.1.1"`)
		require.Contains(t, stdout, `"unassigned": false`)
	})
}

func TestTagsUnassignCommand_Bulk(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		stdin   io.Reader
		seams   unassignSeams
		service func(t *testing.T, ctrl *gomock.Controller) apptags.Service
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "--all with positional assets is a mode conflict",
			args: []string{"alpha", "8.8.8.8", "--all"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be combined with explicit assets")
			},
		},
		{
			name:  "--all with --input-file is a mode conflict",
			args:  []string{"alpha", "--input-file", "-", "--all"},
			stdin: bytes.NewBufferString("8.8.8.8\n"),
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be combined with explicit assets")
			},
		},
		{
			name: "a time filter with positional assets is a mode conflict",
			args: []string{"alpha", "8.8.8.8", "--created-before", "2026-01-01T00:00:00Z"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be combined with explicit assets")
			},
		},
		{
			// --all already means every assignment, so narrowing it is contradictory.
			name: "--all with --created-before is rejected",
			args: []string{"alpha", "--all", "--created-before", "2026-01-01T00:00:00Z"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--all unassigns every assignment")
			},
		},
		{
			name: "--all with --created-after is rejected",
			args: []string{"alpha", "--all", "--created-after", "2026-01-01T00:00:00Z"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--all unassigns every assignment")
			},
		},
		{
			// Caught in PreRun, so it fails without needing credentials; the
			// service keeps its own guard for callers that skip the command layer.
			name: "an inverted time window is rejected before the service",
			args: []string{
				"alpha",
				"--created-before", "2020-01-01T00:00:00Z",
				"--created-after", "2026-01-01T00:00:00Z",
				"--yes",
			},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "created-before must be after created-after")
			},
		},
		{
			name: "--wait without a bulk mode flag is rejected",
			args: []string{"alpha", "8.8.8.8", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--wait only applies to a bulk unassignment")
			},
		},
		{
			name: "--timeout without --wait is rejected",
			args: []string{"alpha", "--all", "--timeout", "5m"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout only applies while polling")
			},
		},
		{
			name:  "non-interactive without --yes refuses to submit",
			args:  []string{"alpha", "--all"},
			seams: unassignSeams{stdinIsTTY: func() bool { return false }},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "confirmation required")
			},
		},
		{
			name: "declining the prompt aborts without submitting",
			args: []string{"alpha", "--all"},
			seams: unassignSeams{
				stdinIsTTY: alwaysTTY(),
				confirm:    func(context.Context, string) (bool, error) { return false, nil },
			},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "Unassign aborted.")
			},
		},
		{
			name: "--all --yes submits an unfiltered removal and reports a track hint",
			args: []string{"alpha", "--all", "--yes"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkUnassignParams) (apptags.BulkUnassignResult, cenclierrors.CencliError) {
						require.Equal(t, "alpha", p.TagID.String())
						// --all must not smuggle a time filter into the request.
						require.False(t, p.CreatedBefore.IsPresent())
						require.False(t, p.CreatedAfter.IsPresent())
						return bulkUnassignSubmitted("pending"), nil
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
			name: "a time filter alone selects bulk mode and is threaded to the service",
			args: []string{"alpha", "--created-before", "2026-01-01T00:00:00Z", "--yes"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkUnassignParams) (apptags.BulkUnassignResult, cenclierrors.CencliError) {
						require.True(t, p.CreatedBefore.IsPresent())
						require.Equal(t,
							time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
							p.CreatedBefore.MustGet().UTC())
						require.False(t, p.CreatedAfter.IsPresent())
						return bulkUnassignSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "both time filters bound the window",
			args: []string{
				"alpha",
				"--created-after", "2026-01-01T00:00:00Z",
				"--created-before", "2026-06-01T00:00:00Z",
				"--yes",
			},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkUnassignParams) (apptags.BulkUnassignResult, cenclierrors.CencliError) {
						require.True(t, p.CreatedBefore.IsPresent())
						require.True(t, p.CreatedAfter.IsPresent())
						return bulkUnassignSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "org id is threaded to the service",
			args: []string{"alpha", "--all", "--yes", "--org-id", "11111111-1111-1111-1111-111111111111"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.BulkUnassignParams) (apptags.BulkUnassignResult, cenclierrors.CencliError) {
						require.True(t, p.OrgID.IsPresent())
						return bulkUnassignSubmitted("pending"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "json output renders the operation payload",
			args: []string{"alpha", "--all", "--yes", "--output-format", "json"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitOnly(ctrl, "pending")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"status": "pending"`)
				require.Contains(t, stdout, `"type": "bulk_delete"`)
			},
		},
		{
			name: "--wait polls the submitted operation and renders the final status",
			args: []string{"alpha", "--all", "--yes", "--wait"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						// The wait must follow the operation the submit just created.
						require.Equal(t, testOperationID, p.OperationID)
						require.True(t, p.Timeout.IsPresent())
						return apptags.GetOperationResult{
							Meta: okMeta(), Operation: finishedDeleteOperation("succeeded"),
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
			args: []string{"alpha", "--all", "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitAndWait(ctrl, "limit_reached")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "asset limit")
				require.Contains(t, stderr, "few minutes")
			},
		},
		{
			name: "--wait ending failed exits non-zero",
			args: []string{"alpha", "--all", "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitAndWait(ctrl, "failed")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				// The payload still renders; only the exit code reports the outcome.
				require.Contains(t, stdout, "failed")
			},
		},
		{
			name: "--wait ending cancelled exits non-zero",
			args: []string{"alpha", "--all", "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitAndWait(ctrl, "cancelled")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, stdout, "cancelled")
			},
		},
		{
			name: "interrupting the wait keeps the job and prints how to follow it",
			args: []string{"alpha", "--all", "--yes", "--wait"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted("pending"), nil)
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
			args: []string{"alpha", "--all", "--yes", "--wait", "--timeout", "5s"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted("pending"), nil)
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
			args: []string{"alpha", "--all", "--yes", "--wait", "--timeout", "0"},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(bulkUnassignSubmitted("pending"), nil)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						// Zero means unbounded, not "give up before the first poll".
						require.False(t, p.Timeout.IsPresent())
						return apptags.GetOperationResult{
							Meta: okMeta(), Operation: finishedDeleteOperation("succeeded"),
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
			args: []string{"alpha", "--all", "--yes", "--wait", "--timeout", "-5m"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout must not be negative")
			},
		},
		{
			name:  "--quiet suppresses the hint and the index-lag note",
			args:  []string{"alpha", "--all", "--yes"},
			seams: unassignSeams{quiet: true},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitOnly(ctrl, "pending")
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
			args: []string{"my tag", "--all", "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return bulkUnassignSubmitOnly(ctrl, "pending")
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, `operations get "my tag" `+testOperationID)
			},
		},
		{
			name: "a failed submit reports the error and nothing to track",
			args: []string{"alpha", "--all", "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().BulkUnassign(gomock.Any(), gomock.Any()).Return(
					apptags.BulkUnassignResult{}, cenclierrors.NewCencliError(errors.New("Permission denied")))
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

			stdout, stderr, err := runUnassignCommand(t, tc.service(t, ctrl), seams, tc.args, tc.stdin)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

// TestTagsUnassignCommand_BulkConfirmationMessage pins what the prompt tells the
// user before they approve a removal they cannot undo. The difference between
// wiping a tag and trimming a window has to be visible in the prompt itself.
func TestTagsUnassignCommand_BulkConfirmationMessage(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "--all names every asset",
			args:     []string{"alpha", "--all"},
			contains: []string{`"alpha"`, "ALL assigned assets"},
		},
		{
			name:     "created-before names the upper bound",
			args:     []string{"alpha", "--created-before", "2026-01-01T00:00:00Z"},
			contains: []string{"created before", "2026-01-01"},
		},
		{
			name:     "created-after names the lower bound",
			args:     []string{"alpha", "--created-after", "2026-01-01T00:00:00Z"},
			contains: []string{"created after", "2026-01-01"},
		},
		{
			name: "both filters name the window",
			args: []string{
				"alpha",
				"--created-after", "2026-01-01T00:00:00Z",
				"--created-before", "2026-06-01T00:00:00Z",
			},
			contains: []string{"created between", "2026-01-01", "2026-06-01"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var prompt string
			seams := unassignSeams{
				stdinIsTTY: alwaysTTY(),
				confirm: func(_ context.Context, message string) (bool, error) {
					prompt = message
					// Declining keeps the test off the submit path.
					return false, nil
				},
			}

			_, _, err := runUnassignCommand(t, bulkUnassignNoCallService(ctrl), seams, tc.args, nil)
			require.NoError(t, err)
			for _, want := range tc.contains {
				require.Contains(t, prompt, want)
			}
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

	_, _, err := runUnassignCommand(t, m, noPromptSeams(t), []string{"alpha", "--input-file", file}, nil)
	require.NoError(t, err)
}
