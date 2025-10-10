package version

import "testing"

func TestBuildInfo(t *testing.T) {
	info := BuildInfo()
	if info.Go == "" || info.OS == "" || info.Arch == "" {
		t.Fatalf("expected Go/OS/Arch to be populated, got: %+v", info)
	}
}
