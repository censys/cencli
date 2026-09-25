package collections

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	collectionsmocks "github.com/censys/cencli/gen/app/collections/mocks"
	appcollections "github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

func TestCollectionsUpdateCommand(t *testing.T) {
	build := func(c *command.Context) command.Command { return NewUpdateCommand(c) }
	expectParams := func(want appcollections.UpdateParams) func(ctrl *gomock.Controller) appcollections.Service {
		return func(ctrl *gomock.Controller) appcollections.Service {
			m := collectionsmocks.NewMockCollectionsService(ctrl)
			m.EXPECT().UpdateCollection(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params appcollections.UpdateParams) (appcollections.UpdateResult, cenclierrors.CencliError) {
					require.Equal(t, testCollectionID, params.CollectionID.String())
					require.Equal(t, want.Name, params.Name)
					require.Equal(t, want.Query, params.Query)
					require.Equal(t, want.Description, params.Description)
					return appcollections.UpdateResult{Meta: okMeta(), Collection: collection("alpha")}, nil
				},
			)
			return m
		}
	}
	noCall := func(ctrl *gomock.Controller) appcollections.Service {
		return collectionsmocks.NewMockCollectionsService(ctrl)
	}
	wantSuccess := func(t *testing.T, stdout, stderr string, err error) {
		require.NoError(t, err)
		require.Contains(t, stdout, "Collection Updated")
	}
	wantErr := func(substr string) func(t *testing.T, stdout, stderr string, err error) {
		return func(t *testing.T, stdout, stderr string, err error) {
			require.Error(t, err)
			require.Contains(t, err.Error(), substr)
		}
	}

	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name:    "name only",
			service: expectParams(appcollections.UpdateParams{Name: mo.Some("renamed")}),
			args:    []string{testCollectionID, "--name", "renamed"},
			assert:  wantSuccess,
		},
		{
			name:    "query only",
			service: expectParams(appcollections.UpdateParams{Query: mo.Some("host.ip=1.1.1.1")}),
			args:    []string{testCollectionID, "--query", "host.ip=1.1.1.1"},
			assert:  wantSuccess,
		},
		{
			name:    "description only",
			service: expectParams(appcollections.UpdateParams{Description: mo.Some("d")}),
			args:    []string{testCollectionID, "--description", "d"},
			assert:  wantSuccess,
		},
		{
			name:    "clear description",
			service: expectParams(appcollections.UpdateParams{Description: mo.Some("")}),
			args:    []string{testCollectionID, "--clear-description"},
			assert:  wantSuccess,
		},
		{
			name:    "no flags",
			service: noCall,
			args:    []string{testCollectionID},
			assert:  wantErr("no fields to update"),
		},
		{
			name:    "blank values count as no flags",
			service: noCall,
			args:    []string{testCollectionID, "--name", "  "},
			assert:  wantErr("no fields to update"),
		},
		{
			name:    "description conflict",
			service: noCall,
			args:    []string{testCollectionID, "--description", "d", "--clear-description"},
			assert:  wantErr("cannot be used together"),
		},
		{
			name:    "invalid ID",
			service: noCall,
			args:    []string{"nope", "--name", "x"},
			assert:  wantErr("is not a valid UUID"),
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().UpdateCollection(gomock.Any(), gomock.Any()).Return(
					appcollections.UpdateResult{Meta: okMeta(), Collection: collection("alpha")}, nil,
				)
				return m
			},
			args: []string{testCollectionID, "--name", "x", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"name": "alpha"`)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stdout, stderr, err := runCommand(t, tc.service(ctrl), build, tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
