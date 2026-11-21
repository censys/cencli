package collections

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// nothingToUpdateError signals that `collections update` was invoked without
// any mutation flag, so there is nothing to change.
type nothingToUpdateError struct{}

func NewNothingToUpdateError() cenclierrors.CencliError { return &nothingToUpdateError{} }

func (e *nothingToUpdateError) Error() string {
	return "no fields to update; specify at least one of --name, --query, --description, --clear-description"
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

// blankFlagError signals that a flag was given with a blank value. A blank
// --name or --query would otherwise be silently treated as "not set".
type blankFlagError struct {
	flag string
}

func NewBlankFlagError(flag string) cenclierrors.CencliError { return &blankFlagError{flag: flag} }

func (e *blankFlagError) Error() string {
	return fmt.Sprintf("--%s cannot be blank", e.flag)
}

func (e *blankFlagError) Title() string { return "Invalid Flag Value" }

func (e *blankFlagError) ShouldPrintUsage() bool { return true }
