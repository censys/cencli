package scan

import (
	"fmt"
	"strings"

	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

// unsupportedScanAssetError signals an asset `scan rescan` cannot target.
// Hosts need the service protocol, which the CLI cannot know without a lookup.
type unsupportedScanAssetError struct {
	raw       string
	assetType assets.AssetType
}

func NewUnsupportedScanAssetError(raw string, assetType assets.AssetType) cenclierrors.CencliError {
	return &unsupportedScanAssetError{raw: raw, assetType: assetType}
}

func (e *unsupportedScanAssetError) Error() string {
	return fmt.Sprintf("scan rescan supports web properties (hostname:port) only; %q is a %s", e.raw, e.assetType)
}

func (e *unsupportedScanAssetError) Title() string { return "Unsupported Asset" }

func (e *unsupportedScanAssetError) ShouldPrintUsage() bool { return true }

// scanFailedError signals that a waited-on scan completed without producing
// results. The scan itself was still rendered; this only drives the exit code.
type scanFailedError struct {
	scan appscan.TrackedScan
}

func NewScanFailedError(s appscan.TrackedScan) cenclierrors.CencliError {
	return &scanFailedError{scan: s}
}

func (e *scanFailedError) Error() string {
	if len(e.scan.Tasks) == 0 {
		return fmt.Sprintf("scan %s completed without running any tasks", e.scan.ID)
	}
	statuses := make([]string, 0, len(e.scan.Tasks))
	for _, t := range e.scan.Tasks {
		statuses = append(statuses, taskStatusLabel(t.Status))
	}
	return fmt.Sprintf("scan %s completed without results (tasks: %s)", e.scan.ID, strings.Join(statuses, ", "))
}

func (e *scanFailedError) Title() string { return "Scan Failed" }

func (e *scanFailedError) ShouldPrintUsage() bool { return false }
