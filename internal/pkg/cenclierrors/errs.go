package cenclierrors

import (
	"context"
	"errors"
	"strings"
)

type CencliError interface {
	// Title is the canonical identifier for the error.
	// Must be short and concise, and not depend on context.
	// Should not produce styled output.
	Title() string
	// Error is the underlying error detail.
	// Should not produce styled output.
	Error() string
	// ShouldPrintUsage indicates whether the error should print usage
	// information for the offending command when this error occurs.
	ShouldPrintUsage() bool
}

var _ error = CencliError(nil)

type cencliError struct {
	err error
}

func NewCencliError(err error) CencliError {
	if err == nil {
		return nil
	}
	// If already a CencliError, return it directly to avoid double-wrapping
	var ce CencliError
	if errors.As(err, &ce) {
		return ce
	}
	return &cencliError{err: err}
}

func (e *cencliError) Error() string {
	return e.err.Error()
}

func (e *cencliError) Unwrap() error {
	return e.err
}

func (e *cencliError) Title() string {
	return "Unknown Error"
}

func (e *cencliError) ShouldPrintUsage() bool {
	return false
}

type PartialError interface {
	CencliError
}

type partialError struct {
	err CencliError
}

// ToPartialError wraps a CencliError in a PartialError.
// If err is nil, it returns nil.
func ToPartialError(err CencliError) PartialError {
	if err == nil {
		return nil
	}
	return &partialError{err: err}
}

func (e *partialError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.err.Error())
	sb.WriteString("\n\nsome data was successfully retrieved before this error occurred")
	return sb.String()
}

func (e *partialError) Title() string {
	return e.err.Title() + " (partial data)"
}

func (e *partialError) ShouldPrintUsage() bool {
	return e.err.ShouldPrintUsage()
}

func (e *partialError) Unwrap() error {
	return e.err
}

// NewUsageError creates a CencliError for command usage errors.
// This should be used for errors like invalid flags, missing arguments, etc.
// These errors will trigger usage information to be printed.
func NewUsageError(err error) CencliError {
	if err == nil {
		return nil
	}
	return &usageError{err: err}
}

type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Title() string {
	return "Usage Error"
}

func (e *usageError) ShouldPrintUsage() bool {
	return true
}

func (e *usageError) Unwrap() error {
	return e.err
}

// NewInterruptedError creates a CencliError for interrupted operations.
// This should used exclusively for context.Canceled errors.
func NewInterruptedError() CencliError {
	return &interruptedError{}
}

type interruptedError struct{}

func (e *interruptedError) Error() string {
	return "the operation's context was cancelled before it completed"
}

func (e *interruptedError) Title() string {
	return "Interrupted"
}

func (e *interruptedError) ShouldPrintUsage() bool {
	return false
}

func (e *interruptedError) Unwrap() error {
	return context.Canceled
}

// NewDeadlineExceededError creates a CencliError for deadline exceeded errors.
// This should used exclusively for context.DeadlineExceeded errors.
func NewDeadlineExceededError() CencliError {
	return &deadlineExceededError{}
}

type deadlineExceededError struct{}

func (e *deadlineExceededError) Error() string {
	return "the operation timed out before it could be completed"
}

func (e *deadlineExceededError) Title() string {
	return "Timeout"
}

func (e *deadlineExceededError) ShouldPrintUsage() bool {
	return false
}

func (e *deadlineExceededError) Unwrap() error {
	return context.DeadlineExceeded
}

// ParseContextError parses a context error into a CencliError.
// This should only be called on errors returned from ctx.Err().
func ParseContextError(err error) CencliError {
	switch {
	case errors.Is(err, context.Canceled):
		return NewInterruptedError()
	case errors.Is(err, context.DeadlineExceeded):
		return NewDeadlineExceededError()
	default:
		return NewCencliError(err)
	}
}

type unwrappableCencliError interface {
	CencliError
	Unwrap() error
}

// IsDeadlineExceeded checks if an error is due to a deadline exceeded error.
func IsDeadlineExceeded(err error) bool {
	if err == nil {
		return false
	}

	var domainError unwrappableCencliError
	if errors.As(err, &domainError) {
		return errors.Is(domainError.Unwrap(), context.DeadlineExceeded)
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// IsInterrupted checks if an error is due to interruption (signal or context cancellation).
func IsInterrupted(err error) bool {
	if err == nil {
		return false
	}
	var domainError unwrappableCencliError
	if errors.As(err, &domainError) {
		return errors.Is(domainError.Unwrap(), context.Canceled)
	}
	return errors.Is(err, context.Canceled)
}

type noOrgIDError struct{}

func (e *noOrgIDError) Error() string {
	return "no organization ID available. Pass --org-id, or run 'censys config org-id add' to store a default"
}

func (e *noOrgIDError) Title() string {
	return "No Organization ID"
}

func (e *noOrgIDError) ShouldPrintUsage() bool {
	return true
}

func NewNoOrgIDError() CencliError {
	return &noOrgIDError{}
}

type organizationRequiredError struct {
	cmdPath string
	scope   string
}

func (e *organizationRequiredError) Error() string {
	return e.cmdPath + " uses an organization API, which only organization-scoped credentials can access. " +
		"Your current log in is scoped to " + e.scope + ". " +
		"To access an organization, authenticate to it. " +
		"(run 'censys auth logout' then 'censys auth login' and scope access to that organization)."
}

func (e *organizationRequiredError) Title() string { return "Organization Required" }

func (e *organizationRequiredError) ShouldPrintUsage() bool { return false }

// NewOrganizationRequiredError is returned when an organization-scoped command
// runs with a credential locked to the user's free account. cmdPath is the
// invoked command (e.g. "censys enrich") and scope is a clause describing the
// credential, e.g. "your OAuth login is scoped to your free account".
func NewOrganizationRequiredError(cmdPath, scope string) CencliError {
	return &organizationRequiredError{cmdPath: cmdPath, scope: scope}
}

type freeAccountRequiredError struct {
	cmdPath     string
	scope       string
	alternative string
}

func (e *freeAccountRequiredError) Error() string {
	msg := e.cmdPath + " uses a free-account API, which only free-account credentials can access. " +
		"Your current log in is scoped to " + e.scope + ". "
	if e.alternative != "" {
		// e.g. "To view your organization's credits run 'censys org credits'"
		msg += e.alternative + " or to read your free user account, authenticate to that."
	} else {
		msg += "To read your free user account, authenticate to that."
	}
	return msg + " (run 'censys auth logout' then 'censys auth login' and scope access to your free account)."
}

func (e *freeAccountRequiredError) Title() string { return "Free User Account Required" }

func (e *freeAccountRequiredError) ShouldPrintUsage() bool { return false }

// NewFreeAccountRequiredError is returned when a free-account-only command runs
// with a credential locked to an organization. cmdPath is the invoked command,
// scope describes the credential, and alternative is an optional sentence
// pointing at the organization equivalent of the command.
func NewFreeAccountRequiredError(cmdPath, scope, alternative string) CencliError {
	return &freeAccountRequiredError{cmdPath: cmdPath, scope: scope, alternative: alternative}
}

type orgIDNotApplicableError struct{ scope string }

func (e *orgIDNotApplicableError) Error() string {
	return "--org-id only applies to personal access tokens, the one credential that is not " +
		"organization-scoped. Your current log in is scoped to " + e.scope + ", so the organization " +
		"cannot be chosen per request. To target a different organization, authenticate to it " +
		"(run 'censys auth logout' then 'censys auth login' and scope access to that organization), " +
		"or switch to a personal access token with 'censys config auth activate <id>'."
}

func (e *orgIDNotApplicableError) Title() string {
	return "Organization ID Not Applicable"
}

func (e *orgIDNotApplicableError) ShouldPrintUsage() bool {
	return true
}

// NewOrgIDNotApplicableError is returned when --org-id is supplied while the
// active credential carries its own organization binding, so the flag (a
// personal-access-token concept) cannot apply. scope is a clause describing what
// the credential is bound to, e.g. "Your OAuth login is scoped to organization
// [Censys]".
func NewOrgIDNotApplicableError(scope string) CencliError {
	return &orgIDNotApplicableError{scope: scope}
}
