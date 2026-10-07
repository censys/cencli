package dns

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/app/pagination"
)

func TestIsAccessDeniedError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "success - the access-denied error",
			err:  NewAccessDeniedError(),
			want: true,
		},
		{
			name: "success - another error",
			err:  pagination.NewInvalidParamsError("x"),
			want: false,
		},
		{
			name: "success - no error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsAccessDeniedError(tc.err))
		})
	}
}
