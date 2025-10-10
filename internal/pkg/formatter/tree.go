package formatter

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/ui/tree"
)

func PrintTree(v any, colored bool) cenclierrors.CencliError {
	data, err := dataToJSON(v)
	if err != nil {
		return newTreeError(err)
	}
	err = tree.Run(data)
	if err != nil {
		return newTreeError(err)
	}
	return nil
}

type TreeError interface {
	cenclierrors.CencliError
}

type treeError struct {
	err error
}

func newTreeError(err error) TreeError {
	return &treeError{err: err}
}

func (e *treeError) Error() string {
	return e.err.Error()
}

func (e *treeError) Title() string {
	return "Tree Error"
}

func (e *treeError) ShouldPrintUsage() bool {
	return false
}
