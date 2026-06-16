package enrich

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type noHostsError struct{}

// NewNoHostsError indicates no host IPs were provided.
func NewNoHostsError() cenclierrors.CencliError { return &noHostsError{} }

func (e *noHostsError) Error() string {
	return "no host IPs provided. Pass one or more IPs as arguments or via --input-file"
}

func (e *noHostsError) Title() string { return "No Hosts Provided" }

func (e *noHostsError) ShouldPrintUsage() bool { return true }

type invalidHostError struct {
	raw string
}

// NewInvalidHostError indicates an input value was not a valid host IP.
func NewInvalidHostError(raw string) cenclierrors.CencliError {
	return &invalidHostError{raw: raw}
}

func (e *invalidHostError) Error() string {
	return fmt.Sprintf("%q is not a valid host IP", e.raw)
}

func (e *invalidHostError) Title() string { return "Invalid Host" }

func (e *invalidHostError) ShouldPrintUsage() bool { return true }
