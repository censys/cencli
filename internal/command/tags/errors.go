package tags

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// nothingToUpdateError signals that `tags update` was invoked without any
// mutation flag, so there is nothing to change.
type nothingToUpdateError struct{}

func NewNothingToUpdateError() cenclierrors.CencliError { return &nothingToUpdateError{} }

func (e *nothingToUpdateError) Error() string {
	return "no fields to update; specify at least one of --name, --privacy, --description, --clear-description"
}

func (e *nothingToUpdateError) Title() string { return "Nothing To Update" }

func (e *nothingToUpdateError) ShouldPrintUsage() bool { return true }

// descriptionConflictError signals that --description and --clear-description
// were used together, which is contradictory.
type descriptionConflictError struct{}

func NewDescriptionConflictError() cenclierrors.CencliError { return &descriptionConflictError{} }

func (e *descriptionConflictError) Error() string {
	return "--description and --clear-description cannot be used together"
}

func (e *descriptionConflictError) Title() string { return "Conflicting Flags" }

func (e *descriptionConflictError) ShouldPrintUsage() bool { return true }

// confirmationRequiredError signals that a destructive command was invoked in a
// non-interactive terminal without --yes, so it cannot prompt for confirmation.
type confirmationRequiredError struct{}

func NewConfirmationRequiredError() cenclierrors.CencliError { return &confirmationRequiredError{} }

func (e *confirmationRequiredError) Error() string {
	return "confirmation required; re-run with --yes to skip the prompt in a non-interactive terminal"
}

func (e *confirmationRequiredError) Title() string { return "Confirmation Required" }

func (e *confirmationRequiredError) ShouldPrintUsage() bool { return true }
