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

func TestCollectionsCreateCommand(t *testing.T) {
	build := func(c *command.Context) command.Command { return NewCreateCommand(c) }
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - short output with search hint",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.CreateParams) (appcollections.CreateResult, cenclierrors.CencliError) {
						require.Equal(t, "alpha", params.Name)
						require.Equal(t, "host.services.protocol=SSH", params.Query)
						require.True(t, params.Description.IsAbsent())
						return appcollections.CreateResult{Meta: okMeta(), Collection: collection("alpha")}, nil
					},
				)
				return m
			},
			args: []string{"alpha", "--query", "host.services.protocol=SSH"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Collection Created")
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stderr, "censys search --collection-id "+testCollectionID)
			},
		},
		{
			name: "description is passed through",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.CreateParams) (appcollections.CreateResult, cenclierrors.CencliError) {
						require.Equal(t, mo.Some("SSH hosts"), params.Description)
						return appcollections.CreateResult{Meta: okMeta(), Collection: collection("alpha")}, nil
					},
				)
				return m
			},
			args: []string{"alpha", "--query", "q", "--description", "SSH hosts"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "missing --query",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				return collectionsmocks.NewMockCollectionsService(ctrl)
			},
			args: []string{"alpha"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "query")
			},
		},
		{
			name: "json output",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
					Return(appcollections.CreateResult{Meta: okMeta(), Collection: collection("alpha")}, nil)
				return m
			},
			args: []string{"alpha", "--query", "q", "--output-format", "json"},
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
