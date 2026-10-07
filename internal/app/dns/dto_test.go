package dns

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/censys/censys-sdk-go/models/components"
)

func TestRecordJSON(t *testing.T) {
	testCases := []struct {
		name   string
		record any
		want   map[string]any
	}{
		{
			name: "success - name record has input beside the SDK fields",
			record: &NameRecord{
				Input:               "censys.com",
				DNSResolutionRecord: &components.DNSResolutionRecord{RecordType: components.DNSResolutionRecordRecordTypeA, IP: strPtr("104.18.10.84"), FirstSeen: from, LastSeen: to},
			},
			want: map[string]any{
				"input":       "censys.com",
				"record_type": "A",
				"ip":          "104.18.10.84",
				"first_seen":  "2026-09-21T00:00:00Z",
				"last_seen":   "2026-09-28T00:00:00Z",
			},
		},
		{
			name: "success - name range record has input beside the SDK fields",
			record: &NameRangeRecord{
				Input:                    "censys.com",
				DNSResolutionRangeRecord: &components.DNSResolutionRangeRecord{RecordType: components.DNSResolutionRangeRecordRecordTypeTxt, Value: strPtr("v=spf1"), FirstObserved: from, LastObserved: to},
			},
			want: map[string]any{
				"input":          "censys.com",
				"record_type":    "TXT",
				"value":          "v=spf1",
				"first_observed": "2026-09-21T00:00:00Z",
				"last_observed":  "2026-09-28T00:00:00Z",
			},
		},
		{
			name: "success - ip record has input beside the SDK fields",
			record: &IPRecord{
				Input:                 "104.18.10.84",
				DNSIPResolutionRecord: &components.DNSIPResolutionRecord{Domain: "censys.com", RecordType: components.DNSIPResolutionRecordRecordTypeA, FirstSeen: from, LastSeen: to},
			},
			want: map[string]any{
				"input":       "104.18.10.84",
				"domain":      "censys.com",
				"record_type": "A",
				"first_seen":  "2026-09-21T00:00:00Z",
				"last_seen":   "2026-09-28T00:00:00Z",
			},
		},
		{
			name: "success - ip range record has input beside the SDK fields",
			record: &IPRangeRecord{
				Input:                      "104.18.10.84",
				DNSIPResolutionRangeRecord: &components.DNSIPResolutionRangeRecord{Domain: "censys.com", RecordType: components.DNSIPResolutionRangeRecordRecordTypeAaaa, FirstSeen: from, LastSeen: to},
			},
			want: map[string]any{
				"input":       "104.18.10.84",
				"domain":      "censys.com",
				"record_type": "AAAA",
				"first_seen":  "2026-09-21T00:00:00Z",
				"last_seen":   "2026-09-28T00:00:00Z",
			},
		},
		{
			name:   "success - a record with no SDK record still has input",
			record: &NameRecord{Input: "censys.com"},
			want:   map[string]any{"input": "censys.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.record)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(data, &got))
			require.Equal(t, tc.want, got)
		})
	}
}
