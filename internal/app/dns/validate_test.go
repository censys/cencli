package dns

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRecordTypes(t *testing.T) {
	tests := []struct {
		name        string
		recordTypes []string
		isIP        bool
		wantErr     string
	}{
		{
			name:        "success - no record types is valid for a name lookup",
			recordTypes: nil,
			isIP:        false,
		},
		{
			name:        "success - a name-only record type is valid for a name lookup",
			recordTypes: []string{"mx"},
			isIP:        false,
		},
		{
			name:        "success - an ip record type is valid for an ip lookup",
			recordTypes: []string{"a"},
			isIP:        true,
		},
		{
			name:        "error - a name-only record type is rejected for an ip lookup",
			recordTypes: []string{"MX"},
			isIP:        true,
			wantErr:     "invalid record type 'MX'",
		},
		{
			name:        "error - an unknown record type is rejected",
			recordTypes: []string{"CNAME"},
			isIP:        false,
			wantErr:     "invalid record type 'CNAME'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordTypes(tt.recordTypes, tt.isIP)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
