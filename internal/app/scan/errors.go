package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// rescanUncertainError signals a rescan failure that does not prove the API
// rejected the request, so the scan may have been accepted and charged.
type rescanUncertainError struct {
	cause cenclierrors.CencliError
}

// NewRescanUncertainError wraps a rescan failure whose outcome is unknown.
func NewRescanUncertainError(cause cenclierrors.CencliError) cenclierrors.CencliError {
	return &rescanUncertainError{cause: cause}
}

func (e *rescanUncertainError) Error() string {
	return fmt.Sprintf(
		"the rescan request failed after it may have been accepted, so 10 credits may have been charged (%s). "+
			"Check `censys org credits` before trying again", e.cause.Error())
}

func (e *rescanUncertainError) Title() string { return "Rescan Outcome Unknown" }

func (e *rescanUncertainError) ShouldPrintUsage() bool { return false }

func (e *rescanUncertainError) Unwrap() error { return e.cause }

// rescanNotEnabledError signals that the API refused the rescan because web
// property rescans are not enabled for the organization. No scan was created.
type rescanNotEnabledError struct{}

// NewRescanNotEnabledError creates a rescan-not-enabled error.
func NewRescanNotEnabledError() cenclierrors.CencliError {
	return &rescanNotEnabledError{}
}

func (e *rescanNotEnabledError) Error() string {
	return "web property rescans are not enabled for your organization, so no scan was started"
}

func (e *rescanNotEnabledError) Title() string { return "Feature Not Enabled" }

func (e *rescanNotEnabledError) ShouldPrintUsage() bool { return false }

// scanWaitTimeoutError signals that --wait gave up before the scan completed.
// It unwraps to context.DeadlineExceeded so the command exits 124.
type scanWaitTimeoutError struct {
	scanID  string
	timeout time.Duration
}

// NewScanWaitTimeoutError creates a wait-timeout error for a scan.
func NewScanWaitTimeoutError(scanID string, timeout time.Duration) cenclierrors.CencliError {
	return &scanWaitTimeoutError{scanID: scanID, timeout: timeout}
}

func (e *scanWaitTimeoutError) Error() string {
	return fmt.Sprintf("timed out after %s waiting for scan %s; it continues server-side", e.timeout, e.scanID)
}

func (e *scanWaitTimeoutError) Title() string { return "Timeout" }

func (e *scanWaitTimeoutError) ShouldPrintUsage() bool { return false }

func (e *scanWaitTimeoutError) Unwrap() error { return context.DeadlineExceeded }

// invalidScanIDError signals a scan ID that is not a UUID.
type invalidScanIDError struct {
	provided string
}

// NewInvalidScanIDError creates an invalid-scan-ID error.
func NewInvalidScanIDError(provided string) cenclierrors.CencliError {
	return &invalidScanIDError{provided: provided}
}

func (e *invalidScanIDError) Error() string {
	if e.provided == "" {
		return "a scan ID is required"
	}
	return fmt.Sprintf("scan ID %q is not a valid UUID", e.provided)
}

func (e *invalidScanIDError) Title() string { return "Invalid Scan ID" }

func (e *invalidScanIDError) ShouldPrintUsage() bool { return true }

// emptyScanError signals a successful response that carried no scan.
type emptyScanError struct{}

func (e *emptyScanError) Error() string { return "the API returned no tracked scan" }

func (e *emptyScanError) Title() string { return "Empty Response" }

func (e *emptyScanError) ShouldPrintUsage() bool { return false }

// RequireScanID rejects a scan ID the endpoint could not accept, before any
// request is spent on it.
func RequireScanID(scanID string) (string, cenclierrors.CencliError) {
	if _, err := uuid.Parse(scanID); err != nil {
		return "", NewInvalidScanIDError(scanID)
	}
	return scanID, nil
}
