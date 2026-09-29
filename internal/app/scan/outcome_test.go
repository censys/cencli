package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackedScan_Succeeded(t *testing.T) {
	testCases := []struct {
		name     string
		statuses []string
		expected bool
	}{
		{name: "no tasks", expected: false},
		{name: "one completed", statuses: []string{TaskStatusCompleted}, expected: true},
		{name: "one scanned", statuses: []string{TaskStatusScanned}, expected: true},
		{name: "completed alongside ignored", statuses: []string{TaskStatusIgnored, TaskStatusCompleted}, expected: true},
		{name: "all rejected", statuses: []string{TaskStatusRejected, TaskStatusRejected}, expected: false},
		{name: "timed out and ignored", statuses: []string{TaskStatusTimedOut, TaskStatusIgnored}, expected: false},
		{name: "unknown status", statuses: []string{""}, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scan := TrackedScan{}
			for _, s := range tc.statuses {
				scan.Tasks = append(scan.Tasks, Task{Status: s})
			}
			assert.Equal(t, tc.expected, scan.Succeeded())
		})
	}
}
