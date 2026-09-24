package collections

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	collectionsmocks "github.com/censys/cencli/gen/app/collections/mocks"
	appcollections "github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

func TestCollectionsGetCommand(t *testing.T) {
	build := func(c *command.Context) command.Command { return NewGetCommand(c) }
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - short output",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().GetCollection(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.GetParams) (appcollections.GetResult, cenclierrors.CencliError) {
						require.Equal(t, testCollectionID, params.CollectionID.String())
						return appcollections.GetResult{Meta: okMeta(), Collection: collection("alpha")}, nil
					},
				)
				return m
			},
			args: []string{testCollectionID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Collection")
				require.Contains(t, stdout, "Name:")
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "host.services.protocol=SSH")
				require.NotContains(t, stdout, "Reason:")
			},
		},
		{
			name: "status reason is shown when present",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				c := collection("alpha")
				reason := "not_enough_credits"
				c.Status = "paused"
				c.StatusReason = &reason
				m.EXPECT().GetCollection(gomock.Any(), gomock.Any()).Return(appcollections.GetResult{Meta: okMeta(), Collection: c}, nil)
				return m
			},
			args: []string{testCollectionID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Reason:")
				require.Contains(t, stdout, "not_enough_credits")
			},
		},
		{
			name: "invalid ID fails before any request",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				return collectionsmocks.NewMockCollectionsService(ctrl)
			},
			args: []string{"not-a-uuid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid collection ID")
			},
		},
		{
			name: "missing argument",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				return collectionsmocks.NewMockCollectionsService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "service error is returned",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().GetCollection(gomock.Any(), gomock.Any()).
					Return(appcollections.GetResult{}, cenclierrors.NewCencliError(errors.New("boom")))
				return m
			},
			args: []string{testCollectionID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
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
