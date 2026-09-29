package dns

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// domainFlagMisuseError signals that --domain was given for a lookup that is
// not an IP timeline, where the API does not support the filter.
type domainFlagMisuseError struct{}

// NewDomainFlagMisuseError creates the error shown when --domain is set without an IP lookup and --timeline.
func NewDomainFlagMisuseError() cenclierrors.CencliError { return &domainFlagMisuseError{} }

func (e *domainFlagMisuseError) Error() string {
	return "--domain applies only to an IP lookup with --timeline"
}

func (e *domainFlagMisuseError) Title() string { return "Conflicting Flags" }

func (e *domainFlagMisuseError) ShouldPrintUsage() bool { return true }
