package scan

// Succeeded reports whether a completed scan produced results: at least one
// task ended scanned or completed.
func (s TrackedScan) Succeeded() bool {
	for _, t := range s.Tasks {
		if t.Status == TaskStatusScanned || t.Status == TaskStatusCompleted {
			return true
		}
	}
	return false
}
