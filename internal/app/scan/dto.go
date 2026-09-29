package scan

import (
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

const (
	TaskStatusScanning  = string(components.TrackedScanTaskStatusScanning)
	TaskStatusScanned   = string(components.TrackedScanTaskStatusScanned)
	TaskStatusRejected  = string(components.TrackedScanTaskStatusRejected)
	TaskStatusTimedOut  = string(components.TrackedScanTaskStatusTimedOut)
	TaskStatusCompleted = string(components.TrackedScanTaskStatusCompleted)
	TaskStatusIgnored   = string(components.TrackedScanTaskStatusIgnored)
)

// RescanParams bundles inputs for requesting a live rescan of a web property.
type RescanParams struct {
	OrgID       identifiers.OrganizationID
	WebProperty assets.WebPropertyID
}

// GetParams bundles inputs for reading one tracked scan.
type GetParams struct {
	OrgID  identifiers.OrganizationID
	ScanID string
}

// WaitParams bundles inputs for polling a tracked scan until it completes. An
// absent Timeout polls until completion or context cancellation.
type WaitParams struct {
	OrgID   identifiers.OrganizationID
	ScanID  string
	Timeout mo.Option[time.Duration]
}

// TrackedScan is the domain representation of a tracked scan. Target keeps the
// API's type so every target kind survives in json and yaml output.
type TrackedScan struct {
	ID         string                            `json:"tracked_scan_id" yaml:"tracked_scan_id"`
	Completed  bool                              `json:"completed" yaml:"completed"`
	CreateTime *string                           `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	Target     *components.TrackedScanScanTarget `json:"target,omitempty" yaml:"target,omitempty"`
	Tasks      []Task                            `json:"tasks" yaml:"tasks"`
}

// Task is one unit of work within a tracked scan.
type Task struct {
	Description string  `json:"description" yaml:"description"`
	Status      string  `json:"status" yaml:"status"`
	UpdateTime  *string `json:"update_time,omitempty" yaml:"update_time,omitempty"`
}

// Result is the outcome of requesting, reading, or waiting on a scan. A failed
// scan is not an error here: the command layer owns the exit code. Scan is nil
// when no tracked scan was received.
type Result struct {
	Meta *responsemeta.ResponseMeta
	Scan *TrackedScan
}
