package censeye

import "github.com/censys/cencli/internal/pkg/cenclierrors"

type CompileRulesError interface {
	cenclierrors.CencliError
}

type compileRulesError struct{ err error }

var _ CompileRulesError = &compileRulesError{}

func newCompileRulesError(err error) CompileRulesError {
	return &compileRulesError{err: err}
}

func (e *compileRulesError) Error() string { return e.err.Error() }

func (e *compileRulesError) Title() string { return "Compile Rules Error" }

func (e *compileRulesError) ShouldPrintUsage() bool { return true }
